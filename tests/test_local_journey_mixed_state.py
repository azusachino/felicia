"""End-to-end publish-invariant tests over a deliberately messy journey seed.

This drives the real pipeline through the built `felicia-cli` binary — package
validate, import --apply (SQLite), static compile — over a fixture that mixes
candidate/draft/authored/published/archived mementos, a memento missing
optional metadata, a fully authored published memento, an archived memento
with a broken media reference, and a second journey with zero published
mementos. It complements (and must not duplicate) the in-memory unit tests in
apps/felicia-publication/public_test.go, which exercise the same publish gate against fake
in-memory fixtures rather than the compiled CLI and a real SQLite database.
"""

from __future__ import annotations

import json
import shutil
import tempfile
import unittest
from argparse import Namespace
from pathlib import Path
from urllib.parse import urlparse

from scripts.local_journey_common import CLI, ensure_cli, run
from scripts.local_journey_package import build_package
from scripts.validate_local_authoring import validate_workspace, validate_workspace_root


FIXTURE_ROOT = Path(__file__).resolve().parent / "fixtures" / "local-journey-mixed-state"
EMPTY_JOURNEY_DIR = FIXTURE_ROOT / "empty-journey"

PUBLISHED_JOURNEY_ID = "0190cbde-f300-7000-9000-a00000000001"
ZERO_PUBLISHED_JOURNEY_ID = "0190cbde-f300-7000-9000-a00000000002"

ALLOWED_MEDIA_EXTENSIONS = {".jpg", ".jpeg", ".png", ".webp"}

# Titles and essay bodies belonging to non-published mementos in the fixture.
# None of these strings may appear anywhere in the compiled public artifact.
NON_PUBLIC_STRINGS = (
    "Unreviewed candidate find",
    "Draft receipt awaiting essay",
    "Authored stamp pending publish",
    "An essay drafted but held back from publish for later review.",
    "Archived receipt with a missing photo",
    "Harbor draft that never got published",
)


class LocalJourneyMixedStateFixtureTest(unittest.TestCase):
    """The fixture itself must stay a valid local-authoring workspace."""

    def test_fixture_validates_as_independent_journeys(self) -> None:
        validate_workspace_root(FIXTURE_ROOT)
        for workspace in (FIXTURE_ROOT, EMPTY_JOURNEY_DIR):
            validate_workspace(workspace)


class LocalJourneyMixedStateWorkflowTest(unittest.TestCase):
    """Runs the real package -> SQLite -> static compile pipeline."""

    @classmethod
    def setUpClass(cls) -> None:
        ensure_cli()

    def test_mixed_state_seed_compiles_to_published_only_artifact(self) -> None:
        with tempfile.TemporaryDirectory(prefix="felicia-mixed-state-") as directory:
            root = Path(directory)
            database = root / "felicia.sqlite"
            media_root = root / "media"
            site = root / "site"

            for source in (FIXTURE_ROOT, EMPTY_JOURNEY_DIR):
                package = self._build_package_from_fixture(source, root / "workspace" / source.name)
                run([str(CLI), "package", "validate", str(package)])
                run(
                    [
                        str(CLI),
                        "import",
                        "--db",
                        str(database),
                        "--media-root",
                        str(media_root),
                        "--apply",
                        str(package),
                    ]
                )

            run(
                [
                    str(CLI),
                    "static",
                    "compile",
                    "--db",
                    str(database),
                    "--media-root",
                    str(media_root),
                    "--out",
                    str(site),
                ]
            )

            self._assert_published_only_index(site)
            self._assert_zero_published_journey_absent(site)
            self._assert_published_content(site)
            self._assert_media_references_are_safe_relative_paths(site)
            self._assert_no_unpublished_content_leaked(site)

    def _build_package_from_fixture(self, source: Path, workspace: Path) -> Path:
        # Copy into a scratch workspace rather than packaging in place: build_package
        # writes journey.zip alongside the authoring files, and the checked-in
        # fixture must stay pristine and safe for concurrent test runs.
        workspace.mkdir(parents=True, exist_ok=True)
        for name in ("journey.json", "stops.json", "mementos.json", "route.gpx"):
            shutil.copy2(source / name, workspace / name)
        return build_package(Namespace(workspace=workspace))

    def _read_json(self, path: Path):
        return json.loads(path.read_text(encoding="utf-8"))

    def _assert_published_only_index(self, site: Path) -> None:
        journeys = self._read_json(site / "api" / "v1" / "journeys.json")
        ids = {journey["id"] for journey in journeys}
        self.assertEqual({PUBLISHED_JOURNEY_ID}, ids)

    def _assert_zero_published_journey_absent(self, site: Path) -> None:
        self.assertFalse((site / "api" / "v1" / "journeys" / f"{ZERO_PUBLISHED_JOURNEY_ID}.json").exists())
        self.assertFalse((site / "api" / "v1" / "journeys" / ZERO_PUBLISHED_JOURNEY_ID / "mementos.json").exists())

    def _assert_published_content(self, site: Path) -> None:
        mementos = self._read_json(site / "api" / "v1" / "journeys" / PUBLISHED_JOURNEY_ID / "mementos.json")
        by_title = {memento["title"]: memento for memento in mementos}
        self.assertEqual({"Published minimal ticket", "Published souvenir with full story"}, set(by_title))

        minimal = by_title["Published minimal ticket"]
        self.assertEqual("LineString", minimal["geom"]["type"])
        self.assertEqual("Asia/Tokyo", minimal["occurred_tz"])
        self.assertNotIn("photos", minimal, "minimal published memento has no photos")

        full = by_title["Published souvenir with full story"]
        self.assertEqual("Temple Gift Shop", full["vendor"])
        self.assertEqual(
            "The story of the souvenir bought at the quiet temple, fully authored for publication.",
            full["essay"],
        )
        self.assertEqual(1500, full["price_amount"])
        self.assertEqual("JPY", full["price_currency"])
        self.assertEqual(1, len(full.get("photos", [])))

    def _assert_media_references_are_safe_relative_paths(self, site: Path) -> None:
        mementos = self._read_json(site / "api" / "v1" / "journeys" / PUBLISHED_JOURNEY_ID / "mementos.json")
        object_keys = [photo["object_key"] for memento in mementos for photo in memento.get("photos", [])]
        self.assertEqual(1, len(object_keys), "only the fully authored published memento carries a photo")

        # The draft/authored/candidate mementos in both journeys reference their
        # own media, but none of it is published; the compiled artifact must
        # contain exactly the media the public API references, nothing more.
        written_media = sorted(path.relative_to(site).as_posix() for path in (site / "media").rglob("*") if path.is_file())
        self.assertEqual(sorted(object_keys), written_media, "unpublished media leaked into the compiled artifact")

        for object_key in object_keys:
            self.assertFalse(object_key.startswith("/"), f"absolute path leaked: {object_key}")
            self.assertNotIn("file://", object_key)
            self.assertEqual("", urlparse(object_key).scheme, f"object key is not a bare relative path: {object_key}")
            self.assertNotIn("..", Path(object_key).parts, f"object key escapes the artifact root: {object_key}")
            resolved = (site / object_key).resolve()
            self.assertTrue(
                str(resolved).startswith(str(site.resolve()) + "/"),
                f"media file resolves outside the artifact root: {object_key}",
            )
            self.assertIn(
                Path(object_key).suffix.lower(),
                ALLOWED_MEDIA_EXTENSIONS,
                f"media file fails the public JPEG/PNG/WebP boundary: {object_key}",
            )
            self.assertTrue(resolved.is_file(), f"referenced media file was not written: {object_key}")

    def _assert_no_unpublished_content_leaked(self, site: Path) -> None:
        haystack = "\n".join(path.read_text(encoding="utf-8") for path in site.rglob("*.json"))
        for forbidden in NON_PUBLIC_STRINGS:
            self.assertNotIn(forbidden, haystack, f"non-published content leaked into the public artifact: {forbidden!r}")


if __name__ == "__main__":
    unittest.main()
