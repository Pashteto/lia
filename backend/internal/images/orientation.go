package images

import (
	"encoding/binary"
	"image"
)

// A phone writes the sensor's pixels unrotated and records how the camera was
// held in EXIF tag 0x0112. Decoders ignore that tag, so a portrait photo
// re-encoded naively comes out lying on its side. This file reads the tag and
// bakes the rotation into the pixels, after which the tag is meaningless and
// can be dropped with the rest of the metadata.
//
// Deliberately minimal: enough of the JPEG/TIFF structure to find one 16-bit
// value, not a general EXIF parser.

const (
	orientationNormal = 1
	orientationMax    = 8
)

// exifOrientation returns the EXIF orientation of a JPEG (1..8), or
// orientationNormal when absent or unparseable — an unreadable tag must never
// be worse than no tag.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return orientationNormal // not a JPEG
	}
	// Walk the marker segments looking for APP1/Exif.
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return orientationNormal // desynchronised; give up
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD9) {
			i += 2
			continue
		}
		if i+4 > len(data) {
			return orientationNormal
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return orientationNormal
		}
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) > 6 && string(segment[:6]) == "Exif\x00\x00" {
			return orientationFromTIFF(segment[6:])
		}
		if marker == 0xDA { // start of scan — no metadata past here
			return orientationNormal
		}
		i += 2 + size
	}
	return orientationNormal
}

// orientationFromTIFF reads tag 0x0112 out of the TIFF header an Exif segment
// wraps.
func orientationFromTIFF(tiff []byte) int {
	if len(tiff) < 8 {
		return orientationNormal
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return orientationNormal
	}
	offset := int(order.Uint32(tiff[4:8]))
	if offset < 8 || offset+2 > len(tiff) {
		return orientationNormal
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	entry := offset + 2
	for n := 0; n < count; n++ {
		if entry+12 > len(tiff) {
			return orientationNormal
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			v := int(order.Uint16(tiff[entry+8 : entry+10]))
			if v >= orientationNormal && v <= orientationMax {
				return v
			}
			return orientationNormal
		}
		entry += 12
	}
	return orientationNormal
}

// applyOrientation rewrites the pixels so the image reads upright.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= orientationNormal || orientation > orientationMax {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Orientations 5..8 transpose the axes, so the canvas swaps dimensions.
	swapped := orientation >= 5
	dw, dh := w, h
	if swapped {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x, b.Min.Y+y)
			var dx, dy int
			switch orientation {
			case 2: // mirrored horizontally
				dx, dy = w-1-x, y
			case 3: // rotated 180°
				dx, dy = w-1-x, h-1-y
			case 4: // mirrored vertically
				dx, dy = x, h-1-y
			case 5: // mirrored horizontally, rotated 270° CW
				dx, dy = y, x
			case 6: // rotated 90° CW
				dx, dy = h-1-y, x
			case 7: // mirrored horizontally, rotated 90° CW
				dx, dy = h-1-y, w-1-x
			case 8: // rotated 270° CW
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, c)
		}
	}
	return dst
}
