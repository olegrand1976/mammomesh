package integration

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/orchestrator"
)

// Repartir d'une campagne finie : contrôles du départ, parents de la génération 1 pris dans
// le front de la génération choisie, départ annoncé au journal.
func TestRepartir(t *testing.T) {
	e := newEnv(t, nil)
	e.user("admin", "mot-de-passe-admin", auth.RoleAdmin)
	admin := e.login("admin", "mot-de-passe-admin")

	// Orchestrateur en arrière-plan pendant tout le test.
	octx, ostop := context.WithCancel(e.ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _ = orchestrator.New(e.st, e.log).Run(octx) }()
	defer func() { ostop(); wg.Wait() }()

	conf := e.call(admin, "GET", "/api/config/default", nil, 200)["config"].(map[string]any)
	conf["evolution"].(map[string]any)["population"] = 40
	conf["evolution"].(map[string]any)["generations_max"] = 12
	conf["simulation"].(map[string]any)["duree_lot_ms"] = 10
	conf["simulation"].(map[string]any)["lots_par_generation"] = 1
	lancer := func(c map[string]any, base any) (int64, map[string]any) {
		t.Helper()
		d := e.call(admin, "POST", "/api/drafts", map[string]any{"config": c, "base_campaign_id": base}, 201)["draft"].(map[string]any)
		v := e.call(admin, "POST", fmt.Sprintf("/api/drafts/%d/verify", int64(d["id"].(float64))), nil, 200)["draft"].(map[string]any)
		if v["verified_ok"] != true {
			return 0, v
		}
		return id(e.call(admin, "POST", fmt.Sprintf("/api/drafts/%d/launch", int64(d["id"].(float64))), nil, 201), "campaign"), v
	}
	attendre := func(cid int64, cond func(map[string]any) bool) map[string]any {
		t.Helper()
		for limite := time.Now().Add(30 * time.Second); time.Now().Before(limite); time.Sleep(50 * time.Millisecond) {
			if c := e.call(admin, "GET", fmt.Sprintf("/api/campaigns/%d", cid), nil, 200)["campaign"].(map[string]any); cond(c) {
				return c
			}
		}
		t.Fatalf("campagne %d : condition non atteinte", cid)
		return nil
	}

	src, _ := lancer(conf, nil)
	// Départ refusé tant que la source tourne.
	enCours := clone(conf)
	enCours["depart"] = map[string]any{"campagne_id": src, "generation": 1}
	if _, v := lancer(enCours, src); v == nil || controle(v, "depart") {
		t.Fatal("départ accepté depuis une campagne en cours")
	}
	fin := attendre(src, func(c map[string]any) bool { return c["state"] == "terminee" })
	finale := int(fin["generation"].(float64))

	// Génération hors bornes refusée.
	hors := clone(conf)
	hors["depart"] = map[string]any{"campagne_id": src, "generation": finale + 1}
	if _, v := lancer(hors, src); controle(v, "depart") {
		t.Fatal("génération hors bornes acceptée")
	}

	// Départ de la génération finale, avec un nouveau réglage.
	suite := clone(conf)
	suite["projet"].(map[string]any)["nom"] = "Suite"
	suite["projet"].(map[string]any)["lignee"] = "LYRA"
	suite["evolution"].(map[string]any)["population"] = 30
	suite["depart"] = map[string]any{"campagne_id": src, "generation": finale}
	fid, v := lancer(suite, src)
	if fid == 0 {
		t.Fatalf("départ valide refusé : %v", v)
	}
	if !controle(v, "depart") {
		t.Fatalf("contrôle du départ absent : %v", v)
	}
	attendre(fid, func(c map[string]any) bool { return c["generation"].(float64) >= 1 })

	var parents []string
	rows, err := e.pool.Query(e.ctx, `SELECT DISTINCT parent_name FROM individuals WHERE campaign_id=$1 AND generation=1`, fid)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var p *string
		_ = rows.Scan(&p)
		if p == nil {
			t.Fatal("individu sans parent à la génération 1 d'une campagne avec départ")
		}
		parents = append(parents, *p)
	}
	front := e.call(admin, "GET", fmt.Sprintf("/api/campaigns/%d/front?generation=%d", src, finale), nil, 200)["front"].([]any)
	var noms []string
	for _, f := range front {
		noms = append(noms, f.(map[string]any)["name"].(string))
	}
	for _, p := range parents {
		if !slices.Contains(noms, p) {
			t.Fatalf("parent %s hors du front de départ %v", p, noms)
		}
	}
	journal := e.call(admin, "GET", fmt.Sprintf("/api/campaigns/%d/events?order=asc&limit=1", fid), nil, 200)["events"].([]any)
	if msg := journal[0].(map[string]any)["message"].(string); !contains(msg, fmt.Sprintf("génération %d de la campagne %d", finale, src)) {
		t.Fatalf("journal de lancement : %s", msg)
	}
}

func clone(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if sub, ok := v.(map[string]any); ok {
			v = clone(sub)
		}
		out[k] = v
	}
	return out
}

// controle dit si le contrôle nommé est présent et réussi dans une vérification.
func controle(draft map[string]any, nom string) bool {
	v, _ := draft["verification"].(map[string]any)
	cs, _ := v["controles"].([]any)
	for _, c := range cs {
		if m := c.(map[string]any); m["nom"] == nom {
			return m["ok"] == true
		}
	}
	return false
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
