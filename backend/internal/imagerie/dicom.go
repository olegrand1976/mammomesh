// Package imagerie importe des mammographies DICOM sans rien garder d'identifiant : seules les
// données de pixels et quelques métadonnées techniques sont conservées (pseudonymisation par
// construction, aucun en-tête patient n'est stocké). Il simule aussi le test d'un individu sur
// une image, en attendant le vrai moteur.
package imagerie

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/frame"
	"github.com/suyashkumar/dicom/pkg/tag"
	"github.com/suyashkumar/dicom/pkg/uid"
)

// Meta : métadonnées techniques conservées (aucune donnée patient, établissement ou date).
type Meta struct {
	Rows           int     `json:"rows"`
	Cols           int     `json:"cols"`
	BitsStored     int     `json:"bits_stored"`
	Photometric    string  `json:"photometric"`
	Invert         bool    `json:"invert"` // MONOCHROME1 : valeurs hautes = noir
	WindowCenter   float64 `json:"window_center"`
	WindowWidth    float64 `json:"window_width"`
	Min            int     `json:"min"`
	Max            int     `json:"max"`
	Modality       string  `json:"modality"`
	View           string  `json:"view"`       // CC, MLO…
	Laterality     string  `json:"laterality"` // L ou R
	Manufacturer   string  `json:"manufacturer"`
	Model          string  `json:"model"`
	TransferSyntax string  `json:"transfer_syntax"`
	Tissue         [4]int  `json:"tissue"` // x, y, largeur, hauteur de la zone de tissu
	PixelSHA256    string  `json:"pixel_sha256"`
}

// Image : métadonnées et pixels (niveaux de gris sur 16 bits, ligne après ligne).
type Image struct {
	Meta
	Pixels []uint16
}

// Erreurs d'import, renvoyées telles quelles à l'utilisateur.
var (
	ErrNotDICOM      = errors.New("fichier illisible comme DICOM")
	ErrBurnedIn      = errors.New("image refusée : annotations incrustées dans les pixels (BurnedInAnnotation = YES), des données identifiantes pourraient y figurer")
	ErrNotGrayscale  = errors.New("image refusée : seules les images en niveaux de gris (un échantillon par pixel) sont prises en charge")
	ErrNoPixels      = errors.New("le fichier ne contient pas de pixels")
	ErrUnsupportedTS = errors.New("compression non prise en charge")
)

// Read lit un DICOM et n'en garde que les pixels et les métadonnées techniques.
func Read(r io.Reader, size int64) (Image, error) {
	ds, err := dicom.Parse(r, size, nil)
	if err != nil {
		return Image{}, fmt.Errorf("%w : %v", ErrNotDICOM, err)
	}
	str := func(t tag.Tag) string {
		el, err := ds.FindElementByTag(t)
		if err != nil {
			return ""
		}
		if v, ok := el.Value.GetValue().([]string); ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	num := func(t tag.Tag, def int) int {
		el, err := ds.FindElementByTag(t)
		if err != nil {
			return def
		}
		if v, ok := el.Value.GetValue().([]int); ok && len(v) > 0 {
			return v[0]
		}
		return def
	}
	flt := func(t tag.Tag, def float64) float64 {
		s := str(t)
		if f, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(s, `\`)[0]), 64); err == nil {
			return f
		}
		return def
	}

	if strings.EqualFold(str(tag.BurnedInAnnotation), "YES") {
		return Image{}, ErrBurnedIn
	}
	if num(tag.SamplesPerPixel, 1) != 1 {
		return Image{}, ErrNotGrayscale
	}
	ts := str(tag.TransferSyntaxUID)
	m := Meta{
		Photometric:    str(tag.PhotometricInterpretation),
		Modality:       str(tag.Modality),
		View:           str(tag.ViewPosition),
		Laterality:     firstNonEmpty(str(tag.ImageLaterality), str(tag.Laterality)),
		Manufacturer:   str(tag.Manufacturer),
		Model:          str(tag.ManufacturerModelName),
		TransferSyntax: transferSyntaxName(ts),
		BitsStored:     num(tag.BitsStored, 16),
	}
	m.Invert = m.Photometric == "MONOCHROME1"

	el, err := ds.FindElementByTag(tag.PixelData)
	if err != nil {
		return Image{}, ErrNoPixels
	}
	info := dicom.MustGetPixelDataInfo(el.Value)
	if info.ParseErr != nil || len(info.Frames) == 0 {
		return Image{}, fmt.Errorf("%w (%s)", ErrUnsupportedTS, m.TransferSyntax)
	}
	values, rows, cols, err := samples(info.Frames[0], m.TransferSyntax)
	if err != nil {
		return Image{}, err
	}
	m.Rows, m.Cols = rows, cols

	// Valeurs utiles (pente et ordonnée de rééchelonnage), décalées pour tenir sur 16 bits non signés.
	slope, intercept := flt(tag.RescaleSlope, 1), flt(tag.RescaleIntercept, 0)
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, v := range values {
		v = v*slope + intercept
		values[i] = v
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	scale := 1.0
	if hi-lo > 65535 {
		scale = 65535 / (hi - lo)
	}
	px := make([]uint16, len(values))
	for i, v := range values {
		px[i] = uint16(math.Round((v - lo) * scale))
	}
	img := Image{Meta: m, Pixels: px}
	img.Min, img.Max = 0, int(math.Round((hi-lo)*scale))

	if wc, ww := flt(tag.WindowCenter, math.NaN()), flt(tag.WindowWidth, math.NaN()); !math.IsNaN(wc) && !math.IsNaN(ww) && ww > 0 {
		img.WindowCenter, img.WindowWidth = (wc-lo)*scale, ww*scale
	} else {
		p1, p99 := percentiles(px, 0.01, 0.99)
		img.WindowCenter, img.WindowWidth = float64(p1+p99)/2, math.Max(1, float64(p99-p1))
	}
	img.Tissue = tissue(img)
	sum := sha256.Sum256(PixelBytes(px))
	img.PixelSHA256 = hex.EncodeToString(sum[:])
	return img, nil
}

// samples renvoie les valeurs du premier cadre, en flottants, avec ses dimensions.
// Lus : pixels non compressés de 8 à 32 bits et JPEG de base ; JPEG 2000 et JPEG-LS sont refusés.
func samples(f *frame.Frame, ts string) ([]float64, int, int, error) {
	if f.Encapsulated {
		img, err := f.GetImage()
		if err != nil {
			return nil, 0, 0, fmt.Errorf("%w (%s) : seules les images non compressées et JPEG de base sont lues", ErrUnsupportedTS, ts)
		}
		b := img.Bounds()
		out := make([]float64, 0, b.Dx()*b.Dy())
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				out = append(out, float64(color.Gray16Model.Convert(img.At(x, y)).(color.Gray16).Y))
			}
		}
		return out, b.Dy(), b.Dx(), nil
	}
	nf, err := f.GetNativeFrame()
	if err != nil {
		return nil, 0, 0, ErrNoPixels
	}
	if nf.SamplesPerPixel() != 1 {
		return nil, 0, 0, ErrNotGrayscale
	}
	var out []float64
	switch d := nf.RawDataSlice().(type) {
	case []uint16:
		out = conv(d)
	case []int16:
		out = conv(d)
	case []uint8:
		out = conv(d)
	case []int8:
		out = conv(d)
	case []uint32:
		out = conv(d)
	case []int32:
		out = conv(d)
	case []int:
		out = conv(d)
	default:
		return nil, 0, 0, fmt.Errorf("%w : type de pixels %T", ErrUnsupportedTS, d)
	}
	if len(out) != nf.Rows()*nf.Cols() {
		return nil, 0, 0, fmt.Errorf("%w : %d pixels pour %d × %d", ErrNoPixels, len(out), nf.Cols(), nf.Rows())
	}
	return out, nf.Rows(), nf.Cols(), nil
}

func conv[T int | int8 | int16 | int32 | uint8 | uint16 | uint32](d []T) []float64 {
	out := make([]float64, len(d))
	for i, v := range d {
		out[i] = float64(v)
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func transferSyntaxName(u string) string {
	if u == "" {
		return ""
	}
	if info, err := uid.Lookup(u); err == nil {
		return info.Name
	}
	return u
}

// percentiles sur l'histogramme 16 bits, en ignorant le fond (valeur minimale).
func percentiles(px []uint16, a, b float64) (int, int) {
	var hist [65536]int
	n := 0
	for _, v := range px {
		if v > 0 {
			hist[v]++
			n++
		}
	}
	if n == 0 {
		return 0, 1
	}
	lo, hi, acc := -1, 0, 0
	for v, c := range hist {
		acc += c
		if lo < 0 && float64(acc) >= a*float64(n) {
			lo = v
		}
		if float64(acc) >= b*float64(n) {
			hi = v
			break
		}
	}
	return lo, hi
}

// tissue : rectangle des pixels plus clairs que le fond (zone du sein), sur une grille de 4 px.
func tissue(img Image) [4]int {
	seuil := img.WindowCenter - img.WindowWidth/2
	if img.Invert {
		seuil = img.WindowCenter + img.WindowWidth/2
	}
	x0, y0, x1, y1 := img.Cols, img.Rows, -1, -1
	for y := 0; y < img.Rows; y += 4 {
		for x := 0; x < img.Cols; x += 4 {
			v := float64(img.Pixels[y*img.Cols+x])
			if (!img.Invert && v > seuil) || (img.Invert && v < seuil) {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return [4]int{0, 0, img.Cols, img.Rows}
	}
	return [4]int{x0, y0, x1 - x0 + 1, y1 - y0 + 1}
}

// PixelBytes : pixels en petit-boutiste, l'ordre attendu par la visionneuse (Uint16Array).
func PixelBytes(px []uint16) []byte {
	b := make([]byte, 2*len(px))
	for i, v := range px {
		binary.LittleEndian.PutUint16(b[2*i:], v)
	}
	return b
}

// Gzip compresse les pixels pour le stockage ; ils sont servis tels quels (Content-Encoding: gzip).
func Gzip(px []uint16) ([]byte, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if _, err := zw.Write(PixelBytes(px)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
