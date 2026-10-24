package media

import (
	_ "embed"
	"image"
	"math"
	"sync"

	"famslide/internal/storage"
	pigo "github.com/esimov/pigo/core"
	"golang.org/x/image/draw"
)

// FaceDetectorVersion changes when the model, thresholds, or padding change.
const FaceDetectorVersion = "pigo-1.4.6-v1"

//go:embed models/facefinder
var faceCascade []byte
var detector = sync.OnceValues(func() (*pigo.Pigo, error) {
	return pigo.NewPigo().Unpack(faceCascade)
})

// DetectFaces returns padded normalized bounds of all accepted face detections.
// Coordinates refer to the same decoded image used by the display derivative.
func DetectFaces(path string) (*storage.Bounds, error) {
	img, err := Decode(path)
	if err != nil {
		return nil, err
	}
	pg, err := detector()
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	scale := math.Min(1, 1024/float64(max(b.Dx(), b.Dy())))
	w, h := max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))
	gray := image.NewGray(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(gray, gray.Bounds(), img, b, draw.Src, nil)
	detections := pg.RunCascade(pigo.CascadeParams{
		MinSize: 20, MaxSize: min(w, h), ShiftFactor: .1, ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{Pixels: gray.Pix, Rows: h, Cols: w, Dim: gray.Stride},
	}, 0)
	return faceBounds(pg.ClusterDetections(detections, .2), w, h), nil
}

func faceBounds(detections []pigo.Detection, w, h int) *storage.Bounds {
	left, top, right, bottom := float64(w), float64(h), 0.0, 0.0
	found := false
	for _, d := range detections {
		if d.Q < 5 {
			continue
		}
		// Pad each square by 20% on every side to leave space around the head.
		radius := float64(d.Scale) * .7
		left = min(left, float64(d.Col)-radius)
		top = min(top, float64(d.Row)-radius)
		right = max(right, float64(d.Col)+radius)
		bottom = max(bottom, float64(d.Row)+radius)
		found = true
	}
	if !found {
		return nil
	}
	left, top = max(0, left), max(0, top)
	right, bottom = min(float64(w), right), min(float64(h), bottom)
	return &storage.Bounds{X: left / float64(w), Y: top / float64(h), Width: (right - left) / float64(w), Height: (bottom - top) / float64(h)}
}
