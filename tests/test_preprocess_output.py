import contextlib
import io
import json
import tempfile
import unittest
from argparse import Namespace
from pathlib import Path
from subprocess import CompletedProcess
from unittest.mock import patch

from scripts.local_journey import preprocess


class PreprocessOutputTest(unittest.TestCase):
    def test_preprocess_uses_resolved_zone_and_preserves_author_override(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            gpx = root / "input.gpx"
            gpx.write_text("<gpx/>")
            workspace = root / "workspace"
            args = Namespace(workspace=workspace, journey="test", journal="journal", slug="trip", title="Trip", gpx=gpx, photos=root, sidecar=None)
            plan = {"stops": [], "mementos": [{"occurred_tz": "Asia/Tokyo", "media": []}]}
            with (
                patch("scripts.local_journey.resolve_identity"),
                patch("scripts.local_journey.ensure_cli"),
                patch("scripts.local_journey.subprocess.run", return_value=CompletedProcess([], 0, json.dumps(plan))),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                preprocess(args)
                path = workspace / "mementos.json"
                document = json.loads(path.read_text())
                self.assertEqual("Asia/Tokyo", document["mementos"][0]["occurred_tz"])
                document["mementos"][0]["occurred_tz"] = "UTC"
                path.write_text(json.dumps(document))
                preprocess(args)
                document = json.loads(path.read_text())
                self.assertEqual("UTC", document["mementos"][0]["occurred_tz"])
                self.assertIn("occurred_tz", document["mementos"][0]["authored_fields"])

    def test_empty_plan_explains_missing_candidates(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            gpx = root / "input.gpx"
            gpx.write_text("<gpx/>")
            args = Namespace(workspace=root / "workspace", journey="test", journal="journal", slug="trip", title="Trip", gpx=gpx, photos=root, sidecar=None)
            output = io.StringIO()
            with (
                patch("scripts.local_journey.resolve_identity"),
                patch("scripts.local_journey.ensure_cli"),
                patch("scripts.local_journey.subprocess.run", return_value=CompletedProcess([], 0, json.dumps({"stops": [], "mementos": []}))),
                contextlib.redirect_stdout(output),
            ):
                preprocess(args)
            self.assertIn("stops=0 mementos=0", output.getvalue())
            self.assertIn("20 minutes within 250 m", output.getvalue())
            self.assertIn("add stops or mementos manually", output.getvalue())
            self.assertNotIn("edit journey.json, stops.json", output.getvalue())
