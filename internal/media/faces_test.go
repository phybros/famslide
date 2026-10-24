package media

import (
	pigo "github.com/esimov/pigo/core"
	"image"
	"math"
	"path/filepath"
	"testing"
)

func TestFaceBoundsUnionAndPadding(t *testing.T) {
	b := faceBounds([]pigo.Detection{
		{Col: 10, Row: 20, Scale: 20, Q: 10},
		{Col: 80, Row: 70, Scale: 20, Q: 8},
		{Col: 99, Row: 99, Scale: 10, Q: 1},
	}, 100, 100)
	if b == nil || b.X != 0 || math.Abs(b.Y-.06) > 1e-9 || b.Width != .94 || b.Height != .78 {
		t.Fatalf("unexpected union: %+v", b)
	}
	if faceBounds(nil, 100, 100) != nil {
		t.Fatal("empty detections should have no bounds")
	}
}

func TestDetectNoFaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.jpg")
	if err := WriteJPEG(path, image.NewRGBA(image.Rect(0, 0, 100, 100))); err != nil {
		t.Fatal(err)
	}
	b, err := DetectFaces(path)
	if err != nil || b != nil {
		t.Fatalf("blank image: bounds=%+v error=%v", b, err)
	}
}

func TestDetectGroupFaces(t *testing.T) {
	b, err := DetectFaces("testdata/faces.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Fatal("no faces detected in group photo")
	}
	// The sample has faces spread across most of the image. A single central
	// detection would not be a useful rectangle for this group.
	if b.Width < .8 || b.Height < .6 || b.X < 0 || b.Y < 0 || b.X+b.Width > 1 || b.Y+b.Height > 1 {
		t.Fatalf("unexpected group bounds: %+v", b)
	}
}
