// Package campaign décrit la configuration d'une campagne d'évolution et ses contrôles avant lancement.
package campaign

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Config est la configuration figée d'une campagne (clé « config » en base).
type Config struct {
	Projet     Projet     `json:"projet"`
	Evolution  Evolution  `json:"evolution"`
	Simulation Simulation `json:"simulation"`
}

// Projet : objectif et garde-fous écrits avant la campagne.
type Projet struct {
	Nom                 string  `json:"nom" jsonschema:"nom de la campagne"`
	Objectif            string  `json:"objectif" jsonschema:"objectif en une phrase"`
	Lignee              string  `json:"lignee" jsonschema:"nom de la lignée fondatrice, par exemple ORION"`
	PlancherSensibilite float64 `json:"plancher_sensibilite" jsonschema:"plancher de sensibilité, de 0,80 à 0,99"`
	MargePoints         float64 `json:"marge_points" jsonschema:"marge de non-infériorité en points, de 1 à 5"`
	TauxRappelCible     float64 `json:"taux_rappel_cible" jsonschema:"taux de rappel cible, de 0,02 à 0,10"`
}

// Evolution : réglages de la boucle rapide.
type Evolution struct {
	Population       int     `json:"population" jsonschema:"taille de la population CephaloMesh, de 20 à 500"`
	GenerationsMax   int     `json:"generations_max" jsonschema:"plafond de générations, de 10 à 2000"`
	ArretStagnation  int     `json:"arret_stagnation" jsonschema:"générations sans progrès sur E-val avant arrêt"`
	BudgetCoeurHeure float64 `json:"budget_coeur_heures" jsonschema:"budget de calcul en cœur-heures (plafond dur)"`
	MutationsMu      float64 `json:"mutations_mu" jsonschema:"μ du nombre de mutations par enfant (1 + Poisson(μ))"`
	MutationsMax     int     `json:"mutations_max" jsonschema:"plafond de mutations par enfant"`
	Iles             int     `json:"iles" jsonschema:"nombre d'îles, de 1 à 8"`
	Graine           int64   `json:"graine" jsonschema:"graine aléatoire"`
}

// Simulation : tant que le CNN et CephaloMesh ne sont pas branchés, la boucle est simulée.
type Simulation struct {
	DureeLotMs        int `json:"duree_lot_ms" jsonschema:"durée simulée d'un lot d'évaluations en millisecondes"`
	LotsParGeneration int `json:"lots_par_generation" jsonschema:"nombre de lots par génération"`
}

// Default renvoie les valeurs par défaut de la note de conception (section Interface de configuration).
func Default() Config {
	return Config{
		Projet: Projet{
			Nom: "Campagne simulée", Objectif: "Valider la chaîne de suivi de bout en bout",
			Lignee: "ORION", PlancherSensibilite: 0.85, MargePoints: 3, TauxRappelCible: 0.05,
		},
		Evolution: Evolution{
			Population: 100, GenerationsMax: 200, ArretStagnation: 15, BudgetCoeurHeure: 50,
			MutationsMu: 1, MutationsMax: 5, Iles: 1, Graine: 42,
		},
		Simulation: Simulation{DureeLotMs: 300, LotsParGeneration: 10},
	}
}

// Check est un contrôle avant lancement.
type Check struct {
	Name   string `json:"nom"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Verification résume les contrôles et l'estimation.
type Verification struct {
	OK              bool    `json:"ok"`
	Checks          []Check `json:"controles"`
	DureeEstimeeMin float64 `json:"duree_estimee_min"`
	CoeurHeures     float64 `json:"coeur_heures_estimees"`
}

func between(name string, v, lo, hi float64, unit string) Check {
	ok := v >= lo && v <= hi
	return Check{Name: name, OK: ok, Detail: fmt.Sprintf("%g %s (plage %g à %g)", v, unit, lo, hi)}
}

// Verify applique les contrôles avant lancement, sans rien lancer.
func Verify(c Config) Verification {
	checks := []Check{
		{Name: "nom", OK: strings.TrimSpace(c.Projet.Nom) != "", Detail: "le nom est obligatoire"},
		{Name: "lignee", OK: validLineage(c.Projet.Lignee), Detail: "lettres majuscules, 3 à 12 caractères"},
		between("plancher_sensibilite", c.Projet.PlancherSensibilite, 0.80, 0.99, ""),
		between("marge_points", c.Projet.MargePoints, 1, 5, "points"),
		between("taux_rappel_cible", c.Projet.TauxRappelCible, 0.02, 0.10, ""),
		between("population", float64(c.Evolution.Population), 20, 500, "individus"),
		between("generations_max", float64(c.Evolution.GenerationsMax), 10, 2000, "générations"),
		between("arret_stagnation", float64(c.Evolution.ArretStagnation), 3, 100, "générations"),
		between("budget_coeur_heures", c.Evolution.BudgetCoeurHeure, 0.1, 100000, "cœur-heures"),
		between("mutations_mu", c.Evolution.MutationsMu, 0, 4, ""),
		between("mutations_max", float64(c.Evolution.MutationsMax), 1, 10, ""),
		between("iles", float64(c.Evolution.Iles), 1, 8, "îles"),
		between("duree_lot_ms", float64(c.Simulation.DureeLotMs), 10, 60000, "ms"),
		between("lots_par_generation", float64(c.Simulation.LotsParGeneration), 1, 100, "lots"),
	}
	v := Verification{OK: true, Checks: checks}
	for _, ch := range checks {
		if !ch.OK {
			v.OK = false
		}
	}
	perGen := time.Duration(c.Simulation.DureeLotMs*c.Simulation.LotsParGeneration) * time.Millisecond
	v.DureeEstimeeMin = round2(perGen.Minutes() * float64(c.Evolution.GenerationsMax))
	v.CoeurHeures = round2(CostPerGeneration(c) * float64(c.Evolution.GenerationsMax))
	if v.CoeurHeures > c.Evolution.BudgetCoeurHeure {
		v.Checks = append(v.Checks, Check{Name: "budget", OK: true,
			Detail: fmt.Sprintf("le plafond de %g cœur-heures arrêtera la campagne après la génération %d",
				c.Evolution.BudgetCoeurHeure, LastGenerationInBudget(c))})
	}
	return v
}

// CostPerGeneration : cœur-heures consommées par une génération simulée.
// Repère de la note : 0,2 à 0,5 ms par examen, 20 000 examens par configuration.
func CostPerGeneration(c Config) float64 {
	return float64(c.Evolution.Population) * 20000 * 0.35e-3 / 3600
}

// LastGenerationInBudget : dernière génération jouée avant l'arrêt par le budget. L'orchestrateur ne
// démarre une génération que si le budget n'est pas encore atteint : la dernière peut le dépasser.
func LastGenerationInBudget(c Config) int {
	return int(math.Ceil(c.Evolution.BudgetCoeurHeure/CostPerGeneration(c) - 1e-9))
}

func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

func validLineage(s string) bool {
	if len(s) < 3 || len(s) > 12 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// Hash renvoie l'empreinte de la configuration (JSON canonique).
func Hash(c Config) string {
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Change décrit un réglage modifié entre deux configurations.
type Change struct {
	Field string `json:"champ"`
	From  any    `json:"avant"`
	To    any    `json:"apres"`
}

// Diff liste les réglages qui diffèrent, en notation pointée (« evolution.population »).
func Diff(a, b Config) []Change {
	fa, fb := flatten(a), flatten(b)
	var keys []string
	for k := range fb {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []Change{} // jamais nil : « [] » en JSON, attendu par l'interface et le schéma MCP
	for _, k := range keys {
		if !reflect.DeepEqual(fa[k], fb[k]) {
			out = append(out, Change{Field: k, From: fa[k], To: fb[k]})
		}
	}
	return out
}

func flatten(c Config) map[string]any {
	b, _ := json.Marshal(c)
	var m map[string]map[string]any
	_ = json.Unmarshal(b, &m)
	out := map[string]any{}
	for sec, fields := range m {
		for k, v := range fields {
			out[sec+"."+k] = v
		}
	}
	return out
}

// Apply modifie une configuration à partir de réglages en notation pointée.
func Apply(base Config, changes map[string]any) (Config, error) {
	b, _ := json.Marshal(base)
	var m map[string]map[string]any
	_ = json.Unmarshal(b, &m)
	for k, v := range changes {
		parts := strings.SplitN(k, ".", 2)
		if len(parts) != 2 {
			return base, fmt.Errorf("réglage %q : attendu section.champ", k)
		}
		sec, ok := m[parts[0]]
		if !ok {
			return base, fmt.Errorf("section inconnue %q", parts[0])
		}
		if _, ok := sec[parts[1]]; !ok {
			return base, fmt.Errorf("réglage inconnu %q", k)
		}
		sec[parts[1]] = v
	}
	b, _ = json.Marshal(m)
	var out Config
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return base, fmt.Errorf("valeur invalide : %w", err)
	}
	return out, nil
}
