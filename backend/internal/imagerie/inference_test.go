package imagerie

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMoteurPredire(t *testing.T) {
	var requete string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer s3cret" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"erreur":"jeton absent ou invalide"}`)
			return
		}
		requete = r.URL.RequestURI()
		b, _ := io.ReadAll(r.Body)
		if string(b) != "pixels" {
			t.Errorf("corps transmis : %q", b)
		}
		_, _ = io.WriteString(w, `{"score":0.72,"seuil":0.46,"signale":true,"duree_ms":1500,"modele":"cnn-x","epoque":4,"fini":true,
			"zones":[{"x":10,"y":20,"largeur":30,"hauteur":40,"score":1,"type":"attention du CNN"}]}`)
	}))
	defer srv.Close()

	m := NewMoteur(srv.URL+"/", "s3cret")
	res, err := m.Predire(context.Background(), Meta{Rows: 4, Cols: 3, BitsStored: 8, Invert: true}, []byte("pixels"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requete, "lignes=4") || !strings.Contains(requete, "colonnes=3") || !strings.Contains(requete, "bits=8") || !strings.Contains(requete, "inverser=1") {
		t.Errorf("requête : %s", requete)
	}
	if res.Simulation || res.Score != 0.72 || res.Modele != "cnn-x" || res.Signale == nil || !*res.Signale || len(res.Zones) != 1 || res.Zones[0].L != 30 {
		t.Errorf("résultat : %+v", res)
	}
	if !strings.Contains(res.Avertissement, "CephaloMesh n'est pas encore branché") {
		t.Errorf("avertissement : %s", res.Avertissement)
	}

	if _, err := NewMoteur(srv.URL, "faux").Predire(context.Background(), Meta{Rows: 1, Cols: 1}, nil); err == nil || !strings.Contains(err.Error(), "jeton") {
		t.Errorf("erreur attendue avec le message du service, reçu %v", err)
	}
}
