package mcpserver

import (
	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// Entrées et sorties typées des outils : Claude reçoit des champs, pas du texte à analyser.

type ListIn struct {
	Limite int `json:"limite,omitempty" jsonschema:"nombre de campagnes, 20 par défaut, 50 au plus"`
}

type CampaignSummary struct {
	ID                       int64    `json:"id"`
	Nom                      string   `json:"nom"`
	Lignee                   string   `json:"lignee"`
	Etat                     string   `json:"etat"`
	Generation               int      `json:"generation"`
	GenerationsMax           int      `json:"generations_max"`
	BudgetUtilise            float64  `json:"budget_utilise_coeur_heures"`
	BudgetTotal              float64  `json:"budget_total_coeur_heures"`
	MeilleureSensibiliteEval *float64 `json:"meilleure_sensibilite_eval"`
	Stagnation               int      `json:"generations_sans_progres"`
	Configuration            string   `json:"empreinte_configuration"`
}

type ListOut struct {
	Campagnes []CampaignSummary `json:"campagnes"`
}

type StateIn struct {
	CampagneID  int64 `json:"campagne_id" jsonschema:"identifiant de la campagne"`
	Generations int   `json:"generations,omitempty" jsonschema:"nombre de dernières générations à inclure, 5 par défaut"`
}

type StateOut struct {
	Campagne    CampaignSummary    `json:"campagne"`
	Config      campaign.Config    `json:"configuration"`
	Generations []store.Generation `json:"generations"`
}

type FrontIn struct {
	CampagneID int64 `json:"campagne_id" jsonschema:"identifiant de la campagne"`
	Generation int   `json:"generation,omitempty" jsonschema:"numéro de génération, la dernière par défaut"`
}

type FrontPoint struct {
	ID              int64    `json:"id"`
	Nom             string   `json:"nom"`
	Sensibilite     float64  `json:"sensibilite_e_ajust"`
	IntervalleBas   float64  `json:"intervalle_bas"`
	IntervalleHaut  float64  `json:"intervalle_haut"`
	SensibiliteEval float64  `json:"sensibilite_e_val"`
	CoutMs          float64  `json:"cout_ms_par_examen"`
	Mutations       []string `json:"mutations"`
}

type FrontOut struct {
	Generation int          `json:"generation"`
	Individus  []FrontPoint `json:"individus"`
	Tronque    bool         `json:"tronque"`
}

type IndividualIn struct {
	IndividuID int64  `json:"individu_id,omitempty" jsonschema:"identifiant de l'individu"`
	CampagneID int64  `json:"campagne_id,omitempty" jsonschema:"campagne, avec le nom"`
	Nom        string `json:"nom,omitempty" jsonschema:"nom de l'individu, par exemple ORION-G03.02"`
}

type CompareIn struct {
	A int64 `json:"a" jsonschema:"identifiant du premier individu"`
	B int64 `json:"b" jsonschema:"identifiant du second individu"`
}

type GeneDiff struct {
	Gene string `json:"gene"`
	A    any    `json:"a"`
	B    any    `json:"b"`
}

type CompareOut struct {
	A       string                `json:"a"`
	B       string                `json:"b"`
	Genome  []GeneDiff            `json:"genome_differences"`
	Mesures map[string][2]float64 `json:"mesures"`
}

type JournalIn struct {
	CampagneID int64    `json:"campagne_id" jsonschema:"identifiant de la campagne"`
	Types      []string `json:"types,omitempty" jsonschema:"types à garder, par exemple nouveau_meilleur, generation_terminee"`
	Limite     int      `json:"limite,omitempty" jsonschema:"nombre d'événements, 20 par défaut, 50 au plus"`
	ApresID    int64    `json:"apres_id,omitempty" jsonschema:"ne lire que les événements postérieurs à cet identifiant"`
}

type JournalLine struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	Phrase string `json:"phrase"`
	Date   string `json:"date"`
}

type JournalOut struct {
	Evenements []JournalLine `json:"evenements"`
}

type DraftIn struct {
	BaseCampagneID int64          `json:"base_campagne_id,omitempty" jsonschema:"campagne de départ ; valeurs par défaut sinon"`
	Reglages       map[string]any `json:"reglages" jsonschema:"réglages modifiés en notation pointée, par exemple {\"evolution.population\": 150}"`
}

type DraftRef struct {
	BrouillonID int64 `json:"brouillon_id" jsonschema:"identifiant du brouillon"`
}

type ControlIn struct {
	CampagneID   int64   `json:"campagne_id" jsonschema:"identifiant de la campagne"`
	Action       string  `json:"action" jsonschema:"pause, reprise, arret ou budget"`
	HausseBudget float64 `json:"hausse_budget,omitempty" jsonschema:"hausse du budget en cœur-heures, pour l'action budget"`
}
