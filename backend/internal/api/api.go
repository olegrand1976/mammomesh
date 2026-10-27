// Package api expose l'API REST, le flux SSE et le serveur MCP.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/campaign"
	"github.com/olegrand1976/mammomesh/backend/internal/config"
	"github.com/olegrand1976/mammomesh/backend/internal/imagerie"
	"github.com/olegrand1976/mammomesh/backend/internal/mcpserver"
	"github.com/olegrand1976/mammomesh/backend/internal/realtime"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

const cookieName = "mm_session"

// Server regroupe les dépendances des gestionnaires.
type Server struct {
	cfg        config.Config
	st         *store.Store
	hub        *realtime.Hub
	log        *slog.Logger
	redis      *auth.Redis   // état des compteurs, affiché par /api/health
	loginLimit *auth.Limiter // par identifiant
	ipLimit    *auth.Limiter // par adresse : plus large, plusieurs personnes peuvent partager un poste
	mcpLimit   *auth.Limiter
	images     imagerie.Store // pixels des images de test ; nil : routes d'images indisponibles
}

// Limits regroupe les limiteurs de débit.
type Limits struct {
	Redis        *auth.Redis // peut être nil : compteurs en mémoire
	LoginPerUser *auth.Limiter
	LoginPerIP   *auth.Limiter
	MCPPerToken  *auth.Limiter
}

// New crée le serveur d'API.
func New(cfg config.Config, st *store.Store, hub *realtime.Hub, log *slog.Logger, l Limits) *Server {
	return &Server{cfg: cfg, st: st, hub: hub, log: log, redis: l.Redis, loginLimit: l.LoginPerUser, ipLimit: l.LoginPerIP, mcpLimit: l.MCPPerToken}
}

// Router construit les routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.st.Pool.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "base injoignable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"etat": "ok", "redis": s.redis.State()})
	})

	r.Group(func(r chi.Router) {
		r.Use(s.sameOrigin)
		r.Post("/api/auth/login", s.login)
		r.Post("/api/auth/logout", s.logout)

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Get("/api/me", s.me)
			r.Post("/api/auth/password", s.changePassword)
			r.Get("/api/campaigns", s.listCampaigns)
			r.Get("/api/campaigns/{id}", s.getCampaign)
			r.Get("/api/campaigns/{id}/generations", s.listGenerations)
			r.Get("/api/campaigns/{id}/front", s.front)
			r.Get("/api/campaigns/{id}/events", s.listEvents)
			r.Get("/api/campaigns/{id}/stream", s.stream)
			r.Post("/api/campaigns/{id}/control", s.control)
			r.Get("/api/individuals/{id}", s.getIndividual)
			r.Get("/api/config/default", s.defaultConfig)
			r.Post("/api/drafts", s.createDraft)
			r.Get("/api/drafts/{id}", s.getDraft)
			r.Post("/api/drafts/{id}/verify", s.verifyDraft)
			r.Post("/api/drafts/{id}/launch", s.launchDraft)
			r.Get("/api/mcp-tokens", s.listTokens)
			r.Post("/api/mcp-tokens", s.createToken)
			r.Delete("/api/mcp-tokens/{id}", s.revokeToken)
			r.Get("/api/images", s.listImages)
			r.Post("/api/images", s.importImage)
			r.Get("/api/images/exemple.dcm", s.exempleDICOM)
			r.Get("/api/images/{id}", s.getImage)
			r.Get("/api/images/{id}/pixels", s.imagePixels)
			r.Delete("/api/images/{id}", s.deleteImage)
			r.Get("/api/images/{id}/tests", s.listImageTests)
			r.Post("/api/images/{id}/tests", s.testImage)
			r.Get("/api/entrainements", s.listEntrainements)
			r.Get("/api/entrainements/{nom}", s.getEntrainement)
		})
	})

	mcpHandler := s.requireMCPToken(mcpserver.Handler(s.st))
	r.Handle("/mcp", mcpHandler)
	r.Handle("/mcp/*", mcpHandler)
	return r
}

// ---------- Intergiciels ----------

type userKey struct{}

func userFrom(ctx context.Context) *auth.User {
	u, _ := ctx.Value(userKey{}).(*auth.User)
	return u
}

// sameOrigin refuse les requêtes d'écriture venues d'une autre origine (en plus du cookie SameSite=Strict).
func (s *Server) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if o := r.Header.Get("Origin"); o != "" && !s.originAllowed(o, r) {
				writeError(w, http.StatusForbidden, "origine refusée")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string, r *http.Request) bool {
	for _, a := range s.cfg.AllowedOrigins {
		if origin == a {
			return true
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return origin == "https://"+host || origin == "http://"+host
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "connexion requise")
			return
		}
		u, err := s.st.SessionUser(r.Context(), c.Value, s.cfg.SessionTTL)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session expirée")
			return
		}
		// Mot de passe par défaut : seuls le profil et le changement de mot de passe restent ouverts.
		if u.MustChangePassword && r.URL.Path != "/api/me" && r.URL.Path != "/api/auth/password" {
			writeError(w, http.StatusForbidden, "changez d'abord le mot de passe par défaut")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// requireMCPToken vérifie l'en-tête Authorization: Bearer, l'origine et le débit.
func (s *Server) requireMCPToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && !s.originAllowed(o, r) {
			writeError(w, http.StatusForbidden, "origine refusée")
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mammomesh"`)
			writeError(w, http.StatusUnauthorized, "jeton MCP requis (en-tête Authorization: Bearer)")
			return
		}
		key := "mcp:" + fmt.Sprintf("%x", auth.HashToken(token, s.cfg.MCPTokenPepper))[:16]
		if !s.mcpLimit.Allow(r.Context(), key) {
			writeError(w, http.StatusTooManyRequests, "trop d'appels MCP ; réessayer dans une minute")
			return
		}
		p, err := s.st.ResolveMCPToken(r.Context(), token)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mammomesh", error="invalid_token"`)
			writeError(w, http.StatusUnauthorized, "jeton MCP invalide, expiré ou révoqué")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r.WithContext(mcpserver.WithPrincipal(r.Context(), p)))
	})
}

// ---------- Authentification ----------

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	ctx := r.Context()
	userKey, ipKey := "login:user:"+strings.ToLower(in.Username), "login:ip:"+clientIP(r)
	if !s.loginLimit.Allow(ctx, userKey) || !s.ipLimit.Allow(ctx, ipKey) {
		writeError(w, http.StatusTooManyRequests, "trop de tentatives ; réessayer dans 15 minutes")
		return
	}
	u, err := s.st.Authenticate(ctx, in.Username, in.Password)
	if err != nil {
		s.st.Audit(ctx, 0, "interface", "connexion_refusee", map[string]any{"identifiant": in.Username, "ip": clientIP(r)})
		writeError(w, http.StatusUnauthorized, "identifiant ou mot de passe incorrect")
		return
	}
	s.loginLimit.Reset(ctx, userKey)
	token, err := s.st.CreateSession(ctx, u.ID, s.cfg.SessionTTL)
	if err != nil {
		s.fail(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: int(s.cfg.SessionTTL.Seconds())})
	s.st.Audit(ctx, u.ID, "interface", "connexion", map[string]any{"ip": clientIP(r)})
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = s.st.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"actuel"`
		Next    string `json:"nouveau"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	u, ctx := userFrom(r.Context()), r.Context()
	key := "password:user:" + strconv.FormatInt(u.ID, 10)
	if !s.loginLimit.Allow(ctx, key) {
		writeError(w, http.StatusTooManyRequests, "trop de tentatives ; réessayer dans 15 minutes")
		return
	}
	c, _ := r.Cookie(cookieName)
	switch err := s.st.ChangePassword(ctx, u.ID, in.Current, in.Next, c.Value); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusForbidden, "mot de passe actuel incorrect")
	case errors.Is(err, auth.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		s.fail(w, err)
	default:
		s.loginLimit.Reset(ctx, key)
		s.st.Audit(ctx, u.ID, "interface", "mot_de_passe_change", map[string]any{"ip": clientIP(r)})
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "permissions": map[string]bool{
		"lancer_campagnes":   auth.CanRunCampaigns(u.Role),
		"proposer_config":    auth.CanProposeConfig(u.Role),
		"jetons_proposition": auth.CanProposeConfig(u.Role),
		"tester_images":      auth.CanTestImages(u.Role),
		"purger_images":      auth.CanPurgeImages(u.Role),
	}})
}

// ---------- Campagnes ----------

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return 0, false
	}
	return id, true
}

func queryInt(r *http.Request, key string, def int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return v
}

func (s *Server) listCampaigns(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListCampaigns(r.Context(), 100)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": list})
}

func (s *Server) getCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	// Dernier événement : point de départ du flux pour le navigateur.
	var lastEvent int64
	_ = s.st.Pool.QueryRow(r.Context(), `SELECT COALESCE(max(id), 0) FROM events WHERE campaign_id=$1`, id).Scan(&lastEvent)
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c, "last_event_id": lastEvent})
}

func (s *Server) listGenerations(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	gens, err := s.st.ListGenerations(r.Context(), id, queryInt(r, "after", 0), min(max(queryInt(r, "limit", 2000), 1), 2000))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"generations": gens})
}

func (s *Server) front(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	g, list, err := s.st.Front(r.Context(), id, queryInt(r, "generation", 0))
	if err != nil {
		s.fail(w, err)
		return
	}
	out := map[string]any{"generation": g, "front": list}
	if g > 1 {
		_, prev, err := s.st.Front(r.Context(), id, g-1)
		if err == nil {
			out["previous"] = prev
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	limit := min(max(queryInt(r, "limit", 100), 1), 500)
	events, err := s.st.ListEvents(r.Context(), id, int64(queryInt(r, "after", 0)), limit, r.URL.Query().Get("order") != "asc")
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.st.GetCampaign(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	token, _ := r.Cookie(cookieName)
	s.hub.Serve(w, r, id, func(ctx context.Context) bool {
		_, err := s.st.SessionUser(ctx, token.Value, s.cfg.SessionTTL)
		return err == nil
	})
}

func (s *Server) control(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanRunCampaigns(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Action       string  `json:"action"`
		HausseBudget float64 `json:"hausse_budget"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	c, err := s.st.Control(r.Context(), id, in.Action, in.HausseBudget)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "campagne."+in.Action, map[string]any{"campagne": id, "hausse_budget": in.HausseBudget})
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}

func (s *Server) getIndividual(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	i, err := s.st.GetIndividual(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"individual": i})
}

// ---------- Brouillons ----------

func (s *Server) defaultConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"config": campaign.Default()})
}

func (s *Server) createDraft(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanProposeConfig(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	var in struct {
		BaseCampaignID *int64          `json:"base_campaign_id"`
		Config         campaign.Config `json:"config"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	d, err := s.st.CreateDraft(r.Context(), u.ID, "interface", in.BaseCampaignID, in.Config)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "brouillon.creation", map[string]any{"brouillon": d.ID})
	writeJSON(w, http.StatusCreated, map[string]any{"draft": d})
}

func (s *Server) getDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, err := s.st.GetDraft(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (s *Server) verifyDraft(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanProposeConfig(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, err := s.st.VerifyDraft(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (s *Server) launchDraft(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanRunCampaigns(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.st.LaunchDraft(r.Context(), id, u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "campagne.lancement", map[string]any{"brouillon": id, "campagne": c.ID})
	writeJSON(w, http.StatusCreated, map[string]any{"campaign": c})
}

// ---------- Jetons MCP ----------

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListMCPTokens(r.Context(), userFrom(r.Context()).ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": list})
}

func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var in struct {
		Name  string `json:"name"`
		Scope string `json:"scope"`
		Days  int    `json:"days"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "nom du jeton requis")
		return
	}
	if in.Scope != "lecture" && in.Scope != "proposition" {
		writeError(w, http.StatusBadRequest, "portée : lecture ou proposition")
		return
	}
	if in.Scope == "proposition" && !auth.CanProposeConfig(u.Role) {
		writeError(w, http.StatusForbidden, "votre rôle ne permet que des jetons en lecture")
		return
	}
	days := min(max(in.Days, 1), 90)
	if in.Days == 0 {
		days = 90
	}
	token, t, err := s.st.CreateMCPToken(r.Context(), u.ID, in.Name, in.Scope, time.Duration(days)*24*time.Hour)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "jeton_mcp.creation", map[string]any{"jeton": t.ID, "portee": t.Scope})
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "info": t})
}

func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.RevokeMCPToken(r.Context(), u.ID, id); err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "jeton_mcp.revocation", map[string]any{"jeton": id})
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Utilitaires ----------

func (s *Server) fail(w http.ResponseWriter, err error) {
	var invalid store.InvalidError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "introuvable")
	case errors.Is(err, store.ErrNotVerified), errors.Is(err, store.ErrAlreadyLaunched), errors.Is(err, store.ErrBadTransition):
		writeError(w, http.StatusConflict, err.Error())
	default:
		s.log.Error("erreur interne", "err", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide : "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"erreur": msg})
}
