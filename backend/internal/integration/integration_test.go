// Test de bout en bout sur une vraie base PostgreSQL : connexion, brouillon, lancement,
// orchestrateur, flux SSE avec reprise, outils MCP et journal d'audit.
// Lancé seulement si DATABASE_URL_TEST est défini ; la base désignée est remise à zéro.
package integration

import (
	"bufio"
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
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/olegrand1976/mammomesh/backend/internal/api"
	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/config"
	"github.com/olegrand1976/mammomesh/backend/internal/db"
	"github.com/olegrand1976/mammomesh/backend/internal/orchestrator"
	"github.com/olegrand1976/mammomesh/backend/internal/realtime"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

type sseEvent struct {
	ID   string
	Type string
	Data string
}

func readSSE(t *testing.T, body io.Reader, out chan<- sseEvent) {
	sc := bufio.NewScanner(body)
	var ev sseEvent
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if ev.Type != "" {
				out <- ev
			}
			ev = sseEvent{}
		case strings.HasPrefix(line, "id: "):
			ev.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			ev.Type = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			ev.Data += strings.TrimPrefix(line, "data: ")
		}
	}
	close(out)
}

type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

func TestEndToEnd(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_TEST non défini")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	st := store.New(pool, "poivre-de-test")
	if _, err := st.CreateUser(ctx, "admin", "Admin", "mot-de-passe-admin", auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, "lecteur", "Lecteur", "mot-de-passe-lecteur", auth.RoleReader); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{SessionTTL: time.Hour, CoalesceInterval: 150 * time.Millisecond, MCPTokenPepper: "poivre-de-test"}
	hub := realtime.NewHub(st, log, cfg.CoalesceInterval)
	hubCtx, hubStop := context.WithCancel(ctx)
	defer hubStop() // avant pool.Close : le relais rend sa connexion d'écoute
	go hub.Run(hubCtx)
	srv := httptest.NewServer(api.New(cfg, st, hub, log, auth.NewLimiter(nil, 10, time.Minute), auth.NewLimiter(nil, 1000, time.Minute)).Router())
	defer srv.Close()

	login := func(user, pw string) *http.Client {
		jar, _ := cookiejar.New(nil)
		c := &http.Client{Jar: jar}
		resp, err := c.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, user, pw)))
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("connexion %s : %v %v", user, err, resp.Status)
		}
		resp.Body.Close()
		return c
	}
	call := func(c *http.Client, method, path string, body any, want int) map[string]any {
		t.Helper()
		var rd io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		}
		req, _ := http.NewRequestWithContext(ctx, method, srv.URL+path, rd)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if resp.StatusCode != want {
			t.Fatalf("%s %s : statut %d, attendu %d (%v)", method, path, resp.StatusCode, want, out)
		}
		return out
	}

	// Mauvais mot de passe refusé, session absente refusée.
	resp, _ := http.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader(`{"username":"admin","password":"faux-mot-de-passe"}`))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("mauvais mot de passe : %d", resp.StatusCode)
	}
	resp, _ = http.Get(srv.URL + "/api/campaigns")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("accès sans session : %d", resp.StatusCode)
	}

	admin := login("admin", "mot-de-passe-admin")
	reader := login("lecteur", "mot-de-passe-lecteur")

	// Brouillon → contrôles → lancement, avec une simulation rapide.
	conf := call(admin, "GET", "/api/config/default", nil, 200)["config"].(map[string]any)
	conf["evolution"].(map[string]any)["population"] = 30
	conf["evolution"].(map[string]any)["generations_max"] = 12
	conf["simulation"].(map[string]any)["duree_lot_ms"] = 40
	conf["simulation"].(map[string]any)["lots_par_generation"] = 3
	draft := call(admin, "POST", "/api/drafts", map[string]any{"config": conf}, 201)["draft"].(map[string]any)
	draftID := int64(draft["id"].(float64))
	call(admin, "POST", fmt.Sprintf("/api/drafts/%d/launch", draftID), nil, 409) // pas encore vérifié
	call(reader, "POST", "/api/drafts", map[string]any{"config": conf}, 403)
	v := call(admin, "POST", fmt.Sprintf("/api/drafts/%d/verify", draftID), nil, 200)["draft"].(map[string]any)
	if v["verified_ok"] != true {
		t.Fatalf("contrôles en échec : %v", v)
	}
	camp := call(admin, "POST", fmt.Sprintf("/api/drafts/%d/launch", draftID), nil, 201)["campaign"].(map[string]any)
	cid := int64(camp["id"].(float64))
	call(admin, "POST", fmt.Sprintf("/api/drafts/%d/launch", draftID), nil, 409) // une seule fois

	// Flux SSE ouvert avant le démarrage de l'orchestrateur.
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/campaigns/%d/stream", srv.URL, cid), nil)
	sresp, err := reader.Do(req)
	if err != nil || sresp.StatusCode != 200 {
		t.Fatalf("flux : %v %v", err, sresp)
	}
	events := make(chan sseEvent, 1000)
	go readSSE(t, sresp.Body, events)

	octx, ostop := context.WithCancel(ctx)
	var owg sync.WaitGroup
	owg.Add(1)
	go func() { defer owg.Done(); _ = orchestrator.New(st, log).Run(octx) }()

	counts := map[string]int{}
	var ids []string
	deadline := time.After(60 * time.Second)
wait:
	for {
		select {
		case e, ok := <-events:
			if !ok {
				t.Fatal("flux fermé trop tôt")
			}
			counts[e.Type]++
			if e.ID != "" {
				ids = append(ids, e.ID)
			}
			if e.Type == "campagne_terminee" {
				break wait
			}
		case <-deadline:
			t.Fatalf("campagne non terminée à temps ; reçus : %v", counts)
		}
	}
	ostop()
	owg.Wait()
	sresp.Body.Close()

	if counts["generation_terminee"] == 0 || counts["progression"] == 0 || counts["nouveau_meilleur"] == 0 {
		t.Fatalf("événements manquants : %v", counts)
	}
	c := call(reader, "GET", fmt.Sprintf("/api/campaigns/%d", cid), nil, 200)["campaign"].(map[string]any)
	if c["state"] != "terminee" {
		t.Fatalf("état final %v", c["state"])
	}
	gens := call(reader, "GET", fmt.Sprintf("/api/campaigns/%d/generations", cid), nil, 200)["generations"].([]any)
	if len(gens) != counts["generation_terminee"] {
		t.Fatalf("%d générations en base, %d annoncées", len(gens), counts["generation_terminee"])
	}

	// Reprise : depuis le 3e événement, le flux rejoue exactement la suite du journal.
	req, _ = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/campaigns/%d/stream", srv.URL, cid), nil)
	req.Header.Set("Last-Event-ID", ids[2])
	rresp, err := reader.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	replay := make(chan sseEvent, 1000)
	go readSSE(t, rresp.Body, replay)
	var got []string
	timeout := time.After(3 * time.Second)
collect:
	for len(got) < len(ids)-3 {
		select {
		case e := <-replay:
			if e.ID != "" {
				got = append(got, e.ID)
			}
		case <-timeout:
			break collect
		}
	}
	rresp.Body.Close()
	if strings.Join(got, ",") != strings.Join(ids[3:], ",") {
		t.Fatalf("reprise incorrecte :\nreçu    %v\nattendu %v", got, ids[3:])
	}

	// Réglage à chaud refusé au lecteur, état terminé non modifiable.
	call(reader, "POST", fmt.Sprintf("/api/campaigns/%d/control", cid), map[string]any{"action": "pause"}, 403)
	call(admin, "POST", fmt.Sprintf("/api/campaigns/%d/control", cid), map[string]any{"action": "pause"}, 409)

	// MCP : jeton en lecture, puis jeton « proposition ».
	readTok := call(admin, "POST", "/api/mcp-tokens", map[string]any{"name": "test lecture", "scope": "lecture"}, 201)["token"].(string)
	call(reader, "POST", "/api/mcp-tokens", map[string]any{"name": "x", "scope": "proposition"}, 403)
	propTok := call(admin, "POST", "/api/mcp-tokens", map[string]any{"name": "test proposition", "scope": "proposition"}, 201)["token"].(string)

	resp, _ = http.Post(srv.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/mcp sans jeton : %d", resp.StatusCode)
	}

	connect := func(token string) *mcp.ClientSession {
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp",
			HTTPClient: &http.Client{Transport: bearer{token: token, base: http.DefaultTransport}}}, nil)
		if err != nil {
			t.Fatalf("connexion MCP : %v", err)
		}
		return cs
	}
	tool := func(cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		return res
	}

	rs := connect(readTok)
	tools, err := rs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 10 {
		t.Fatalf("outils MCP : %v %v", err, tools)
	}
	if res := tool(rs, "campagnes_lister", nil); res.IsError {
		t.Fatalf("campagnes_lister : %v", res.Content)
	}
	front := tool(rs, "front_pareto", map[string]any{"campagne_id": cid})
	if front.IsError || len(front.StructuredContent.(map[string]any)["individus"].([]any)) == 0 {
		t.Fatalf("front_pareto : %+v", front)
	}
	j := tool(rs, "journal_lire", map[string]any{"campagne_id": cid, "types": []string{"nouveau_meilleur"}, "limite": 2})
	if lines := j.StructuredContent.(map[string]any)["evenements"].([]any); len(lines) == 0 || len(lines) > 2 {
		t.Fatalf("journal_lire : %v", lines)
	}
	if res := tool(rs, "config_brouillon", map[string]any{"base_campagne_id": cid, "reglages": map[string]any{"evolution.population": 60}}); !res.IsError {
		t.Fatal("un jeton en lecture a créé un brouillon")
	}
	rep, err := rs.ReadResource(ctx, &mcp.ReadResourceParams{URI: fmt.Sprintf("mammomesh://campagne/%d/rapport", cid)})
	if err != nil || !strings.Contains(rep.Contents[0].Text, "Campagne") {
		t.Fatalf("ressource rapport : %v", err)
	}
	rs.Close()

	ps := connect(propTok)
	d := tool(ps, "config_brouillon", map[string]any{"base_campagne_id": cid, "reglages": map[string]any{"evolution.population": 60, "simulation.duree_lot_ms": 20}})
	if d.IsError {
		t.Fatalf("config_brouillon : %v", d.Content)
	}
	newDraft := int64(d.StructuredContent.(map[string]any)["id"].(float64))
	if diff := d.StructuredContent.(map[string]any)["diff"].([]any); len(diff) != 2 {
		t.Fatalf("diff du brouillon : %v", diff)
	}
	if res := tool(ps, "campagne_lancer", map[string]any{"brouillon_id": newDraft}); !res.IsError {
		t.Fatal("brouillon non vérifié lancé")
	}
	if res := tool(ps, "config_verifier", map[string]any{"brouillon_id": newDraft}); res.IsError {
		t.Fatalf("config_verifier : %v", res.Content)
	}
	launched := tool(ps, "campagne_lancer", map[string]any{"brouillon_id": newDraft})
	if launched.IsError {
		t.Fatalf("campagne_lancer : %v", launched.Content)
	}
	newID := int64(launched.StructuredContent.(map[string]any)["id"].(float64))
	if res := tool(ps, "campagne_reglage", map[string]any{"campagne_id": newID, "action": "arret"}); res.IsError {
		t.Fatalf("campagne_reglage : %v", res.Content)
	}
	ps.Close()

	var mcpAudit int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE source='mcp'`).Scan(&mcpAudit); err != nil || mcpAudit < 8 {
		t.Fatalf("journal d'audit MCP : %d lignes, %v", mcpAudit, err)
	}

	// Jeton révoqué : refusé.
	tokens := call(admin, "GET", "/api/mcp-tokens", nil, 200)["tokens"].([]any)
	for _, tk := range tokens {
		call(admin, "DELETE", fmt.Sprintf("/api/mcp-tokens/%d", int64(tk.(map[string]any)["id"].(float64))), nil, 204)
	}
	r2, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{}`))
	r2.Header.Set("Authorization", "Bearer "+propTok)
	resp, _ = http.DefaultClient.Do(r2)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("jeton révoqué accepté : %d", resp.StatusCode)
	}
}
