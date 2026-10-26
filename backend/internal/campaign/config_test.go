package campaign

import (
	"encoding/json"
	"testing"
)

func TestDefaultPassesChecks(t *testing.T) {
	v := Verify(Default())
	if !v.OK {
		t.Fatalf("la configuration par défaut échoue : %+v", v.Checks)
	}
	if v.DureeEstimeeMin <= 0 || v.CoeurHeures <= 0 {
		t.Fatalf("estimation absente : %+v", v)
	}
}

func TestVerifyRejectsOutOfRange(t *testing.T) {
	c := Default()
	c.Projet.PlancherSensibilite = 0.5
	c.Projet.Lignee = "orion"
	v := Verify(c)
	if v.OK {
		t.Fatal("plancher hors plage accepté")
	}
	failed := map[string]bool{}
	for _, ch := range v.Checks {
		if !ch.OK {
			failed[ch.Name] = true
		}
	}
	if !failed["plancher_sensibilite"] || !failed["lignee"] {
		t.Fatalf("contrôles en échec inattendus : %v", failed)
	}
}

func TestApplyAndDiff(t *testing.T) {
	base := Default()
	c, err := Apply(base, map[string]any{"evolution.population": 150, "projet.lignee": "VEGA"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Evolution.Population != 150 || c.Projet.Lignee != "VEGA" {
		t.Fatalf("réglages non appliqués : %+v", c)
	}
	d := Diff(base, c)
	if len(d) != 2 || d[0].Field != "evolution.population" || d[1].Field != "projet.lignee" {
		t.Fatalf("diff inattendu : %+v", d)
	}
	if Hash(base) == Hash(c) {
		t.Fatal("empreinte inchangée")
	}
}

func TestApplyRejectsUnknown(t *testing.T) {
	for _, k := range []string{"evolution.inconnu", "inconnue.x", "sansPoint"} {
		if _, err := Apply(Default(), map[string]any{k: 1}); err == nil {
			t.Fatalf("réglage %q accepté", k)
		}
	}
	if _, err := Apply(Default(), map[string]any{"evolution.population": "beaucoup"}); err == nil {
		t.Fatal("valeur de mauvais type acceptée")
	}
}

// Configuration inchangée : la différence est une liste vide, jamais null en JSON
// (l'interface lit diff.length ; le schéma MCP attend un tableau).
func TestDiffEmptyIsArray(t *testing.T) {
	d := Diff(Default(), Default())
	b, _ := json.Marshal(d)
	if string(b) != "[]" {
		t.Fatalf("diff vide : %s", b)
	}
}

// Le budget arrête après la génération qui l'atteint : 500 individus (0,97 cœur-h) et 50 cœur-h → 52.
func TestLastGenerationInBudget(t *testing.T) {
	c := Default()
	c.Evolution.Population = 500
	if g := LastGenerationInBudget(c); g != 52 {
		t.Fatalf("dernière génération : %d", g)
	}
	c.Evolution.BudgetCoeurHeure = CostPerGeneration(c) * 10
	if g := LastGenerationInBudget(c); g != 10 {
		t.Fatalf("budget exact pour 10 générations : %d", g)
	}
}
