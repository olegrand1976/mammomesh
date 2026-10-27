package entrainements

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDemandes(t *testing.T) {
	d := t.TempDir()
	ecrire(t, filepath.Join(d, "fini", "rapport.json"), `{"validation":{"auc_sein":0.8}}`, t0)
	ecrire(t, filepath.Join(d, "fini", "parametres.json"), `{}`, t0)
	ecrire(t, filepath.Join(d, "actif", "etat.json"), `{"statut":"en_cours","maj":"2026-10-27T11:59:00Z"}`, t0)

	if _, err := Demander(d, "fini", Relancer, "marie", t0); !errors.Is(err, ErrDemandesFermees) {
		t.Fatalf("dossier des demandes absent : %v", err)
	}
	if err := os.MkdirAll(filepath.Join(d, DossierDemandes), 0o777); err != nil {
		t.Fatal(err)
	}
	ecrire(t, filepath.Join(d, DossierDemandes, ".gitkeep"), "", t0)

	if _, err := Demander(d, "fini", "effacer-tout", "marie", t0); !errors.Is(err, ErrActionInconnue) {
		t.Fatalf("action inconnue : %v", err)
	}
	if _, err := Demander(d, "../x", Supprimer, "marie", t0); !errors.Is(err, ErrIntrouvable) {
		t.Fatalf("nom invalide : %v", err)
	}
	if _, err := Demander(d, "actif", Relancer, "marie", t0); !errors.Is(err, ErrEnCours) {
		t.Fatalf("relancer un entraînement en cours : %v", err)
	}
	dem, err := Demander(d, "fini", Relancer, "marie", t0)
	if err != nil || dem.Par != "marie" || dem.Action != Relancer {
		t.Fatalf("demande : %+v %v", dem, err)
	}
	if _, err := Demander(d, "fini", Supprimer, "marie", t0.Add(time.Second)); !errors.Is(err, ErrDejaDemande) {
		t.Fatalf("deuxième demande : %v", err)
	}
	if _, err := Demander(d, "actif", Supprimer, "paul", t0); err != nil {
		t.Fatalf("supprimer un entraînement en cours (le veilleur l'arrête d'abord) : %v", err)
	}

	l, _ := Lister(d, t0)
	vus := map[string]string{}
	for _, r := range l {
		if r.Demande != nil {
			vus[r.Nom] = r.Demande.Action
		}
	}
	if vus["fini"] != Relancer || vus["actif"] != Supprimer || len(vus) != 2 {
		t.Fatalf("demandes dans la liste : %v", vus)
	}
	r, _ := Lire(d, "fini", t0)
	if r.Demande == nil || !r.AvecParametres {
		t.Fatalf("détail : %+v", r.Resume)
	}
	if r, _ := Lire(d, "actif", t0); r.AvecParametres {
		t.Fatal("pas de parametres.json")
	}
}
