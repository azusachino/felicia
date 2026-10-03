import os
import plistlib
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts import go_tasks
from scripts.go_tasks import discover_modules


class DesktopDeploymentTest(unittest.TestCase):
    def test_bundle_requires_macos_26(self):
        info = go_tasks.ROOT / "apps/felicia-desktop/Info.plist"
        self.assertEqual(plistlib.loads(info.read_bytes())["LSMinimumSystemVersion"], "26.0")

    def test_desktop_compile_and_link_follow_bundle_floor(self):
        with patch.object(go_tasks.sys, "platform", "darwin"), patch.dict(
            os.environ, {"CGO_CFLAGS": "-O2 -g", "CGO_LDFLAGS": "-g"}
        ):
            environment = go_tasks.module_environment("apps/felicia-desktop")
        self.assertEqual(environment["MACOSX_DEPLOYMENT_TARGET"], "26.0")
        self.assertEqual(environment["CGO_CFLAGS"], "-O2 -g -mmacosx-version-min=26.0")
        self.assertEqual(environment["CGO_LDFLAGS"], "-g -mmacosx-version-min=26.0")

    def test_other_modules_and_platforms_keep_environment(self):
        for platform, module, goos in (
            ("darwin", "apps/felicia-cli", "darwin"),
            ("linux", "apps/felicia-desktop", "linux"),
            ("darwin", "apps/felicia-desktop", "windows"),
        ):
            environment = {"CGO_CFLAGS": "-O2", "GOOS": goos}
            with self.subTest(platform=platform, module=module, goos=goos), \
                 patch.object(go_tasks.sys, "platform", platform), \
                 patch.dict(os.environ, environment, clear=True):
                self.assertEqual(go_tasks.module_environment(module), environment)


class DiscoverModulesTest(unittest.TestCase):
    def test_parses_use_block(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory) / "go.work"
            work.write_text("go 1.27\n\nuse (\n\t./apps/felicia-core\n\t./apps/felicia-server\n)\n")
            self.assertEqual(discover_modules(work), ["apps/felicia-core", "apps/felicia-server"])

    def test_parses_single_line_use(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory) / "go.work"
            work.write_text("go 1.27\n\nuse ./apps/felicia-core\n")
            self.assertEqual(discover_modules(work), ["apps/felicia-core"])

    def test_workspace_modules_have_manifests(self):
        root = Path(__file__).resolve().parents[1]
        for module in discover_modules(root / "go.work"):
            self.assertTrue((root / module / "go.mod").is_file(), module)
