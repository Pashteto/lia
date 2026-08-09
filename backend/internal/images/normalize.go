// Package images normalises uploaded photos: it accepts what phones actually
// produce (including Apple's HEIC) and stores one predictable format.
//
// Uploads used to be stored verbatim, which forced two unhappy constraints: a
// small byte cap, because a modern phone photo is 3-8 MB, and a format
// whitelist that rejected every HEIC an iPhone hands over. Re-encoding removes
// both — the cap can be generous because what lands on disk is a downscaled
// JPEG, and any format we can decode is a format we can accept.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register the PNG decoder
	"io"

	"github.com/gen2brain/heic"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the WebP decoder
)

// white is the backdrop transparent sources are flattened onto.
var white = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

// MaxDimension caps the longest side of a stored image. Covers are rendered at
// most a few hundred CSS pixels wide; 2000 leaves room for retina and crops
// without keeping a 12-megapixel original around.
const MaxDimension = 2000

// JPEGQuality is the re-encode quality. 82 is the usual "no visible loss at
// photographic detail" point and roughly a tenth the size of a phone original.
const JPEGQuality = 82

// ErrUnsupported means the bytes are not an image format we can decode.
var ErrUnsupported = errors.New("unsupported image format")

// heicBrands are the ISO-BMFF brands that carry HEIF/HEIC payloads. The decoder
// package registers only some of these with image.RegisterFormat, and notably
// not "mif1" — the brand iPhone stills most often use — so sniffing is done
// here rather than relying on image.Decode's format table.
var heicBrands = map[string]bool{
	"heic": true, "heix": true, "hevc": true, "hevx": true,
	"mif1": true, "msf1": true, "heim": true, "heis": true,
	"hevm": true, "hevs": true,
}

// isHEIC reports whether data looks like an ISO-BMFF file with a HEIF brand:
// [4]byte size, "ftyp", [4]byte major brand.
func isHEIC(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	if string(data[4:8]) != "ftyp" {
		return false
	}
	return heicBrands[string(data[8:12])]
}

// Normalize decodes an uploaded image and re-encodes it as a JPEG no larger
// than MaxDimension on its longest side. It returns the encoded bytes plus the
// content type and file extension to store them under.
func Normalize(data []byte) (out []byte, contentType, ext string, err error) {
	img, err := decode(data)
	if err != nil {
		return nil, "", "", err
	}

	img = fitWithin(img, MaxDimension)
	img = flatten(img)

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, "", "", fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), "image/jpeg", ".jpg", nil
}

// decode turns the bytes into an image, routing HEIC past the stdlib registry.
func decode(data []byte) (image.Image, error) {
	if isHEIC(data) {
		img, err := heic.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%w: heic: %s", ErrUnsupported, err)
		}
		return img, nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		if errors.Is(err, image.ErrFormat) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrUnsupported
		}
		return nil, fmt.Errorf("%w: %s", ErrUnsupported, err)
	}
	// A camera JPEG records orientation in EXIF rather than in the pixels, so a
	// portrait phone photo decodes on its side unless it is rotated back.
	return applyOrientation(img, exifOrientation(data)), nil
}

// fitWithin scales img down so neither side exceeds max. Images already within
// bounds are returned untouched — upscaling would only add bytes.
func fitWithin(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	nw, nh := w, h
	if w >= h {
		nw, nh = max, int(float64(h)*float64(max)/float64(w))
	} else {
		nw, nh = int(float64(w)*float64(max)/float64(h)), max
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// flatten composites onto white. JPEG has no alpha channel, so a transparent
// PNG would otherwise come out with black where it used to be see-through.
func flatten(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(white), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Over)
	return dst
}
