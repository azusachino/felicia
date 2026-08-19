import tempfile
import unittest
from pathlib import Path

from scripts.go_tasks import discover_modules


class DiscoverModulesTest(unittest.TestCase):
    def test_parses_use_block(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory) / "go.work"
            work.write_text("go 1.26\n\nuse (\n\t./apps/felicia-core\n\t./apps/felicia-server\n)\n")
            self.assertEqual(discover_modules(work), ["apps/felicia-core", "apps/felicia-server"])

    def test_parses_single_line_use(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory) / "go.work"
            work.write_text("go 1.26\n\nuse ./apps/felicia-core\n")
            self.assertEqual(discover_modules(work), ["apps/felicia-core"])

    def test_workspace_modules_have_manifests(self):
        root = Path(__file__).resolve().parents[1]
        for module in discover_modules(root / "go.work"):
            self.assertTrue((root / module / "go.mod").is_file(), module)
