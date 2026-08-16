"""The stops a CLI-imported trip carries must reach the admin intake inbox.

Issue #79: the importer learned to consume a `stops.yaml` package member
(ADR-0034), but the package never emitted one, so the intake inbox stayed
empty for every CLI-imported trip and the surface built for naming, merging,
and discarding stops could not be used on them. These tests cover the
producer half and prove the chain end to end, against the real compiled CLI
and the resulting SQLite database rather than a hand-written workspace.
"""

from __future__ import annotations

import json
import sqlite3
import sys
import tempfile
import unittest
import zipfile
from argparse import Namespace
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from local_journey import preprocess  # noqa: E402
from local_journey_common import CLI, ensure_cli, run  # noqa: E402
from local_journey_package import build_package, package_stops  # noqa: E402

JOURNAL_ID = "0190cbde-f300-7000-8000-000000000000"
PHOTO_AT = "2026-05-02T09:20:00Z"

# The checked-in two-dwell track the raw-intake test already proves the real
# planner derives stops from. Reused rather than synthesised: the planner's
# DefaultMinimumStopDwell is 20 minutes, and a hand-rolled track that misses it
# yields an empty plan and a vacuously passing test.
FIXTURE = ROOT / "tests" / "fixtures" / "local-journey-raw"


class PackageStopsProjectionTest(unittest.TestCase):
    """package_stops decides what travels; it must not leak review decisions."""

    def _stop(self, key: str, **overrides: object) -> dict:
        stop = {
            "candidate_key": key,
            "derivation_version": "gpx-stops-v1",
            "selected": True,
            "label": "",
            "coord": [139.7, 35.68],
            "arrive": "2026-05-02T09:00:00Z",
            "depart": "2026-05-02T09:30:00Z",
            "confidence": 0.5,
            "state": "kept",
            "merged_into": "someone-else",
        }
        stop.update(overrides)
        return stop

    def test_only_selected_stops_travel(self) -> None:
        data = {"stops": [self._stop("a"), self._stop("b")]}
        self.assertEqual(["a"], [stop["candidate_key"] for stop in package_stops(data, {"a"})])

    def test_review_owned_fields_are_never_emitted(self) -> None:
        emitted = package_stops({"stops": [self._stop("a", label="Named by the author")]}, {"a"})[0]
        for field in ("state", "merged_into", "selected", "authored_fields", "review_note"):
            self.assertNotIn(field, emitted, f"{field} is the author's decision and must not be seeded by an import")
        # The label is carried: it seeds the inbox, and the importer guards it
        # behind the stored authored mask rather than the package's word.
        self.assertEqual("Named by the author", emitted["label"])

    def test_derivation_version_is_carried_not_defaulted(self) -> None:
        emitted = package_stops({"stops": [self._stop("a", derivation_version="gpx-stops-v9")]}, {"a"})[0]
        self.assertEqual("gpx-stops-v9", emitted["derivation_version"])

    def test_an_unexpressable_kept_stop_fails_loudly(self) -> None:
        # A stop silently missing from the inbox is indistinguishable from one
        # the planner never found, so each of these must raise.
        for overrides, expected in (
            ({"derivation_version": ""}, "derivation_version"),
            ({"coord": None}, "coordinate"),
            ({"coord": [139.7]}, "coordinate"),
            ({"arrive": ""}, "arrive/depart"),
            ({"depart": ""}, "arrive/depart"),
        ):
            with self.subTest(overrides=overrides):
                with self.assertRaises(SystemExit) as caught:
                    package_stops({"stops": [self._stop("a", **overrides)]}, {"a"})
                self.assertIn(expected, str(caught.exception))

    def test_a_deselected_unexpressable_stop_is_not_an_error(self) -> None:
        data = {"stops": [self._stop("a", coord=None, selected=False)]}
        self.assertEqual([], package_stops(data, set()))


class StopCandidateEndToEndTest(unittest.TestCase):
    """preprocess -> package -> import -> the inbox the admin GUI queries."""

    @classmethod
    def setUpClass(cls) -> None:
        ensure_cli()

    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory(prefix="felicia-stops-")
        self.root = Path(self._tmp.name)
        self.addCleanup(self._tmp.cleanup)
        self.database = self.root / "felicia.sqlite"
        self.media_root = self.root / "media"

    def _import_a_trip(self) -> Path:
        photos = self.root / "photos"
        photos.mkdir(exist_ok=True)
        (photos / "shot.jpg").write_bytes(b"\xff\xd8\xff\xd9")
        sidecar = self.root / "sidecar.jsonl"
        sidecar.write_text(json.dumps({"path": "shot.jpg", "at": PHOTO_AT, "title": "A photo"}) + "\n")
        workspace = self.root / "workspace"
        preprocess(
            Namespace(
                workspace=workspace,
                journey=None,
                journal=JOURNAL_ID,
                slug=None,
                title=None,
                gpx=FIXTURE / "route.gpx",
                photos=photos,
                sidecar=sidecar,
            )
        )
        package = build_package(Namespace(workspace=workspace))
        run([str(CLI), "package", "validate", str(package)])
        run([str(CLI), "import", "--db", str(self.database), "--media-root", str(self.media_root), "--apply", str(package)])
        return workspace

    def _candidates(self) -> list[sqlite3.Row]:
        connection = sqlite3.connect(self.database)
        try:
            connection.row_factory = sqlite3.Row
            return connection.execute(
                "SELECT candidate_key, derivation_version, label, state, provenance, authored_fields "
                "FROM tb_stop_candidates ORDER BY candidate_key"
            ).fetchall()
        finally:
            connection.close()

    def test_the_package_carries_a_stops_member(self) -> None:
        workspace = self._import_a_trip()
        with zipfile.ZipFile(workspace / "journey.zip") as archive:
            self.assertIn("stops.yaml", archive.namelist())
            carried = json.loads(archive.read("stops.yaml"))
        self.assertTrue(carried, "a trip whose planner found stops must carry them")
        for stop in carried:
            self.assertTrue(stop["derivation_version"])

    def test_imported_stops_reach_the_intake_inbox(self) -> None:
        self._import_a_trip()
        rows = self._candidates()
        self.assertTrue(rows, "the admin intake inbox must not be empty for a CLI-imported trip")
        for row in rows:
            self.assertTrue(row["derivation_version"])
            # An import seeds the inbox; it never claims a review decision.
            self.assertEqual("proposed", row["state"])
            self.assertEqual([], json.loads(row["authored_fields"]))
            provenance = json.loads(row["provenance"])
            self.assertTrue(provenance, "an imported candidate must say where it came from")
            self.assertTrue(
                any(str(entry.get("source", {}).get("system", "")).startswith("package:") for entry in provenance),
                f"provenance must name the package as the source: {provenance}",
            )
            # ADR-0010: evidence must be dated, not left at a zero timestamp.
            self.assertTrue(all(entry.get("observed_at") for entry in provenance), provenance)

    def test_reimport_does_not_duplicate_or_resurrect_a_review_decision(self) -> None:
        workspace = self._import_a_trip()
        before = self._candidates()

        # The author names and discards a stop in the GUI.
        connection = sqlite3.connect(self.database)
        try:
            connection.execute(
                "UPDATE tb_stop_candidates SET label = ?, authored_fields = ?, state = ? WHERE candidate_key = ?",
                ("Named by hand", json.dumps(["label"]), "ignored", before[0]["candidate_key"]),
            )
            connection.commit()
        finally:
            connection.close()

        package = build_package(Namespace(workspace=workspace))
        run([str(CLI), "import", "--db", str(self.database), "--media-root", str(self.media_root), "--apply", str(package)])

        after = self._candidates()
        self.assertEqual(len(before), len(after), "re-import must not duplicate candidates")
        reviewed = next(row for row in after if row["candidate_key"] == before[0]["candidate_key"])
        self.assertEqual("Named by hand", reviewed["label"], "re-import must not overwrite an authored label")
        self.assertEqual("ignored", reviewed["state"], "re-import must not resurrect a discarded stop")


if __name__ == "__main__":
    unittest.main()
