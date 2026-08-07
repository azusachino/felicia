package publication

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	_ "golang.org/x/image/webp" // register the WebP decoder for the assertions below
)

// The fixture's GPSLatitude / GPSLongitude payloads double as sentinels: if
// either 24-byte run survives anywhere in a derivative, the photographer's
// coordinates leaked. 35 39' 36.00" N, 139 41' 30.00" E.
var (
	gpsLatitudeSentinel  = rationals(35, 1, 39, 1, 3600, 100)
	gpsLongitudeSentinel = rationals(139, 1, 41, 1, 3000, 100)
)

func rationals(values ...uint32) []byte {
	out := make([]byte, 0, len(values)*4)
	for _, value := range values {
		out = binary.LittleEndian.AppendUint32(out, value)
	}
	return out
}

// buildExifTIFF assembles a real little-endian TIFF block carrying an
// Orientation tag in IFD0 and a GPS IFD with latitude/longitude, at the fixed
// offsets the layout comment below describes.
func buildExifTIFF(orientation uint16) []byte {
	const (
		ifd0Offset     = 8
		gpsIFDOffset   = 38  // header 8 + count 2 + 2 entries 24 + next-IFD 4
		latValueOffset = 92  // gpsIFD 38 + count 2 + 4 entries 48 + next-IFD 4
		lonValueOffset = 116 // latValueOffset + 3 rationals
	)
	buf := new(bytes.Buffer)
	buf.WriteString("II")
	buf.Write(binary.LittleEndian.AppendUint16(nil, 0x002A))
	buf.Write(binary.LittleEndian.AppendUint32(nil, ifd0Offset))

	buf.Write(binary.LittleEndian.AppendUint16(nil, 2)) // IFD0 entry count
	writeEntry(buf, exifOrientationTag, 3, 1, binary.LittleEndian.AppendUint16(nil, orientation))
	writeEntry(buf, 0x8825, 4, 1, binary.LittleEndian.AppendUint32(nil, gpsIFDOffset)) // GPSInfo
	buf.Write(binary.LittleEndian.AppendUint32(nil, 0))                                // no IFD1

	buf.Write(binary.LittleEndian.AppendUint16(nil, 4)) // GPS IFD entry count
	writeEntry(buf, 0x0001, 2, 2, []byte("N\x00"))      // GPSLatitudeRef
	writeEntry(buf, 0x0002, 5, 3, binary.LittleEndian.AppendUint32(nil, latValueOffset))
	writeEntry(buf, 0x0003, 2, 2, []byte("E\x00")) // GPSLongitudeRef
	writeEntry(buf, 0x0004, 5, 3, binary.LittleEndian.AppendUint32(nil, lonValueOffset))
	buf.Write(binary.LittleEndian.AppendUint32(nil, 0))

	buf.Write(gpsLatitudeSentinel)
	buf.Write(gpsLongitudeSentinel)
	return buf.Bytes()
}

func writeEntry(buf *bytes.Buffer, tag, entryType uint16, count uint32, value []byte) {
	buf.Write(binary.LittleEndian.AppendUint16(nil, tag))
	buf.Write(binary.LittleEndian.AppendUint16(nil, entryType))
	buf.Write(binary.LittleEndian.AppendUint32(nil, count))
	padded := make([]byte, 4)
	copy(padded, value)
	buf.Write(padded)
}

// jpegWithGPSExif encodes img as a genuine JPEG and splices an APP1/Exif
// segment in directly after SOI, producing a file that both image/jpeg and a
// real EXIF reader accept.
func jpegWithGPSExif(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode fixture jpeg: %v", err)
	}
	base := encoded.Bytes()
	payload := append([]byte(exifSignature), buildExifTIFF(orientation)...)

	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	segment = append(segment, payload...)

	fixture := make([]byte, 0, len(base)+len(segment))
	fixture = append(fixture, base[:2]...) // SOI
	fixture = append(fixture, segment...)
	fixture = append(fixture, base[2:]...)

	// Guard against a vacuous test: the fixture must really carry the EXIF and
	// the GPS bytes we are about to demand are gone.
	tiff, ok := jpegExifTIFF(fixture)
	if !ok {
		t.Fatal("fixture jpeg does not carry an APP1/Exif segment")
	}
	if !bytes.Contains(tiff, gpsLatitudeSentinel) || !bytes.Contains(tiff, gpsLongitudeSentinel) {
		t.Fatal("fixture EXIF does not carry the GPS coordinates")
	}
	if got := exifOrientation(tiff); got != int(orientation) {
		t.Fatalf("fixture EXIF orientation = %d, want %d", got, orientation)
	}
	return fixture
}

// gradient builds an asymmetric test image by filling Pix directly — Set() per
// pixel is too slow under -race for the larger resize fixtures.
func gradient(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for index := 0; index < len(img.Pix); index += 4 {
		pixel := index / 4
		img.Pix[index] = uint8(pixel % 256)
		img.Pix[index+1] = uint8(pixel / width % 256)
		img.Pix[index+2] = 0x40
		img.Pix[index+3] = 0xFF
	}
	return img
}

// TestSanitizePublicImageStripsGPSExifFromJPEG is the acceptance proof for the
// documented privacy invariant: a real GPS-tagged JPEG must not reach dist/
// with its coordinates intact.
func TestSanitizePublicImageStripsGPSExifFromJPEG(t *testing.T) {
	fixture := jpegWithGPSExif(t, gradient(120, 80), 1)

	derivative, err := SanitizePublicImage("media/trip.jpg", bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}

	if tiff, ok := jpegExifTIFF(derivative); ok {
		t.Errorf("derivative still carries an APP1/Exif segment (%d bytes of TIFF)", len(tiff))
	}
	if bytes.Contains(derivative, []byte(exifSignature)) {
		t.Error("derivative still contains the Exif APP1 signature")
	}
	if bytes.Contains(derivative, gpsLatitudeSentinel) {
		t.Error("derivative still contains the GPS latitude bytes")
	}
	if bytes.Contains(derivative, gpsLongitudeSentinel) {
		t.Error("derivative still contains the GPS longitude bytes")
	}

	decoded, format, err := image.Decode(bytes.NewReader(derivative))
	if err != nil {
		t.Fatalf("derivative does not decode: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("derivative format = %q, want jpeg", format)
	}
	if got := decoded.Bounds().Size(); got.X != 120 || got.Y != 80 {
		t.Errorf("derivative size = %v, want 120x80", got)
	}
}

func TestResizeToFitCapsLongestEdgeWithoutUpscaling(t *testing.T) {
	cases := []struct {
		name                  string
		width, height         int
		wantWidth, wantHeight int
	}{
		// 1000 * 2048/3000 rounds to 683.
		{"oversized landscape", 3000, 1000, maxPublicEdge, 683},
		{"oversized portrait", 1000, 3000, 683, maxPublicEdge},
		{"already within the cap is never upscaled", 640, 480, 640, 480},
		{"exactly at the cap keeps its dimensions", maxPublicEdge, 64, maxPublicEdge, 64},
		{"an extreme panorama keeps at least one pixel of height", 40960, 1, maxPublicEdge, 1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := resizeToFit(image.NewRGBA(image.Rect(0, 0, testCase.width, testCase.height))).Bounds().Size()
			if got.X != testCase.wantWidth || got.Y != testCase.wantHeight {
				t.Errorf("size = %v, want %dx%d", got, testCase.wantWidth, testCase.wantHeight)
			}
		})
	}
}

// TestSanitizePublicImageResizesAndReEncodes proves the resize rule is actually
// wired into the published derivative, for both the oversized and the
// already-small case (the latter must still be re-encoded, hence stripped).
func TestSanitizePublicImageResizesAndReEncodes(t *testing.T) {
	cases := []struct {
		name                  string
		width, height         int
		wantWidth, wantHeight int
	}{
		// 400 * 2048/2600 rounds to 315.
		{"oversized", 2600, 400, maxPublicEdge, 315},
		{"already small", 320, 200, 320, 200},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var source bytes.Buffer
			if err := png.Encode(&source, gradient(testCase.width, testCase.height)); err != nil {
				t.Fatalf("encode fixture png: %v", err)
			}
			derivative, err := SanitizePublicImage("media/big.png", bytes.NewReader(source.Bytes()))
			if err != nil {
				t.Fatalf("sanitize: %v", err)
			}
			decoded, err := png.Decode(bytes.NewReader(derivative))
			if err != nil {
				t.Fatalf("derivative does not decode: %v", err)
			}
			got := decoded.Bounds().Size()
			if got.X != testCase.wantWidth || got.Y != testCase.wantHeight {
				t.Errorf("derivative size = %v, want %dx%d", got, testCase.wantWidth, testCase.wantHeight)
			}
		})
	}
}

// TestSanitizePNGDropsTextChunks proves the PNG round trip removes ancillary
// metadata chunks, which is where a PNG would carry eXIf or a tEXt comment.
func TestSanitizePNGDropsTextChunks(t *testing.T) {
	var source bytes.Buffer
	if err := png.Encode(&source, gradient(32, 16)); err != nil {
		t.Fatalf("encode fixture png: %v", err)
	}
	fixture := pngWithTextChunk(source.Bytes(), "Comment", "home address 1-2-3 Setagaya")

	derivative, err := SanitizePublicImage("media/note.png", bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if !bytes.Contains(fixture, []byte("home address 1-2-3 Setagaya")) {
		t.Fatal("fixture png does not carry the tEXt chunk")
	}
	if bytes.Contains(derivative, []byte("home address 1-2-3 Setagaya")) {
		t.Error("derivative still contains the PNG tEXt metadata")
	}
	if _, err := png.Decode(bytes.NewReader(derivative)); err != nil {
		t.Fatalf("derivative does not decode: %v", err)
	}
}

// pngWithTextChunk inserts a tEXt chunk after the 8-byte signature + IHDR.
func pngWithTextChunk(raw []byte, keyword, text string) []byte {
	const insertAt = 8 + 8 + 13 + 4 // signature + IHDR header + IHDR data + CRC
	payload := append([]byte(keyword+"\x00"), text...)
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(payload)))
	chunk = append(chunk, "tEXt"...)
	chunk = append(chunk, payload...)
	chunk = binary.BigEndian.AppendUint32(chunk, pngCRC(append([]byte("tEXt"), payload...)))

	out := make([]byte, 0, len(raw)+len(chunk))
	out = append(out, raw[:insertAt]...)
	out = append(out, chunk...)
	return append(out, raw[insertAt:]...)
}

func pngCRC(data []byte) uint32 {
	crc := ^uint32(0)
	for _, b := range data {
		crc ^= uint32(b)
		for range 8 {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// TestApplyOrientationTransformsPixels checks all eight EXIF orientations
// against a 2x3 image whose every pixel is distinguishable, so a swapped axis
// or an inverted mirror cannot pass.
func TestApplyOrientationTransformsPixels(t *testing.T) {
	// a b
	// c d
	// e f
	const (
		a, b = 10, 20
		c, d = 30, 40
		e, f = 50, 60
	)
	source := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for index, value := range []uint8{a, b, c, d, e, f} {
		source.Set(index%2, index/2, color.RGBA{R: value, A: 0xFF})
	}

	cases := []struct {
		orientation int
		want        [][]uint8
	}{
		{1, [][]uint8{{a, b}, {c, d}, {e, f}}},
		{2, [][]uint8{{b, a}, {d, c}, {f, e}}},
		{3, [][]uint8{{f, e}, {d, c}, {b, a}}},
		{4, [][]uint8{{e, f}, {c, d}, {a, b}}},
		{5, [][]uint8{{a, c, e}, {b, d, f}}},
		{6, [][]uint8{{e, c, a}, {f, d, b}}},
		{7, [][]uint8{{f, d, b}, {e, c, a}}},
		{8, [][]uint8{{b, d, f}, {a, c, e}}},
	}
	for _, testCase := range cases {
		got := applyOrientation(source, testCase.orientation)
		bounds := got.Bounds()
		if bounds.Dy() != len(testCase.want) || bounds.Dx() != len(testCase.want[0]) {
			t.Errorf("orientation %d: size = %v, want %dx%d", testCase.orientation, bounds.Size(), len(testCase.want[0]), len(testCase.want))
			continue
		}
		for y, row := range testCase.want {
			for x, want := range row {
				red, _, _, _ := got.At(x, y).RGBA()
				if uint8(red>>8) != want {
					t.Errorf("orientation %d: pixel (%d,%d) = %d, want %d", testCase.orientation, x, y, red>>8, want)
				}
			}
		}
	}
}

// TestSanitizeJPEGBakesOrientationIntoPixels guards the classic naive-strip
// bug: dropping EXIF also drops the rotation hint, so a portrait phone photo
// must come out physically rotated, not merely stripped.
func TestSanitizeJPEGBakesOrientationIntoPixels(t *testing.T) {
	fixture := jpegWithGPSExif(t, gradient(120, 60), 6) // 6 = rotate 90 clockwise

	derivative, err := SanitizePublicImage("media/portrait.jpg", bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(derivative))
	if err != nil {
		t.Fatalf("derivative does not decode: %v", err)
	}
	if got := decoded.Bounds().Size(); got.X != 60 || got.Y != 120 {
		t.Errorf("derivative size = %v, want the axes swapped to 60x120", got)
	}
}

// TestJPEGOrientationDefaultsToOneOnMalformedMetadata: orientation is cosmetic,
// so unreadable metadata must never fail a publish.
func TestJPEGOrientationDefaultsToOneOnMalformedMetadata(t *testing.T) {
	cases := map[string][]byte{
		"not a jpeg":          []byte("just some bytes"),
		"jpeg without exif":   {0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x02},
		"truncated app1":      {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x20, 'E', 'x'},
		"exif with bad order": append([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10}, append([]byte(exifSignature), 'Z', 'Z', 0, 0, 0, 0, 0, 0)...),
	}
	for name, raw := range cases {
		if got := jpegOrientation(raw); got != 1 {
			t.Errorf("%s: orientation = %d, want the safe default 1", name, got)
		}
	}
}

// minimalWebP is a genuine 34-byte lossless (VP8L) 1x1 WebP file.
const minimalWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func TestSanitizeWebPDropsMetadataChunks(t *testing.T) {
	base, err := base64.StdEncoding.DecodeString(minimalWebP)
	if err != nil {
		t.Fatalf("decode fixture webp: %v", err)
	}
	if _, _, decodeErr := image.Decode(bytes.NewReader(base)); decodeErr != nil {
		t.Fatalf("fixture webp does not decode: %v", decodeErr)
	}

	fixture := webpWithChunks(base, riffChunk("EXIF", append([]byte(exifSignature), gpsLatitudeSentinel...)), riffChunk("XMP ", []byte("<x:xmpmeta/>")))
	if !bytes.Contains(fixture, gpsLatitudeSentinel) {
		t.Fatal("fixture webp does not carry the GPS bytes")
	}

	derivative, err := SanitizePublicImage("media/stub.webp", bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if bytes.Contains(derivative, []byte("EXIF")) || bytes.Contains(derivative, gpsLatitudeSentinel) {
		t.Error("derivative still contains the EXIF chunk")
	}
	if bytes.Contains(derivative, []byte("XMP ")) {
		t.Error("derivative still contains the XMP chunk")
	}
	if !bytes.Equal(derivative, base) {
		t.Errorf("derivative = %x, want the metadata-free container %x", derivative, base)
	}
	if _, _, err := image.Decode(bytes.NewReader(derivative)); err != nil {
		t.Fatalf("derivative does not decode: %v", err)
	}
}

// TestSanitizeWebPClearsVP8XMetadataFlags: an extended container that keeps
// advertising ICC/EXIF/XMP after the chunks are gone is malformed.
func TestSanitizeWebPClearsVP8XMetadataFlags(t *testing.T) {
	vp8x := riffChunk("VP8X", []byte{0x2C, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // ICC|EXIF|XMP set
	fixture := webpWithChunks(nil, vp8x, riffChunk("ICCP", []byte("profile")), riffChunk("EXIF", []byte("exifdata")))

	derivative, err := SanitizePublicImage("media/extended.webp", bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if bytes.Contains(derivative, []byte("ICCP")) || bytes.Contains(derivative, []byte("EXIF")) {
		t.Error("derivative still contains metadata chunks")
	}
	// riffHeaderSize + 8-byte chunk header lands on the VP8X feature byte.
	if flags := derivative[riffHeaderSize+8]; flags&vp8xMetadataFlags != 0 {
		t.Errorf("VP8X feature byte = %#02x, want the ICC/EXIF/XMP bits cleared", flags)
	}
}

func riffChunk(fourCC string, payload []byte) []byte {
	chunk := append([]byte(fourCC), binary.LittleEndian.AppendUint32(nil, uint32(len(payload)))...)
	chunk = append(chunk, payload...)
	if len(payload)%2 == 1 {
		chunk = append(chunk, 0)
	}
	return chunk
}

// webpWithChunks rebuilds a RIFF/WEBP container from base's existing chunks
// (base may be nil) plus extra ones, fixing up the RIFF size.
func webpWithChunks(base []byte, extra ...[]byte) []byte {
	body := []byte(nil)
	if len(base) > riffHeaderSize {
		body = append(body, base[riffHeaderSize:]...)
	}
	for _, chunk := range extra {
		body = append(body, chunk...)
	}
	out := append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(len(body)+4))...)
	out = append(out, "WEBP"...)
	return append(out, body...)
}

func TestSanitizePublicImageRejectsUnsanitizableFormats(t *testing.T) {
	cases := map[string]string{
		"gif":               "media/animation.gif",
		"tiff":              "media/scan.tiff",
		"heic":              "media/photo.heic",
		"no extension":      "media/photo",
		"mislabelled bytes": "media/photo.jpg", // named .jpg but not a JPEG
	}
	for name, objectKey := range cases {
		if _, err := SanitizePublicImage(objectKey, strings.NewReader("not an image")); err == nil {
			t.Errorf("%s: sanitize succeeded, want a hard error so packaging fails instead of leaking", name)
		}
	}
}
