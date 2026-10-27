package imagerie

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------- Test simulé d'un individu sur une image ----------

// Zone : zone signalée, en pixels de l'image.
type Zone struct {
	X     int     `json:"x"`
	Y     int     `json:"y"`
	L     int     `json:"largeur"`
	H     int     `json:"hauteur"`
	Score float64 `json:"score"`
	Type  string  `json:"type"`
}

// Resultat d'un test : simulé (aucun moteur configuré) ou score du CNN entraîné (INFERENCE_URL).
type Resultat struct {
	Simulation    bool     `json:"simulation"`
	Score         float64  `json:"score"` // simulé, ou probabilité de malignité donnée par le CNN
	DureeMs       float64  `json:"duree_ms"`
	Zones         []Zone   `json:"zones"`
	Avertissement string   `json:"avertissement"`
	Modele        string   `json:"modele,omitempty"`  // CNN : dossier du modèle utilisé
	Epoque        *int     `json:"epoque,omitempty"`  // CNN : époque retenue
	Seuil         *float64 `json:"seuil,omitempty"`   // CNN : seuil choisi à l'entraînement (sensibilité cible)
	Signale       *bool    `json:"signale,omitempty"` // CNN : score au-dessus du seuil
}

// Individu : ce que le test utilise d'un individu.
type Individu struct {
	ID          int64
	Sensibilite float64 // E-ajust
	CoutMs      float64
}

var typesZone = []string{"masse", "microcalcifications", "distorsion architecturale", "asymétrie"}

// Simuler produit un résultat déterministe : même image et même individu, même résultat.
// Un individu plus sensible signale davantage de zones, avec des scores plus élevés.
func Simuler(m Meta, ind Individu) Resultat {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d", m.PixelSHA256, ind.ID)
	r := rand.New(rand.NewPCG(h.Sum64(), uint64(ind.ID)))
	gain := math.Max(0, (ind.Sensibilite-0.80)/0.15) // 0 à ~1,2
	n := r.IntN(3)
	if r.Float64() < gain*0.6 {
		n++
	}
	tx, ty, tl, th := m.Tissue[0], m.Tissue[1], m.Tissue[2], m.Tissue[3]
	out := Resultat{Simulation: true, Zones: []Zone{},
		Avertissement: "Simulation : le CNN et CephaloMesh ne sont pas branchés ; ce résultat n'a aucune valeur, ni diagnostique ni de recherche."}
	for k := 0; k < n; k++ {
		taille := int(float64(min(tl, th)) * (0.04 + 0.06*r.Float64()))
		z := Zone{
			X:     tx + int(float64(tl)*(0.15+0.7*r.Float64())) - taille/2,
			Y:     ty + int(float64(th)*(0.15+0.7*r.Float64())) - taille/2,
			L:     max(taille, 8),
			H:     max(taille, 8),
			Score: math.Round(math.Min(0.99, 0.15+0.5*r.Float64()+0.3*gain*r.Float64())*1000) / 1000,
			Type:  typesZone[r.IntN(len(typesZone))],
		}
		out.Zones = append(out.Zones, z)
		out.Score = math.Max(out.Score, z.Score)
	}
	if n == 0 {
		out.Score = math.Round((0.02+0.08*r.Float64())*1000) / 1000
	}
	out.DureeMs = math.Round(ind.CoutMs*(0.9+0.2*r.Float64())*10) / 10
	return out
}

// ---------- Stockage des pixels ----------

// Store garde les pixels compressés (gzip) sous une clé.
type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// ErrAbsent : objet introuvable dans le stockage.
var ErrAbsent = errors.New("image absente du stockage")

// DirStore : un dossier local (développement, tests).
type DirStore struct{ Dir string }

func (d DirStore) path(key string) string { return filepath.Join(d.Dir, filepath.Base(key)) }

func (d DirStore) Put(_ context.Context, key string, data []byte) error {
	if err := os.MkdirAll(d.Dir, 0o750); err != nil {
		return err
	}
	tmp := d.path(key) + ".part"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, d.path(key))
}

func (d DirStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(d.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrAbsent
	}
	return f, err
}

func (d DirStore) Delete(_ context.Context, key string) error {
	err := os.Remove(d.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// FilerStore : le filer HTTP de SeaweedFS (PUT, GET, DELETE sur un chemin), sur le réseau interne.
type FilerStore struct {
	Base   string // par exemple http://s3:8888/mammomesh-images
	Client *http.Client
}

func (f FilerStore) url(key string) string {
	return strings.TrimRight(f.Base, "/") + "/" + filepath.Base(key)
}

func (f FilerStore) do(ctx context.Context, method, key string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, f.url(key), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	c := f.Client
	if c == nil {
		c = defaultFilerClient
	}
	return c.Do(req)
}

// Le client ne demande ni ne défait aucune compression : les octets stockés sont déjà en gzip
// et doivent revenir tels quels.
var defaultFilerClient = &http.Client{Timeout: 2 * time.Minute, Transport: &http.Transport{DisableCompression: true}}

// NewFilerStore prépare un FilerStore avec le client adapté.
func NewFilerStore(base string) FilerStore { return FilerStore{Base: base, Client: defaultFilerClient} }

// Put écrit l'objet ; une erreur 5xx (SeaweedFS encore en démarrage) est retentée deux fois.
func (f FilerStore) Put(ctx context.Context, key string, data []byte) error {
	var last error
	for essai := 0; essai < 3; essai++ {
		if essai > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(essai) * time.Second):
			}
		}
		resp, err := f.do(ctx, http.MethodPut, key, data)
		if err != nil {
			last = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		last = fmt.Errorf("stockage : %s", resp.Status)
		if resp.StatusCode < 500 {
			return last
		}
	}
	return last
}

func (f FilerStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := f.do(ctx, http.MethodGet, key, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrAbsent
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("stockage : %s", resp.Status)
	}
	return resp.Body, nil
}

func (f FilerStore) Delete(ctx context.Context, key string) error {
	resp, err := f.do(ctx, http.MethodDelete, key, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("stockage : %s", resp.Status)
	}
	return nil
}
