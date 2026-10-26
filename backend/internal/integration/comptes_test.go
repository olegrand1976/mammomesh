package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/olegrand1976/mammomesh/backend/internal/api"
	"github.com/olegrand1976/mammomesh/backend/internal/auth"
)

// Premier administrateur au mot de passe par défaut : tout est fermé jusqu'au changement,
// qui ferme les autres sessions et invalide l'ancien mot de passe.
func TestDefaultAdmin(t *testing.T) {
	e := newEnv(t, nil)
	if _, err := e.st.CreateDefaultAdmin(e.ctx, "admin"); err != nil {
		t.Fatal(err)
	}
	a := e.login("admin", auth.DefaultAdminPassword)
	other := e.login("admin", auth.DefaultAdminPassword)

	me := e.call(a, "GET", "/api/me", nil, 200)["user"].(map[string]any)
	if me["must_change_password"] != true || me["role"] != auth.RoleAdmin {
		t.Fatalf("profil : %v", me)
	}
	e.call(a, "GET", "/api/campaigns", nil, 403)
	e.call(a, "POST", "/api/mcp-tokens", map[string]any{"name": "x", "scope": "lecture"}, 403)

	pw := func(cur, next string, want int) {
		t.Helper()
		e.call(a, "POST", "/api/auth/password", map[string]any{"actuel": cur, "nouveau": next}, want)
	}
	pw("faux", "un-mot-de-passe-solide", 403)
	pw(auth.DefaultAdminPassword, "court", 400)
	pw(auth.DefaultAdminPassword, auth.DefaultAdminPassword, 400)
	pw(auth.DefaultAdminPassword, "un-mot-de-passe-solide", 204)

	if me := e.call(a, "GET", "/api/me", nil, 200)["user"].(map[string]any); me["must_change_password"] != false {
		t.Fatalf("drapeau non levé : %v", me)
	}
	e.call(a, "GET", "/api/campaigns", nil, 200)
	e.call(other, "GET", "/api/me", nil, 401) // autre session fermée
	if _, code := e.tryLogin("admin", auth.DefaultAdminPassword); code != 401 {
		t.Fatalf("ancien mot de passe accepté : %d", code)
	}
	e.login("admin", "un-mot-de-passe-solide")

	var n int
	if err := e.pool.QueryRow(e.ctx, `SELECT count(*) FROM audit_log WHERE action='mot_de_passe_change'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit du changement : %d %v", n, err)
	}

	// Déconnexion : la session ne vaut plus rien.
	e.call(a, "POST", "/api/auth/logout", nil, 204)
	e.call(a, "GET", "/api/me", nil, 401)
}

// Santé, métriques, origine refusée et limitation des tentatives (avec Redis si REDIS_URL_TEST).
func TestAccessAndLimits(t *testing.T) {
	var rdb *auth.Redis
	if url := os.Getenv("REDIS_URL_TEST"); url != "" {
		opt, err := redis.ParseURL(url)
		if err != nil {
			t.Fatal(err)
		}
		c := redis.NewClient(opt)
		t.Cleanup(func() { c.Close() })
		c.Del(context.Background(), "mammomesh:limite:login:user:lecteur")
		rdb = auth.NewRedis(c, nil)
		rdb.Probe(context.Background())
	}
	e := newEnv(t, rdb)
	e.user("lecteur", "mot-de-passe-lecteur", auth.RoleReader)

	h := e.call(nil, "GET", "/api/health", nil, 200)
	want := "desactive"
	if rdb != nil {
		want = "ok"
	}
	if h["etat"] != "ok" || h["redis"] != want {
		t.Fatalf("santé : %v", h)
	}

	rec := httptest.NewRecorder()
	api.MetricsHandler(e.hub, rdb).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	for _, m := range []string{"mammomesh_sse_clients", "mammomesh_notify_lag_ms", "mammomesh_redis_up"} {
		if !strings.Contains(string(body), "\n"+m+" ") {
			t.Fatalf("métrique %s absente", m)
		}
	}

	// Origine étrangère refusée, origine déclarée acceptée.
	login := func(origin string) int {
		req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/login",
			strings.NewReader(`{"username":"lecteur","password":"mot-de-passe-lecteur"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := login("https://ailleurs.example"); code != 403 {
		t.Fatalf("origine étrangère : %d", code)
	}
	if code := login("https://mammomesh.lan"); code != 200 {
		t.Fatalf("origine déclarée : %d", code)
	}

	// 10 échecs par identifiant, puis 429 même avec le bon mot de passe.
	start := time.Now()
	for i := 0; i < 10; i++ {
		if _, code := e.tryLogin("lecteur", "mauvais-mot-de-passe"); code != 401 {
			t.Fatalf("échec %d : %d", i+1, code)
		}
	}
	if _, code := e.tryLogin("lecteur", "mot-de-passe-lecteur"); code != 429 {
		t.Fatalf("11e tentative : %d, attendu 429", code)
	}
	if rdb != nil {
		var n int
		n, _ = rdb.Client.Get(e.ctx, "mammomesh:limite:login:user:lecteur").Int()
		if n < 11 {
			t.Fatalf("compteur Redis : %d", n)
		}
		rdb.Client.Del(e.ctx, "mammomesh:limite:login:user:lecteur", "mammomesh:limite:login:ip:127.0.0.1")
	}
	t.Logf("11 connexions en %v (Redis : %s)", time.Since(start), rdb.State())
}
