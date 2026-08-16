"""Cover the sidecar generator against a fabricated EXIF JPEG.

The repository's sample images are EXIF-stripped, so the fixture is built here:
a real APP1 segment with DateTimeOriginal and a GPS IFD, in both byte orders.
"""

import struct
import tempfile
import unittest
from pathlib import Path
from zoneinfo import ZoneInfo

from scripts.photo_sidecar import build, read_capture


def _ifd(order: str, entries: list[tuple[int, int, int, bytes]], next_offset: int = 0) -> bytes:
    """Pack one IFD; entries are (tag, type, count, inline-or-offset bytes)."""
    packed = struct.pack(order + "H", len(entries))
    for tag, kind, count, raw in entries:
        packed += struct.pack(order + "HHI", tag, kind, count) + raw.ljust(4, b"\x00")[:4]
    return packed + struct.pack(order + "I", next_offset)


def exif_jpeg(order: str = "<", stamp: bytes = b"2026:04:18 00:25:00\x00", gps: bool = True) -> bytes:
    """A JPEG carrying a genuine EXIF APP1 segment."""
    mark = b"II\x2a\x00" if order == "<" else b"MM\x00\x2a"
    # Layout: header(8) | IFD0 | ExifIFD | GPS IFD | heap
    heap: list[bytes] = []
    heap_start = 8

    ifd0_entries = 2 if gps else 1
    ifd0_size = 2 + ifd0_entries * 12 + 4
    exif_size = 2 + 1 * 12 + 4
    gps_size = 2 + 4 * 12 + 4 if gps else 0
    exif_offset = heap_start + ifd0_size
    gps_offset = exif_offset + exif_size
    heap_start = gps_offset + gps_size

    stamp_offset = heap_start
    heap.append(stamp)
    cursor = stamp_offset + len(stamp)

    latitude_offset = cursor
    if gps:
        heap.append(b"".join(struct.pack(order + "II", value, 1) for value in (35, 1, 30)))
        cursor += 24
        longitude_offset = cursor
        heap.append(b"".join(struct.pack(order + "II", value, 1) for value in (135, 47, 0)))
        cursor += 24

    ifd0 = [(0x8769, 4, 1, struct.pack(order + "I", exif_offset))]
    if gps:
        ifd0.append((0x8825, 4, 1, struct.pack(order + "I", gps_offset)))
    body = _ifd(order, ifd0)
    body += _ifd(order, [(0x9003, 2, len(stamp), struct.pack(order + "I", stamp_offset))])
    if gps:
        body += _ifd(
            order,
            [
                (0x0001, 2, 2, b"N\x00"),
                (0x0002, 5, 3, struct.pack(order + "I", latitude_offset)),
                (0x0003, 2, 2, b"E\x00"),
                (0x0004, 5, 3, struct.pack(order + "I", longitude_offset)),
            ],
        )
    tiff = mark + struct.pack(order + "I", 8) + body + b"".join(heap)
    app1 = b"Exif\x00\x00" + tiff
    return (
        b"\xff\xd8"
        + b"\xff\xe1"
        + struct.pack(">H", len(app1) + 2)
        + app1
        + b"\xff\xd9"
    )


class PhotoSidecarTest(unittest.TestCase):
    def test_reads_timestamp_and_gps_in_both_byte_orders(self):
        for order in ("<", ">"):
            with self.subTest(order=order), tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "shot.jpg"
                path.write_bytes(exif_jpeg(order))
                stamp, coord = read_capture(path)
                self.assertEqual(stamp, "2026:04:18 00:25:00")
                self.assertIsNotNone(coord)
                longitude, latitude = coord
                # 35° 1' 30" and 135° 47' 0", as packed by exif_jpeg.
                self.assertAlmostEqual(latitude, 35.025, places=4)
                self.assertAlmostEqual(longitude, 135.7833333, places=4)

    def test_applies_the_requested_zone_and_keeps_relative_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "day-one").mkdir()
            (root / "day-one" / "shot.jpg").write_bytes(exif_jpeg())
            records, missing = build(root, ZoneInfo("Asia/Tokyo"))
            self.assertEqual(missing, [])
            self.assertEqual(len(records), 1)
            self.assertEqual(records[0]["path"], "day-one/shot.jpg")
            self.assertEqual(records[0]["at"], "2026-04-18T00:25:00+09:00")
            self.assertEqual(records[0]["title"], "shot")

    def test_reports_photos_without_capture_time_instead_of_dropping_them(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "with-exif.jpg").write_bytes(exif_jpeg())
            (root / "stripped.jpg").write_bytes(b"\xff\xd8\xff\xd9")
            records, missing = build(root, ZoneInfo("UTC"))
            self.assertEqual([record["path"] for record in records], ["with-exif.jpg"])
            self.assertEqual(missing, ["stripped.jpg"], "a silent omission is the bug this avoids")


if __name__ == "__main__":
    unittest.main()
