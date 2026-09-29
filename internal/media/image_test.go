package media

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestDerivativeUsesLongestEdgeForEitherOrientation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{"landscape", 2000, 1000, 1920, 960},
		{"portrait", 1000, 2000, 960, 1920},
		{"small", 500, 250, 500, 250},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "original.jpg")
			file, err := os.Create(original)
			if err != nil {
				t.Fatal(err)
			}
			if err := jpeg.Encode(file, image.NewRGBA(image.Rect(0, 0, tc.width, tc.height)), nil); err != nil {
				file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			display := filepath.Join(dir, "display.jpg")
			width, height, err := Derivative(original, display)
			if err != nil || width != tc.width || height != tc.height {
				t.Fatalf("original dimensions: %d x %d, %v", width, height, err)
			}
			out, err := Decode(display)
			if err != nil {
				t.Fatal(err)
			}
			if got := out.Bounds().Size(); got.X != tc.wantWidth || got.Y != tc.wantHeight {
				t.Fatalf("display dimensions: %d x %d, want %d x %d", got.X, got.Y, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}
