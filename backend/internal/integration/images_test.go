package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
)

// envoyer importe un DICOM en multipart et renvoie le statut et la réponse.
func envoyer(t *testing.T, e *env, c *http.Client, dicom []byte, libelle string) (int, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("fichier", "DUPONT_Marie_1970.dcm") // nom identifiant : jamais gardé
	_, _ = fw.Write(dicom)
	if libelle != "" {
		_ = mw.WriteField("libelle", libelle)
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/images", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// Import pseudonymisé, dédoublonnage, pixels servis compressés, test simulé tracé, droits, purge.
func TestImages(t *testing.T) {
	e := newEnv(t, nil)
	e.user("admin", "mot-de-passe-admin", auth.RoleAdmin)
	e.user("radio", "mot-de-passe-radio", auth.RoleRadiologist)
	e.user("lecteur", "mot-de-passe-lecteur", auth.RoleReader)
	admin, radio, lecteur := e.login("admin", "mot-de-passe-admin"), e.login("radio", "mot-de-passe-radio"), e.login("lecteur", "mot-de-passe-lecteur")

	// Exemple téléchargeable, puis importé tel quel.
	resp, err := radio.Get(e.srv.URL + "/api/images/exemple.dcm?vue=MLO&cote=R")
	if err != nil || resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Disposition"), "exemple-synthetique-R-MLO") {
		t.Fatalf("exemple : %v %v", err, resp)
	}
	dcm, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if code, _ := envoyer(t, e, lecteur, dcm, ""); code != 403 {
		t.Fatalf("import par un lecteur : %d", code)
	}
	if code, body := envoyer(t, e, radio, []byte("pas un dicom"), ""); code != 422 || !strings.Contains(body, "DICOM") {
		t.Fatalf("fichier invalide : %d %s", code, body)
	}
	code, body := envoyer(t, e, radio, dcm, "")
	if code != 201 {
		t.Fatalf("import : %d %s", code, body)
	}
	for _, phi := range []string{"EXEMPLE", "SYN-0001", "19700101", "DUPONT", "Centre fictif"} {
		if strings.Contains(body, phi) {
			t.Fatalf("donnée identifiante renvoyée : %s", phi)
		}
	}
	if code, body := envoyer(t, e, radio, dcm, "autre nom"); code != 200 || !strings.Contains(body, `"existe":true`) {
		t.Fatalf("doublon : %d %s", code, body)
	}

	list := e.call(lecteur, "GET", "/api/images", nil, 200)["images"].([]any)
	if len(list) != 1 {
		t.Fatalf("images : %v", list)
	}
	im := list[0].(map[string]any)
	imID := int64(im["id"].(float64))
	meta := im["meta"].(map[string]any)
	if im["libelle"] != "MG R MLO" || meta["view"] != "MLO" || meta["laterality"] != "R" || meta["rows"].(float64) != 1280 {
		t.Fatalf("image : %v", im)
	}
	var phiDB int
	_ = e.pool.QueryRow(e.ctx, `SELECT count(*) FROM images WHERE meta::text ~ 'EXEMPLE|SYN-0001|19700101|DUPONT' OR libelle ~ 'DUPONT'`).Scan(&phiDB)
	if phiDB != 0 {
		t.Fatal("donnée identifiante en base")
	}

	// Pixels : uint16 petit-boutiste, décompressés par le client HTTP.
	resp, err = lecteur.Get(fmt.Sprintf("%s/api/images/%d/pixels", e.srv.URL, imID))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("pixels : %v %v", err, resp)
	}
	px, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(px) != 1280*1024*2 {
		t.Fatalf("%d octets de pixels", len(px))
	}

	// Test simulé d'un individu (créé à la main, sans orchestrateur).
	var campID, indID int64
	err = e.pool.QueryRow(e.ctx, `INSERT INTO campaigns (name, lineage, config, config_hash, state, max_generations, budget_total)
		VALUES ('Images', 'ORION', '{}', 'x', 'terminee', 10, 1) RETURNING id`).Scan(&campID)
	if err == nil {
		err = e.pool.QueryRow(e.ctx, `INSERT INTO individuals (campaign_id, generation, name, mutations, genome, sensitivity,
			sensitivity_low, sensitivity_high, sensitivity_eval, cost_ms, feasible, on_front)
			VALUES ($1, 1, 'ORION-G01.01', '{}', '{}', 0.9, 0.88, 0.92, 0.89, 250, true, true) RETURNING id`, campID).Scan(&indID)
	}
	if err != nil {
		t.Fatal(err)
	}
	e.call(lecteur, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": indID}, 403)
	t1 := e.call(radio, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": indID}, 201)["test"].(map[string]any)
	t2 := e.call(radio, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": indID}, 201)["test"].(map[string]any)
	r1, r2 := t1["resultat"].(map[string]any), t2["resultat"].(map[string]any)
	if r1["simulation"] != true || fmt.Sprint(r1["zones"]) != fmt.Sprint(r2["zones"]) || t1["individu"] != "ORION-G01.01" {
		t.Fatalf("test simulé : %v / %v", t1, t2)
	}
	e.call(radio, "POST", fmt.Sprintf("/api/images/%d/tests", imID), map[string]any{"individual_id": 999999}, 404)
	if hist := e.call(lecteur, "GET", fmt.Sprintf("/api/images/%d/tests", imID), nil, 200)["tests"].([]any); len(hist) != 2 {
		t.Fatalf("historique : %d tests", len(hist))
	}

	// Purge : réservée, efface aussi les tests et les pixels.
	e.call(radio, "DELETE", fmt.Sprintf("/api/images/%d", imID), nil, 403)
	e.call(admin, "DELETE", fmt.Sprintf("/api/images/%d", imID), nil, 204)
	e.call(lecteur, "GET", fmt.Sprintf("/api/images/%d", imID), nil, 404)
	var n int
	_ = e.pool.QueryRow(e.ctx, `SELECT count(*) FROM image_tests`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d tests restants", n)
	}
	var audit int
	_ = e.pool.QueryRow(e.ctx, `SELECT count(*) FROM audit_log WHERE action LIKE 'image.%'`).Scan(&audit)
	if audit < 4 {
		t.Fatalf("audit des images : %d lignes", audit)
	}
}
