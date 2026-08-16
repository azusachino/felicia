#!/usr/bin/env python3
"""Build a photo sidecar from the capture metadata already in the files.

Local photo folders are timestamp-less to felicia: EXIF is only decoded on the
Immich path, so `journey plan` cannot attach a bare folder to a track. Until
provider-level extraction lands, this reads DateTimeOriginal and GPS straight
out of the JPEGs and writes the JSONL sidecar that `--sidecar` consumes.

    uv run python scripts/photo_sidecar.py ~/trip/photos --tz Asia/Tokyo

Photos whose timestamp cannot be read are reported, never skipped silently --
a sidecar that quietly omits half a trip is the failure this exists to avoid.
"""

from __future__ import annotations

import argparse
import json
import struct
import sys
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo

# HEIC is what an iPhone actually produces, so it is scanned even though the
# public package boundary will not accept it un-converted; PNG and WebP are
# scanned so a screenshot without capture time is reported rather than ignored.
SUFFIXES = {".jpg", ".jpeg", ".heic", ".heif", ".png", ".webp"}
TYPE_SIZES = {1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 7: 1, 9: 4, 10: 8}
TAG_DATETIME = 0x0132
TAG_EXIF_IFD = 0x8769
TAG_GPS_IFD = 0x8825
TAG_DATETIME_ORIGINAL = 0x9003


def _entries(data: bytes, order: str, offset: int) -> dict[int, tuple[int, int, bytes]]:
    """Read one IFD into {tag: (type, count, raw value bytes)}."""
    if offset + 2 > len(data):
        return {}
    (count,) = struct.unpack_from(order + "H", data, offset)
    found: dict[int, tuple[int, int, bytes]] = {}
    for index in range(count):
        entry = offset + 2 + index * 12
        if entry + 12 > len(data):
            break
        tag, kind, length = struct.unpack_from(order + "HHI", data, entry)
        size = TYPE_SIZES.get(kind, 0) * length
        if size == 0:
            continue
        if size <= 4:
            raw = data[entry + 8 : entry + 8 + size]
        else:
            (value_offset,) = struct.unpack_from(order + "I", data, entry + 8)
            raw = data[value_offset : value_offset + size]
        found[tag] = (kind, length, raw)
    return found


def _ascii(raw: bytes) -> str:
    return raw.split(b"\x00")[0].decode("ascii", "replace")


def _rationals(raw: bytes, order: str, count: int) -> list[float]:
    values = []
    for index in range(count):
        numerator, denominator = struct.unpack_from(order + "II", raw, index * 8)
        values.append(numerator / denominator if denominator else 0.0)
    return values


def _degrees(raw: bytes, order: str, reference: str) -> float | None:
    parts = _rationals(raw, order, 3)
    if len(parts) < 3:
        return None
    value = parts[0] + parts[1] / 60 + parts[2] / 3600
    return -value if reference in {"S", "W"} else value


def _tiff_offset(data: bytes) -> int | None:
    """Find the TIFF header of the embedded EXIF block.

    JPEG carries it in an APP1 segment, HEIC in a metadata item, and HEIC also
    spells "Exif" once more in the box that declares the item's type -- so the
    marker alone is not enough. Anchor on the one followed by a byte-order mark.
    """
    cursor = 0
    while True:
        marker = data.find(b"Exif\x00\x00", cursor)
        if marker < 0:
            return None
        start = marker + 6
        if data[start : start + 4] in {b"II\x2a\x00", b"MM\x00\x2a"}:
            return start
        cursor = marker + 1


def read_capture(path: Path) -> tuple[str | None, list[float] | None]:
    """Return (EXIF datetime string, [lng, lat]) for one photo."""
    data = path.read_bytes()
    offset = _tiff_offset(data)
    if offset is None:
        return None, None
    tiff = data[offset:]
    if len(tiff) < 8:
        return None, None
    order = "<" if tiff[:2] == b"II" else ">"
    (first,) = struct.unpack_from(order + "I", tiff, 4)
    ifd0 = _entries(tiff, order, first)

    stamp = None
    if TAG_EXIF_IFD in ifd0:
        (pointer,) = struct.unpack_from(order + "I", ifd0[TAG_EXIF_IFD][2], 0)
        exif = _entries(tiff, order, pointer)
        if TAG_DATETIME_ORIGINAL in exif:
            stamp = _ascii(exif[TAG_DATETIME_ORIGINAL][2])
    if stamp is None and TAG_DATETIME in ifd0:
        stamp = _ascii(ifd0[TAG_DATETIME][2])

    coord = None
    if TAG_GPS_IFD in ifd0:
        (pointer,) = struct.unpack_from(order + "I", ifd0[TAG_GPS_IFD][2], 0)
        gps = _entries(tiff, order, pointer)
        if 0x0002 in gps and 0x0004 in gps:
            latitude = _degrees(gps[0x0002][2], order, _ascii(gps.get(0x0001, (2, 1, b"N"))[2]))
            longitude = _degrees(gps[0x0004][2], order, _ascii(gps.get(0x0003, (2, 1, b"E"))[2]))
            if latitude is not None and longitude is not None:
                coord = [round(longitude, 7), round(latitude, 7)]  # [lng, lat]
    return stamp, coord


def build(photos: Path, zone: ZoneInfo) -> tuple[list[dict], list[str]]:
    records: list[dict] = []
    missing: list[str] = []
    for path in sorted(photos.rglob("*")):
        if not path.is_file() or path.suffix.lower() not in SUFFIXES:
            continue
        relative = path.relative_to(photos).as_posix()
        stamp, coord = read_capture(path)
        if not stamp:
            missing.append(relative)
            continue
        try:
            naive = datetime.strptime(stamp, "%Y:%m:%d %H:%M:%S")
        except ValueError:
            missing.append(relative)
            continue
        record = {
            "path": relative,
            "at": naive.replace(tzinfo=zone).isoformat().replace("+00:00", "Z"),
            "title": path.stem,
        }
        if coord:
            record["coord"] = coord
        records.append(record)
    return records, missing


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("photos", type=Path, help="photo folder passed to PHOTOS=")
    parser.add_argument("--tz", default="UTC", help="zone the camera clock was in (default UTC)")
    parser.add_argument("--out", type=Path, help="write here instead of stdout")
    arguments = parser.parse_args()

    records, missing = build(arguments.photos.resolve(), ZoneInfo(arguments.tz))
    payload = "".join(json.dumps(record, ensure_ascii=False) + "\n" for record in records)
    if arguments.out:
        arguments.out.write_text(payload, encoding="utf-8")
        print(f"sidecar written: {arguments.out} ({len(records)} photos)", file=sys.stderr)
    else:
        sys.stdout.write(payload)
    if missing:
        print(f"no capture time in {len(missing)} file(s):", file=sys.stderr)
        for name in missing:
            print(f"  {name}", file=sys.stderr)
        print("add these by hand, or they will not attach to the track.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
