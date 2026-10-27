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

func TestNouveau(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, DossierDemandes), 0o777); err != nil {
		t.Fatal(err)
	}
	r := ReglagesParDefaut()
	if err := r.Verifier(); err != nil {
		t.Fatalf("réglages par défaut refusés : %v", err)
	}
	for _, mauvais := range []func(*Reglages){
		func(r *Reglages) { r.Epoques = 0 }, func(r *Reglages) { r.Resolution = "4000x3000" },
		func(r *Reglages) { r.Patience = 99 }, func(r *Reglages) { r.LR = 1 },
		func(r *Reglages) { r.SensibiliteCible = 1 }, func(r *Reglages) { r.Limite = 3 }, func(r *Reglages) { r.Graine = -1 },
	} {
		x := ReglagesParDefaut()
		mauvais(&x)
		var inv InvalideError
		if _, err := DemanderNouveau(d, x, "marie", t0); !errors.As(err, &inv) {
			t.Fatalf("réglages %+v acceptés : %v", x, err)
		}
	}
	r.Limite, r.Epoques = 100, 2
	dem, err := DemanderNouveau(d, r, "marie", t0)
	if err != nil || dem.Reglages == nil || dem.Reglages.Epoques != 2 || dem.Nom != "" {
		t.Fatalf("nouveau : %+v %v", dem, err)
	}
	if _, err := DemanderNouveau(d, r, "paul", t0.Add(time.Second)); err != nil {
		t.Fatalf("deux nouveaux à la suite : %v", err)
	}
	if n := NouveauxEnAttente(d); len(n) != 2 || n[0].Par != "marie" {
		t.Fatalf("nouveaux en attente : %+v", n)
	}
	if len(Demandes(d)) != 0 {
		t.Fatal("un nouveau ne se rattache à aucun entraînement existant")
	}
}
