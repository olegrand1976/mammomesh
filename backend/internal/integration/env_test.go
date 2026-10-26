package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olegrand1976/mammomesh/backend/internal/api"
	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/config"
	"github.com/olegrand1976/mammomesh/backend/internal/db"
	"github.com/olegrand1976/mammomesh/backend/internal/imagerie"
	"github.com/olegrand1976/mammomesh/backend/internal/realtime"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// env : base remise à zéro, relais temps réel et API sur un serveur HTTP de test.
type env struct {
	t     *testing.T
	ctx   context.Context
	log   *slog.Logger
	pool  *pgxpool.Pool
	st    *store.Store
	hub   *realtime.Hub
	redis *auth.Redis
	srv   *httptest.Server
	mux   http.Handler
}

// newEnv exige DATABASE_URL_TEST ; rdb peut être nil (compteurs en mémoire).
func newEnv(t *testing.T, rdb *auth.Redis) *env {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_TEST non défini")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	st := store.New(pool, "poivre-de-test")
	cfg := config.Config{SessionTTL: time.Hour, CoalesceInterval: 150 * time.Millisecond, MCPTokenPepper: "poivre-de-test",
		AllowedOrigins: []string{"https://mammomesh.lan"}}
	hub := realtime.NewHub(st, log, cfg.CoalesceInterval)
	hubCtx, hubStop := context.WithCancel(ctx)
	go hub.Run(hubCtx)
	mux := api.New(cfg, st, hub, log, api.Limits{Redis: rdb, LoginPerUser: auth.NewLimiter(rdb, 10, time.Minute),
		LoginPerIP: auth.NewLimiter(rdb, 50, time.Minute), MCPPerToken: auth.NewLimiter(rdb, 1000, time.Minute)}).
		WithImages(imagerie.DirStore{Dir: t.TempDir()}).Router()
	srv := httptest.NewServer(mux)
	// Ordre de fermeture : serveur, relais (il rend sa connexion d'écoute), puis base.
	t.Cleanup(func() { srv.Close(); hubStop(); pool.Close(); cancel() })
	return &env{t: t, ctx: ctx, log: log, pool: pool, st: st, hub: hub, redis: rdb, srv: srv, mux: mux}
}

func (e *env) user(name, password, role string) {
	e.t.Helper()
	if _, err := e.st.CreateUser(e.ctx, name, "", password, role); err != nil {
		e.t.Fatal(err)
	}
}

// tryLogin renvoie un client à cookies et le statut de la connexion.
func (e *env) tryLogin(user, pw string) (*http.Client, int) {
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	resp, err := c.Post(e.srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, user, pw)))
	if err != nil {
		e.t.Fatal(err)
	}
	resp.Body.Close()
	return c, resp.StatusCode
}

func (e *env) login(user, pw string) *http.Client {
	e.t.Helper()
	c, code := e.tryLogin(user, pw)
	if code != http.StatusOK {
		e.t.Fatalf("connexion %s : statut %d", user, code)
	}
	return c
}

// call envoie une requête JSON et vérifie le statut ; c nil = sans cookie.
func (e *env) call(c *http.Client, method, path string, body any, want int) map[string]any {
	e.t.Helper()
	if c == nil {
		c = http.DefaultClient
	}
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(e.ctx, method, e.srv.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != want {
		e.t.Fatalf("%s %s : statut %d, attendu %d (%v)", method, path, resp.StatusCode, want, out)
	}
	return out
}

func id(m map[string]any, key string) int64 { return int64(m[key].(map[string]any)["id"].(float64)) }
