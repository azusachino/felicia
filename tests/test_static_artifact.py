import contextlib
import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts.verify_static_artifact import main


class StaticArtifactTest(unittest.TestCase):
    def verify(self, base, asset):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "index.html").write_text(f'<script type="module" src="{asset}"></script>')
            api = root / "api/v1/journeys/j1"
            api.mkdir(parents=True)
            (api.parent.parent / "journeys.json").write_text(json.dumps([{"id": "j1"}]))
            (api.parent / "j1.json").write_text(json.dumps({"id": "j1"}))
            (api / "mementos.json").write_text("[]")
            with patch.dict(os.environ, {"SITE_DIST": directory, "BASE_PATH": base}), contextlib.redirect_stdout(io.StringIO()):
                main()

    def test_custom_output_and_base(self):
        self.verify("/travels", "/travels/assets/app.js")

    def test_root_base(self):
        self.verify("/", "/assets/app.js")

    def test_root_rejects_nested_base(self):
        with self.assertRaisesRegex(AssertionError, "does not match base path"):
            self.verify("/", "/travels/assets/app.js")

    def test_nested_base_rejects_root_assets(self):
        with self.assertRaisesRegex(AssertionError, "does not match base path"):
            self.verify("/travels/", "/assets/app.js")
