import json
import tempfile
import unittest
import zipfile
from argparse import Namespace
from pathlib import Path

from scripts.local_journey import build_package
from scripts.validate_local_authoring import validate_workspace


class LocalJourneyWorkflowTest(unittest.TestCase):
    def test_preview_workspaces_validate_as_independent_journeys(self):
        root = Path(__file__).resolve().parents[1] / "examples" / "preview" / "local-journey"
        workspaces = [root, root / "kansai-ramble"]
        for workspace in workspaces:
            validate_workspace(workspace)
            self.assertTrue((workspace / "journey.json").is_file())
            self.assertTrue((workspace / "stops.json").is_file())
            self.assertTrue((workspace / "mementos.json").is_file())

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
