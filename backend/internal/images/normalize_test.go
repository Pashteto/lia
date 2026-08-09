package images

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// solidPNG encodes a w×h PNG of one colour.
func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestNormalizeReencodesToJPEG(t *testing.T) {
	out, ct, ext, err := Normalize(solidPNG(t, 40, 20, color.RGBA{R: 200, G: 30, B: 30, A: 255}))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if ct != "image/jpeg" || ext != ".jpg" {
		t.Fatalf("content type %q ext %q; want image/jpeg .jpg", ct, ext)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not a decodable jpeg: %v", err)
	}
}

func TestNormalizeDownscalesOversizeImages(t *testing.T) {
	// 3000×1500 stands in for a phone photo: wider than MaxDimension.
	out, _, _, err := Normalize(solidPNG(t, 3000, 1500, color.RGBA{R: 10, G: 10, B: 200, A: 255}))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != MaxDimension {
		t.Errorf("width = %d; want %d", cfg.Width, MaxDimension)
	}
	if cfg.Height != MaxDimension/2 {
		t.Errorf("height = %d; want %d (aspect preserved)", cfg.Height, MaxDimension/2)
	}
}

func TestNormalizeLeavesSmallImagesAtTheirSize(t *testing.T) {
	out, _, _, err := Normalize(solidPNG(t, 300, 200, color.RGBA{A: 255}))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != 300 || cfg.Height != 200 {
		t.Errorf("size = %dx%d; want 300x200 (no upscaling)", cfg.Width, cfg.Height)
	}
}

func TestNormalizeFlattensTransparencyOntoWhite(t *testing.T) {
	// Fully transparent input: without flattening this comes out black.
	out, _, _, err := Normalize(solidPNG(t, 10, 10, color.RGBA{}))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	r, g, b, _ := img.At(5, 5).RGBA()
	if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Errorf("pixel = %d,%d,%d; want near-white", r>>8, g>>8, b>>8)
	}
}

func TestNormalizeRejectsNonImages(t *testing.T) {
	if _, _, _, err := Normalize([]byte("this is definitely not an image")); err == nil {
		t.Fatal("expected an error for non-image bytes")
	}
}

func TestIsHEICAcceptsTheBrandsPhonesUse(t *testing.T) {
	// iPhone stills are commonly "mif1", which the decoder package does not
	// register with image.RegisterFormat — the reason sniffing lives here.
	for _, brand := range []string{"heic", "mif1", "msf1", "heix"} {
		data := append([]byte{0, 0, 0, 0x18}, []byte("ftyp"+brand)...)
		data = append(data, make([]byte, 16)...)
		if !isHEIC(data) {
			t.Errorf("brand %q not recognised as HEIC", brand)
		}
	}
	if isHEIC(append([]byte{0, 0, 0, 0x18}, []byte("ftypqt  ")...)) {
		t.Error("a QuickTime brand was recognised as HEIC")
	}
}
