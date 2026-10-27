// Package entrainements lit, en lecture seule, les dossiers d'entraînement du CNN (MODELES_DIR) :
// etat.json (phase, progression), journal.csv (une ligne par époque) et rapport.json (à la fin).
// Un entraînement lancé avant l'apparition d'etat.json n'a que journal.csv : son état se déduit des dates.
package entrainements

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Statuts.
const (
	EnCours      = "en_cours"
	Termine      = "termine"
	Echec        = "echec"
	Interrompu   = "interrompu"
	SansNouvelle = "sans_nouvelles"
	EnAttente    = "en_attente_gpu" // écrit par le veilleur (mammomesh_engine.cnn.auto) toutes les minutes
)

// Délais au-delà desquels un entraînement « en cours » est dit sans nouvelles : etat.json est
// réécrit toutes les 10 s ; sans lui, seule la fin de chaque époque (parfois longue) laisse une trace.
const (
	SilenceAvecEtat = 15 * time.Minute
	SilenceSansEtat = 2 * time.Hour
)

var nomValide = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ErrIntrouvable : pas de dossier d'entraînement à ce nom.
var ErrIntrouvable = errors.New("entraînement introuvable")

// Epoque : une ligne de journal.csv.
type Epoque struct {
	Epoque   int      `json:"epoque"`
	Perte    float64  `json:"perte"`
	AUCImage *float64 `json:"auc_image"`
	AUCSein  *float64 `json:"auc_sein"`
	DureeS   float64  `json:"duree_s"`
}

// Resume : ce que montre la liste.
type Resume struct {
	Nom               string     `json:"nom"`
	Statut            string     `json:"statut"`
	Phase             string     `json:"phase"`
	Etape             string     `json:"etape,omitempty"`
	Debut             *time.Time `json:"debut,omitempty"`
	Maj               *time.Time `json:"maj,omitempty"`
	Epoque            int        `json:"epoque"`
	Epoques           int        `json:"epoques,omitempty"`
	MeilleureAUCSein  *float64   `json:"meilleure_auc_sein"`
	AUCSeinValidation *float64   `json:"auc_sein_validation"`
	Message           string     `json:"message,omitempty"`
	AvecEtat          bool       `json:"avec_etat"`
	// Pour l'estimation des temps de fin (calculée par l'interface).
	DureeEpoqueS *float64   `json:"duree_epoque_s"` // durée typique d'une époque (la 1re, qui compile, est écartée si possible)
	Pas          int        `json:"pas"`
	PasParEpoque int        `json:"pas_par_epoque"`
	DebutEpoque  *time.Time `json:"debut_epoque,omitempty"`
	DebutPhase   *time.Time `json:"debut_phase,omitempty"`
	Patience     int        `json:"patience,omitempty"`
}

// Detail : la page d'un entraînement.
type Detail struct {
	Resume
	Etat    json.RawMessage `json:"etat"`
	Journal []Epoque        `json:"journal"`
	Rapport json.RawMessage `json:"rapport"`
}

type etatFichier struct {
	Statut           string     `json:"statut"`
	Phase            string     `json:"phase"`
	Etape            string     `json:"etape"`
	Debut            *time.Time `json:"debut"`
	Maj              *time.Time `json:"maj"`
	Epoque           int        `json:"epoque"`
	Epoques          int        `json:"epoques"`
	MeilleureAUCSein *float64   `json:"meilleure_auc_sein"`
	Message          string     `json:"message"`
	Pas              int        `json:"pas"`
	PasParEpoque     int        `json:"pas_par_epoque"`
	DebutEpoque      *time.Time `json:"debut_epoque"`
	DebutPhase       *time.Time `json:"debut_phase"`
	Patience         int        `json:"patience"`
}

type rapportFichier struct {
	Validation struct {
		AUCSein *float64 `json:"auc_sein"`
	} `json:"validation"`
	Reglages struct {
		EpoquesMax int `json:"epoques_max"`
	} `json:"reglages"`
}

// Lister renvoie les entraînements du dossier, le plus récent en tête. Dossier absent : liste vide.
func Lister(dossier string, maintenant time.Time) ([]Resume, error) {
	entrees, err := os.ReadDir(dossier)
	if errors.Is(err, fs.ErrNotExist) {
		return []Resume{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Resume{}
	for _, e := range entrees {
		if !e.IsDir() || !nomValide.MatchString(e.Name()) {
			continue
		}
		d, err := lire(dossier, e.Name(), maintenant)
		if err != nil {
			continue // dossier illisible : ignoré plutôt que de bloquer la liste
		}
		out = append(out, d.Resume)
	}
	sort.SliceStable(out, func(i, j int) bool { return dateTri(out[i]).After(dateTri(out[j])) })
	return out, nil
}

func dateTri(r Resume) time.Time {
	if r.Debut != nil {
		return *r.Debut
	}
	if r.Maj != nil {
		return *r.Maj
	}
	return time.Time{}
}

// Lire renvoie le détail d'un entraînement.
func Lire(dossier, nom string, maintenant time.Time) (Detail, error) {
	if !nomValide.MatchString(nom) {
		return Detail{}, ErrIntrouvable
	}
	return lire(dossier, nom, maintenant)
}

func lire(dossier, nom string, maintenant time.Time) (Detail, error) {
	rep := filepath.Join(dossier, nom)
	info, err := os.Stat(rep)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && !info.IsDir()) {
		return Detail{}, ErrIntrouvable
	}
	if err != nil {
		return Detail{}, err
	}
	d := Detail{Resume: Resume{Nom: nom, Statut: EnCours, Phase: "preparation"}, Journal: []Epoque{}}
	derniere := info.ModTime()

	if b, mt, ok := lireFichier(rep, "journal.csv"); ok {
		d.Journal = lireJournal(b)
		if mt.After(derniere) {
			derniere = mt
		}
	}
	for _, ep := range d.Journal {
		d.Epoque = ep.Epoque
		if ep.AUCSein != nil && (d.MeilleureAUCSein == nil || *ep.AUCSein > *d.MeilleureAUCSein) {
			v := *ep.AUCSein
			d.MeilleureAUCSein = &v
		}
	}
	if len(d.Journal) > 0 {
		d.Phase = "entrainement"
	}
	d.DureeEpoqueS = dureeTypique(d.Journal)

	if b, mt, ok := lireFichier(rep, "etat.json"); ok {
		var e etatFichier
		if json.Unmarshal(b, &e) == nil {
			d.Etat, d.AvecEtat = json.RawMessage(b), true
			d.Statut, d.Phase, d.Etape, d.Message = e.Statut, e.Phase, e.Etape, e.Message
			d.Debut, d.Epoques = e.Debut, e.Epoques
			d.Pas, d.PasParEpoque, d.DebutEpoque, d.DebutPhase, d.Patience = e.Pas, e.PasParEpoque, e.DebutEpoque, e.DebutPhase, e.Patience
			if e.Epoque > d.Epoque {
				d.Epoque = e.Epoque
			}
			if e.MeilleureAUCSein != nil {
				d.MeilleureAUCSein = e.MeilleureAUCSein
			}
			if e.Maj != nil {
				derniere = *e.Maj
			} else if mt.After(derniere) {
				derniere = mt
			}
		}
	}

	if b, mt, ok := lireFichier(rep, "rapport.json"); ok {
		var r rapportFichier
		if json.Unmarshal(b, &r) == nil {
			d.Rapport = json.RawMessage(b)
			d.AUCSeinValidation = r.Validation.AUCSein
			if d.Epoques == 0 {
				d.Epoques = r.Reglages.EpoquesMax
			}
			if d.Statut == EnCours {
				d.Statut = Termine
			}
			if mt.After(derniere) {
				derniere = mt
			}
		}
	}

	if d.Statut == Termine {
		d.Phase, d.Etape = "termine", ""
	}
	m := derniere.UTC()
	d.Maj = &m
	if d.Statut == EnCours || d.Statut == EnAttente {
		silence := SilenceSansEtat
		if d.AvecEtat {
			silence = SilenceAvecEtat
		}
		if maintenant.Sub(derniere) > silence {
			d.Statut = SansNouvelle
		}
	}
	return d, nil
}

const tailleMax = 4 << 20 // aucun de ces fichiers n'approche 4 Mio

func lireFichier(rep, nom string) ([]byte, time.Time, bool) {
	f := filepath.Join(rep, nom)
	info, err := os.Lstat(f)
	if err != nil || !info.Mode().IsRegular() || info.Size() > tailleMax {
		return nil, time.Time{}, false
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, time.Time{}, false
	}
	return b, info.ModTime(), true
}

func lireJournal(b []byte) []Epoque {
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	lignes, _ := r.ReadAll() // une dernière ligne en cours d'écriture est simplement ignorée
	out := []Epoque{}
	for i, l := range lignes {
		if i == 0 || len(l) < 5 {
			continue
		}
		ep, err := strconv.Atoi(l[0])
		if err != nil {
			continue
		}
		e := Epoque{Epoque: ep, Perte: nombre(l[1]), AUCImage: nombreOuNul(l[2]), AUCSein: nombreOuNul(l[3]), DureeS: nombre(l[4])}
		out = append(out, e)
	}
	return out
}

// dureeTypique : moyenne des trois dernières époques, sans la première (compilation, chargement
// des caches) dès qu'il y en a au moins deux.
func dureeTypique(j []Epoque) *float64 {
	if len(j) > 1 {
		j = j[1:]
	}
	if len(j) > 3 {
		j = j[len(j)-3:]
	}
	s, n := 0.0, 0
	for _, e := range j {
		if e.DureeS > 0 {
			s += e.DureeS
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := s / float64(n)
	return &v
}

func nombre(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func nombreOuNul(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
