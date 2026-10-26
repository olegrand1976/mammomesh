package orchestrator

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
)

// Simulation de la boucle rapide, en attendant le vrai couple CNN + CephaloMesh.
// Elle est déterministe : (graine, campagne, génération) donne toujours le même résultat,
// ce qui rend la reprise après panne exacte.

var mutationCodes = []string{"SEU↑", "SEU↓", "TMP↑C", "TMP↓B", "LAM↑", "LAM↓", "RAY↑", "RAY↓", "BUD↑", "BUD↓",
	"SYM+", "TEM+", "CON+", "BIR+", "CUB16×8", "DEG+", "COU-BIFPN", "BLK+", "RES1536"}

// simIndividual est un individu simulé, avant écriture en base.
type simIndividual struct {
	Name            string
	Parent          string
	Mutations       []string
	Genome          map[string]any
	Sensitivity     float64 // E-ajust, fenêtre de rappel
	SensitivityLow  float64
	SensitivityHigh float64
	SensitivityEval float64 // E-val
	CostMs          float64
	Feasible        bool
	OnFront         bool
}

// simGeneration est le résultat d'une génération simulée.
type simGeneration struct {
	Individuals []simIndividual
	Best        *simIndividual // meilleur réalisable sur E-val
	Feasible    int
	MedianSens  float64
	MedianCost  float64
	BestCost    float64
	EvalGap     float64
}

func rngFor(seed, campaignID int64, generation int) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed)^uint64(campaignID)*0x9E3779B97F4A7C15, uint64(generation)+1))
}

func poisson(r *rand.Rand, mu float64) int {
	l, k, p := math.Exp(-mu), 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// simulate produit la génération g (1, 2, …) d'une campagne.
func simulate(cfg campaign.Config, campaignID int64, g int, prevFront []string) simGeneration {
	r := rngFor(cfg.Evolution.Graine, campaignID, g)
	ceiling := 0.80 + 0.10*(1-math.Exp(-float64(g)/35)) // progrès qui ralentit
	floor := cfg.Projet.PlancherSensibilite - cfg.Projet.MargePoints/100

	out := simGeneration{}
	for i := 1; i <= cfg.Evolution.Population; i++ {
		cost := 110 + 390*r.Float64()
		sens := ceiling - math.Abs(r.NormFloat64())*0.035 + 0.035*(cost-110)/390
		sens = math.Min(sens, 0.985)
		half := 0.022 + 0.01*r.Float64()
		gap := 0.008 + 0.012*r.Float64()

		m := 1 + poisson(r, cfg.Evolution.MutationsMu)
		if m > cfg.Evolution.MutationsMax {
			m = cfg.Evolution.MutationsMax
		}
		muts := make([]string, m)
		for k := range muts {
			muts[k] = mutationCodes[r.IntN(len(mutationCodes))]
		}
		parent := ""
		if len(prevFront) > 0 {
			parent = prevFront[r.IntN(len(prevFront))]
		}
		ind := simIndividual{
			Name:            fmt.Sprintf("%s-G%02d.%02d", cfg.Projet.Lignee, g, i),
			Parent:          parent,
			Mutations:       muts,
			Sensitivity:     round4(sens),
			SensitivityLow:  round4(sens - half),
			SensitivityHigh: round4(math.Min(sens+half, 1)),
			SensitivityEval: round4(sens - gap),
			CostMs:          math.Round(cost*10) / 10,
			Genome: map[string]any{
				"seuil":            round4(0.3 + 0.4*r.Float64()),
				"tau":              round4(2 + 8*r.Float64()),
				"lambda":           round4(0.5 + 2*r.Float64()),
				"budget_activite":  round4(0.02 + 0.28*r.Float64()),
				"rayon_stimuli_mm": round4(4 + 10*r.Float64()),
				"tours":            1 + r.IntN(2),
				"controles":        []string{"profil", "concordance", "symetrie", "temporel", "birads", "retour_cnn"}[:3+r.IntN(4)],
			},
		}
		ind.Feasible = ind.SensitivityLow >= floor
		out.Individuals = append(out.Individuals, ind)
	}

	markFront(out.Individuals)

	var sens, costs, gaps []float64
	for i := range out.Individuals {
		ind := &out.Individuals[i]
		sens = append(sens, ind.Sensitivity)
		costs = append(costs, ind.CostMs)
		if ind.Feasible {
			out.Feasible++
			gaps = append(gaps, ind.Sensitivity-ind.SensitivityEval)
			if out.Best == nil || ind.SensitivityEval > out.Best.SensitivityEval {
				out.Best = ind
			}
		}
	}
	out.MedianSens = round4(median(sens))
	out.MedianCost = math.Round(median(costs)*10) / 10
	out.BestCost = math.Round(minOf(costs)*10) / 10
	if len(gaps) > 0 {
		out.EvalGap = round4(mean(gaps))
	}
	return out
}

// markFront marque les individus réalisables non dominés (sensibilité max, coût min).
func markFront(list []simIndividual) {
	idx := make([]int, 0, len(list))
	for i := range list {
		if list[i].Feasible {
			idx = append(idx, i)
		}
	}
	sort.Slice(idx, func(a, b int) bool { return list[idx[a]].CostMs < list[idx[b]].CostMs })
	best := -1.0
	for _, i := range idx {
		if list[i].Sensitivity > best {
			list[i].OnFront = true
			best = list[i].Sensitivity
		}
	}
}

func round4(f float64) float64 { return math.Round(f*10000) / 10000 }

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

func mean(v []float64) float64 {
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t / float64(len(v))
}

func minOf(v []float64) float64 {
	m := math.Inf(1)
	for _, x := range v {
		m = math.Min(m, x)
	}
	return m
}
