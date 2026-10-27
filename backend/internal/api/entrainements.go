package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/entrainements"
)

// Suivi des entraînements du CNN : lecture seule de MODELES_DIR, ouvert à tous les rôles.

func (s *Server) listEntrainements(w http.ResponseWriter, r *http.Request) {
	commun := map[string]any{"reglages_defaut": entrainements.ReglagesParDefaut(), "resolutions": entrainements.Resolutions}
	if s.cfg.ModelesDir == "" {
		commun["configure"], commun["entrainements"], commun["nouveaux_en_attente"] = false, []entrainements.Resume{}, []entrainements.Demande{}
		writeJSON(w, http.StatusOK, commun)
		return
	}
	l, err := entrainements.Lister(s.cfg.ModelesDir, time.Now())
	if err != nil {
		s.log.Warn("dossier des entraînements illisible", "dossier", s.cfg.ModelesDir, "err", err)
		writeError(w, http.StatusServiceUnavailable, "dossier des entraînements illisible")
		return
	}
	commun["configure"], commun["entrainements"], commun["nouveaux_en_attente"] = true, l, entrainements.NouveauxEnAttente(s.cfg.ModelesDir)
	writeJSON(w, http.StatusOK, commun)
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

// demanderEntrainement dépose une demande (relancer, supprimer) que le veilleur de l'hôte exécutera.
func (s *Server) demanderEntrainement(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanManageTrainings(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	if s.cfg.ModelesDir == "" {
		writeError(w, http.StatusNotFound, "suivi des entraînements non configuré (MODELES_DIR)")
		return
	}
	var in struct {
		Action string `json:"action"`
		Mode   string `json:"mode"` // pause : fin_epoque (défaut) ou maintenant
	}
	if !readJSON(w, r, &in) {
		return
	}
	nom := chi.URLParam(r, "nom")
	dem, err := entrainements.DemanderMode(s.cfg.ModelesDir, nom, in.Action, in.Mode, u.Username, time.Now())
	switch {
	case errors.Is(err, entrainements.ErrIntrouvable):
		writeError(w, http.StatusNotFound, "introuvable")
	case errors.Is(err, entrainements.ErrActionInconnue):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, entrainements.ErrEnCours), errors.Is(err, entrainements.ErrDejaDemande),
		errors.Is(err, entrainements.ErrPasEnCours), errors.Is(err, entrainements.ErrSansReprise):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, entrainements.ErrDemandesFermees):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case err != nil:
		s.fail(w, err)
	default:
		s.st.Audit(r.Context(), u.ID, "interface", "entrainement."+dem.Action, map[string]any{"entrainement": nom, "mode": dem.Mode})
		writeJSON(w, http.StatusAccepted, map[string]any{"demande": dem})
	}
}

// nouvelEntrainement dépose la demande d'un nouvel entraînement ; le veilleur le crée et le lance
// dès que le GPU est libre.
func (s *Server) nouvelEntrainement(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanManageTrainings(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	if s.cfg.ModelesDir == "" {
		writeError(w, http.StatusNotFound, "suivi des entraînements non configuré (MODELES_DIR)")
		return
	}
	in := entrainements.ReglagesParDefaut()
	if !readJSON(w, r, &in) {
		return
	}
	dem, err := entrainements.DemanderNouveau(s.cfg.ModelesDir, in, u.Username, time.Now())
	var invalide entrainements.InvalideError
	switch {
	case errors.As(err, &invalide):
		writeError(w, http.StatusBadRequest, invalide.Error())
	case errors.Is(err, entrainements.ErrDemandesFermees):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case err != nil:
		s.fail(w, err)
	default:
		s.st.Audit(r.Context(), u.ID, "interface", "entrainement.nouveau", in)
		writeJSON(w, http.StatusAccepted, map[string]any{"demande": dem})
	}
}
