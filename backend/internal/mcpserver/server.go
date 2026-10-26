// Package mcpserver expose MammoMesh à Claude Code par le protocole MCP.
//
// C'est un adaptateur au-dessus du store : mêmes règles que l'interface, aucun SQL propre.
// Absents par construction : lecture du test caché ou d'un jeu externe, promotion, suppression,
// images, SQL libre. Les actions exigent la portée « proposition » et un rôle habilité.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// Version du serveur MCP annoncée aux clients.
const Version = "0.1.0"

const instructions = `Serveur MCP de MammoMesh, outil de recherche (aucun usage clinique).
Règles : E-val ne sert jamais à optimiser ; aucune conclusion ne se tire de la sélection ;
le test caché n'est pas accessible ; une variante de configuration passe toujours par
config_brouillon puis config_verifier avant campagne_lancer, qui demande une confirmation humaine.
Les réponses sont courtes par défaut : demander le détail outil par outil.`

// Handler renvoie le gestionnaire HTTP de /mcp (HTTP en flux, sans état).
// Le jeton est vérifié à chaque requête en amont ; le serveur construit pour un principal
// (jeton, utilisateur, rôle, portée) est réutilisé quelques minutes : sa construction coûte
// plusieurs millisecondes et plus d'un mégaoctet, à ne pas payer à chaque appel.
func Handler(st *store.Store) http.Handler {
	c := &serverCache{st: st, ttl: 5 * time.Minute, m: map[principalKey]cachedServer{}}
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		p, _ := PrincipalFrom(r.Context())
		return c.get(p)
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

type principalKey struct {
	token, user int64
	role, scope string
}

type cachedServer struct {
	s     *mcp.Server
	until time.Time
}

type serverCache struct {
	st  *store.Store
	ttl time.Duration
	mu  sync.Mutex
	m   map[principalKey]cachedServer
}

func (c *serverCache) get(p *store.MCPPrincipal) *mcp.Server {
	if p == nil {
		return NewServer(c.st, nil)
	}
	k := principalKey{p.TokenID, p.User.ID, p.User.Role, p.Scope}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok && now.Before(e.until) {
		return e.s
	}
	for k2, e := range c.m { // purge des entrées expirées : la table reste petite
		if now.After(e.until) {
			delete(c.m, k2)
		}
	}
	s := NewServer(c.st, p)
	c.m[k] = cachedServer{s: s, until: now.Add(c.ttl)}
	return s
}

type ctxKey struct{}

// WithPrincipal place le principal MCP dans le contexte.
func WithPrincipal(ctx context.Context, p *store.MCPPrincipal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFrom lit le principal MCP du contexte.
func PrincipalFrom(ctx context.Context) (*store.MCPPrincipal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*store.MCPPrincipal)
	return p, ok && p != nil
}

var errScope = errors.New("ce jeton est en lecture seule : créez un jeton « proposition » dans l'interface (page Jetons MCP)")
var errRole = errors.New("votre rôle ne permet pas ce geste (responsable scientifique ou administrateur requis)")

func ptr[T any](v T) *T { return &v }

// tool enregistre un outil et journalise chaque appel dans audit_log (source « mcp »).
func tool[In, Out any](s *mcp.Server, st *store.Store, p *store.MCPPrincipal, t *mcp.Tool, h func(context.Context, In) (Out, error)) {
	mcp.AddTool(s, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var uid int64
		if p != nil {
			uid = p.User.ID
		}
		out, err := h(ctx, in)
		details := map[string]any{"outil": t.Name, "arguments": req.Params.Arguments, "ok": err == nil}
		if err != nil {
			details["erreur"] = err.Error()
		}
		st.Audit(context.WithoutCancel(ctx), uid, "mcp", "mcp."+t.Name, details)
		return nil, out, err
	})
}

// NewServer construit le serveur MCP pour un principal donné.
func NewServer(st *store.Store, p *store.MCPPrincipal) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "mammomesh", Title: "MammoMesh", Version: Version},
		&mcp.ServerOptions{Instructions: instructions})
	if p == nil {
		return s
	}
	read := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(false)}
	propose := &mcp.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)}
	act := &mcp.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: ptr(false)}

	needPropose := func() error {
		if p.Scope != "proposition" {
			return errScope
		}
		if !auth.CanProposeConfig(p.User.Role) {
			return errRole
		}
		return nil
	}
	needAct := func() error {
		if p.Scope != "proposition" {
			return errScope
		}
		if !auth.CanRunCampaigns(p.User.Role) {
			return errRole
		}
		return nil
	}

	tool(s, st, p, &mcp.Tool{Name: "campagnes_lister", Title: "Lister les campagnes", Annotations: read,
		Description: "Campagnes, des plus récentes aux plus anciennes : état, génération, budget, meilleure sensibilité sur E-val."},
		func(ctx context.Context, in ListIn) (ListOut, error) {
			limit := clamp(in.Limite, 1, 50, 20)
			list, err := st.ListCampaigns(ctx, limit)
			if err != nil {
				return ListOut{}, err
			}
			out := ListOut{Campagnes: []CampaignSummary{}}
			for _, c := range list {
				out.Campagnes = append(out.Campagnes, summarize(c))
			}
			return out, nil
		})

	tool(s, st, p, &mcp.Tool{Name: "campagne_etat", Title: "État d'une campagne", Annotations: read,
		Description: "État d'une campagne, configuration figée et agrégats des dernières générations (5 par défaut, 50 au plus)."},
		func(ctx context.Context, in StateIn) (StateOut, error) {
			c, err := st.GetCampaign(ctx, in.CampagneID)
			if err != nil {
				return StateOut{}, notFound("campagne", in.CampagneID, err)
			}
			n := clamp(in.Generations, 1, 50, 5)
			gens, err := st.ListGenerations(ctx, c.ID, max(0, c.Generation-n), n)
			if err != nil {
				return StateOut{}, err
			}
			return StateOut{Campagne: summarize(c), Config: c.Config, Generations: gens}, nil
		})

	tool(s, st, p, &mcp.Tool{Name: "front_pareto", Title: "Front de Pareto", Annotations: read,
		Description: "Individus réalisables non dominés d'une génération (la dernière par défaut) : sensibilité sur E-ajust avec intervalle, sensibilité sur E-val, coût par examen. 50 au plus."},
		func(ctx context.Context, in FrontIn) (FrontOut, error) {
			if _, err := st.GetCampaign(ctx, in.CampagneID); err != nil {
				return FrontOut{}, notFound("campagne", in.CampagneID, err)
			}
			g, list, err := st.Front(ctx, in.CampagneID, in.Generation)
			if err != nil {
				return FrontOut{}, err
			}
			out := FrontOut{Generation: g, Individus: []FrontPoint{}}
			for i, ind := range list {
				if i == 50 {
					out.Tronque = true
					break
				}
				out.Individus = append(out.Individus, FrontPoint{ID: ind.ID, Nom: ind.Name, Sensibilite: ind.Sensitivity,
					IntervalleBas: ind.SensitivityLow, IntervalleHaut: ind.SensitivityHigh, SensibiliteEval: ind.SensitivityEval,
					CoutMs: ind.CostMs, Mutations: ind.Mutations})
			}
			return out, nil
		})

	tool(s, st, p, &mcp.Tool{Name: "individu_fiche", Title: "Fiche d'un individu", Annotations: read,
		Description: "Génome CephaloMesh, mesures et parent d'un individu, par identifiant ou par nom (avec la campagne)."},
		func(ctx context.Context, in IndividualIn) (store.Individual, error) {
			return findIndividual(ctx, st, in)
		})

	tool(s, st, p, &mcp.Tool{Name: "individus_comparer", Title: "Comparer deux individus", Annotations: read,
		Description: "Différences de génome et de mesures entre deux individus (identifiants)."},
		func(ctx context.Context, in CompareIn) (CompareOut, error) {
			a, err := st.GetIndividual(ctx, in.A)
			if err != nil {
				return CompareOut{}, notFound("individu", in.A, err)
			}
			b, err := st.GetIndividual(ctx, in.B)
			if err != nil {
				return CompareOut{}, notFound("individu", in.B, err)
			}
			return compare(a, b), nil
		})

	tool(s, st, p, &mcp.Tool{Name: "journal_lire", Title: "Lire le journal", Annotations: read,
		Description: "Événements en phrases d'une campagne, du plus récent au plus ancien (20 par défaut, 50 au plus), filtrables par type."},
		func(ctx context.Context, in JournalIn) (JournalOut, error) {
			limit := clamp(in.Limite, 1, 50, 20)
			events, err := st.ListEvents(ctx, in.CampagneID, in.ApresID, 500, true)
			if err != nil {
				return JournalOut{}, err
			}
			out := JournalOut{Evenements: []JournalLine{}}
			for _, e := range events {
				if len(in.Types) > 0 && !contains(in.Types, e.Type) {
					continue
				}
				out.Evenements = append(out.Evenements, JournalLine{ID: e.ID, Type: e.Type, Phrase: e.Message, Date: e.CreatedAt.Format("2006-01-02 15:04:05")})
				if len(out.Evenements) == limit {
					break
				}
			}
			return out, nil
		})

	tool(s, st, p, &mcp.Tool{Name: "config_brouillon", Title: "Créer un brouillon de configuration", Annotations: propose,
		Description: "Crée un brouillon à partir d'une campagne (ou des valeurs par défaut) et de réglages en notation pointée, par exemple {\"evolution.population\": 150}. depart_campagne_id fait repartir du front d'une campagne finie (dernière génération par défaut, ou depart_generation). Ne lance rien."},
		func(ctx context.Context, in DraftIn) (store.Draft, error) {
			if err := needPropose(); err != nil {
				return store.Draft{}, err
			}
			base := campaign.Default()
			var baseID *int64
			if in.BaseCampagneID > 0 {
				c, err := st.GetCampaign(ctx, in.BaseCampagneID)
				if err != nil {
					return store.Draft{}, notFound("campagne", in.BaseCampagneID, err)
				}
				base, baseID = c.Config, &c.ID
				base.Depart = nil // le départ ne s'hérite pas : il se demande explicitement
			}
			cfg, err := campaign.Apply(base, in.Reglages)
			if err != nil {
				return store.Draft{}, err
			}
			if in.DepartCampagneID > 0 {
				src, err := st.GetCampaign(ctx, in.DepartCampagneID)
				if err != nil {
					return store.Draft{}, notFound("campagne", in.DepartCampagneID, err)
				}
				g := in.DepartGeneration
				if g <= 0 {
					g = src.Generation // par défaut : la génération finale
				}
				cfg.Depart = &campaign.Depart{CampagneID: src.ID, Generation: g}
				if baseID == nil {
					baseID = &src.ID
				}
			}
			return st.CreateDraft(ctx, p.User.ID, "mcp", baseID, cfg)
		})

	tool(s, st, p, &mcp.Tool{Name: "config_verifier", Title: "Vérifier un brouillon", Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: true, OpenWorldHint: ptr(false)},
		Description: "Lance les contrôles avant lancement d'un brouillon et l'estimation de durée et de cœur-heures, sans rien lancer."},
		func(ctx context.Context, in DraftRef) (store.Draft, error) {
			if err := needPropose(); err != nil {
				return store.Draft{}, err
			}
			d, err := st.VerifyDraft(ctx, in.BrouillonID)
			if err != nil {
				return d, notFound("brouillon", in.BrouillonID, err)
			}
			return d, nil
		})

	tool(s, st, p, &mcp.Tool{Name: "campagne_lancer", Title: "Lancer une campagne", Annotations: act,
		Description: "Lance un brouillon vérifié : la configuration est figée et la campagne démarre. Geste sensible, à confirmer par l'utilisateur."},
		func(ctx context.Context, in DraftRef) (CampaignSummary, error) {
			if err := needAct(); err != nil {
				return CampaignSummary{}, err
			}
			c, err := st.LaunchDraft(ctx, in.BrouillonID, p.User.ID)
			if err != nil {
				return CampaignSummary{}, notFound("brouillon", in.BrouillonID, err)
			}
			return summarize(c), nil
		})

	tool(s, st, p, &mcp.Tool{Name: "campagne_reglage", Title: "Réglage à chaud", Annotations: act,
		Description: "Seuls réglages admis pendant une campagne : pause, reprise, arret, budget (hausse en cœur-heures). Geste sensible, à confirmer par l'utilisateur."},
		func(ctx context.Context, in ControlIn) (CampaignSummary, error) {
			if err := needAct(); err != nil {
				return CampaignSummary{}, err
			}
			c, err := st.Control(ctx, in.CampagneID, in.Action, in.HausseBudget)
			if err != nil {
				return CampaignSummary{}, notFound("campagne", in.CampagneID, err)
			}
			return summarize(c), nil
		})

	s.AddResourceTemplate(&mcp.ResourceTemplate{Name: "rapport-campagne", Title: "Rapport de campagne",
		URITemplate: "mammomesh://campagne/{id}/rapport", MIMEType: "text/markdown",
		Description: "Résumé d'une campagne en Markdown : état, meilleur individu, dernières générations, journal récent."},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			uri := req.Params.URI
			idStr := strings.TrimSuffix(strings.TrimPrefix(uri, "mammomesh://campagne/"), "/rapport")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return nil, mcp.ResourceNotFoundError(uri)
			}
			text, err := report(ctx, st, id)
			if err != nil {
				return nil, mcp.ResourceNotFoundError(uri)
			}
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: text}}}, nil
		})

	s.AddPrompt(&mcp.Prompt{Name: "analyser-generation", Title: "Analyser une génération",
		Description: "Analyse la dernière génération d'une campagne : progrès, compromis du front, signes de sur-adaptation.",
		Arguments:   []*mcp.PromptArgument{{Name: "campagne_id", Description: "identifiant de la campagne", Required: true}}},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			id := req.Params.Arguments["campagne_id"]
			return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: fmt.Sprintf(
				"Analyse la campagne MammoMesh %s : appelle campagne_etat, front_pareto et journal_lire. "+
					"Dis si la sensibilité sur E-val progresse encore, quels compromis offre le front, si l'écart E-ajust − E-val grandit "+
					"(signe de sur-adaptation) et quelles mutations reviennent chez les meilleurs. Ne conclus rien sur la sélection ni sur le test caché.", id)}}}}, nil
		})

	s.AddPrompt(&mcp.Prompt{Name: "proposer-variante", Title: "Proposer une variante de configuration",
		Description: "Propose une variante de configuration à partir d'une campagne, sans la lancer.",
		Arguments:   []*mcp.PromptArgument{{Name: "campagne_id", Description: "campagne de départ", Required: true}}},
		func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			id := req.Params.Arguments["campagne_id"]
			return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: fmt.Sprintf(
				"À partir de la campagne MammoMesh %s, propose une variante de configuration justifiée par ses résultats. "+
					"Crée-la avec config_brouillon, vérifie-la avec config_verifier et présente le diff et l'estimation. Ne la lance pas.", id)}}}}, nil
		})

	return s
}

func notFound(what string, id int64, err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("%s %d introuvable", what, id)
	}
	return err
}

func clamp(v, lo, hi, def int) int {
	if v == 0 {
		return def
	}
	return min(max(v, lo), hi)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func summarize(c store.Campaign) CampaignSummary {
	return CampaignSummary{ID: c.ID, Nom: c.Name, Lignee: c.Lineage, Etat: c.State, Generation: c.Generation,
		GenerationsMax: c.MaxGenerations, BudgetUtilise: round2(c.BudgetUsed), BudgetTotal: c.BudgetTotal,
		MeilleureSensibiliteEval: c.BestSensitivity, Stagnation: c.Stagnation, Configuration: c.ConfigHash[:8]}
}

func opt(f *float64) string {
	if f == nil {
		return "—"
	}
	return fmt.Sprintf("%.3f", *f)
}

func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

func findIndividual(ctx context.Context, st *store.Store, in IndividualIn) (store.Individual, error) {
	if in.IndividuID > 0 {
		i, err := st.GetIndividual(ctx, in.IndividuID)
		return i, notFound("individu", in.IndividuID, err)
	}
	if in.CampagneID > 0 && in.Nom != "" {
		i, err := st.GetIndividualByName(ctx, in.CampagneID, in.Nom)
		if errors.Is(err, store.ErrNotFound) {
			return i, fmt.Errorf("individu %q introuvable dans la campagne %d", in.Nom, in.CampagneID)
		}
		return i, err
	}
	return store.Individual{}, errors.New("donner individu_id, ou campagne_id et nom")
}

func compare(a, b store.Individual) CompareOut {
	out := CompareOut{A: a.Name, B: b.Name, Genome: []GeneDiff{},
		Mesures: map[string][2]float64{
			"sensibilite":      {a.Sensitivity, b.Sensitivity},
			"sensibilite_eval": {a.SensitivityEval, b.SensitivityEval},
			"cout_ms":          {a.CostMs, b.CostMs},
		}}
	keys := map[string]bool{}
	for k := range a.Genome {
		keys[k] = true
	}
	for k := range b.Genome {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		va, vb := fmt.Sprint(a.Genome[k]), fmt.Sprint(b.Genome[k])
		if va != vb {
			out.Genome = append(out.Genome, GeneDiff{Gene: k, A: a.Genome[k], B: b.Genome[k]})
		}
	}
	return out
}

func report(ctx context.Context, st *store.Store, id int64) (string, error) {
	c, err := st.GetCampaign(ctx, id)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Campagne %d · %s\n\n", c.ID, c.Name)
	fmt.Fprintf(&b, "- État : %s ; génération %d sur %d au plus\n", c.State, c.Generation, c.MaxGenerations)
	fmt.Fprintf(&b, "- Budget : %.2f sur %.2f cœur-heures\n", c.BudgetUsed, c.BudgetTotal)
	fmt.Fprintf(&b, "- Configuration : %s\n", c.ConfigHash[:8])
	if c.BestSensitivity != nil {
		fmt.Fprintf(&b, "- Meilleure sensibilité sur E-val : %.3f\n", *c.BestSensitivity)
	}
	gens, _ := st.ListGenerations(ctx, id, max(0, c.Generation-5), 5)
	if len(gens) > 0 {
		b.WriteString("\n| Génération | Réalisables | Meilleure (E-val) | Médiane | Écart E-ajust − E-val |\n|---|---|---|---|---|\n")
		for _, g := range gens {
			fmt.Fprintf(&b, "| %d | %d / %d | %s | %.3f | %s |\n", g.Number, g.Feasible, g.Population, opt(g.BestSensitivity), g.MedianSensitivity, opt(g.EvalGap))
		}
	}
	events, _ := st.ListEvents(ctx, id, 0, 10, true)
	if len(events) > 0 {
		b.WriteString("\n## Journal récent\n\n")
		for _, e := range events {
			fmt.Fprintf(&b, "- %s\n", e.Message)
		}
	}
	return b.String(), nil
}
