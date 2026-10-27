package entrainements

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

func ecrire(t *testing.T, chemin, contenu string, date time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(chemin, date, date); err != nil {
		t.Fatal(err)
	}
}

const journal = "epoque,perte,auc_image_ajustement,auc_sein_ajustement,duree_s\n" +
	"1,0.6800,0.7000,0.7200,600\n2,0.6100,0.7400,0.7900,590\n3,0.5900,0.7300,0.7700,595\n4,0.58"

func TestSansEtatDeduitDesDates(t *testing.T) {
	d := t.TempDir()
	rep := filepath.Join(d, "cnn-cmmd-20261027-1315")
	ecrire(t, filepath.Join(rep, "journal.csv"), journal, t0.Add(-30*time.Minute))
	os.Chtimes(rep, t0.Add(-3*time.Hour), t0.Add(-3*time.Hour))

	r, err := Lire(d, "cnn-cmmd-20261027-1315", t0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Statut != EnCours || r.Phase != "entrainement" || r.Epoque != 3 || len(r.Journal) != 3 {
		t.Fatalf("en cours attendu, époque 3 et ligne incomplète ignorée : %+v", r.Resume)
	}
	if r.MeilleureAUCSein == nil || *r.MeilleureAUCSein != 0.79 {
		t.Fatalf("meilleure AUC : %v", r.MeilleureAUCSein)
	}
	r, _ = Lire(d, "cnn-cmmd-20261027-1315", t0.Add(3*time.Hour))
	if r.Statut != SansNouvelle {
		t.Fatalf("sans nouvelles attendu après 2 h de silence : %s", r.Statut)
	}
}

func TestAvecEtatEtRapport(t *testing.T) {
	d := t.TempDir()
	rep := filepath.Join(d, "a")
	ecrire(t, filepath.Join(rep, "journal.csv"), journal, t0)
	ecrire(t, filepath.Join(rep, "etat.json"), `{"statut":"en_cours","phase":"entrainement","etape":"apprentissage",
		"debut":"2026-10-27T10:00:00+00:00","maj":"2026-10-27T11:59:50+00:00","epoque":4,"epoques":15,"pas":120,"pas_par_epoque":660}`, t0)

	r, _ := Lire(d, "a", t0)
	if r.Statut != EnCours || r.Epoque != 4 || r.Epoques != 15 || r.Etape != "apprentissage" || !r.AvecEtat || len(r.Etat) == 0 {
		t.Fatalf("état mal lu : %+v", r.Resume)
	}
	if r2, _ := Lire(d, "a", t0.Add(20*time.Minute)); r2.Statut != SansNouvelle {
		t.Fatalf("15 min sans mise à jour d'etat.json : %s", r2.Statut)
	}

	ecrire(t, filepath.Join(rep, "rapport.json"), `{"validation":{"auc_sein":0.81},"reglages":{"epoques_max":15}}`, t0)
	r, _ = Lire(d, "a", t0.Add(24*time.Hour))
	if r.Statut != Termine || r.AUCSeinValidation == nil || *r.AUCSeinValidation != 0.81 || len(r.Rapport) == 0 {
		t.Fatalf("terminé attendu : %+v", r.Resume)
	}

	ecrire(t, filepath.Join(rep, "etat.json"), `{"statut":"echec","phase":"entrainement","message":"OutOfMemoryError : CUDA"}`, t0)
	if r, _ = Lire(d, "a", t0); r.Statut != Echec || r.Message == "" {
		t.Fatalf("échec attendu : %+v", r.Resume)
	}
}

func TestListeEtNoms(t *testing.T) {
	d := t.TempDir()
	ecrire(t, filepath.Join(d, "ancien", "etat.json"), `{"statut":"termine","phase":"termine","debut":"2026-10-20T10:00:00Z","maj":"2026-10-20T12:00:00Z"}`, t0)
	ecrire(t, filepath.Join(d, "recent", "etat.json"), `{"statut":"en_cours","phase":"calibrage","debut":"2026-10-27T11:00:00Z","maj":"2026-10-27T11:59:00Z"}`, t0)
	ecrire(t, filepath.Join(d, ".cache", "x"), "", t0)
	ecrire(t, filepath.Join(d, "fichier.txt"), "", t0)

	l, err := Lister(d, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 2 || l[0].Nom != "recent" || l[1].Nom != "ancien" {
		t.Fatalf("liste : %+v", l)
	}
	if l, err := Lister(filepath.Join(d, "absent"), t0); err != nil || len(l) != 0 {
		t.Fatalf("dossier absent : %v %v", l, err)
	}
	for _, nom := range []string{"../a", "a/b", "", ".cache", "fichier.txt", "inconnu"} {
		if _, err := Lire(d, nom, t0); !errors.Is(err, ErrIntrouvable) {
			t.Fatalf("%q : %v", nom, err)
		}
	}
}
