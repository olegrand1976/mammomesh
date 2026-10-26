// Package store regroupe les accès à PostgreSQL. L'API, l'orchestrateur et le serveur MCP
// passent tous par lui : les règles d'accès aux données sont écrites une seule fois.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
)

// ErrNotFound : objet absent ou invisible pour l'utilisateur.
var ErrNotFound = errors.New("introuvable")

// InvalidError : demande mal formée, renvoyée telle quelle à l'utilisateur (HTTP 400).
type InvalidError string

func (e InvalidError) Error() string { return string(e) }

// Store encapsule le pool de connexions.
type Store struct {
	Pool   *pgxpool.Pool
	Pepper string
}

// New crée un Store.
func New(pool *pgxpool.Pool, pepper string) *Store { return &Store{Pool: pool, Pepper: pepper} }

// ---------- Comptes et sessions ----------

// CreateUser ajoute un compte (commande d'administration, pas d'inscription libre).
func (s *Store) CreateUser(ctx context.Context, username, displayName, password, role string) (int64, error) {
	if !auth.ValidRole(role) {
		return 0, InvalidError("rôle inconnu")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO users (username, display_name, password_hash, role)
		VALUES ($1, $2, $3, $4) RETURNING id`, username, displayName, hash, role).Scan(&id)
	return id, err
}

// CreateDefaultAdmin crée le premier administrateur avec le mot de passe par défaut, à changer.
func (s *Store) CreateDefaultAdmin(ctx context.Context, username string) (int64, error) {
	hash, err := auth.HashDefaultPassword()
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, must_change_password)
		VALUES ($1, $2, 'admin', TRUE) RETURNING id`, username, hash).Scan(&id)
	return id, err
}

// ErrSamePassword : le nouveau mot de passe reprend l'ancien.
var ErrSamePassword error = InvalidError("le nouveau mot de passe doit différer de l'actuel")

// ChangePassword vérifie le mot de passe actuel, enregistre le nouveau (12 caractères au moins)
// et ferme les autres sessions du compte ; keepToken est la session en cours.
func (s *Store) ChangePassword(ctx context.Context, userID int64, current, next, keepToken string) error {
	var hash string
	if err := s.Pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1 AND active`, userID).Scan(&hash); err != nil {
		return err
	}
	if ok, err := auth.CheckPassword(hash, current); err != nil || !ok {
		return ErrNotFound
	}
	if next == current {
		return ErrSamePassword
	}
	newHash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, must_change_password=FALSE WHERE id=$1`, userID, newHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, userID, auth.HashToken(keepToken, s.Pepper)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CountUsers sert à créer le premier administrateur.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

// Authenticate vérifie identifiant et mot de passe.
func (s *Store) Authenticate(ctx context.Context, username, password string) (*auth.User, error) {
	var u auth.User
	var hash string
	err := s.Pool.QueryRow(ctx, `SELECT id, username, display_name, role, must_change_password, password_hash FROM users
		WHERE username=$1 AND active`, username).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.MustChangePassword, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		// Calcul factice : même durée de réponse qu'un compte existant.
		_, _ = auth.CheckPassword("$argon2id$v=19$m=65536,t=2,p=2$c2VsLWZhY3RpY2U$ZmFjdGljZQ", password)
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ok, err := auth.CheckPassword(hash, password)
	if err != nil || !ok {
		return nil, ErrNotFound
	}
	return &u, nil
}

// CreateSession ouvre une session et renvoie le jeton à placer dans le cookie.
func (s *Store) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	token, err := auth.NewToken("")
	if err != nil {
		return "", err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		auth.HashToken(token, s.Pepper), userID, time.Now().Add(ttl))
	return token, err
}

// SessionUser renvoie l'utilisateur d'une session valide et prolonge son activité.
func (s *Store) SessionUser(ctx context.Context, token string, idle time.Duration) (*auth.User, error) {
	var u auth.User
	err := s.Pool.QueryRow(ctx, `UPDATE sessions se SET last_seen_at = now()
		FROM users u
		WHERE se.token_hash=$1 AND se.user_id=u.id AND u.active
		  AND se.expires_at > now() AND se.last_seen_at > now() - $2::interval
		RETURNING u.id, u.username, u.display_name, u.role, u.must_change_password`,
		auth.HashToken(token, s.Pepper), idle.String()).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.MustChangePassword)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// DeleteSession ferme une session (déconnexion).
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, auth.HashToken(token, s.Pepper))
	return err
}

// ---------- Jetons MCP ----------

// MCPToken décrit un jeton (jamais sa valeur).
type MCPToken struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Scope      string     `json:"scope"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// CreateMCPToken crée un jeton ; la valeur n'est montrée qu'une fois.
func (s *Store) CreateMCPToken(ctx context.Context, userID int64, name, scope string, ttl time.Duration) (string, MCPToken, error) {
	token, err := auth.NewToken("mmk_")
	if err != nil {
		return "", MCPToken{}, err
	}
	var t MCPToken
	err = s.Pool.QueryRow(ctx, `INSERT INTO mcp_tokens (user_id, name, token_hash, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, name, scope, created_at, expires_at`,
		userID, name, auth.HashToken(token, s.Pepper), scope, time.Now().Add(ttl)).
		Scan(&t.ID, &t.Name, &t.Scope, &t.CreatedAt, &t.ExpiresAt)
	return token, t, err
}

// ListMCPTokens liste les jetons d'un utilisateur.
func (s *Store) ListMCPTokens(ctx context.Context, userID int64) ([]MCPToken, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, scope, created_at, expires_at, last_used_at, revoked_at
		FROM mcp_tokens WHERE user_id=$1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (MCPToken, error) {
		var t MCPToken
		err := r.Scan(&t.ID, &t.Name, &t.Scope, &t.CreatedAt, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt)
		return t, err
	})
}

// RevokeMCPToken révoque un jeton de l'utilisateur.
func (s *Store) RevokeMCPToken(ctx context.Context, userID, id int64) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE mcp_tokens SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// MCPPrincipal : utilisateur et portée d'un jeton MCP valide.
type MCPPrincipal struct {
	User    auth.User
	Scope   string
	TokenID int64
}

// ResolveMCPToken vérifie un jeton présenté en en-tête Authorization.
func (s *Store) ResolveMCPToken(ctx context.Context, token string) (*MCPPrincipal, error) {
	var p MCPPrincipal
	err := s.Pool.QueryRow(ctx, `UPDATE mcp_tokens t SET last_used_at=now()
		FROM users u
		WHERE t.token_hash=$1 AND t.user_id=u.id AND u.active AND t.revoked_at IS NULL AND t.expires_at > now()
		RETURNING t.id, t.scope, u.id, u.username, u.display_name, u.role`,
		auth.HashToken(token, s.Pepper)).Scan(&p.TokenID, &p.Scope, &p.User.ID, &p.User.Username, &p.User.DisplayName, &p.User.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

// ---------- Journal d'audit ----------

// Audit écrit un geste dans audit_log.
func (s *Store) Audit(ctx context.Context, userID int64, source, action string, details any) {
	b, _ := json.Marshal(details)
	var uid any
	if userID > 0 {
		uid = userID
	}
	_, _ = s.Pool.Exec(ctx, `INSERT INTO audit_log (user_id, source, action, details) VALUES ($1, $2, $3, $4)`, uid, source, action, b)
}

// ---------- Campagnes ----------

// Campaign est la vue d'une campagne.
type Campaign struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Lineage         string          `json:"lineage"`
	State           string          `json:"state"`
	Generation      int             `json:"generation"`
	MaxGenerations  int             `json:"max_generations"`
	BudgetTotal     float64         `json:"budget_total"`
	BudgetUsed      float64         `json:"budget_used"`
	BestSensitivity *float64        `json:"best_sensitivity"`
	Stagnation      int             `json:"stagnation"`
	ConfigHash      string          `json:"config_hash"`
	Config          campaign.Config `json:"config"`
	CreatedAt       time.Time       `json:"created_at"`
	StartedAt       *time.Time      `json:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

const campaignCols = `id, name, lineage, state, generation, max_generations, budget_total, budget_used,
	best_sensitivity, stagnation, config_hash, config, created_at, started_at, finished_at, updated_at`

func scanCampaign(r pgx.Row) (Campaign, error) {
	var c Campaign
	var cfg []byte
	err := r.Scan(&c.ID, &c.Name, &c.Lineage, &c.State, &c.Generation, &c.MaxGenerations, &c.BudgetTotal, &c.BudgetUsed,
		&c.BestSensitivity, &c.Stagnation, &c.ConfigHash, &cfg, &c.CreatedAt, &c.StartedAt, &c.FinishedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err == nil {
		err = json.Unmarshal(cfg, &c.Config)
	}
	return c, err
}

// ListCampaigns renvoie les campagnes, les plus récentes d'abord.
func (s *Store) ListCampaigns(ctx context.Context, limit int) ([]Campaign, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+campaignCols+` FROM campaigns ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Campaign, error) { return scanCampaign(r) })
}

// GetCampaign renvoie une campagne.
func (s *Store) GetCampaign(ctx context.Context, id int64) (Campaign, error) {
	return scanCampaign(s.Pool.QueryRow(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE id=$1`, id))
}

// CampaignsInState sert à l'orchestrateur.
func (s *Store) CampaignsInState(ctx context.Context, state string) ([]Campaign, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE state=$1 ORDER BY id`, state)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Campaign, error) { return scanCampaign(r) })
}

// ---------- Brouillons de configuration ----------

// Draft est un brouillon de configuration.
type Draft struct {
	ID                 int64                  `json:"id"`
	BaseCampaignID     *int64                 `json:"base_campaign_id"`
	Config             campaign.Config        `json:"config"`
	Diff               []campaign.Change      `json:"diff"`
	Source             string                 `json:"source"`
	Verification       *campaign.Verification `json:"verification"`
	VerifiedOK         bool                   `json:"verified_ok"`
	LaunchedCampaignID *int64                 `json:"launched_campaign_id"`
	CreatedAt          time.Time              `json:"created_at"`
}

// CreateDraft enregistre un brouillon, avec son diff par rapport à la base.
func (s *Store) CreateDraft(ctx context.Context, authorID int64, source string, baseID *int64, cfg campaign.Config) (Draft, error) {
	base := campaign.Default()
	if baseID != nil {
		c, err := s.GetCampaign(ctx, *baseID)
		if err != nil {
			return Draft{}, err
		}
		base = c.Config
	}
	diff := campaign.Diff(base, cfg)
	cb, _ := json.Marshal(cfg)
	db, _ := json.Marshal(diff)
	d := Draft{BaseCampaignID: baseID, Config: cfg, Diff: diff, Source: source}
	err := s.Pool.QueryRow(ctx, `INSERT INTO config_drafts (base_campaign_id, config, diff, author_id, source)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`, baseID, cb, db, authorID, source).Scan(&d.ID, &d.CreatedAt)
	return d, err
}

// GetDraft renvoie un brouillon.
func (s *Store) GetDraft(ctx context.Context, id int64) (Draft, error) {
	var d Draft
	var cfg, diff, checks []byte
	err := s.Pool.QueryRow(ctx, `SELECT id, base_campaign_id, config, diff, source, checks, verified_ok, launched_campaign_id, created_at
		FROM config_drafts WHERE id=$1`, id).Scan(&d.ID, &d.BaseCampaignID, &cfg, &diff, &d.Source, &checks, &d.VerifiedOK, &d.LaunchedCampaignID, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	_ = json.Unmarshal(cfg, &d.Config)
	_ = json.Unmarshal(diff, &d.Diff)
	if len(checks) > 0 {
		d.Verification = &campaign.Verification{}
		_ = json.Unmarshal(checks, d.Verification)
	}
	return d, nil
}

// VerifyDraft lance les contrôles avant lancement et les garde avec le brouillon.
func (s *Store) VerifyDraft(ctx context.Context, id int64) (Draft, error) {
	d, err := s.GetDraft(ctx, id)
	if err != nil {
		return d, err
	}
	v := campaign.Verify(d.Config)
	b, _ := json.Marshal(v)
	if _, err := s.Pool.Exec(ctx, `UPDATE config_drafts SET checks=$2, verified_ok=$3, verified_at=now() WHERE id=$1`, id, b, v.OK); err != nil {
		return d, err
	}
	d.Verification, d.VerifiedOK = &v, v.OK
	return d, nil
}

// ErrNotVerified : un brouillon ne se lance qu'après des contrôles réussis.
var ErrNotVerified = errors.New("brouillon non vérifié ou contrôles en échec")

// ErrAlreadyLaunched : un brouillon ne se lance qu'une fois.
var ErrAlreadyLaunched = errors.New("brouillon déjà lancé")

// LaunchDraft fige la configuration et crée la campagne, dans une transaction.
func (s *Store) LaunchDraft(ctx context.Context, id, userID int64) (Campaign, error) {
	var out Campaign
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var cfgB []byte
		var ok bool
		var launched *int64
		err := tx.QueryRow(ctx, `SELECT config, verified_ok, launched_campaign_id FROM config_drafts WHERE id=$1 FOR UPDATE`, id).
			Scan(&cfgB, &ok, &launched)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if launched != nil {
			return ErrAlreadyLaunched
		}
		if !ok {
			return ErrNotVerified
		}
		var cfg campaign.Config
		if err := json.Unmarshal(cfgB, &cfg); err != nil {
			return err
		}
		var cid int64
		err = tx.QueryRow(ctx, `INSERT INTO campaigns (name, lineage, config, config_hash, state, max_generations, budget_total, created_by, started_at)
			VALUES ($1, $2, $3, $4, 'en_cours', $5, $6, $7, now()) RETURNING id`,
			cfg.Projet.Nom, cfg.Projet.Lignee, cfgB, campaign.Hash(cfg), cfg.Evolution.GenerationsMax, cfg.Evolution.BudgetCoeurHeure, userID).Scan(&cid)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE config_drafts SET launched_campaign_id=$2 WHERE id=$1`, id, cid); err != nil {
			return err
		}
		if err := InsertEvent(ctx, tx, &cid, "campagne_lancee",
			"Campagne « "+cfg.Projet.Nom+" » lancée : lignée "+cfg.Projet.Lignee+", configuration "+campaign.Hash(cfg)[:8]+".",
			map[string]any{"config_hash": campaign.Hash(cfg)}); err != nil {
			return err
		}
		out, err = scanCampaign(tx.QueryRow(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE id=$1`, cid))
		return err
	})
	return out, err
}

// ErrBadTransition : réglage à chaud refusé dans cet état.
var ErrBadTransition = errors.New("changement d'état impossible depuis l'état actuel")

// Control applique un réglage à chaud : pause, reprise, arrêt, hausse du budget.
func (s *Store) Control(ctx context.Context, id int64, action string, budgetIncrease float64) (Campaign, error) {
	var out Campaign
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		c, err := scanCampaign(tx.QueryRow(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE id=$1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		var next, msg string
		switch action {
		case "pause":
			if c.State != "en_cours" {
				return ErrBadTransition
			}
			next, msg = "en_pause", "Campagne mise en pause."
		case "reprise":
			if c.State != "en_pause" {
				return ErrBadTransition
			}
			next, msg = "en_cours", "Campagne reprise."
		case "arret":
			if c.State != "en_cours" && c.State != "en_pause" {
				return ErrBadTransition
			}
			next, msg = "arretee", "Campagne arrêtée à la demande, génération "+itoa(c.Generation)+"."
		case "budget":
			if budgetIncrease <= 0 {
				return InvalidError("le budget ne peut qu'augmenter")
			}
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET budget_total=budget_total+$2, updated_at=now() WHERE id=$1`, id, budgetIncrease); err != nil {
				return err
			}
			if err := InsertEvent(ctx, tx, &id, "budget_augmente", "Budget augmenté de "+ftoa(budgetIncrease)+" cœur-heures.",
				map[string]any{"hausse": budgetIncrease}); err != nil {
				return err
			}
			out, err = scanCampaign(tx.QueryRow(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE id=$1`, id))
			return err
		default:
			return InvalidError("action inconnue : pause, reprise, arret ou budget")
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET state=$2, updated_at=now(),
			finished_at = CASE WHEN $2='arretee' THEN now() ELSE finished_at END WHERE id=$1`, id, next); err != nil {
			return err
		}
		if err := InsertEvent(ctx, tx, &id, "etat_change", msg, map[string]any{"avant": c.State, "apres": next}); err != nil {
			return err
		}
		out, err = scanCampaign(tx.QueryRow(ctx, `SELECT `+campaignCols+` FROM campaigns WHERE id=$1`, id))
		return err
	})
	return out, err
}

// ---------- Générations, individus, front ----------

// Generation : agrégats d'une génération.
type Generation struct {
	Number            int       `json:"number"`
	Kind              string    `json:"kind"`
	StartedAt         time.Time `json:"started_at"`
	FinishedAt        time.Time `json:"finished_at"`
	Population        int       `json:"population"`
	Feasible          int       `json:"feasible"`
	BestSensitivity   *float64  `json:"best_sensitivity"` // nil : aucun individu réalisable
	MedianSensitivity float64   `json:"median_sensitivity"`
	BestCostMs        float64   `json:"best_cost_ms"`
	MedianCostMs      float64   `json:"median_cost_ms"`
	EvalGap           *float64  `json:"eval_gap"`
}

// ListGenerations renvoie les agrégats après la génération « after ».
func (s *Store) ListGenerations(ctx context.Context, campaignID int64, after, limit int) ([]Generation, error) {
	rows, err := s.Pool.Query(ctx, `SELECT number, kind, started_at, finished_at, population, feasible, best_sensitivity,
		median_sensitivity, best_cost_ms, median_cost_ms, eval_gap FROM generations
		WHERE campaign_id=$1 AND number > $2 ORDER BY number LIMIT $3`, campaignID, after, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Generation, error) {
		var g Generation
		err := r.Scan(&g.Number, &g.Kind, &g.StartedAt, &g.FinishedAt, &g.Population, &g.Feasible, &g.BestSensitivity,
			&g.MedianSensitivity, &g.BestCostMs, &g.MedianCostMs, &g.EvalGap)
		return g, err
	})
}

// Individual : un couple CNN + CephaloMesh évalué.
type Individual struct {
	ID              int64          `json:"id"`
	CampaignID      int64          `json:"campaign_id"`
	Generation      int            `json:"generation"`
	Name            string         `json:"name"`
	ParentName      *string        `json:"parent_name"`
	Mutations       []string       `json:"mutations"`
	Genome          map[string]any `json:"genome"`
	Sensitivity     float64        `json:"sensitivity"`
	SensitivityLow  float64        `json:"sensitivity_low"`
	SensitivityHigh float64        `json:"sensitivity_high"`
	SensitivityEval float64        `json:"sensitivity_eval"`
	CostMs          float64        `json:"cost_ms"`
	Feasible        bool           `json:"feasible"`
	OnFront         bool           `json:"on_front"`
}

const individualCols = `id, campaign_id, generation, name, parent_name, mutations, genome, sensitivity,
	sensitivity_low, sensitivity_high, sensitivity_eval, cost_ms, feasible, on_front`

func scanIndividual(r pgx.Row) (Individual, error) {
	var i Individual
	var genome []byte
	err := r.Scan(&i.ID, &i.CampaignID, &i.Generation, &i.Name, &i.ParentName, &i.Mutations, &genome, &i.Sensitivity,
		&i.SensitivityLow, &i.SensitivityHigh, &i.SensitivityEval, &i.CostMs, &i.Feasible, &i.OnFront)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, ErrNotFound
	}
	if err == nil {
		err = json.Unmarshal(genome, &i.Genome)
	}
	return i, err
}

// Front renvoie le front d'une génération (la dernière si generation <= 0).
func (s *Store) Front(ctx context.Context, campaignID int64, generation int) (int, []Individual, error) {
	if generation <= 0 {
		if err := s.Pool.QueryRow(ctx, `SELECT COALESCE(max(number), 0) FROM generations WHERE campaign_id=$1`, campaignID).Scan(&generation); err != nil {
			return 0, nil, err
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+individualCols+` FROM individuals
		WHERE campaign_id=$1 AND generation=$2 AND on_front ORDER BY cost_ms`, campaignID, generation)
	if err != nil {
		return generation, nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Individual, error) { return scanIndividual(r) })
	return generation, list, err
}

// GetIndividual renvoie un individu par identifiant ou par nom.
func (s *Store) GetIndividual(ctx context.Context, id int64) (Individual, error) {
	return scanIndividual(s.Pool.QueryRow(ctx, `SELECT `+individualCols+` FROM individuals WHERE id=$1`, id))
}

// GetIndividualByName renvoie un individu par son nom dans une campagne.
func (s *Store) GetIndividualByName(ctx context.Context, campaignID int64, name string) (Individual, error) {
	return scanIndividual(s.Pool.QueryRow(ctx, `SELECT `+individualCols+` FROM individuals WHERE campaign_id=$1 AND name=$2`, campaignID, name))
}

// ---------- Événements ----------

// Event est une ligne du journal.
type Event struct {
	ID         int64          `json:"id"`
	CampaignID *int64         `json:"campaign_id"`
	Type       string         `json:"type"`
	Message    string         `json:"message"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  time.Time      `json:"created_at"`
}

// InsertEvent ajoute un événement ; le déclencheur SQL envoie l'avis temps réel au commit.
func InsertEvent(ctx context.Context, tx pgx.Tx, campaignID *int64, typ, message string, payload any) error {
	b, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `INSERT INTO events (campaign_id, type, message, payload) VALUES ($1, $2, $3, $4)`, campaignID, typ, message, b)
	return err
}

// ListEvents renvoie les événements après « afterID », dans l'ordre (reprise du flux, MCP).
func (s *Store) ListEvents(ctx context.Context, campaignID, afterID int64, limit int, newestFirst bool) ([]Event, error) {
	order := "ASC"
	if newestFirst {
		order = "DESC"
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, campaign_id, type, message, payload, created_at FROM events
		WHERE campaign_id=$1 AND id > $2 ORDER BY id `+order+` LIMIT $3`, campaignID, afterID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanEvent)
}

// GetEvent renvoie un événement.
func (s *Store) GetEvent(ctx context.Context, id int64) (Event, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, campaign_id, type, message, payload, created_at FROM events WHERE id=$1`, id)
	if err != nil {
		return Event{}, err
	}
	e, err := pgx.CollectExactlyOneRow(rows, scanEvent)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func scanEvent(r pgx.CollectableRow) (Event, error) {
	var e Event
	var payload []byte
	err := r.Scan(&e.ID, &e.CampaignID, &e.Type, &e.Message, &payload, &e.CreatedAt)
	if err == nil {
		_ = json.Unmarshal(payload, &e.Payload)
	}
	return e, err
}
