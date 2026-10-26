package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/olegrand1976/mammomesh/backend/internal/imagerie"
)

// ImageRow : une image de test et ses métadonnées techniques.
type ImageRow struct {
	ID           int64         `json:"id"`
	Libelle      string        `json:"libelle"`
	Meta         imagerie.Meta `json:"meta"`
	StorageKey   string        `json:"-"`
	TailleOctets int64         `json:"taille_octets"`
	NbTests      int           `json:"nb_tests"`
	CreatedAt    time.Time     `json:"created_at"`
}

// ImageTest : un test d'individu sur une image.
type ImageTest struct {
	ID           int64             `json:"id"`
	ImageID      int64             `json:"image_id"`
	IndividualID int64             `json:"individual_id"`
	Individu     string            `json:"individu"`
	CampaignID   int64             `json:"campaign_id"`
	Generation   int               `json:"generation"`
	Resultat     imagerie.Resultat `json:"resultat"`
	CreatedAt    time.Time         `json:"created_at"`
}

const imageCols = `i.id, i.libelle, i.meta, i.storage_key, i.taille_octets,
	(SELECT count(*) FROM image_tests t WHERE t.image_id = i.id), i.created_at`

func scanImage(r pgx.Row) (ImageRow, error) {
	var im ImageRow
	var meta []byte
	err := r.Scan(&im.ID, &im.Libelle, &meta, &im.StorageKey, &im.TailleOctets, &im.NbTests, &im.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return im, ErrNotFound
	}
	if err == nil {
		err = json.Unmarshal(meta, &im.Meta)
	}
	return im, err
}

// CreateImage enregistre une image ; une image aux mêmes pixels n'est pas dupliquée (existe = vrai).
func (s *Store) CreateImage(ctx context.Context, libelle string, m imagerie.Meta, key string, taille int64, userID int64) (ImageRow, bool, error) {
	mb, _ := json.Marshal(m)
	var id int64
	err := s.Pool.QueryRow(ctx, `INSERT INTO images (pixel_sha256, libelle, meta, storage_key, taille_octets, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (pixel_sha256) DO NOTHING RETURNING id`,
		m.PixelSHA256, libelle, mb, key, taille, userID).Scan(&id)
	existe := errors.Is(err, pgx.ErrNoRows)
	if existe {
		err = s.Pool.QueryRow(ctx, `SELECT id FROM images WHERE pixel_sha256=$1`, m.PixelSHA256).Scan(&id)
	}
	if err != nil {
		return ImageRow{}, false, err
	}
	im, err := s.GetImage(ctx, id)
	return im, existe, err
}

// ListImages : les plus récentes d'abord.
func (s *Store) ListImages(ctx context.Context) ([]ImageRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+imageCols+` FROM images i ORDER BY i.created_at DESC, i.id DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (ImageRow, error) { return scanImage(r) })
}

// GetImage renvoie une image.
func (s *Store) GetImage(ctx context.Context, id int64) (ImageRow, error) {
	return scanImage(s.Pool.QueryRow(ctx, `SELECT `+imageCols+` FROM images i WHERE i.id=$1`, id))
}

// DeleteImage supprime une image et ses tests ; renvoie la clé de stockage à effacer.
func (s *Store) DeleteImage(ctx context.Context, id int64) (string, error) {
	var key string
	err := s.Pool.QueryRow(ctx, `DELETE FROM images WHERE id=$1 RETURNING storage_key`, id).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

// InsertImageTest garde la trace d'un test.
func (s *Store) InsertImageTest(ctx context.Context, imageID, individualID int64, res imagerie.Resultat, userID int64) (int64, error) {
	b, _ := json.Marshal(res)
	var id int64
	err := s.Pool.QueryRow(ctx, `INSERT INTO image_tests (image_id, individual_id, resultat, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`, imageID, individualID, b, userID).Scan(&id)
	return id, err
}

// ListImageTests : tests d'une image, les plus récents d'abord.
func (s *Store) ListImageTests(ctx context.Context, imageID int64) ([]ImageTest, error) {
	rows, err := s.Pool.Query(ctx, `SELECT t.id, t.image_id, t.individual_id, ind.name, ind.campaign_id, ind.generation, t.resultat, t.created_at
		FROM image_tests t JOIN individuals ind ON ind.id = t.individual_id
		WHERE t.image_id=$1 ORDER BY t.created_at DESC, t.id DESC LIMIT 100`, imageID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (ImageTest, error) {
		var t ImageTest
		var res []byte
		err := r.Scan(&t.ID, &t.ImageID, &t.IndividualID, &t.Individu, &t.CampaignID, &t.Generation, &res, &t.CreatedAt)
		if err == nil {
			err = json.Unmarshal(res, &t.Resultat)
		}
		return t, err
	})
}
