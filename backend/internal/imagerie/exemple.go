package imagerie

import (
	"io"
	"math"
	"math/rand/v2"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/frame"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// Exemple écrit une mammographie DICOM synthétique (aucune patiente réelle) : silhouette de sein,
// texture et quelques foyers plus denses. Ses en-têtes portent volontairement de fausses données
// patient, pour vérifier qu'elles disparaissent à l'import. vue : « CC » ou « MLO » ; cote : « L » ou « R ».
func Exemple(w io.Writer, vue, cote string, graine uint64) error {
	const rows, cols = 1280, 1024
	r := rand.New(rand.NewPCG(graine, 0x6D616D6D6F))
	px := make([]uint16, rows*cols)
	gauche := cote != "R" // sein gauche : paroi thoracique à droite de l'image
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Distance au bord de la paroi thoracique, normalisée, et silhouette en demi-ellipse.
			dx := float64(x) / cols
			if gauche {
				dx = 1 - dx
			}
			dy := (float64(y) - rows/2) / (rows * 0.46)
			in := dx*dx/(0.82*0.82) + dy*dy
			v := 60.0 + 20*r.Float64() // fond
			if in < 1 {
				bord := math.Min(1, (1-in)*6)
				v = 900 + 1700*bord + 380*math.Sin(float64(x)/23+float64(y)/41)*math.Cos(float64(y)/17) + 160*r.NormFloat64()
				if vue == "MLO" && dx > 0.86 && float64(y) < rows*0.45 { // muscle pectoral
					v += 900
				}
			}
			px[y*cols+x] = uint16(math.Max(0, math.Min(4095, v)))
		}
	}
	// Foyers denses (pour la visionneuse seulement : sans valeur diagnostique).
	for k := 0; k < 3; k++ {
		cx, cy, rad := 250+r.IntN(400), 300+r.IntN(650), 10+r.IntN(25)
		if !gauche {
			cx = cols - cx
		}
		for y := cy - rad; y <= cy+rad; y++ {
			for x := cx - rad; x <= cx+rad; x++ {
				if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= rad*rad && x >= 0 && y >= 0 && x < cols && y < rows {
					px[y*cols+x] = uint16(math.Min(4095, float64(px[y*cols+x])+700))
				}
			}
		}
	}

	el := func(t tag.Tag, v any) *dicom.Element {
		e, err := dicom.NewElement(t, v)
		if err != nil {
			panic(err)
		}
		return e
	}
	ds := dicom.Dataset{Elements: []*dicom.Element{
		el(tag.MediaStorageSOPClassUID, []string{"1.2.840.10008.5.1.4.1.1.1.2"}),
		el(tag.MediaStorageSOPInstanceUID, []string{"1.2.826.0.1.3680043.10.1234.1"}),
		el(tag.TransferSyntaxUID, []string{"1.2.840.10008.1.2.1"}),
		el(tag.SOPClassUID, []string{"1.2.840.10008.5.1.4.1.1.1.2"}),
		el(tag.SOPInstanceUID, []string{"1.2.826.0.1.3680043.10.1234.1"}),
		el(tag.StudyDate, []string{"20260101"}),
		el(tag.Modality, []string{"MG"}),
		el(tag.Manufacturer, []string{"MammoMesh (synthétique)"}),
		el(tag.InstitutionName, []string{"Centre fictif"}),
		el(tag.PatientName, []string{"EXEMPLE^SYNTHETIQUE"}),
		el(tag.PatientID, []string{"SYN-0001"}),
		el(tag.PatientBirthDate, []string{"19700101"}),
		el(tag.ViewPosition, []string{vue}),
		el(tag.ImageLaterality, []string{cote}),
		el(tag.SamplesPerPixel, []int{1}),
		el(tag.PhotometricInterpretation, []string{"MONOCHROME2"}),
		el(tag.Rows, []int{rows}),
		el(tag.Columns, []int{cols}),
		el(tag.BitsAllocated, []int{16}),
		el(tag.BitsStored, []int{12}),
		el(tag.HighBit, []int{11}),
		el(tag.PixelRepresentation, []int{0}),
		el(tag.WindowCenter, []string{"2000"}),
		el(tag.WindowWidth, []string{"3200"}),
		el(tag.PixelData, dicom.PixelDataInfo{Frames: []*frame.Frame{{NativeData: &frame.NativeFrame[uint16]{
			InternalBitsPerSample: 16, InternalRows: rows, InternalCols: cols, InternalSamplesPerPixel: 1, RawData: px,
		}}}}),
	}}
	return dicom.Write(w, ds)
}
