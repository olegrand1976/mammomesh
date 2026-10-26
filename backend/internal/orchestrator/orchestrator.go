// Package orchestrator fait avancer les campagnes sans intervention : il est le seul à décider du travail,
// reprend seul après un redémarrage et n'a aucun droit de promotion.
package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// lockID garantit un seul orchestrateur actif (verrou consultatif PostgreSQL).
const lockID = 727002

// ProgressChannel porte les messages de progression, fréquents et non journalisés.
const ProgressChannel = "mammomesh_progress"

// Orchestrator pilote les campagnes en cours.
type Orchestrator struct {
	st   *store.Store
	log  *slog.Logger
	poll time.Duration

	mu      sync.Mutex
	running map[int64]context.CancelFunc
}

// New crée l'orchestrateur.
func New(st *store.Store, log *slog.Logger) *Orchestrator {
	return &Orchestrator{st: st, log: log, poll: time.Second, running: map[int64]context.CancelFunc{}}
}

// Run prend le verrou puis lance une boucle par campagne en cours, jusqu'à l'arrêt du contexte.
func (o *Orchestrator) Run(ctx context.Context) error {
	conn, err := o.st.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	for {
		var ok bool
		if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&ok); err != nil {
			return err
		}
		if ok {
			break
		}
		o.log.Info("un autre orchestrateur est actif ; attente du verrou")
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}
	o.log.Info("orchestrateur actif")
	if err := o.announceResumes(ctx); err != nil {
		o.log.Error("annonce des reprises", "err", err)
	}

	var wg sync.WaitGroup
	ticker := time.NewTicker(o.poll)
	defer ticker.Stop()
	for {
		if err := o.schedule(ctx, &wg); err != nil && !errors.Is(err, context.Canceled) {
			o.log.Error("planification", "err", err)
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case <-ticker.C:
		}
	}
}

func (o *Orchestrator) announceResumes(ctx context.Context) error {
	list, err := o.st.CampaignsInState(ctx, "en_cours")
	if err != nil {
		return err
	}
	for _, c := range list {
		if c.Generation == 0 {
			continue
		}
		id := c.ID
		err := pgx.BeginFunc(ctx, o.st.Pool, func(tx pgx.Tx) error {
			return store.InsertEvent(ctx, tx, &id, "reprise_apres_redemarrage",
				fmt.Sprintf("Reprise après redémarrage : génération %d reprise, rien n'est recalculé.", c.Generation+1),
				map[string]any{"generation": c.Generation + 1})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Orchestrator) schedule(ctx context.Context, wg *sync.WaitGroup) error {
	list, err := o.st.CampaignsInState(ctx, "en_cours")
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, c := range list {
		if _, ok := o.running[c.ID]; ok {
			continue
		}
		rctx, cancel := context.WithCancel(ctx)
		o.running[c.ID] = cancel
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			defer func() {
				o.mu.Lock()
				delete(o.running, id)
				o.mu.Unlock()
				cancel()
			}()
			if err := o.runCampaign(rctx, id); err != nil && !errors.Is(err, context.Canceled) {
				o.log.Error("campagne en échec", "campagne", id, "err", err)
				o.fail(context.Background(), id, err)
			}
		}(c.ID)
	}
	return nil
}

// runCampaign enchaîne les générations tant que la campagne reste en cours.
func (o *Orchestrator) runCampaign(ctx context.Context, id int64) error {
	for {
		c, err := o.st.GetCampaign(ctx, id)
		if err != nil {
			return err
		}
		if c.State != "en_cours" {
			return nil
		}
		if reason := stopReason(c); reason != "" {
			return o.finish(ctx, c, reason)
		}
		g := c.Generation + 1
		started := time.Now()
		lots := c.Config.Simulation.LotsParGeneration
		perLot := campaign.CostPerGeneration(c.Config) / float64(lots)
		for lot := 1; lot <= lots; lot++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(c.Config.Simulation.DureeLotMs) * time.Millisecond):
			}
			state, err := o.state(ctx, id)
			if err != nil {
				return err
			}
			if state != "en_cours" {
				return nil // pause ou arrêt : la génération en cours sera refaite à l'identique
			}
			o.progress(ctx, id, g, lot, lots, c.BudgetUsed+perLot*float64(lot), c.BudgetTotal)
		}
		prev, err := o.parents(ctx, c)
		if err != nil {
			return err
		}
		sim := simulate(c.Config, id, g, prev)
		if err := o.persist(ctx, c, g, started, sim); err != nil {
			return err
		}
	}
}

func stopReason(c store.Campaign) string {
	switch {
	case c.Generation >= c.MaxGenerations:
		return fmt.Sprintf("plafond de %d générations atteint", c.MaxGenerations)
	case c.BudgetUsed >= c.BudgetTotal:
		return fmt.Sprintf("budget de %s cœur-heures atteint", fr(c.BudgetTotal, 1))
	case c.Stagnation >= c.Config.Evolution.ArretStagnation:
		return fmt.Sprintf("E-val stable depuis %d générations", c.Stagnation)
	}
	return ""
}

func (o *Orchestrator) state(ctx context.Context, id int64) (string, error) {
	var s string
	err := o.st.Pool.QueryRow(ctx, `SELECT state FROM campaigns WHERE id=$1`, id).Scan(&s)
	return s, err
}

// parents : noms du front précédent ; à la génération 1, le front de départ s'il y en a un.
func (o *Orchestrator) parents(ctx context.Context, c store.Campaign) ([]string, error) {
	if c.Generation == 0 {
		if d := c.Config.Depart; d != nil {
			return o.frontNames(ctx, d.CampagneID, d.Generation)
		}
		return nil, nil
	}
	return o.frontNames(ctx, c.ID, c.Generation)
}

func (o *Orchestrator) frontNames(ctx context.Context, id int64, g int) ([]string, error) {
	if g == 0 {
		return nil, nil
	}
	rows, err := o.st.Pool.Query(ctx, `SELECT name FROM individuals WHERE campaign_id=$1 AND generation=$2 AND on_front`, id, g)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// progress envoie un avis de progression, sans ligne de journal : l'API le fusionne avant envoi.
func (o *Orchestrator) progress(ctx context.Context, id int64, g, lot, lots int, used, total float64) {
	b, _ := json.Marshal(map[string]any{
		"campaign_id": id, "type": "progression", "generation": g, "lot": lot, "lots": lots,
		"budget_used": used, "budget_total": total,
	})
	if _, err := o.st.Pool.Exec(ctx, `SELECT pg_notify($1, $2)`, ProgressChannel, string(b)); err != nil {
		o.log.Warn("avis de progression", "err", err)
	}
}

// persist écrit la génération, les individus, les agrégats et le journal dans une seule transaction.
func (o *Orchestrator) persist(ctx context.Context, c store.Campaign, g int, started time.Time, sim simGeneration) error {
	cfg := c.Config
	return pgx.BeginFunc(ctx, o.st.Pool, func(tx pgx.Tx) error {
		rows := make([][]any, 0, len(sim.Individuals))
		for _, ind := range sim.Individuals {
			genome, _ := json.Marshal(ind.Genome)
			var parent any
			if ind.Parent != "" {
				parent = ind.Parent
			}
			rows = append(rows, []any{c.ID, g, ind.Name, parent, ind.Mutations, string(genome), ind.Sensitivity,
				ind.SensitivityLow, ind.SensitivityHigh, ind.SensitivityEval, ind.CostMs, ind.Feasible, ind.OnFront})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"individuals"},
			[]string{"campaign_id", "generation", "name", "parent_name", "mutations", "genome", "sensitivity",
				"sensitivity_low", "sensitivity_high", "sensitivity_eval", "cost_ms", "feasible", "on_front"},
			pgx.CopyFromRows(rows)); err != nil {
			return err
		}

		var bestEval, evalGap *float64
		bestCost := sim.BestCost
		if sim.Best != nil {
			bestEval, evalGap = &sim.Best.SensitivityEval, &sim.EvalGap
			bestCost = sim.Best.CostMs
		}
		if _, err := tx.Exec(ctx, `INSERT INTO generations (campaign_id, number, kind, started_at, finished_at, population,
			feasible, best_sensitivity, median_sensitivity, best_cost_ms, median_cost_ms, eval_gap)
			VALUES ($1, $2, 'rapide', $3, now(), $4, $5, $6, $7, $8, $9, $10)`,
			c.ID, g, started, len(sim.Individuals), sim.Feasible, bestEval, sim.MedianSens, bestCost, sim.MedianCost, evalGap); err != nil {
			return err
		}

		improved := sim.Best != nil && (c.BestSensitivity == nil || sim.Best.SensitivityEval > *c.BestSensitivity+0.001)
		stagnation := c.Stagnation + 1
		best := c.BestSensitivity
		if improved {
			stagnation = 0
			v := sim.Best.SensitivityEval
			best = &v
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET generation=$2, budget_used=budget_used+$3, best_sensitivity=$4,
			stagnation=$5, updated_at=now() WHERE id=$1`, c.ID, g, campaign.CostPerGeneration(cfg), best, stagnation); err != nil {
			return err
		}

		id := c.ID
		ev := func(typ, msg string, payload map[string]any) error {
			payload["generation"] = g
			return store.InsertEvent(ctx, tx, &id, typ, msg, payload)
		}
		if improved {
			if err := ev("nouveau_meilleur", fmt.Sprintf(
				"Génération %d : nouveau meilleur compromis, %s, sensibilité %s sur E-val (mesurée au taux de rappel cible fixé à %s %%), %s ms par examen ; mutations %s.",
				g, sim.Best.Name, fr(sim.Best.SensitivityEval, 3), fr(cfg.Projet.TauxRappelCible*100, 1), fr(sim.Best.CostMs, 0),
				strings.Join(sim.Best.Mutations, " › ")),
				map[string]any{"individu": sim.Best.Name, "sensibilite": sim.Best.SensitivityEval, "cout_ms": sim.Best.CostMs}); err != nil {
				return err
			}
		}
		if rejected := len(sim.Individuals) - sim.Feasible; rejected > 0 {
			if err := ev("individus_ecartes", fmt.Sprintf("Génération %d : %d individus écartés, sensibilité sous le plancher.", g, rejected),
				map[string]any{"nombre": rejected}); err != nil {
				return err
			}
		}
		if left := cfg.Evolution.ArretStagnation - stagnation; !improved && left > 0 && left <= 3 {
			if err := ev("eval_stable", fmt.Sprintf("E-val stable depuis %d générations : arrêt prévu dans %d génération(s).", stagnation, left),
				map[string]any{"stagnation": stagnation}); err != nil {
				return err
			}
		}
		msg := fmt.Sprintf("Génération %d terminée : aucun individu réalisable sur %d, médiane %s.", g, len(sim.Individuals), fr(sim.MedianSens, 3))
		if sim.Feasible > 0 {
			msg = fmt.Sprintf("Génération %d terminée : %d réalisables sur %d, médiane %s, écart E-ajust − E-val %s point(s).",
				g, sim.Feasible, len(sim.Individuals), fr(sim.MedianSens, 3), fr(sim.EvalGap*100, 1))
		}
		return ev("generation_terminee", msg,
			map[string]any{"realisables": sim.Feasible, "population": len(sim.Individuals)})
	})
}

func (o *Orchestrator) finish(ctx context.Context, c store.Campaign, reason string) error {
	id := c.ID
	return pgx.BeginFunc(ctx, o.st.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET state='terminee', finished_at=now(), updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		return store.InsertEvent(ctx, tx, &id, "campagne_terminee",
			fmt.Sprintf("Campagne terminée après %d générations : %s. Candidats prêts pour la décision humaine.", c.Generation, reason),
			map[string]any{"raison": reason, "generation": c.Generation})
	})
}

func (o *Orchestrator) fail(ctx context.Context, id int64, cause error) {
	_ = pgx.BeginFunc(ctx, o.st.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET state='en_echec', updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		return store.InsertEvent(ctx, tx, &id, "campagne_en_echec", "Campagne en échec : "+cause.Error()+".", map[string]any{})
	})
}

// fr écrit un nombre avec une virgule décimale.
func fr(f float64, digits int) string {
	return strings.Replace(strconv.FormatFloat(f, 'f', digits, 64), ".", ",", 1)
}
