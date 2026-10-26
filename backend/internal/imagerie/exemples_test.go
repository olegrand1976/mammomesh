package imagerie

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExemples importe les DICOM de examples/ (voir examples/README.md) et vérifie le résultat
// attendu du manifeste. Ignoré tant que ./examples/telecharger.sh n'a pas été lancé.
func TestExemples(t *testing.T) {
	racine := filepath.Join("..", "..", "..", "examples")
	brut, err := os.ReadFile(filepath.Join(racine, "manifeste.json"))
	if err != nil {
		t.Skipf("manifeste absent : %v", err)
	}
	var m struct {
		Github []struct {
			Fichier string `json:"fichier"`
			Attendu struct {
				Statut     string `json:"statut"`
				Motif      string `json:"motif"`
				Modalite   string `json:"modalite"`
				Vue        string `json:"vue"`
				Lateralite string `json:"lateralite"`
				Cols       int    `json:"cols"`
				Rows       int    `json:"rows"`
				Inverse    bool   `json:"inverse"`
			} `json:"attendu"`
		} `json:"github"`
	}
	if err := json.Unmarshal(brut, &m); err != nil {
		t.Fatalf("manifeste illisible : %v", err)
	}
	lire := func(chemin string) (Image, error) {
		f, err := os.Open(chemin)
		if err != nil {
			return Image{}, err
		}
		defer f.Close()
		st, _ := f.Stat()
		return Read(f, st.Size())
	}

	trouves := 0
	for _, e := range m.Github {
		chemin := filepath.Join(racine, e.Fichier)
		if _, err := os.Stat(chemin); err != nil {
			continue
		}
		trouves++
		t.Run(filepath.Base(e.Fichier), func(t *testing.T) {
			img, err := lire(chemin)
			a := e.Attendu
			if a.Statut == "refuse" {
				if err == nil || !strings.Contains(err.Error(), a.Motif) {
					t.Fatalf("refus attendu (« %s »), obtenu : %v", a.Motif, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}
			if img.Modality != a.Modalite || img.Cols != a.Cols || img.Rows != a.Rows || img.Invert != a.Inverse ||
				(a.Vue != "" && img.View != a.Vue) || (a.Lateralite != "" && img.Laterality != a.Lateralite) {
				t.Fatalf("métadonnées inattendues : %+v", img.Meta)
			}
			if len(img.Pixels) != img.Cols*img.Rows {
				t.Fatalf("%d pixels pour %d × %d", len(img.Pixels), img.Cols, img.Rows)
			}
		})
	}

	tcia, _ := filepath.Glob(filepath.Join(racine, "tcia", "*", "*.dcm"))
	for _, chemin := range tcia {
		trouves++
		t.Run("tcia/"+filepath.Base(chemin), func(t *testing.T) {
			img, err := lire(chemin)
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}
			if img.Modality != "MG" {
				t.Fatalf("modalité %q, MG attendue", img.Modality)
			}
			t.Logf("%s %s %d × %d, %d bits, %s", img.View, img.Laterality, img.Cols, img.Rows, img.BitsStored, img.TransferSyntax)
		})
	}
	if trouves == 0 {
		t.Skip("aucun fichier : lancez ./examples/telecharger.sh")
	}
}
