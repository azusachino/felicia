"""Guard issue #72 (a second trip overwrites the first) and issue #75 (media
paths collide across trips) end to end.

This drives the real pipeline through the built `felicia-cli` binary --
`journey plan` (via `preprocess`), `package validate`, and `import --apply` --
for two distinct trips, then reads the resulting SQLite database directly to
prove both journeys, both mementos, and both same-named-but-different photos
survive side by side. It complements the unit-level coverage in
`tests/test_local_journey.py` (identity derivation, content-addressed keys in
isolation) with the end-to-end claim the issues actually make: a second trip
must coexist with the first, not replace it.
"""

from __future__ import annotations

import base64
import json
import sqlite3
import tempfile
import unittest
from argparse import Namespace
from pathlib import Path

from scripts.local_journey import build_package, preprocess
from scripts.local_journey_common import CLI, ensure_cli, run
from scripts.validate_local_authoring import validate_workspace

FIXTURE = Path(__file__).resolve().parent / "fixtures" / "local-journey-raw"
JOURNAL_ID = "0190cbde-f300-7000-8000-000000000000"

# Two distinct 1x1 JPEGs -- same eventual basename ("shot.jpg") in each trip's
# photo folder, different bytes, so a content-addressed key is the only thing
# that can keep them from colliding (issue #75).
PHOTO_A = base64.b64decode(
    "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHR"
    "ofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QA"
    "FAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AKp//2Q=="
)
PHOTO_B = base64.b64decode(
    "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHR"
    "ofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAIBAREA/8QA"
    "FAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AKp//2Q=="
)

# Inside the fixture track's first 20+ minute dwell (see route.gpx).
PHOTO_AT = "2026-04-18T00:25:00Z"


class MultiTripImportTest(unittest.TestCase):
    """Two trips, imported into the same database, must coexist."""

    @classmethod
    def setUpClass(cls) -> None:
        ensure_cli()

    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory(prefix="felicia-multi-trip-")
        self.addCleanup(self._tmp.cleanup)
        self.root = Path(self._tmp.name)
        self.database = self.root / "felicia.sqlite"
        self.media_root = self.root / "media"

    def _single_dwell_gpx(self, label: str) -> Path:
        """One trip's track: only the fixture's *first* 20+ minute dwell (the
        fixture has two), so exactly one stop/memento is derived -- keeping
        this test's expectations independent of the fixture's own dwell
        count. Each label produces different bytes (different track name),
        so two labels hash to two different identities.
        """
        original = (FIXTURE / "route.gpx").read_text(encoding="utf-8")
        marker = '<trkpt lat="35.025000" lon="135.788000"><time>2026-04-18T00:40:00Z</time></trkpt>'
        cutoff = original.index(marker) + len(marker)
        trimmed = original[:cutoff] + "\n    </trkseg>\n  </trk>\n</gpx>"
        trimmed = trimmed.replace("two-dwell walk", f"single-dwell walk ({label})")
        path = self.root / f"route-{label}.gpx"
        path.write_text(trimmed, encoding="utf-8")
        return path

    def _preprocess_trip(self, label: str, gpx: Path, photo_bytes: bytes) -> Path:
        workspace = self.root / f"workspace-{label}"
        photos = self.root / f"photos-{label}"
        photos.mkdir(parents=True, exist_ok=True)
        (photos / "shot.jpg").write_bytes(photo_bytes)
        sidecar = self.root / f"sidecar-{label}.jsonl"
        sidecar.write_text(json.dumps({"path": "shot.jpg", "at": PHOTO_AT, "title": f"Trip {label}"}) + "\n")
        preprocess(
            Namespace(
                workspace=workspace,
                journey=None,
                journal=JOURNAL_ID,
                slug=None,
                title=None,
                gpx=gpx,
                photos=photos,
                sidecar=sidecar,
            )
        )
        validate_workspace(workspace)
        return workspace

    def _import(self, workspace: Path) -> None:
        package = build_package(Namespace(workspace=workspace))
        run([str(CLI), "package", "validate", str(package)])
        run(
            [
                str(CLI),
                "import",
                "--db",
                str(self.database),
                "--media-root",
                str(self.media_root),
                "--apply",
                str(package),
            ]
        )

    def _journeys(self) -> list[sqlite3.Row]:
        connection = sqlite3.connect(self.database)
        try:
            connection.row_factory = sqlite3.Row
            return connection.execute("SELECT id, slug FROM tb_journeys ORDER BY slug").fetchall()
        finally:
            connection.close()

    def _photo_object_keys(self) -> list[str]:
        connection = sqlite3.connect(self.database)
        try:
            return [row[0] for row in connection.execute("SELECT object_key FROM tb_memento_photos ORDER BY object_key")]
        finally:
            connection.close()

    def test_two_trips_get_distinct_ids_and_both_survive_import(self) -> None:
        workspace_a = self._preprocess_trip("a", self._single_dwell_gpx("a"), PHOTO_A)
        workspace_b = self._preprocess_trip("b", self._single_dwell_gpx("b"), PHOTO_B)

        journey_a = json.loads((workspace_a / "journey.json").read_text())
        journey_b = json.loads((workspace_b / "journey.json").read_text())
        self.assertNotEqual(journey_a["id"], journey_b["id"], "distinct trips must not share a journey id")
        self.assertNotEqual(journey_a["slug"], journey_b["slug"], "distinct trips must not share a slug")

        mementos_a = {m["id"] for m in json.loads((workspace_a / "mementos.json").read_text())["mementos"]}
        mementos_b = {m["id"] for m in json.loads((workspace_b / "mementos.json").read_text())["mementos"]}
        self.assertTrue(mementos_a, "trip A should have derived at least one memento")
        self.assertTrue(mementos_b, "trip B should have derived at least one memento")
        self.assertEqual(set(), mementos_a & mementos_b, "memento ids must not collide across trips")

        self._import(workspace_a)
        self._import(workspace_b)

        rows = self._journeys()
        self.assertEqual(2, len(rows), f"both journeys should have their own row, got {[dict(r) for r in rows]}")
        self.assertEqual({journey_a["id"], journey_b["id"]}, {row["id"] for row in rows})

        # issue #75: same basename ("shot.jpg"), different bytes -- must land
        # at two different object keys, and the Go importer (which stores the
        # package's photo path verbatim, see runtime/importer/package.go and
        # cli/cmd/felicia/main.go) must actually have written both files.
        object_keys = self._photo_object_keys()
        self.assertEqual(2, len(object_keys), f"expected one photo per trip, got {object_keys!r}")
        self.assertEqual(len(set(object_keys)), len(object_keys), "same-named photos from different trips collided on one key")
        for key in object_keys:
            self.assertTrue((self.media_root / key).is_file(), f"imported media missing on disk at {key}")

    def test_rerunning_the_same_trip_stays_idempotent(self) -> None:
        gpx = self._single_dwell_gpx("a")
        workspace_first = self._preprocess_trip("a", gpx, PHOTO_A)
        journey_first = json.loads((workspace_first / "journey.json").read_text())
        self._import(workspace_first)

        # Re-run preprocess for the *same* trip (same gpx bytes, same photo,
        # same default workspace directory -- "a") and import again.
        workspace_again = self._preprocess_trip("a", gpx, PHOTO_A)
        journey_again = json.loads((workspace_again / "journey.json").read_text())
        self.assertEqual(workspace_first, workspace_again, "the same trip must resolve to the same workspace")
        self.assertEqual(journey_first["id"], journey_again["id"], "re-running the same trip must not mint a new journey id")
        self.assertEqual(journey_first["slug"], journey_again["slug"])
        self._import(workspace_again)

        rows = self._journeys()
        self.assertEqual(1, len(rows), f"re-importing the same trip must not duplicate its journey row, got {[dict(r) for r in rows]}")


if __name__ == "__main__":
    unittest.main()
