import json
import tempfile
import unittest
import zipfile
from argparse import Namespace
from pathlib import Path

from scripts.local_journey import build_package
from scripts.validate_local_authoring import validate_document, validate_workspace, validate_workspace_root


class LocalJourneyWorkflowTest(unittest.TestCase):
    def test_preview_workspaces_validate_as_independent_journeys(self):
        root = Path(__file__).resolve().parents[1] / "examples" / "preview" / "local-journey"
        validate_workspace_root(root)
        workspaces = [root, root / "kansai-ramble"]
        for workspace in workspaces:
            validate_workspace(workspace)
            self.assertTrue((workspace / "journey.json").is_file())
            self.assertTrue((workspace / "stops.json").is_file())
            self.assertTrue((workspace / "mementos.json").is_file())

    def test_plan_schema_types_all_evidence_arrays(self):
        with tempfile.TemporaryDirectory() as directory:
            plan_path = Path(directory) / "plan.json"
            plan_path.write_text(
                json.dumps(
                    {
                        "journey_id": "0190cbde-f300-7000-8000-111111111111",
                        "schema": "felicia.intake.plan",
                        "version": "1",
                        "routes": [
                            {
                                "Line": [[135.5, 34.7], [135.6, 34.8]],
                                "Points": [{"Coord": [135.5, 34.7], "At": "2026-04-01T09:00:00Z"}],
                                "From": "2026-04-01T09:00:00Z",
                                "To": "2026-04-01T10:00:00Z",
                                "DistanceM": 100,
                                "Mode": "walking",
                                "SourceRef": "gpx:segment-1",
                                "Provenance": {
                                    "source": {"system": "gpx", "external_id": "segment-1"},
                                    "observed_at": "2026-04-01T09:00:00Z",
                                    "confidence": 1,
                                },
                            }
                        ],
                        "visits": [
                            {
                                "Coord": [135.5, 34.7],
                                "Label": "Osaka",
                                "Arrive": "2026-04-01T09:00:00Z",
                                "Depart": "2026-04-01T10:00:00Z",
                                "Confidence": 0.9,
                                "SourceRef": "dawarich:visit-1",
                                "Provenance": {
                                    "source": {"system": "dawarich", "external_id": "visit-1"},
                                    "observed_at": "2026-04-01T09:00:00Z",
                                    "confidence": 0.9,
                                },
                            }
                        ],
                        "stops": [
                            {
                                "ID": "0190cbde-f300-7000-8000-222222222222",
                                "JourneyID": "0190cbde-f300-7000-8000-111111111111",
                                "Identity": {"derivation_version": "gpx-stops-v1", "key": "visit-1"},
                                "Label": "Osaka",
                                "Coord": [135.5, 34.7],
                                "Arrive": "2026-04-01T09:00:00Z",
                                "Depart": "2026-04-01T10:00:00Z",
                                "Confidence": 0.9,
                                "Evidence": [
                                    {
                                        "kind": "visit",
                                        "source": {"system": "dawarich", "external_id": "visit-1"},
                                        "locator": "visit-1",
                                    }
                                ],
                                "State": "proposed",
                                "Revision": 0,
                                "CreatedAt": "2026-04-01T09:00:00Z",
                                "UpdatedAt": "2026-04-01T09:00:00Z",
                            }
                        ],
                        "mementos": [
                            {
                                "source": {"system": "immich", "external_id": "asset-1"},
                                "stop_key": "visit-1",
                                "kind": "goods",
                                "occurred_at": "2026-04-01T09:30:00Z",
                                "occurred_tz": "Asia/Tokyo",
                                "geom": [135.5, 34.7],
                                "title": "Souvenir",
                                "place": "Osaka",
                                "kind_data": {},
                                "media": [],
                                "memory_links": [],
                                "provenance": {
                                    "source": {"system": "immich", "external_id": "asset-1"},
                                    "observed_at": "2026-04-01T09:30:00Z",
                                    "confidence": 0.8,
                                },
                            }
                        ],
                        "issues": [{"severity": "warning", "code": "review", "message": "check stop"}],
                    }
                ),
                encoding="utf-8",
            )

            validate_document(plan_path)
            document = json.loads(plan_path.read_text(encoding="utf-8"))
            document["routes"][0]["DistanceM"] = "100"
            plan_path.write_text(json.dumps(document), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "routes.0.DistanceM"):
                validate_document(plan_path)

    def test_workspace_manifest_rejects_duplicate_journey_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "workspace.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.local.workspace.v1",
                        "version": "1",
                        "journal_id": "0190cbde-f300-7000-8000-000000000000",
                        "journeys": [
                            {"path": ".", "id": "0190cbde-f300-7000-8000-111111111111"},
                            {"path": ".", "id": "0190cbde-f300-7000-8000-222222222222"},
                        ],
                    }
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValueError, "path is duplicated"):
                validate_workspace_root(root)

    def test_package_keeps_only_selected_stops_and_copies_media(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.local.journey.v1",
                        "id": "0190cbde-f300-7000-8000-111111111111",
                        "journal_id": "0190cbde-f300-7000-8000-000000000000",
                        "slug": "osaka-five-days",
                        "title": "Osaka five days",
                        "place": "Osaka · Kobe · Nara",
                        "date_start": "2026-04-01",
                        "date_end": "2026-04-05",
                    }
                )
            )
            (workspace / "stops.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.local.stops.v1",
                        "stops": [
                            {"candidate_key": "osaka", "selected": True, "label": "Osaka"},
                            {"candidate_key": "noise", "selected": False, "label": "Noise"},
                        ]
                    }
                )
            )
            (workspace / "mementos.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.local.mementos.v1",
                        "mementos": [
                            {
                                "id": "0190cbde-f300-7000-8000-a00000000001",
                                "stop_key": "osaka",
                                "seq": 1,
                                "kind": "receipt",
                                "occurred_at": "2026-04-01T09:00:00Z",
                                "occurred_tz": "UTC",
                                "title": "Dotonbori receipt",
                                "place": "Osaka",
                                "geom": [135.5, 34.7],
                                "state": "published",
                                "vendor": "Dotonbori Kitchen",
                                "essay": "A receipt from the first night.",
                                "price_amount": 1200,
                                "price_currency": "JPY",
                                "authored_fields": ["title", "vendor", "essay", "price_amount", "price_currency"],
                                "kind_data": {"vendor": "sample"},
                                "media": [{"path": "ticket.jpg", "caption": "ticket"}],
                            },
                            {
                                "id": "0190cbde-f300-7000-8000-a00000000002",
                                "stop_key": "noise",
                                "seq": 2,
                                "kind": "goods",
                                "occurred_at": "2026-04-01T10:00:00Z",
                                "occurred_tz": "UTC",
                                "title": "Noise",
                                "state": "draft",
                                "kind_data": {},
                                "media": [],
                            },
                        ]
                    }
                )
            )
            (workspace / "route.gpx").write_text("<gpx />")
            (workspace / "ticket.jpg").write_bytes(b"ticket")

            output = build_package(Namespace(workspace=workspace))

            with zipfile.ZipFile(output) as archive:
                mementos = json.loads(archive.read("mementos.yaml"))
                self.assertEqual(["Dotonbori receipt"], [m["title"] for m in mementos])
                self.assertEqual("A receipt from the first night.", mementos[0]["essay"])
                self.assertEqual(["title", "vendor", "essay", "price_amount", "price_currency"], mementos[0]["authored_fields"])
                self.assertEqual(b"ticket", archive.read("media/ticket.jpg"))
                self.assertIn(b"sha256:", archive.read("manifest.yaml"))

    def test_package_rejects_private_and_unsupported_media(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(json.dumps({"schema": "felicia.local.journey.v1", "id": "0190cbde-f300-7000-8000-111111111111", "journal_id": "0190cbde-f300-7000-8000-000000000000", "slug": "test", "title": "Test", "date_start": "2026-04-01", "date_end": "2026-04-01"}))
            (workspace / "stops.json").write_text(json.dumps({"schema": "felicia.local.stops.v1", "stops": [{"candidate_key": "stop", "selected": True, "label": "Stop"}]}))
            (workspace / "mementos.json").write_text(json.dumps({"schema": "felicia.local.mementos.v1", "mementos": [{"id": "0190cbde-f300-7000-8000-a00000000001", "stop_key": "stop", "seq": 1, "kind": "goods", "occurred_at": "2026-04-01T09:00:00Z", "state": "draft", "title": "Video", "kind_data": {}, "media": [{"path": "secret.mp4", "kind": "video"}]}]}))
            (workspace / "route.gpx").write_text("<gpx />")
            (workspace / "secret.mp4").write_bytes(b"video")

            with self.assertRaisesRegex(SystemExit, "unsupported public media kind"):
                build_package(Namespace(workspace=workspace))


if __name__ == "__main__":
    unittest.main()
