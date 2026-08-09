package images

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// jpegWithOrientation encodes img as JPEG and splices in an Exif APP1 segment
// declaring the given orientation — what a phone camera writes.
func jpegWithOrientation(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	raw := buf.Bytes()

	// TIFF header (little endian) + one IFD entry for tag 0x0112.
	tiff := make([]byte, 0, 26)
	tiff = append(tiff, 'I', 'I', 42, 0)
	tiff = binary.LittleEndian.AppendUint32(tiff, 8) // IFD0 at offset 8
	tiff = binary.LittleEndian.AppendUint16(tiff, 1) // one entry
	tiff = binary.LittleEndian.AppendUint16(tiff, 0x0112)
	tiff = binary.LittleEndian.AppendUint16(tiff, 3) // SHORT
	tiff = binary.LittleEndian.AppendUint32(tiff, 1) // count
	tiff = binary.LittleEndian.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0)                        // pad the 4-byte value field
	tiff = binary.LittleEndian.AppendUint32(tiff, 0) // next IFD: none

	payload := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	segment = append(segment, payload...)

	// Insert straight after SOI so the walker meets it first.
	out := append([]byte{}, raw[:2]...)
	out = append(out, segment...)
	return append(out, raw[2:]...)
}

func TestExifOrientationReadsTheTag(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	data := jpegWithOrientation(t, img, 6)
	if got := exifOrientation(data); got != 6 {
		t.Fatalf("exifOrientation = %d; want 6", got)
	}
}

func TestExifOrientationDefaultsWhenAbsent(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got := exifOrientation(buf.Bytes()); got != orientationNormal {
		t.Fatalf("exifOrientation = %d; want %d", got, orientationNormal)
	}
	if got := exifOrientation([]byte("not a jpeg at all")); got != orientationNormal {
		t.Fatalf("exifOrientation(garbage) = %d; want %d", got, orientationNormal)
	}
}

func TestApplyOrientationRotatesQuarterTurns(t *testing.T) {
	// Landscape source with a marked top-left pixel.
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	mark := color.RGBA{R: 255, A: 255}
	src.Set(0, 0, mark)

	got := applyOrientation(src, 6) // 90° clockwise
	if w, h := got.Bounds().Dx(), got.Bounds().Dy(); w != 4 || h != 8 {
		t.Fatalf("size = %dx%d; want 4x8 (axes swapped)", w, h)
	}
	// Turning clockwise sends the top-left pixel to the top-right corner.
	if r, _, _, _ := got.At(3, 0).RGBA(); r>>8 != 255 {
		t.Errorf("marked pixel did not land top-right after a 90° turn")
	}
}

func TestNormalizeUprightsAPortraitPhoto(t *testing.T) {
	// 8 wide × 4 tall on the sensor, tagged as needing a quarter turn: the
	// stored image must come out 4×8.
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	out, _, _, err := Normalize(jpegWithOrientation(t, src, 6))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != 4 || cfg.Height != 8 {
		t.Errorf("size = %dx%d; want 4x8 (rotation baked in)", cfg.Width, cfg.Height)
	}
}
