package entrainements

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Demandes d'action déposées par l'API dans DossierDemandes et exécutées sur l'hôte par le veilleur
// (python -m mammomesh_engine.cnn.auto), seul à pouvoir lancer, arrêter ou effacer un entraînement.
const DossierDemandes = ".demandes"

// Actions.
const (
	Relancer  = "relancer"  // reprendre là où il s'est arrêté, ou relancer avec les mêmes réglages
	Supprimer = "supprimer" // arrêter s'il tourne, puis effacer le dossier et son journal
	Nouveau   = "nouveau"   // nouvel entraînement, avec les réglages donnés, en file d'attente du GPU
	Pause     = "pause"     // mettre en pause (Mode : en fin d'époque ou tout de suite) ; « relancer » reprend
)

// Modes de pause.
const (
	PauseFinEpoque  = "fin_epoque" // après l'époque en cours : rien n'est perdu
	PauseMaintenant = "maintenant" // tout de suite : l'époque en cours sera refaite
)

// Erreurs des demandes.
var (
	ErrActionInconnue  = errors.New("action inconnue")
	ErrDejaDemande     = errors.New("une demande est déjà en attente pour cet entraînement")
	ErrEnCours         = errors.New("entraînement en cours : il n'y a rien à relancer")
	ErrPasEnCours      = errors.New("seul un entraînement en cours ou en attente du GPU se met en pause")
	ErrSansReprise     = errors.New("entraînement lancé avant les points de reprise : une pause le perdrait, le supprimer ou le laisser finir")
	ErrDemandesFermees = errors.New("dossier des demandes inaccessible en écriture (lancer ./scripts/demarrer-pile.sh sur l'hôte)")
)

// Demande : une action en attente du veilleur.
type Demande struct {
	Action   string    `json:"action"`
	Mode     string    `json:"mode,omitempty"` // action « pause » seulement
	Nom      string    `json:"nom,omitempty"`
	Par      string    `json:"par"`
	Date     time.Time `json:"date"`
	Reglages *Reglages `json:"reglages,omitempty"` // action « nouveau » seulement
}

// Reglages d'un nouvel entraînement, mêmes noms que les options de python -m mammomesh_engine.cnn.
// La banque est celle du serveur (BANQUE_DIR du veilleur, sinon ./banque).
type Reglages struct {
	Epoques             int     `json:"epoques"`
	Resolution          string  `json:"resolution"`
	Patience            int     `json:"patience"`
	LR                  float64 `json:"lr"`
	Graine              int     `json:"graine"`
	SensibiliteCible    float64 `json:"sensibilite_cible"`
	Limite              int     `json:"limite"` // patientes, 0 = toutes (sinon essai rapide)
	SansPreentrainement bool    `json:"sans_preentrainement"`
}

// ReglagesParDefaut : ceux de python -m mammomesh_engine.cnn sans option.
func ReglagesParDefaut() Reglages {
	return Reglages{Epoques: 15, Resolution: "1024x832", Patience: 5, LR: 3e-4, Graine: 42, SensibiliteCible: 0.85}
}

// Resolutions proposées : celles que le moteur sait descendre si la mémoire manque.
var Resolutions = []string{"1024x832", "768x624", "512x416"}

// InvalideError : réglage hors plage.
type InvalideError struct{ Msg string }

func (e InvalideError) Error() string { return e.Msg }

// Verifier contrôle chaque réglage.
func (r Reglages) Verifier() error {
	bon := false
	for _, x := range Resolutions {
		bon = bon || r.Resolution == x
	}
	switch {
	case r.Epoques < 1 || r.Epoques > 100:
		return InvalideError{"époques : de 1 à 100"}
	case !bon:
		return InvalideError{"résolution : " + strings.Join(Resolutions, ", ")}
	case r.Patience < 1 || r.Patience > 50:
		return InvalideError{"patience : de 1 à 50 époques"}
	case r.LR < 1e-5 || r.LR > 1e-2:
		return InvalideError{"taux d'apprentissage : de 0,00001 à 0,01"}
	case r.Graine < 0 || r.Graine > 1<<31-1:
		return InvalideError{"graine : entier positif"}
	case r.SensibiliteCible < 0.5 || r.SensibiliteCible > 0.99:
		return InvalideError{"sensibilité cible : de 0,50 à 0,99"}
	case r.Limite != 0 && (r.Limite < 10 || r.Limite > 100000):
		return InvalideError{"limite : 0 (toutes les patientes) ou de 10 à 100 000"}
	}
	return nil
}

// Demandes renvoie les demandes en attente sur un entraînement existant, par entraînement
// (la plus ancienne si plusieurs).
func Demandes(dossier string) map[string]Demande {
	out := map[string]Demande{}
	for _, d := range toutes(dossier) {
		if _, deja := out[d.Nom]; !deja && d.Nom != "" {
			out[d.Nom] = d
		}
	}
	return out
}

// NouveauxEnAttente : nouveaux entraînements demandés que le veilleur n'a pas encore créés.
func NouveauxEnAttente(dossier string) []Demande {
	out := []Demande{}
	for _, d := range toutes(dossier) {
		if d.Action == Nouveau {
			out = append(out, d)
		}
	}
	return out
}

func toutes(dossier string) []Demande {
	out := []Demande{}
	entrees, err := os.ReadDir(filepath.Join(dossier, DossierDemandes))
	if err != nil {
		return out
	}
	sort.Slice(entrees, func(i, j int) bool { return entrees[i].Name() < entrees[j].Name() })
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, _, ok := lireFichier(filepath.Join(dossier, DossierDemandes), e.Name())
		var d Demande
		if !ok || json.Unmarshal(b, &d) != nil || (d.Nom != "" && !nomValide.MatchString(d.Nom)) || (d.Nom == "" && d.Action != Nouveau) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// Demander dépose une demande, après vérification : l'entraînement existe, l'action a un sens,
// aucune autre demande n'attend pour lui.
func Demander(dossier, nom, action, par string, maintenant time.Time) (Demande, error) {
	return DemanderMode(dossier, nom, action, "", par, maintenant)
}

// DemanderMode : comme Demander, avec le mode d'une pause (défaut : en fin d'époque).
func DemanderMode(dossier, nom, action, mode, par string, maintenant time.Time) (Demande, error) {
	if action != Relancer && action != Supprimer && action != Pause {
		return Demande{}, ErrActionInconnue
	}
	if action == Pause {
		if mode == "" {
			mode = PauseFinEpoque
		}
		if mode != PauseFinEpoque && mode != PauseMaintenant {
			return Demande{}, ErrActionInconnue
		}
	} else {
		mode = ""
	}
	d, err := Lire(dossier, nom, maintenant)
	if err != nil {
		return Demande{}, err
	}
	if action == Relancer && d.Statut == EnCours {
		return Demande{}, ErrEnCours
	}
	if action == Pause && d.Statut != EnCours && d.Statut != EnAttente {
		return Demande{}, ErrPasEnCours
	}
	if action == Pause && !d.AvecParametres {
		return Demande{}, ErrSansReprise
	}
	if _, deja := Demandes(dossier)[nom]; deja {
		return Demande{}, ErrDejaDemande
	}
	return deposer(dossier, Demande{Action: action, Mode: mode, Nom: nom, Par: par, Date: maintenant.UTC()}, maintenant)
}

// DemanderNouveau dépose la demande d'un nouvel entraînement, après vérification des réglages.
func DemanderNouveau(dossier string, r Reglages, par string, maintenant time.Time) (Demande, error) {
	if err := r.Verifier(); err != nil {
		return Demande{}, err
	}
	return deposer(dossier, Demande{Action: Nouveau, Par: par, Date: maintenant.UTC(), Reglages: &r}, maintenant)
}

func deposer(dossier string, dem Demande, maintenant time.Time) (Demande, error) {
	b, _ := json.Marshal(dem)
	rep := filepath.Join(dossier, DossierDemandes)
	fichier := fmt.Sprintf("%d-%s-%s.json", maintenant.UnixMilli(), dem.Action, dem.Nom)
	tmp := filepath.Join(rep, "."+fichier+".tmp")
	if err := os.WriteFile(tmp, b, 0o666); err != nil {
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return Demande{}, ErrDemandesFermees
		}
		return Demande{}, err
	}
	if err := os.Rename(tmp, filepath.Join(rep, fichier)); err != nil {
		_ = os.Remove(tmp)
		return Demande{}, err
	}
	return dem, nil
}
