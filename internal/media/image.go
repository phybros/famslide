package media

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func Decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func WriteJPEG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".image-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 87}); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func Derivative(original, dst string, maxW, maxH int) (int, int, error) {
	img, err := Decode(original)
	if err != nil {
		return 0, 0, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 || w > 16000 || h > 16000 {
		return 0, 0, fmt.Errorf("invalid image dimensions")
	}
	scale := min(1.0, min(float64(maxW)/float64(w), float64(maxH)/float64(h)))
	dw, dh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	out := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Over, nil)
	return w, h, WriteJPEG(dst, out)
}
