package service

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateThumbnail_DoesNotDarkenBottomHalf(t *testing.T) {
	dir := t.TempDir()
	name := "ui-ux-pro-max"

	fileName, err := GenerateThumbnail(name, "", dir)
	if err != nil {
		t.Fatalf("generate thumbnail: %v", err)
	}

	f, err := os.Open(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatalf("open thumbnail: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}

	const (
		width  = 300
		height = 200
		x      = 280
		y      = 170
	)

	palette := gradientPalettes[hashString(name)%len(gradientPalettes)]
	tValue := (float64(x)/float64(width) + float64(y)/float64(height)) / 2.0
	expected := lerpColor(palette[0], palette[1], tValue)
	actual := img.At(x, y)
	r16, g16, b16, _ := actual.RGBA()

	if uint8(r16>>8) != expected.R || uint8(g16>>8) != expected.G || uint8(b16>>8) != expected.B {
		t.Fatalf("expected bottom-right background pixel to remain gradient color %v, got rgb(%d,%d,%d)", expected, uint8(r16>>8), uint8(g16>>8), uint8(b16>>8))
	}
}
