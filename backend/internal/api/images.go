package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/imagerie"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// Taille maximale d'un DICOM importé (une mammographie fait de 10 à 60 Mo).
const maxDICOM = 300 << 20

// WithImages branche le stockage des pixels ; sans lui, les routes d'images répondent 503.
func (s *Server) WithImages(st imagerie.Store) *Server {
	s.images = st
	return s
}

// WithMoteur branche le service d'inférence du CNN ; sans lui, les tests d'images sont simulés.
func (s *Server) WithMoteur(m *imagerie.Moteur) *Server {
	s.moteur = m
	return s
}

// etatInference dit à l'interface si les tests sont réels (CNN) ou simulés, et quel modèle répond.
func (s *Server) etatInference(w http.ResponseWriter, r *http.Request) {
	if s.moteur == nil {
		writeJSON(w, http.StatusOK, map[string]any{"mode": "simulation"})
		return
	}
	e, err := s.moteur.Etat(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"mode": "cnn", "joignable": false, "erreur": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": "cnn", "joignable": true, "etat": e})
}

func (s *Server) imagesReady(w http.ResponseWriter) bool {
	if s.images == nil {
		writeError(w, http.StatusServiceUnavailable, "stockage des images non configuré (IMAGES_URL ou IMAGES_DIR)")
		return false
	}
	return true
}

// importImage lit un DICOM envoyé en multipart (champ « fichier », « libelle » facultatif).
// Le nom du fichier n'est jamais gardé : il peut contenir le nom d'une patiente.
func (s *Server) importImage(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanTestImages(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	if !s.imagesReady(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDICOM+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "envoi multipart attendu (champ « fichier »)")
		return
	}
	var data bytes.Buffer
	libelle := ""
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "envoi interrompu ou trop gros (300 Mo au plus)")
			return
		}
		switch part.FormName() {
		case "fichier":
			if _, err := io.Copy(&data, io.LimitReader(part, maxDICOM+1)); err != nil {
				writeError(w, http.StatusBadRequest, "envoi interrompu ou trop gros (300 Mo au plus)")
				return
			}
		case "libelle":
			b, _ := io.ReadAll(io.LimitReader(part, 200))
			libelle = strings.TrimSpace(string(b))
		}
	}
	if data.Len() == 0 || data.Len() > maxDICOM {
		writeError(w, http.StatusBadRequest, "fichier absent ou trop gros (300 Mo au plus)")
		return
	}
	img, err := imagerie.Read(bytes.NewReader(data.Bytes()), int64(data.Len()))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	gz, err := imagerie.Gzip(img.Pixels)
	if err != nil {
		s.fail(w, err)
		return
	}
	key := img.PixelSHA256 + ".px.gz"
	if err := s.images.Put(r.Context(), key, gz); err != nil {
		s.log.Error("stockage d'une image", "err", err)
		writeError(w, http.StatusBadGateway, "stockage des images injoignable")
		return
	}
	if libelle == "" {
		libelle = strings.TrimSpace(strings.Join([]string{img.Modality, img.Laterality, img.View}, " "))
		if libelle == "" {
			libelle = "Image"
		}
	}
	row, existe, err := s.st.CreateImage(r.Context(), libelle, img.Meta, key, int64(len(gz)), u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "image.import", map[string]any{"image": row.ID, "existe": existe})
	status := http.StatusCreated
	if existe {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"image": row, "existe": existe})
}

func (s *Server) listImages(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListImages(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"images": list})
}

func (s *Server) getImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	im, err := s.st.GetImage(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"image": im})
}

// imagePixels sert les pixels (uint16, petit-boutiste) compressés : le navigateur les décompresse seul.
func (s *Server) imagePixels(w http.ResponseWriter, r *http.Request) {
	if !s.imagesReady(w) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	im, err := s.st.GetImage(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	rc, err := s.images.Get(r.Context(), im.StorageKey)
	if errors.Is(err, imagerie.ErrAbsent) {
		writeError(w, http.StatusNotFound, "pixels absents du stockage")
		return
	}
	if err != nil {
		s.log.Error("lecture d'une image", "err", err)
		writeError(w, http.StatusBadGateway, "stockage des images injoignable")
		return
	}
	defer rc.Close()
	h := w.Header()
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Encoding", "gzip")
	h.Set("Cache-Control", "private, max-age=86400, immutable") // la clé est l'empreinte des pixels
	h.Set("X-Image-Dimensions", fmt.Sprintf("%dx%d", im.Meta.Cols, im.Meta.Rows))
	_, _ = io.Copy(w, rc)
}

func (s *Server) deleteImage(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanPurgeImages(u.Role) {
		writeError(w, http.StatusForbidden, "suppression réservée à l'administrateur et au data manager")
		return
	}
	if !s.imagesReady(w) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	key, err := s.st.DeleteImage(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.images.Delete(r.Context(), key); err != nil {
		s.log.Warn("pixels non effacés du stockage", "cle", key, "err", err)
	}
	s.st.Audit(r.Context(), u.ID, "interface", "image.suppression", map[string]any{"image": id})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testImage(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	if !auth.CanTestImages(u.Role) {
		writeError(w, http.StatusForbidden, "rôle insuffisant")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		IndividualID int64 `json:"individual_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	im, err := s.st.GetImage(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	ind, err := s.st.GetIndividual(r.Context(), in.IndividualID)
	if err != nil {
		s.fail(w, err)
		return
	}
	var res imagerie.Resultat
	if s.moteur == nil {
		res = imagerie.Simuler(im.Meta, imagerie.Individu{ID: ind.ID, Sensibilite: ind.Sensitivity, CoutMs: ind.CostMs})
	} else {
		if !s.imagesReady(w) {
			return
		}
		rc, err := s.images.Get(r.Context(), im.StorageKey)
		if errors.Is(err, imagerie.ErrAbsent) {
			writeError(w, http.StatusNotFound, "pixels absents du stockage")
			return
		}
		if err != nil {
			s.log.Error("lecture d'une image", "err", err)
			writeError(w, http.StatusBadGateway, "stockage des images injoignable")
			return
		}
		gz, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			writeError(w, http.StatusBadGateway, "lecture des pixels interrompue")
			return
		}
		if res, err = s.moteur.Predire(r.Context(), im.Meta, gz); err != nil {
			s.log.Warn("inférence du CNN", "err", err)
			writeError(w, http.StatusBadGateway, "moteur d'inférence injoignable ou en erreur (service mammomesh-inference) : "+err.Error())
			return
		}
	}
	tid, err := s.st.InsertImageTest(r.Context(), im.ID, ind.ID, res, u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.st.Audit(r.Context(), u.ID, "interface", "image.test", map[string]any{"image": im.ID, "individu": ind.ID, "simulation": res.Simulation, "modele": res.Modele})
	writeJSON(w, http.StatusCreated, map[string]any{"test": store.ImageTest{ID: tid, ImageID: im.ID, IndividualID: ind.ID,
		Individu: ind.Name, CampaignID: ind.CampaignID, Generation: ind.Generation, Resultat: res}})
}

func (s *Server) listImageTests(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := s.st.ListImageTests(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tests": list})
}

// exempleDICOM fournit une mammographie synthétique à télécharger, pour essayer la visionneuse.
func (s *Server) exempleDICOM(w http.ResponseWriter, r *http.Request) {
	vue, cote := strings.ToUpper(r.URL.Query().Get("vue")), strings.ToUpper(r.URL.Query().Get("cote"))
	if vue != "MLO" {
		vue = "CC"
	}
	if cote != "R" {
		cote = "L"
	}
	var buf bytes.Buffer
	graine := uint64(queryInt(r, "graine", 1))
	if err := imagerie.Exemple(&buf, vue, cote, graine); err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/dicom")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="exemple-synthetique-%s-%s-%d.dcm"`, cote, vue, graine))
	_, _ = w.Write(buf.Bytes())
}
