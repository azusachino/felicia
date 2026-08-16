package local

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// --- minimal little-endian TIFF/EXIF builder for fixtures -----------------
//
// decodeExif only cares about byte layout, not about being embedded in a
// real JPEG/HEIC container, so these helpers build the smallest TIFF blob
// that exercises each EXIF tag this project trusts (mirrors the tags read by
// scripts/photo_sidecar.py) rather than committing real photo files.

type exifFieldSpec struct {
	tag   uint16
	kind  uint16
	count uint32
	value []byte
}

func asciiField(tag uint16, s string) exifFieldSpec {
	value := append([]byte(s), 0)
	return exifFieldSpec{tag: tag, kind: 2, count: uint32(len(value)), value: value}
}

func longField(tag uint16, v uint32) exifFieldSpec {
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, v)
	return exifFieldSpec{tag: tag, kind: 4, count: 1, value: value}
}

func rationalField(tag uint16, pairs [][2]uint32) exifFieldSpec {
	value := make([]byte, 0, len(pairs)*8)
	for _, pair := range pairs {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[0:4], pair[0])
		binary.LittleEndian.PutUint32(b[4:8], pair[1])
		value = append(value, b...)
	}
	return exifFieldSpec{tag: tag, kind: 5, count: uint32(len(pairs)), value: value}
}

// encodeIFD serializes one little-endian IFD whose block starts at the
// absolute (tiff-relative) offset base, matching the TIFF/EXIF layout that
// decodeExif's readIFD expects.
func encodeIFD(base int, fields []exifFieldSpec) []byte {
	entryAreaLen := 2 + len(fields)*12 + 4
	var entries, extra bytes.Buffer
	for _, f := range fields {
		var head [8]byte
		binary.LittleEndian.PutUint16(head[0:2], f.tag)
		binary.LittleEndian.PutUint16(head[2:4], f.kind)
		binary.LittleEndian.PutUint32(head[4:8], f.count)
		entries.Write(head[:])
		if len(f.value) <= 4 {
			var inline [4]byte
			copy(inline[:], f.value)
			entries.Write(inline[:])
		} else {
			var off [4]byte
			binary.LittleEndian.PutUint32(off[:], uint32(base+entryAreaLen+extra.Len()))
			entries.Write(off[:])
			extra.Write(f.value)
		}
	}
	var out bytes.Buffer
	var count [2]byte
	binary.LittleEndian.PutUint16(count[:], uint16(len(fields)))
	out.Write(count[:])
	out.Write(entries.Bytes())
	out.Write(make([]byte, 4)) // next-IFD offset: none
	out.Write(extra.Bytes())
	return out.Bytes()
}

// buildTIFF assembles a full little-endian TIFF header + IFD0, optionally
// with an Exif sub-IFD and/or a GPS sub-IFD, and returns it prefixed with the
// "Exif\x00\x00" marker decodeExif scans for (plus a little unrelated junk,
// standing in for surrounding JPEG/HEIC container bytes).
func buildTIFF(ifd0Fields, exifFields, gpsFields []exifFieldSpec) []byte {
	const headerLen = 8
	fields := append([]exifFieldSpec{}, ifd0Fields...)
	if exifFields != nil {
		fields = append(fields, longField(tagExifIFD, 0))
	}
	if gpsFields != nil {
		fields = append(fields, longField(tagGPSIFD, 0))
	}
	ifd0Len := len(encodeIFD(headerLen, fields))
	exifOffset := headerLen + ifd0Len
	var exifBytes []byte
	if exifFields != nil {
		exifBytes = encodeIFD(exifOffset, exifFields)
	}
	gpsOffset := exifOffset + len(exifBytes)
	var gpsBytes []byte
	if gpsFields != nil {
		gpsBytes = encodeIFD(gpsOffset, gpsFields)
	}
	for i, f := range fields {
		switch f.tag {
		case tagExifIFD:
			fields[i] = longField(tagExifIFD, uint32(exifOffset))
		case tagGPSIFD:
			fields[i] = longField(tagGPSIFD, uint32(gpsOffset))
		}
	}
	ifd0Bytes := encodeIFD(headerLen, fields)

	var header [8]byte
	copy(header[0:2], "II")
	binary.LittleEndian.PutUint16(header[2:4], 0x2A)
	binary.LittleEndian.PutUint32(header[4:8], headerLen)

	var tiff bytes.Buffer
	tiff.Write(header[:])
	tiff.Write(ifd0Bytes)
	tiff.Write(exifBytes)
	tiff.Write(gpsBytes)

	var out bytes.Buffer
	out.WriteString("junk-container-bytes-before-marker")
	out.WriteString("Exif\x00\x00")
	out.Write(tiff.Bytes())
	return out.Bytes()
}

func gpsRef(tag uint16, ref string) exifFieldSpec { return asciiField(tag, ref) }

// degreesToRational converts decimal degrees into an unsigned D/M/S rational
// triple (the sign is carried separately by the ref tag, per EXIF).
func degreesToRational(value float64) [][2]uint32 {
	if value < 0 {
		value = -value
	}
	degrees := uint32(value)
	minutesFull := (value - float64(degrees)) * 60
	minutes := uint32(minutesFull)
	seconds := (minutesFull - float64(minutes)) * 60
	return [][2]uint32{{degrees, 1}, {minutes, 1}, {uint32(seconds * 1000), 1000}}
}

func TestDecodeExif(t *testing.T) {
	wantTime := time.Date(2026, 4, 1, 9, 15, 30, 0, time.UTC)

	tests := []struct {
		name      string
		data      []byte
		wantAt    time.Time
		wantCoord bool
		wantLon   float64
		wantLat   float64
	}{
		{
			name:   "DateTimeOriginal in Exif sub-IFD",
			data:   buildTIFF(nil, []exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")}, nil),
			wantAt: wantTime,
		},
		{
			name:   "falls back to IFD0 DateTime when no DateTimeOriginal",
			data:   buildTIFF([]exifFieldSpec{asciiField(tagDateTime, "2026:04:01 09:15:30")}, nil, nil),
			wantAt: wantTime,
		},
		{
			name: "GPS present alongside a timestamp",
			data: buildTIFF(nil,
				[]exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")},
				[]exifFieldSpec{
					gpsRef(tagGPSLatitudeRef, "N"),
					rationalField(tagGPSLatitude, degreesToRational(35.0116)),
					gpsRef(tagGPSLongitudeRef, "E"),
					rationalField(tagGPSLongitude, degreesToRational(135.7681)),
				}),
			wantAt:    wantTime,
			wantCoord: true,
			wantLon:   135.7681,
			wantLat:   35.0116,
		},
		{
			name: "southern and western hemisphere refs negate degrees",
			data: buildTIFF(nil, nil, []exifFieldSpec{
				gpsRef(tagGPSLatitudeRef, "S"),
				rationalField(tagGPSLatitude, degreesToRational(33.8688)),
				gpsRef(tagGPSLongitudeRef, "W"),
				rationalField(tagGPSLongitude, degreesToRational(151.2093)),
			}),
			wantCoord: true,
			wantLon:   -151.2093,
			wantLat:   -33.8688,
		},
		{
			name: "no Exif marker at all degrades without error",
			data: []byte("plain bytes, not a photo with any embedded metadata"),
		},
		{
			name: "Exif marker present but truncated immediately after the TIFF header",
			data: func() []byte {
				full := buildTIFF(nil, []exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")}, nil)
				marker := bytes.Index(full, []byte("Exif\x00\x00"))
				return full[:marker+len("Exif\x00\x00")+6]
			}(),
		},
		{
			// The single IFD0 field is the auto-added Exif sub-IFD pointer;
			// corrupting its offset to point far past the buffer must make
			// the sub-IFD read fail cleanly rather than reading garbage or
			// panicking, and DateTimeOriginal must not be found some other
			// way.
			name: "Exif marker present but the sub-IFD pointer is corrupt/out of range",
			data: func() []byte {
				full := buildTIFF(nil, []exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")}, nil)
				marker := bytes.Index(full, []byte("Exif\x00\x00"))
				tiffStart := marker + len("Exif\x00\x00")
				// IFD0 layout here: header(8) + count(2) + [tag(2) kind(2) count(4) value(4)] for the one field.
				valueOffset := tiffStart + 8 + 2 + 8
				corrupt := append([]byte{}, full...)
				binary.LittleEndian.PutUint32(corrupt[valueOffset:valueOffset+4], 0x7FFFFFFF)
				return corrupt
			}(),
		},
		{
			name: "empty input degrades without error",
			data: []byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			at, coord := decodeExif(tc.data)
			if !at.Equal(tc.wantAt) {
				t.Fatalf("at = %v, want %v", at, tc.wantAt)
			}
			if tc.wantCoord {
				if coord == nil {
					t.Fatalf("coord = nil, want lon=%v lat=%v", tc.wantLon, tc.wantLat)
				}
				if lon, lat := coord.Lon(), coord.Lat(); !almostEqual(lon, tc.wantLon) || !almostEqual(lat, tc.wantLat) {
					t.Fatalf("coord = (%v, %v), want (%v, %v)", lon, lat, tc.wantLon, tc.wantLat)
				}
			} else if coord != nil {
				t.Fatalf("coord = %v, want nil", coord)
			}
		})
	}
}

func almostEqual(a, b float64) bool {
	const epsilon = 1e-4
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}
