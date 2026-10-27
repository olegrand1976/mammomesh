package imagerie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------- Inférence réelle : le service du moteur (python -m mammomesh_engine.inference) ----------

// Moteur appelle le service d'inférence du CNN entraîné.
type Moteur struct {
	URL    string // par exemple http://host.docker.internal:8765
	Jeton  string // facultatif : INFERENCE_JETON, envoyé en Bearer
	Client *http.Client
}

// NewMoteur prépare le client ; une inférence sur CPU prend quelques secondes.
func NewMoteur(base, jeton string) *Moteur {
	return &Moteur{URL: strings.TrimRight(base, "/"), Jeton: jeton, Client: &http.Client{Timeout: 90 * time.Second}}
}

// EtatMoteur : modèle chargé par le service.
type EtatMoteur struct {
	OK       bool     `json:"ok"`
	Modele   string   `json:"modele"`
	Epoque   *int     `json:"epoque"`
	Seuil    *float64 `json:"seuil"`
	Fini     bool     `json:"fini"`
	Appareil string   `json:"appareil"`
}

func (m *Moteur) requete(ctx context.Context, method, chemin string, corps []byte) (*http.Response, error) {
	var body io.Reader
	if corps != nil {
		body = bytes.NewReader(corps)
	}
	req, err := http.NewRequestWithContext(ctx, method, m.URL+chemin, body)
	if err != nil {
		return nil, err
	}
	if corps != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if m.Jeton != "" {
		req.Header.Set("Authorization", "Bearer "+m.Jeton)
	}
	return m.Client.Do(req)
}

// lire décode une réponse JSON ; une erreur du service garde son message.
func lire(resp *http.Response, dest any) error {
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Erreur string `json:"erreur"`
		}
		if json.Unmarshal(b, &e) == nil && e.Erreur != "" {
			return fmt.Errorf("moteur d'inférence : %s", e.Erreur)
		}
		return fmt.Errorf("moteur d'inférence : %s", resp.Status)
	}
	return json.Unmarshal(b, dest)
}

// Etat interroge /sante.
func (m *Moteur) Etat(ctx context.Context) (EtatMoteur, error) {
	var e EtatMoteur
	resp, err := m.requete(ctx, http.MethodGet, "/sante", nil)
	if err != nil {
		return e, err
	}
	return e, lire(resp, &e)
}

// Predire envoie les pixels stockés (uint16 petit-boutiste, gzip) et renvoie le résultat du CNN.
func (m *Moteur) Predire(ctx context.Context, meta Meta, pixelsGzip []byte) (Resultat, error) {
	q := url.Values{}
	q.Set("lignes", strconv.Itoa(meta.Rows))
	q.Set("colonnes", strconv.Itoa(meta.Cols))
	q.Set("bits", strconv.Itoa(meta.BitsStored))
	if meta.Invert {
		q.Set("inverser", "1")
	}
	resp, err := m.requete(ctx, http.MethodPost, "/predire?"+q.Encode(), pixelsGzip)
	if err != nil {
		return Resultat{}, err
	}
	var r struct {
		Score   float64  `json:"score"`
		Seuil   *float64 `json:"seuil"`
		Signale *bool    `json:"signale"`
		Zones   []Zone   `json:"zones"`
		DureeMs float64  `json:"duree_ms"`
		Modele  string   `json:"modele"`
		Epoque  *int     `json:"epoque"`
		Fini    bool     `json:"fini"`
	}
	if err := lire(resp, &r); err != nil {
		return Resultat{}, err
	}
	if r.Zones == nil {
		r.Zones = []Zone{}
	}
	return Resultat{
		Simulation: false, Score: r.Score, DureeMs: r.DureeMs, Zones: r.Zones,
		Modele: r.Modele, Epoque: r.Epoque, Seuil: r.Seuil, Signale: r.Signale,
		Avertissement: avertissementCNN(r.Modele, r.Epoque, r.Fini),
	}, nil
}

func avertissementCNN(modele string, epoque *int, fini bool) string {
	ep := ""
	if epoque != nil {
		ep = fmt.Sprintf(", époque %d", *epoque)
	}
	cours := ""
	if !fini {
		cours = " Entraînement encore en cours : meilleure version à ce jour."
	}
	return fmt.Sprintf("Score du CNN seul (modèle %s%s). CephaloMesh n'est pas encore branché : l'individu choisi n'influence pas ce résultat. "+
		"Les zones montrent où le CNN a regardé, ce ne sont pas des lésions détectées. Outil de recherche, sans valeur diagnostique.%s", modele, ep, cours)
}
