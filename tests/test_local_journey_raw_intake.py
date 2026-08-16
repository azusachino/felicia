"""Guard the raw-input entry path: GPX + photo folder -> workspace -> package.

Every other local-journey fixture in this repository is a hand-written
workspace, so nothing exercised the producer itself. That let the planner's
output drift away from `schemas/local-authoring-v1.schema.json` field by field
until `make journey-local` could not be packaged at all, and let a key-casing
mismatch drop every photo without a word. This test runs the real compiled CLI
and asserts the producer's own output against the repository's own schema.
"""

import base64
import json
import tempfile
import unittest
from argparse import Namespace
from pathlib import Path

from scripts.local_journey import build_package, preprocess
from scripts.validate_local_authoring import validate_workspace

FIXTURE = Path(__file__).resolve().parent / "fixtures" / "local-journey-raw"
JOURNEY_ID = "0190cbde-f300-7000-8000-111111111111"
JOURNAL_ID = "0190cbde-f300-7000-8000-000000000000"

# A 1x1 JPEG. Packaging checks the media boundary by extension and MIME and
# hashes the bytes, so the fixture only has to be a real, tiny file.
PIXEL_JPEG = base64.b64decode(
    "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHR"
    "ofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QA"
    "FAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AKp//2Q=="
)

# Both timestamps sit inside a dwell in the fixture track.
SIDECAR = [
    {"path": "first-stop.jpg", "at": "2026-04-18T00:25:00Z", "title": "First stop"},
    {"path": "second-stop.jpg", "at": "2026-04-18T01:40:00Z", "title": "Second stop"},
]


class RawIntakeWorkflowTest(unittest.TestCase):
    def setUp(self):
        self._temporary = tempfile.TemporaryDirectory()
        root = Path(self._temporary.name)
        self.workspace = root / "workspace"
        self.photos = root / "photos"
        self.photos.mkdir(parents=True)
        for record in SIDECAR:
            (self.photos / record["path"]).write_bytes(PIXEL_JPEG)
        self.sidecar = root / "photos.jsonl"
        self.sidecar.write_text("\n".join(json.dumps(record) for record in SIDECAR) + "\n")
        self.addCleanup(self._temporary.cleanup)

    def _preprocess(self):
        preprocess(
            Namespace(
                workspace=self.workspace,
                journey=JOURNEY_ID,
                journal=JOURNAL_ID,
                slug="raw-intake",
                title="Raw intake",
                gpx=FIXTURE / "route.gpx",
                photos=self.photos,
                sidecar=self.sidecar,
            )
        )

    def test_producer_output_satisfies_the_authoring_schema(self):
        self._preprocess()
        # Validates journey/stops/mementos and, crucially, the plan the Go
        # planner just wrote -- the document that silently drifted.
        validate_workspace(self.workspace)

    def test_dwell_track_yields_stops_and_mementos(self):
        self._preprocess()
        stops = json.loads((self.workspace / "stops.json").read_text())["stops"]
        mementos = json.loads((self.workspace / "mementos.json").read_text())["mementos"]
        self.assertEqual(len(stops), 2, "the fixture track has two 20+ minute dwells")
        self.assertTrue(mementos, "each derived stop should propose a memento")
        for memento in mementos:
            self.assertTrue(memento["occurred_tz"], "a blank zone fails packaging")

    def test_journey_dates_come_from_the_track_not_from_today(self):
        self._preprocess()
        journey = json.loads((self.workspace / "journey.json").read_text())
        self.assertEqual(journey["date_start"], "2026-04-18")
        self.assertEqual(journey["date_end"], "2026-04-18")

    def test_sidecar_photos_reach_the_workspace_and_the_package(self):
        self._preprocess()
        mementos = json.loads((self.workspace / "mementos.json").read_text())["mementos"]
        attached = [asset for memento in mementos for asset in memento["media"]]
        self.assertTrue(attached, "sidecar-timed photos must not be dropped silently")
        for asset in attached:
            self.assertTrue(asset["path"], "media entries need a resolvable path")

        package = build_package(Namespace(workspace=self.workspace))
        import zipfile

        with zipfile.ZipFile(package) as archive:
            packaged = [name for name in archive.namelist() if name.startswith("media/")]
        self.assertTrue(packaged, "the package must carry the attached photos")


if __name__ == "__main__":
    unittest.main()
