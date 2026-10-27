package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/imagerie"
)

// campagneAvecIndividu crée une campagne dans l'état donné, avec un individu du front.
func campagneAvecIndividu(t *testing.T, e *env, nom, etat string) (int64, int64) {
	t.Helper()
	var campID, indID int64
	err := e.pool.QueryRow(e.ctx, `INSERT INTO campaigns (name, lineage, config, config_hash, state, generation, max_generations, budget_total)
		VALUES ($1, 'ORION', '{}', 'x', $2, 1, 10, 1) RETURNING id`, nom, etat).Scan(&campID)
	if err == nil {
		err = e.pool.QueryRow(e.ctx, `INSERT INTO individuals (campaign_id, generation, name, mutations, genome, sensitivity,
			sensitivity_low, sensitivity_high, sensitivity_eval, cost_ms, feasible, on_front)
			VALUES ($1, 1, $2, '{}', '{}', 0.9, 0.88, 0.92, 0.89, 250, true, true) RETURNING id`, campID, nom+"-G01.01").Scan(&indID)
	}
	if err != nil {
		t.Fatal(err)
	}
	return campID, indID
}

func noms(m map[string]any) string {
	var out []string
	for _, c := range m["campaigns"].([]any) {
		out = append(out, c.(map[string]any)["name"].(string))
	}
	return strings.Join(out, ",")
}

// Désactiver : la campagne sort des listes et des départs, reste consultable ; supprimer : tout part avec elle.
func TestCampagnesDesactiverSupprimer(t *testing.T) {
	e := newEnv(t, nil)
	e.user("admin", "mot-de-passe-admin", auth.RoleAdmin)
	e.user("radio", "mot-de-passe-radio", auth.RoleRadiologist)
	admin, radio := e.login("admin", "mot-de-passe-admin"), e.login("radio", "mot-de-passe-radio")
	enCours, _ := campagneAvecIndividu(t, e, "Active", "en_cours")
	finie, indFini := campagneAvecIndividu(t, e, "Finie", "terminee")

	ctl := func(c *http.Client, id int64, action string, want int) map[string]any {
		return e.call(c, "POST", fmt.Sprintf("/api/campaigns/%d/control", id), map[string]any{"action": action}, want)
	}
	ctl(radio, finie, "desactiver", 403)
	ctl(admin, enCours, "desactiver", 400)
	c := ctl(admin, finie, "desactiver", 200)["campaign"].(map[string]any)
	if c["disabled_at"] == nil {
		t.Fatalf("désactivation : %v", c)
	}
	if l := noms(e.call(radio, "GET", "/api/campaigns", nil, 200)); l != "Active" {
		t.Fatalf("liste par défaut : %s", l)
	}
	if l := noms(e.call(radio, "GET", "/api/campaigns?desactivees=1", nil, 200)); l != "Finie,Active" {
		t.Fatalf("liste complète : %s", l)
	}
	e.call(radio, "GET", fmt.Sprintf("/api/campaigns/%d", finie), nil, 200)
	conf := e.call(admin, "GET", "/api/config/default", nil, 200)["config"].(map[string]any)
	conf["depart"] = map[string]any{"campagne_id": finie, "generation": 1}
	d := e.call(admin, "POST", "/api/drafts", map[string]any{"config": conf}, 201)["draft"].(map[string]any)
	v := e.call(admin, "POST", fmt.Sprintf("/api/drafts/%d/verify", int64(d["id"].(float64))), nil, 200)["draft"].(map[string]any)
	if controle(v, "depart") || !strings.Contains(fmt.Sprint(v["verification"]), "désactivée") {
		t.Fatalf("départ depuis une campagne désactivée accepté : %v", v["verification"])
	}
	if c := ctl(admin, finie, "reactiver", 200)["campaign"].(map[string]any); c["disabled_at"] != nil {
		t.Fatalf("réactivation : %v", c)
	}

	// Suppression : refusée en cours, puis tout part (individus, tests d'images), les brouillons perdent le lien.
	var adminID int64
	_ = e.pool.QueryRow(e.ctx, `SELECT id FROM users WHERE username='admin'`).Scan(&adminID)
	if _, err := e.pool.Exec(e.ctx, `INSERT INTO config_drafts (base_campaign_id, config, author_id, source) VALUES ($1, '{}', $2, 'interface')`, finie, adminID); err != nil {
		t.Fatal(err)
	}
	var imID int64
	if err := e.pool.QueryRow(e.ctx, `INSERT INTO images (pixel_sha256, libelle, meta, storage_key, taille_octets) VALUES ('abc', 'x', '{}', 'k', 1) RETURNING id`).Scan(&imID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(e.ctx, `INSERT INTO image_tests (image_id, individual_id, resultat) VALUES ($1, $2, '{}')`, imID, indFini); err != nil {
		t.Fatal(err)
	}
	e.call(radio, "DELETE", fmt.Sprintf("/api/campaigns/%d", finie), nil, 403)
	e.call(admin, "DELETE", fmt.Sprintf("/api/campaigns/%d", enCours), nil, 400)
	e.call(admin, "DELETE", fmt.Sprintf("/api/campaigns/%d", finie), nil, 204)
	e.call(admin, "GET", fmt.Sprintf("/api/campaigns/%d", finie), nil, 404)
	var ind, tests, liens int
	_ = e.pool.QueryRow(e.ctx, `SELECT (SELECT count(*) FROM individuals WHERE campaign_id=$1), (SELECT count(*) FROM image_tests),
		(SELECT count(*) FROM config_drafts WHERE base_campaign_id IS NOT NULL)`, finie).Scan(&ind, &tests, &liens)
	if ind != 0 || tests != 0 || liens != 0 {
		t.Fatalf("après suppression : %d individus, %d tests, %d liens de brouillon", ind, tests, liens)
	}
}

// Test d'une image par le CNN : pixels transmis au service d'inférence, résultat réel tracé.
func TestImagesAvecMoteur(t *testing.T) {
	var vu string
	faux := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sante":
			_, _ = w.Write([]byte(`{"ok":true,"modele":"cnn-x","epoque":3,"seuil":0.5,"fini":true,"appareil":"cpu"}`))
		case "/predire":
			vu = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"score":0.81,"seuil":0.5,"signale":true,"duree_ms":900,"modele":"cnn-x","epoque":3,"fini":true,"zones":[]}`))
		}
	}))
	defer faux.Close()
	e := newEnvMoteur(t, nil, imagerie.NewMoteur(faux.URL, ""))
	e.user("radio", "mot-de-passe-radio", auth.RoleRadiologist)
	radio := e.login("radio", "mot-de-passe-radio")
	resp, err := radio.Get(e.srv.URL + "/api/images/exemple.dcm?vue=CC&cote=L")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("exemple : %v %v", err, resp)
	}
	dcm, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if code, body := envoyer(t, e, radio, dcm, ""); code != 201 {
		t.Fatalf("import : %d %s", code, body)
	}
	imID := int64(e.call(radio, "GET", "/api/images", nil, 200)["images"].([]any)[0].(map[string]any)["id"].(float64))
	_, indID := campagneAvecIndividu(t, e, "C", "terminee")

	if etat := e.call(radio, "GET", "/api/inference", nil, 200); etat["mode"] != "cnn" || etat["joignable"] != true {
		t.Fatalf("état de l'inférence : %v", etat)
	}
	r := e.call(radio, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": indID}, 201)["test"].(map[string]any)["resultat"].(map[string]any)
	if r["simulation"] != false || r["score"] != 0.81 || r["modele"] != "cnn-x" || r["signale"] != true {
		t.Fatalf("résultat : %v", r)
	}
	if !strings.Contains(vu, "lignes=1280") || !strings.Contains(vu, "colonnes=1024") {
		t.Fatalf("paramètres transmis : %s", vu)
	}
	faux.Close()
	e.call(radio, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": indID}, 502)
	if etat := e.call(radio, "GET", "/api/inference", nil, 200); etat["joignable"] != false {
		t.Fatalf("moteur arrêté : %v", etat)
	}
}
