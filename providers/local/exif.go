package local

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/paulmach/orb"
)

// This file decodes the same EXIF tags, by the same byte-scanning method, as
// scripts/photo_sidecar.py — so a photo never gets one timestamp from the
// script's sidecar and a different one from this provider. Any change here
// should stay in lockstep with that script.
//
// exifDateLayout is EXIF's zone-less "YYYY:MM:DD HH:MM:SS" wall-clock format.
const exifDateLayout = "2006:01:02 15:04:05"

// EXIF tag IDs this project trusts (matches TAG_* in photo_sidecar.py).
const (
	tagDateTime         = 0x0132
	tagExifIFD          = 0x8769
	tagGPSIFD           = 0x8825
	tagDateTimeOriginal = 0x9003
	tagGPSLatitudeRef   = 0x0001
	tagGPSLatitude      = 0x0002
	tagGPSLongitudeRef  = 0x0003
	tagGPSLongitude     = 0x0004
)

// exifTypeSizes mirrors TYPE_SIZES in photo_sidecar.py: byte width per EXIF
// field type, keyed by the numeric type tag (BYTE, ASCII, SHORT, LONG,
// RATIONAL, UNDEFINED, SLONG, SRATIONAL).
var exifTypeSizes = map[uint16]int{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 7: 1, 9: 4, 10: 8}

// decodeExif recovers a capture timestamp and GPS coordinate from an image's
// embedded EXIF block, in that order of trust: EXIF DateTimeOriginal, then
// the coarser DateTime tag; GPS is read only if both latitude and longitude
// are present. It never errors — corrupt, truncated, or absent EXIF simply
// yields a zero time and a nil coordinate, matching pre-EXIF behavior for
// files this cannot read.
//
// at is a naive wall-clock reading with no UTC offset — EXIF DateTimeOriginal
// carries none. It is returned in time.UTC purely as a placeholder Location
// so callers get a valid time.Time; the numeric fields (year..second) are
// the camera's own clock reading, unconverted. Resolving that wall-clock
// value to a real instant (e.g. from GPS, per issue #58) is explicitly out of
// scope here — this function does not guess a zone.
func decodeExif(data []byte) (at time.Time, coord *orb.Point) {
	// Corrupt EXIF must degrade the import, never panic it; every read below
	// is already bounds-checked, but this is a deliberate second line of
	// defense against a case the checks above missed.
	defer func() {
		if recover() != nil {
			at, coord = time.Time{}, nil
		}
	}()
	offset, ok := tiffOffset(data)
	if !ok {
		return time.Time{}, nil
	}
	tiff := data[offset:]
	if len(tiff) < 8 {
		return time.Time{}, nil
	}
	order, ok := byteOrder(tiff)
	if !ok {
		return time.Time{}, nil
	}
	first, ok := readUint32(tiff, order, 4)
	if !ok {
		return time.Time{}, nil
	}
	ifd0, ok := readIFD(tiff, order, int(first))
	if !ok {
		return time.Time{}, nil
	}

	at = readDateTime(tiff, order, ifd0)
	coord = readGPS(tiff, order, ifd0)
	return at, coord
}

// tiffOffset finds the TIFF header of an embedded EXIF block. JPEG carries it
// in an APP1 segment, HEIC in a metadata item — in both containers it is
// introduced by the literal "Exif\x00\x00" marker followed by a byte-order
// mark, so a plain byte scan for that marker locates it in either container
// without needing to parse the surrounding box/segment structure. Mirrors
// _tiff_offset in photo_sidecar.py.
func tiffOffset(data []byte) (int, bool) {
	marker := []byte("Exif\x00\x00")
	cursor := 0
	for {
		index := bytes.Index(data[cursor:], marker)
		if index < 0 {
			return 0, false
		}
		start := cursor + index + len(marker)
		if start+4 <= len(data) {
			head := data[start : start+4]
			if bytes.Equal(head, []byte("II\x2a\x00")) || bytes.Equal(head, []byte("MM\x00\x2a")) {
				return start, true
			}
		}
		cursor += index + 1
	}
}

func byteOrder(tiff []byte) (binary.ByteOrder, bool) {
	switch {
	case bytes.HasPrefix(tiff, []byte("II")):
		return binary.LittleEndian, true
	case bytes.HasPrefix(tiff, []byte("MM")):
		return binary.BigEndian, true
	default:
		return nil, false
	}
}

type exifEntry struct {
	kind  uint16
	count uint32
	raw   []byte
}

// readIFD reads one Image File Directory into {tag: entry}, mirroring
// _entries in photo_sidecar.py.
func readIFD(tiff []byte, order binary.ByteOrder, offset int) (map[uint16]exifEntry, bool) {
	if offset < 0 || offset+2 > len(tiff) {
		return nil, false
	}
	count := order.Uint16(tiff[offset : offset+2])
	found := make(map[uint16]exifEntry, count)
	for index := 0; index < int(count); index++ {
		entry := offset + 2 + index*12
		if entry+12 > len(tiff) {
			break
		}
		tag := order.Uint16(tiff[entry : entry+2])
		kind := order.Uint16(tiff[entry+2 : entry+4])
		length := order.Uint32(tiff[entry+4 : entry+8])
		size := exifTypeSizes[kind] * int(length)
		if size <= 0 {
			continue
		}
		var raw []byte
		if size <= 4 {
			if entry+8+size > len(tiff) {
				continue
			}
			raw = tiff[entry+8 : entry+8+size]
		} else {
			valueOffset, ok := readUint32(tiff, order, entry+8)
			if !ok {
				continue
			}
			if int(valueOffset)+size > len(tiff) || valueOffset > uint32(len(tiff)) {
				continue
			}
			raw = tiff[valueOffset : int(valueOffset)+size]
		}
		found[tag] = exifEntry{kind: kind, count: length, raw: raw}
	}
	return found, true
}

func readUint32(data []byte, order binary.ByteOrder, offset int) (uint32, bool) {
	if offset < 0 || offset+4 > len(data) {
		return 0, false
	}
	return order.Uint32(data[offset : offset+4]), true
}

func asciiValue(raw []byte) string {
	if index := bytes.IndexByte(raw, 0); index >= 0 {
		raw = raw[:index]
	}
	return string(raw)
}

func readDateTime(tiff []byte, order binary.ByteOrder, ifd0 map[uint16]exifEntry) time.Time {
	var stamp string
	if entry, ok := ifd0[tagExifIFD]; ok && len(entry.raw) >= 4 {
		pointer := order.Uint32(entry.raw[:4])
		if exif, ok := readIFD(tiff, order, int(pointer)); ok {
			if dto, ok := exif[tagDateTimeOriginal]; ok {
				stamp = asciiValue(dto.raw)
			}
		}
	}
	if stamp == "" {
		if dt, ok := ifd0[tagDateTime]; ok {
			stamp = asciiValue(dt.raw)
		}
	}
	if stamp == "" {
		return time.Time{}
	}
	parsed, err := time.ParseInLocation(exifDateLayout, stamp, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func readGPS(tiff []byte, order binary.ByteOrder, ifd0 map[uint16]exifEntry) *orb.Point {
	entry, ok := ifd0[tagGPSIFD]
	if !ok || len(entry.raw) < 4 {
		return nil
	}
	pointer := order.Uint32(entry.raw[:4])
	gps, ok := readIFD(tiff, order, int(pointer))
	if !ok {
		return nil
	}
	latRaw, ok := gps[tagGPSLatitude]
	if !ok {
		return nil
	}
	lonRaw, ok := gps[tagGPSLongitude]
	if !ok {
		return nil
	}
	latRef := "N"
	if entry, ok := gps[tagGPSLatitudeRef]; ok {
		latRef = asciiValue(entry.raw)
	}
	lonRef := "E"
	if entry, ok := gps[tagGPSLongitudeRef]; ok {
		lonRef = asciiValue(entry.raw)
	}
	latitude, ok := degrees(latRaw.raw, order, latRef)
	if !ok {
		return nil
	}
	longitude, ok := degrees(lonRaw.raw, order, lonRef)
	if !ok {
		return nil
	}
	if longitude < -180 || longitude > 180 || latitude < -90 || latitude > 90 {
		return nil
	}
	point := orb.Point{longitude, latitude}
	return &point
}

// rationals reads count consecutive 8-byte EXIF RATIONAL values.
func rationals(raw []byte, order binary.ByteOrder, count int) ([]float64, bool) {
	if len(raw) < count*8 {
		return nil, false
	}
	values := make([]float64, 0, count)
	for index := 0; index < count; index++ {
		numerator := order.Uint32(raw[index*8 : index*8+4])
		denominator := order.Uint32(raw[index*8+4 : index*8+8])
		if denominator == 0 {
			values = append(values, 0)
			continue
		}
		values = append(values, float64(numerator)/float64(denominator))
	}
	return values, true
}

// degrees converts a GPSLatitude/GPSLongitude triple (degrees, minutes,
// seconds as EXIF RATIONALs) into signed decimal degrees.
func degrees(raw []byte, order binary.ByteOrder, ref string) (float64, bool) {
	parts, ok := rationals(raw, order, 3)
	if !ok || len(parts) < 3 {
		return 0, false
	}
	value := parts[0] + parts[1]/60 + parts[2]/3600
	if ref == "S" || ref == "W" {
		value = -value
	}
	return value, true
}
