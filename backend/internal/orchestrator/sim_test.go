package orchestrator

import (
	"reflect"
	"testing"

	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
)

func TestSimulateIsDeterministic(t *testing.T) {
	cfg := campaign.Default()
	a := simulate(cfg, 7, 3, []string{"ORION-G02.01"})
	b := simulate(cfg, 7, 3, []string{"ORION-G02.01"})
	if !reflect.DeepEqual(a.Individuals, b.Individuals) {
		t.Fatal("même graine, même génération : résultats différents (la reprise ne serait pas exacte)")
	}
	c := simulate(cfg, 8, 3, nil)
	if reflect.DeepEqual(a.Individuals[0].Sensitivity, c.Individuals[0].Sensitivity) && a.Individuals[0].CostMs == c.Individuals[0].CostMs {
		t.Fatal("deux campagnes différentes donnent le même individu")
	}
}

func TestFrontIsNonDominated(t *testing.T) {
	g := simulate(campaign.Default(), 1, 10, nil)
	var front []simIndividual
	for _, i := range g.Individuals {
		if i.OnFront {
			if !i.Feasible {
				t.Fatal("individu non réalisable sur le front")
			}
			front = append(front, i)
		}
	}
	if len(front) == 0 {
		t.Fatal("front vide")
	}
	for _, f := range front {
		for _, o := range g.Individuals {
			if o.Feasible && o.Sensitivity > f.Sensitivity && o.CostMs < f.CostMs {
				t.Fatalf("%s est dominé par %s", f.Name, o.Name)
			}
		}
	}
}

func TestMutationsRespectCap(t *testing.T) {
	cfg := campaign.Default()
	cfg.Evolution.MutationsMu = 4
	cfg.Evolution.MutationsMax = 3
	for _, i := range simulate(cfg, 1, 1, nil).Individuals {
		if len(i.Mutations) < 1 || len(i.Mutations) > 3 {
			t.Fatalf("%d mutations pour %s", len(i.Mutations), i.Name)
		}
	}
}
