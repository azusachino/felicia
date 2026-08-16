"""A CLI-imported `transit` memento must be possible at all (issue #77).

`core/kinds/transit.yaml` declares `anchor: edge`, which the write boundary
enforces as a two-or-more-point line, but the journey package could only express
a single coordinate per memento. So the importer accepted a `transit` memento,
the compiler published it, and every later save from the admin GUI answered
`anchor_mismatch` — the memento was unauthorable for the rest of its life.

These tests cover the producer half of the fix against the real compiled CLI and
a real SQLite database: the workspace and the package can carry edge geometry, it
survives the whole chain as a LineString, and a `transit` memento offered as a
single point is refused at import instead of silently corrupted.
"""

from __future__ import annotations

import json
import sqlite3
import subprocess
import sys
import tempfile
import unittest
import zipfile
from argparse import Namespace
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from local_journey_common import CLI, ensure_cli, run  # noqa: E402
from local_journey_package import build_package  # noqa: E402
from validate_local_authoring import validate_workspace  # noqa: E402

JOURNAL_ID = "0190cbde-f300-7000-8000-000000000000"

KYOTO = [135.7681, 35.0116]
OSAKA = [135.5023, 34.7025]

ROUTE_GPX = (
    '<?xml version="1.0"?>\n'
    '<gpx version="1.1" creator="felicia test" xmlns="http://www.topografix.com/GPX/1/1">\n'
    "  <trk><name>kyoto to osaka</name><trkseg>\n"
    f'    <trkpt lat="{KYOTO[1]}" lon="{KYOTO[0]}"><time>2026-04-01T09:00:00Z</time></trkpt>\n'
    f'    <trkpt lat="{OSAKA[1]}" lon="{OSAKA[0]}"><time>2026-04-01T09:40:00Z</time></trkpt>\n'
    "  </trkseg></trk>\n"
    "</gpx>\n"
)


class TransitMementoWorkflowTest(unittest.TestCase):
    """workspace -> package -> real CLI import -> the stored geometry."""

    @classmethod
    def setUpClass(cls) -> None:
        ensure_cli()

    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory(prefix="felicia-transit-")
        self.addCleanup(self._tmp.cleanup)
        self.root = Path(self._tmp.name)
        self.database = self.root / "felicia.sqlite"
        self.media_root = self.root / "media"

    def _workspace(self, geom: object, name: str = "trip", suffix: str = "1", state: str = "published") -> Path:
        """Write one hand-authored trip whose only memento is a transit ticket.

        Hand-authored rather than planned: the planner has no notion of a transit
        leg, so this is the shape an author edits into mementos.json.
        """
        workspace = self.root / f"workspace-{name}"
        workspace.mkdir(parents=True, exist_ok=True)
        (workspace / "journey.json").write_text(
            json.dumps(
                {
                    "schema": "felicia.local.journey.v1",
                    "id": f"0190cbde-f300-7000-8000-c0000000000{suffix}",
                    "journal_id": JOURNAL_ID,
                    "slug": f"kyoto-osaka-{name}",
                    "title": "Kyoto to Osaka",
                    "date_start": "2026-04-01",
                    "date_end": "2026-04-01",
                }
            )
        )
        (workspace / "stops.json").write_text(
            json.dumps(
                {
                    "schema": "felicia.local.stops.v1",
                    "stops": [
                        {
                            "candidate_key": f"kyoto-{name}",
                            "derivation_version": "gpx-stops-v1",
                            "selected": True,
                            "label": "Kyoto",
                            "coord": KYOTO,
                            "arrive": "2026-04-01T09:00:00Z",
                            "depart": "2026-04-01T09:30:00Z",
                            "confidence": 0.9,
                        }
                    ],
                }
            )
        )
        (workspace / "mementos.json").write_text(
            json.dumps(
                {
                    "schema": "felicia.local.mementos.v1",
                    "mementos": [
                        {
                            "id": f"0190cbde-f300-7000-8000-d0000000000{suffix}",
                            "stop_key": f"kyoto-{name}",
                            "seq": 1,
                            "kind": "transit",
                            "occurred_at": "2026-04-01T09:15:00Z",
                            "occurred_tz": "Asia/Tokyo",
                            "state": state,
                            "title": "Kyoto to Osaka",
                            "place": "Kyoto",
                            "geom": geom,
                            # An edge anchor needs two resolved coord-bearing
                            # fields as well as a two-point geometry, so a
                            # station is an object with coords, not a bare name.
                            "kind_data": {
                                "operator": "JR West",
                                "line": "Kyoto Line",
                                "from": {"name": "Kyoto", "coords": KYOTO},
                                "to": {"name": "Osaka", "coords": OSAKA},
                            },
                            "media": [],
                        }
                    ],
                }
            )
        )
        (workspace / "route.gpx").write_text(ROUTE_GPX)
        return workspace

    def _import(self, package: Path) -> subprocess.CompletedProcess:
        return subprocess.run(
            [
                str(CLI),
                "import",
                "--db",
                str(self.database),
                "--media-root",
                str(self.media_root),
                "--apply",
                str(package),
            ],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )

    def _count(self, table: str) -> int:
        if not self.database.exists():
            return 0
        connection = sqlite3.connect(self.database)
        try:
            return int(connection.execute(f"SELECT count(*) FROM {table}").fetchone()[0])
        finally:
            connection.close()

    def _stored_geometry(self) -> dict:
        connection = sqlite3.connect(self.database)
        try:
            row = connection.execute("SELECT geom FROM tb_mementos").fetchone()
        finally:
            connection.close()
        return json.loads(row[0])

    def test_an_edge_anchored_memento_travels_and_is_stored_as_a_line(self) -> None:
        workspace = self._workspace([KYOTO, OSAKA])
        validate_workspace(workspace)
        package = build_package(Namespace(workspace=workspace))

        with zipfile.ZipFile(package) as archive:
            carried = json.loads(archive.read("mementos.yaml"))
        self.assertEqual([KYOTO, OSAKA], carried[0]["geom"], "the package must carry the whole line, not a point")

        run([str(CLI), "package", "validate", str(package)])
        result = self._import(package)
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

        self.assertEqual(1, self._count("tb_mementos"))
        stored = self._stored_geometry()
        self.assertEqual("LineString", stored["type"])
        self.assertEqual([KYOTO, OSAKA], stored["coordinates"])

    def test_a_single_point_transit_memento_is_refused_and_writes_nothing(self) -> None:
        # A point on an edge-anchored kind is the corruption issue #77 records:
        # accepted by the importer, published by the compiler, then rejected by
        # every GUI save. Import a good trip first so the assertions below are
        # about a live database, not a missing file.
        good = self._workspace([KYOTO, OSAKA], name="good", suffix="1")
        accepted = self._import(build_package(Namespace(workspace=good)))
        self.assertEqual(0, accepted.returncode, accepted.stdout + accepted.stderr)
        mementos, stops = self._count("tb_mementos"), self._count("tb_stop_candidates")

        bad = self._workspace(KYOTO, name="bad", suffix="2")
        validate_workspace(bad)
        result = self._import(build_package(Namespace(workspace=bad)))

        self.assertNotEqual(0, result.returncode, "a point-anchored transit memento must not import")
        message = result.stdout + result.stderr
        self.assertIn("anchor_mismatch", message, message)
        self.assertIn("transit", message, message)
        self.assertEqual(mementos, self._count("tb_mementos"), "a rejected package must persist no memento")
        # ApplyPackage writes stop candidates before it reaches the memento loop
        # and runs in no transaction (issue #76), which is why the rejection has
        # to happen at decode time.
        self.assertEqual(stops, self._count("tb_stop_candidates"), "a rejected package must persist no stop either")


if __name__ == "__main__":
    unittest.main()
