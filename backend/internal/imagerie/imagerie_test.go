package imagerie

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

func exemple(t *testing.T, vue, cote string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Exemple(&buf, vue, cote, 7); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// L'import ne garde que pixels et métadonnées techniques : aucune donnée patient.
func TestReadPseudonymise(t *testing.T) {
	b := exemple(t, "MLO", "L")
	if !bytes.Contains(b, []byte("EXEMPLE^SYNTHETIQUE")) {
		t.Fatal("l'exemple devrait porter de fausses données patient")
	}
	img, err := Read(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if img.Rows != 1280 || img.Cols != 1024 || len(img.Pixels) != 1280*1024 {
		t.Fatalf("dimensions %d × %d, %d pixels", img.Cols, img.Rows, len(img.Pixels))
	}
	if img.Modality != "MG" || img.View != "MLO" || img.Laterality != "L" || img.Invert {
		t.Fatalf("métadonnées : %+v", img.Meta)
	}
	if img.WindowWidth != 3200 || img.TransferSyntax == "" || len(img.PixelSHA256) != 64 {
		t.Fatalf("fenêtre ou syntaxe : %+v", img.Meta)
	}
	meta, _ := json.Marshal(img.Meta)
	for _, phi := range []string{"EXEMPLE", "SYNTHETIQUE", "SYN-0001", "19700101", "20260101", "Centre fictif", "1.2.826"} {
		if strings.Contains(string(meta), phi) {
			t.Fatalf("donnée identifiante conservée : %s dans %s", phi, meta)
		}
	}
	// Zone de tissu : à gauche de l'image pour un sein gauche, dans les bornes.
	tx, ty, tl, th := img.Tissue[0], img.Tissue[1], img.Tissue[2], img.Tissue[3]
	if tl < 400 || th < 800 || tx+tl > img.Cols || ty+th > img.Rows {
		t.Fatalf("zone de tissu : %v", img.Tissue)
	}
	// Même pixels, même empreinte.
	again, _ := Read(bytes.NewReader(b), int64(len(b)))
	if again.PixelSHA256 != img.PixelSHA256 {
		t.Fatal("empreinte instable")
	}
}

func TestReadRefus(t *testing.T) {
	if _, err := Read(strings.NewReader("pas un dicom"), 12); !errors.Is(err, ErrNotDICOM) {
		t.Fatalf("fichier quelconque : %v", err)
	}
	// Annotations incrustées : refus.
	b := exemple(t, "CC", "R")
	ds, err := dicom.Parse(bytes.NewReader(b), int64(len(b)), nil)
	if err != nil {
		t.Fatal(err)
	}
	burned, _ := dicom.NewElement(tag.BurnedInAnnotation, []string{"YES"})
	ds.Elements = append(ds.Elements, burned)
	var out bytes.Buffer
	if err := dicom.Write(&out, ds); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(bytes.NewReader(out.Bytes()), int64(out.Len())); !errors.Is(err, ErrBurnedIn) {
		t.Fatalf("annotations incrustées : %v", err)
	}
}

func TestSimulerDeterministe(t *testing.T) {
	b := exemple(t, "CC", "L")
	img, _ := Read(bytes.NewReader(b), int64(len(b)))
	a := Simuler(img.Meta, Individu{ID: 42, Sensibilite: 0.9, CoutMs: 300})
	if !a.Simulation || a.Avertissement == "" {
		t.Fatal("résultat non marqué comme simulé")
	}
	if b2 := Simuler(img.Meta, Individu{ID: 42, Sensibilite: 0.9, CoutMs: 300}); !equalJSON(a, b2) {
		t.Fatal("même image et même individu : résultats différents")
	}
	tx, ty, tl, th := img.Tissue[0], img.Tissue[1], img.Tissue[2], img.Tissue[3]
	for _, z := range a.Zones {
		if z.X < tx-z.L || z.Y < ty-z.H || z.X > tx+tl || z.Y > ty+th || z.Score <= 0 || z.Score >= 1 {
			t.Fatalf("zone hors du tissu ou score invalide : %+v", z)
		}
	}
	// Sur 200 individus, les plus sensibles signalent en moyenne plus de zones.
	var peu, plus float64
	for id := int64(1); id <= 200; id++ {
		peu += float64(len(Simuler(img.Meta, Individu{ID: id, Sensibilite: 0.80, CoutMs: 200}).Zones))
		plus += float64(len(Simuler(img.Meta, Individu{ID: id, Sensibilite: 0.95, CoutMs: 200}).Zones))
	}
	if plus <= peu {
		t.Fatalf("zones moyennes : %v (sensible) contre %v", plus/200, peu/200)
	}
}

func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// Les deux stockages se comportent pareil : écrire, relire, absent, supprimer.
func TestStores(t *testing.T) {
	var mu sync.Mutex
	objets := map[string][]byte{}
	filer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			objets[r.URL.Path] = b
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			b, ok := objets[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(b)
		case http.MethodDelete:
			delete(objets, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer filer.Close()

	for nom, s := range map[string]Store{"dossier": DirStore{Dir: t.TempDir()}, "filer": FilerStore{Base: filer.URL + "/images"}} {
		ctx := context.Background()
		if err := s.Put(ctx, "a.px.gz", []byte("pixels")); err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		rc, err := s.Get(ctx, "a.px.gz")
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		if string(b) != "pixels" {
			t.Fatalf("%s : relu %q", nom, b)
		}
		if err := s.Delete(ctx, "a.px.gz"); err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		if _, err := s.Get(ctx, "a.px.gz"); !errors.Is(err, ErrAbsent) {
			t.Fatalf("%s : après suppression %v", nom, err)
		}
		// Une clé ne sort jamais du dossier.
		if err := s.Put(ctx, "../../evasion", []byte("x")); err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
	}
}

// Contre un vrai filer SeaweedFS (MAMMOMESH_FILER_TEST=http://127.0.0.1:8888/essais) : les octets
// gzip reviennent identiques, même gros (découpés en morceaux par le filer).
func TestFilerReel(t *testing.T) {
	base := os.Getenv("MAMMOMESH_FILER_TEST")
	if base == "" {
		t.Skip("MAMMOMESH_FILER_TEST absent")
	}
	b := exemple(t, "CC", "L")
	img, _ := Read(bytes.NewReader(b), int64(len(b)))
	gz, _ := Gzip(append(img.Pixels, img.Pixels...)) // ~2,6 Mo de pixels, plusieurs morceaux de 4 Mo une fois doublés
	big := bytes.Repeat(gz, 4)
	s := NewFilerStore(base)
	ctx := context.Background()
	if err := s.Put(ctx, "gros.px.gz", big); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Get(ctx, "gros.px.gz")
	if err != nil {
		t.Fatal(err)
	}
	back, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(back, big) {
		t.Fatalf("%d octets relus pour %d écrits", len(back), len(big))
	}
	if err := s.Delete(ctx, "gros.px.gz"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "gros.px.gz"); !errors.Is(err, ErrAbsent) {
		t.Fatalf("après suppression : %v", err)
	}
}
