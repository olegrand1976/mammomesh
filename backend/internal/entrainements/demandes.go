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
)

// Erreurs des demandes.
var (
	ErrActionInconnue  = errors.New("action inconnue")
	ErrDejaDemande     = errors.New("une demande est déjà en attente pour cet entraînement")
	ErrEnCours         = errors.New("entraînement en cours : il n'y a rien à relancer")
	ErrDemandesFermees = errors.New("dossier des demandes inaccessible en écriture (lancer ./scripts/demarrer-pile.sh sur l'hôte)")
)

// Demande : une action en attente du veilleur.
type Demande struct {
	Action string    `json:"action"`
	Nom    string    `json:"nom"`
	Par    string    `json:"par"`
	Date   time.Time `json:"date"`
}

// Demandes renvoie les demandes en attente, par entraînement (la plus ancienne si plusieurs).
func Demandes(dossier string) map[string]Demande {
	out := map[string]Demande{}
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
		if !ok || json.Unmarshal(b, &d) != nil || !nomValide.MatchString(d.Nom) {
			continue
		}
		if _, deja := out[d.Nom]; !deja {
			out[d.Nom] = d
		}
	}
	return out
}

// Demander dépose une demande, après vérification : l'entraînement existe, l'action a un sens,
// aucune autre demande n'attend pour lui.
func Demander(dossier, nom, action, par string, maintenant time.Time) (Demande, error) {
	if action != Relancer && action != Supprimer {
		return Demande{}, ErrActionInconnue
	}
	d, err := Lire(dossier, nom, maintenant)
	if err != nil {
		return Demande{}, err
	}
	if action == Relancer && d.Statut == EnCours {
		return Demande{}, ErrEnCours
	}
	if _, deja := Demandes(dossier)[nom]; deja {
		return Demande{}, ErrDejaDemande
	}
	dem := Demande{Action: action, Nom: nom, Par: par, Date: maintenant.UTC()}
	b, _ := json.Marshal(dem)
	rep := filepath.Join(dossier, DossierDemandes)
	fichier := fmt.Sprintf("%d-%s-%s.json", maintenant.UnixMilli(), action, nom)
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
