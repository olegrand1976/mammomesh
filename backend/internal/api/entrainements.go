package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/olegrand1976/mammomesh/backend/internal/entrainements"
)

// Suivi des entraînements du CNN : lecture seule de MODELES_DIR, ouvert à tous les rôles.

func (s *Server) listEntrainements(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ModelesDir == "" {
		writeJSON(w, http.StatusOK, map[string]any{"configure": false, "entrainements": []entrainements.Resume{}})
		return
	}
	l, err := entrainements.Lister(s.cfg.ModelesDir, time.Now())
	if err != nil {
		s.log.Warn("dossier des entraînements illisible", "dossier", s.cfg.ModelesDir, "err", err)
		writeError(w, http.StatusServiceUnavailable, "dossier des entraînements illisible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configure": true, "entrainements": l})
}

func (s *Server) getEntrainement(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ModelesDir == "" {
		writeError(w, http.StatusNotFound, "suivi des entraînements non configuré (MODELES_DIR)")
		return
	}
	d, err := entrainements.Lire(s.cfg.ModelesDir, chi.URLParam(r, "nom"), time.Now())
	if errors.Is(err, entrainements.ErrIntrouvable) {
		writeError(w, http.StatusNotFound, "introuvable")
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entrainement": d})
}
