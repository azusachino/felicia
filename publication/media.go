package publication

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// maxPublicEdge caps the longest edge of a published raster derivative.
// Publishing full-resolution originals is both a bandwidth and a privacy
// problem (a 48MP frame carries far more incidental detail than the story
// needs), so derivatives are bounded — see ADR-0026 §"public derivatives".
const maxPublicEdge = 2048

// publicJPEGQuality trades a little fidelity for a much smaller artifact at a
// size where the difference is not visible in the reader's gallery.
const publicJPEGQuality = 85

// SanitizePublicImage turns a source media object into the derivative that is
// safe to publish, and is the *only* way media bytes reach an ArtifactWriter.
//
// It is a plain function rather than an injectable interface on purpose:
// stripping EXIF (and with it the embedded GPS that would otherwise disclose
// where the photographer lives) is a privacy invariant of publication, not a
// policy a caller may swap out, forget to wire, or pass nil for. Every
// supported format is either fully re-encoded or rewritten at the container
// level; anything we cannot sanitize is a hard error, so a compile fails
// instead of leaking an unprocessed original.
func SanitizePublicImage(objectKey string, source io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	extension := strings.ToLower(path.Ext(objectKey))
	switch extension {
	case ".jpg", ".jpeg":
		return sanitizeJPEG(raw)
	case ".png":
		return sanitizePNG(raw)
	case ".webp":
		return sanitizeWebP(raw)
	default:
		return nil, fmt.Errorf("unsupported public image format %q: only .jpg, .jpeg, .png and .webp can be stripped of metadata", extension)
	}
}

// sanitizeJPEG decodes and re-encodes: Go's JPEG encoder emits no EXIF/XMP at
// all, so the round trip *is* the strip. Decoding also rejects a file that is
// merely named .jpg but is some other (unsanitizable) format.
func sanitizeJPEG(raw []byte) ([]byte, error) {
	decoded, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode jpeg: %w", err)
	}
	// The EXIF Orientation tag is about to be discarded along with the rest of
	// the metadata, so it has to be baked into the pixels — otherwise every
	// portrait phone photo publishes sideways.
	//
	// Resizing first is a pure optimisation: rotation never changes which edge
	// is longest, so capping before or after the transform yields the same
	// derivative, but the per-pixel orientation pass then runs over a bounded
	// 2048px image instead of the full 48MP original.
	oriented := applyOrientation(resizeToFit(decoded), jpegOrientation(raw))
	var out bytes.Buffer
	if err := jpeg.Encode(&out, oriented, &jpeg.Options{Quality: publicJPEGQuality}); err != nil {
		return nil, fmt.Errorf("encode jpeg derivative: %w", err)
	}
	return out.Bytes(), nil
}

// sanitizePNG decodes and re-encodes, which drops every ancillary chunk
// (tEXt/iTXt/eXIf and friends) because Go's PNG encoder writes none of them.
func sanitizePNG(raw []byte) ([]byte, error) {
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, resizeToFit(decoded)); err != nil {
		return nil, fmt.Errorf("encode png derivative: %w", err)
	}
	return out.Bytes(), nil
}

// resizeToFit scales an image down so its longest edge is at most
// maxPublicEdge, preserving aspect ratio. An image that already fits is
// returned untouched — it is still re-encoded by the caller, so it is still
// stripped; only upscaling (which adds no information) is refused.
func resizeToFit(source image.Image) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	longest := max(width, height)
	if longest <= maxPublicEdge {
		return source
	}
	scale := float64(maxPublicEdge) / float64(longest)
	target := image.Rect(0, 0, scaledEdge(width, scale), scaledEdge(height, scale))
	scaled := image.NewRGBA(target)
	xdraw.CatmullRom.Scale(scaled, target, source, bounds, xdraw.Src, nil)
	return scaled
}

func scaledEdge(edge int, scale float64) int {
	return max(1, int(math.Round(float64(edge)*scale)))
}

// applyOrientation physically rotates/flips pixels for the given EXIF
// Orientation value (1-8). Orientation 1 (and any value we could not trust)
// is a no-op and returns the source untouched.
func applyOrientation(source image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return source
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	// Orientations 5-8 exchange the axes, so the derivative is transposed.
	target := image.Rect(0, 0, width, height)
	if orientation >= 5 {
		target = image.Rect(0, 0, height, width)
	}
	transformed := image.NewRGBA(target)
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			sourceX, sourceY := orientationSource(orientation, x, y, width, height)
			transformed.Set(x, y, source.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY))
		}
	}
	return transformed
}

// orientationSource maps a destination pixel back to its source pixel for one
// of the eight EXIF orientations, expressed as the inverse of the transform
// each orientation asks a viewer to apply.
func orientationSource(orientation, x, y, width, height int) (int, int) {
	switch orientation {
	case 2: // mirror horizontally
		return width - 1 - x, y
	case 3: // rotate 180
		return width - 1 - x, height - 1 - y
	case 4: // mirror vertically
		return x, height - 1 - y
	case 5: // transpose (mirror across the main diagonal)
		return y, x
	case 6: // rotate 90 clockwise
		return y, height - 1 - x
	case 7: // transverse (mirror across the anti-diagonal)
		return width - 1 - y, height - 1 - x
	case 8: // rotate 90 counter-clockwise
		return width - 1 - y, x
	default:
		return x, y
	}
}

const exifSignature = "Exif\x00\x00"

// exifOrientationTag is the TIFF tag number of EXIF Orientation.
const exifOrientationTag = 0x0112

// jpegOrientation reads the EXIF Orientation of a JPEG, defaulting to 1 (no
// transform) whenever the file has no EXIF, no Orientation tag, or metadata we
// cannot parse confidently. Orientation is a cosmetic hint: a malformed tag
// must never fail a publish.
func jpegOrientation(raw []byte) int {
	tiff, ok := jpegExifTIFF(raw)
	if !ok {
		return 1
	}
	return exifOrientation(tiff)
}

// jpegExifTIFF returns the TIFF block of the first APP1/Exif segment. It is
// also the assertion hook the publication tests use to prove a derivative
// carries no EXIF segment at all.
func jpegExifTIFF(raw []byte) ([]byte, bool) {
	// Skip the SOI marker; every segment below is FF <marker> <2-byte length>.
	offset := 2
	if len(raw) < offset || raw[0] != 0xFF || raw[1] != 0xD8 {
		return nil, false
	}
	for offset+4 <= len(raw) {
		if raw[offset] != 0xFF {
			return nil, false // not at a marker boundary — stop rather than guess
		}
		marker := raw[offset+1]
		// SOS starts entropy-coded data and SOI/EOI carry no length; no
		// metadata segment can follow, so parsing is done.
		if marker == 0xDA || marker == 0xD9 {
			return nil, false
		}
		length := int(binary.BigEndian.Uint16(raw[offset+2 : offset+4]))
		if length < 2 || offset+2+length > len(raw) {
			return nil, false
		}
		payload := raw[offset+4 : offset+2+length]
		if marker == 0xE1 && len(payload) > len(exifSignature) && string(payload[:len(exifSignature)]) == exifSignature {
			return payload[len(exifSignature):], true
		}
		offset += 2 + length
	}
	return nil, false
}

// exifOrientation walks IFD0 of a TIFF block for the Orientation tag.
func exifOrientation(tiff []byte) int {
	const headerSize = 8
	if len(tiff) < headerSize {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:4]) != 0x002A {
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < headerSize || ifd+2 > len(tiff) {
		return 1
	}
	entries := int(order.Uint16(tiff[ifd : ifd+2]))
	for index := range entries {
		const entrySize = 12
		start := ifd + 2 + index*entrySize
		if start+entrySize > len(tiff) {
			return 1
		}
		entry := tiff[start : start+entrySize]
		if order.Uint16(entry[:2]) != exifOrientationTag {
			continue
		}
		// Orientation is a single SHORT, so it lives inline in the value
		// field's first two bytes (in the file's byte order).
		orientation := int(order.Uint16(entry[8:10]))
		if orientation < 1 || orientation > 8 {
			return 1
		}
		return orientation
	}
	return 1
}

// WebP has no pure-Go encoder, so a decode/re-encode round trip is not
// available. Metadata is instead removed at the RIFF container level, which is
// lossless and leaves the image bitstream byte-identical.
//
// isWebPMetadataChunk reports whether a FourCC names a chunk that must not be
// published. RIFF FourCCs are always exactly four bytes, so the XMP identifier
// is space-padded: the trailing space is part of the format, not a typo.
func isWebPMetadataChunk(fourCC string) bool {
	switch fourCC {
	case "EXIF", "XMP ", "ICCP":
		return true
	default:
		return false
	}
}

// vp8xMetadataFlags are the ICC (0x20), EXIF (0x08) and XMP (0x04) presence
// bits of the VP8X feature byte; they must be cleared when their chunks go,
// or the rewritten container advertises metadata it no longer contains.
const vp8xMetadataFlags = 0x2C

const riffHeaderSize = 12

// sanitizeWebP rewrites a WebP container without its metadata chunks. No
// resize is applied: without an encoder the only alternative would be to
// transcode to another format, which would change the object key.
func sanitizeWebP(raw []byte) ([]byte, error) {
	if len(raw) < riffHeaderSize || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WEBP" {
		return nil, fmt.Errorf("decode webp: not a RIFF/WEBP container")
	}
	declared := int(binary.LittleEndian.Uint32(raw[4:8]))
	end := min(len(raw), 8+declared)
	if end < riffHeaderSize {
		return nil, fmt.Errorf("decode webp: RIFF size %d is shorter than the header", declared)
	}

	kept := make([]byte, 0, len(raw))
	for offset := riffHeaderSize; offset+8 <= end; {
		fourCC := string(raw[offset : offset+4])
		// Sizes are computed in uint64 so a hostile 4GB size cannot wrap.
		size := uint64(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		// A chunk payload is padded to an even length; the pad byte is part of
		// the container but not of the declared size.
		stride := 8 + size + size%2
		if uint64(offset)+stride > uint64(end) {
			return nil, fmt.Errorf("decode webp: chunk %q overruns the container", fourCC)
		}
		if !isWebPMetadataChunk(fourCC) {
			chunk := raw[offset : offset+int(stride)]
			if fourCC == "VP8X" && size > 0 {
				chunk = clearVP8XMetadataFlags(chunk)
			}
			kept = append(kept, chunk...)
		}
		offset += int(stride)
	}

	out := make([]byte, 0, riffHeaderSize+len(kept))
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(kept)+4)) // +4 for "WEBP"
	out = append(out, "WEBP"...)
	return append(out, kept...), nil
}

func clearVP8XMetadataFlags(chunk []byte) []byte {
	cleared := bytes.Clone(chunk)
	cleared[8] &^= vp8xMetadataFlags
	return cleared
}
