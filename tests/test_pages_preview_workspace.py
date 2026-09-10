"""The Pages preview must never delete an author's original media.

Regression cover for the destructive-reset defect: `publish()` emptied the
shared authoring media root, and did so whenever `PAGES_DB` was unset, which
took an explicitly configured `PAGES_MEDIA_ROOT` with it. Every test here
redirects each root into a temporary directory and stops the build at its first
external command, so nothing real is touched and no toolchain is needed.
"""

import re
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from scripts import felicia


class PagesPreviewWorkspaceTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        root = Path(self.directory.name)
        self.authoring_media = root / "authoring-media"
        self.preview_media = root / "preview-media"
        self.authoring_media.mkdir()
        self.sentinel = self.authoring_media / "original.jpg"
        self.sentinel.write_bytes(b"the only copy")
        # Redirect every path publish() would otherwise resolve under the repo,
        # and stop at the first external command so no build or import runs.
        self.patches = [
            mock.patch.object(felicia, "DEFAULT_MEDIA_ROOT", self.preview_media),
            mock.patch.object(felicia, "DEFAULT_DB", root / "preview.sqlite"),
            mock.patch.object(felicia, "DEFAULT_DIST", root / "dist"),
            mock.patch.object(felicia, "CLI", root / "bin" / "felicia-cli"),
            mock.patch.object(felicia, "WEB", root / "web"),
            mock.patch.object(felicia, "run", side_effect=StopBuild),
        ]
        for patch in self.patches:
            patch.start()
            self.addCleanup(patch.stop)

    def publish(self, **environment):
        """Run publish() with a clean environment, stopping at the first command."""
        with mock.patch.dict("os.environ", environment, clear=False):
            for name in ("PAGES_DB", "PAGES_MEDIA_ROOT"):
                if name not in environment:
                    felicia.os.environ.pop(name, None)
            with self.assertRaises(StopBuild):
                felicia.publish("/")

    def test_default_run_leaves_the_authoring_media_root_untouched(self):
        """AC1: the shared authoring root is not the preview's to reset."""
        self.publish()
        self.assertEqual(self.sentinel.read_bytes(), b"the only copy")

    def test_configured_media_root_is_never_emptied(self):
        """AC2: an explicit PAGES_MEDIA_ROOT is used as given, not wiped."""
        self.publish(PAGES_MEDIA_ROOT=str(self.authoring_media))
        self.assertEqual(self.sentinel.read_bytes(), b"the only copy")

    def test_configured_media_root_survives_an_unset_pages_db(self):
        """AC2: the media reset no longer keys off the database variable."""
        self.publish(PAGES_MEDIA_ROOT=str(self.authoring_media))
        self.assertTrue(self.sentinel.is_file())

    def test_owned_workspace_is_reset_between_runs(self):
        """AC3: the disposable workspace really is emptied, or previews go stale."""
        self.preview_media.mkdir()
        (self.preview_media / felicia.WORKSPACE_MARKER).touch()
        stale = self.preview_media / "stale.webp"
        stale.write_bytes(b"from a previous run")
        self.publish()
        self.assertFalse(stale.exists())
        self.assertTrue((self.preview_media / felicia.WORKSPACE_MARKER).is_file())

    def test_unmarked_workspace_aborts_without_deleting(self):
        """AC4: no ownership proof means refuse, and remove nothing."""
        self.preview_media.mkdir()
        occupied = self.preview_media / "unexplained.jpg"
        occupied.write_bytes(b"provenance unknown")
        with self.assertRaises(SystemExit) as raised:
            felicia.reset_disposable(self.preview_media)
        self.assertIn(felicia.WORKSPACE_MARKER, str(raised.exception))
        self.assertEqual(occupied.read_bytes(), b"provenance unknown")

    def test_missing_workspace_is_adopted_and_marked(self):
        """A fresh checkout has no workspace yet; creating one is not a refusal."""
        felicia.reset_disposable(self.preview_media)
        self.assertTrue((self.preview_media / felicia.WORKSPACE_MARKER).is_file())

    def test_failed_build_leaves_originals_intact(self):
        """AC5: the reset happens before the build, so a failure cannot undo it."""
        self.publish()
        self.assertEqual(
            sorted(path.name for path in self.authoring_media.iterdir()),
            ["original.jpg"],
        )


class PreviewDefaultsTest(unittest.TestCase):
    """The shipped defaults, unpatched.

    The tests above redirect `DEFAULT_MEDIA_ROOT` into a temporary directory,
    which is what makes them safe to run -- and also means they cannot see the
    original defect, which was the value of the constant itself. These assert on
    the real one.
    """

    def test_preview_media_root_is_not_the_authoring_media_root(self):
        """AC1: the preview's default is a workspace of its own."""
        self.assertNotEqual(
            felicia.DEFAULT_MEDIA_ROOT,
            felicia.ROOT / ".felicia" / "media",
        )

    def test_preview_media_root_differs_from_the_server_default(self):
        """Drift guard: the server owns the authoring root, and it may move.

        Reads the authoring default out of the Go config rather than restating
        it, so repointing either side at the other fails here instead of during
        a preview.
        """
        config = (
            felicia.ROOT / "apps" / "felicia-server" / "config" / "config.go"
        ).read_text(encoding="utf-8")
        declared = re.search(r'defaultMediaRoot\s*=\s*"([^"]+)"', config)
        self.assertIsNotNone(declared, "defaultMediaRoot not found in config.go")
        authoring_root = (felicia.ROOT / declared.group(1)).resolve()
        self.assertNotEqual(felicia.DEFAULT_MEDIA_ROOT.resolve(), authoring_root)

    def test_preview_database_is_not_the_authoring_database(self):
        """The database already had a dedicated name; keep it that way."""
        self.assertNotEqual(
            felicia.DEFAULT_DB,
            felicia.ROOT / ".felicia" / "felicia.sqlite",
        )


class StopBuild(Exception):
    """Raised in place of the first external command, ending the run early."""


if __name__ == "__main__":
    unittest.main()
