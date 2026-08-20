import hashlib
import json
import tempfile
import unittest
import zipfile
from argparse import Namespace
from pathlib import Path

from scripts.local_journey import build_package, guard_workspace_identity, resolve_identity
from scripts.local_journey_common import DEFAULT_WORKSPACE_ROOT, derive_journey_identity
from scripts.validate_local_authoring import validate_document, validate_workspace, validate_workspace_root


class LocalJourneyWorkflowTest(unittest.TestCase):
    def test_izu_publication_workspace_validates(self):
        root = Path(__file__).resolve().parents[1] / "publication" / "journeys"
        validate_workspace_root(root)
        journey = root / "izu-trip-2026-08-01"
        validate_workspace(journey)
        self.assertEqual("izu-trip-2026-08-01", json.loads((journey / "journey.json").read_text())["slug"])
        self.assertTrue((journey / "stops.json").is_file())
        self.assertTrue((journey / "mementos.json").is_file())

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
                                "id": "0190cbde-f300-7000-8000-222222222222",
                                "journey_id": "0190cbde-f300-7000-8000-111111111111",
                                "identity": {"derivation_version": "gpx-stops-v1", "key": "visit-1"},
                                "label": "Osaka",
                                "coord": [135.5, 34.7],
                                "arrive": "2026-04-01T09:00:00Z",
                                "depart": "2026-04-01T10:00:00Z",
                                "confidence": 0.9,
                                "evidence": [
                                    {
                                        "kind": "visit",
                                        "source": {"system": "dawarich", "external_id": "visit-1"},
                                        "locator": "visit-1",
                                    }
                                ],
                                "state": "proposed",
                                "revision": 0,
                                "created_at": "2026-04-01T09:00:00Z",
                                "updated_at": "2026-04-01T09:00:00Z",
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
                        "schema": "felicia.workspace.v1",
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
                        "schema": "felicia.journey.v1",
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
                        "schema": "felicia.stops.v1",
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
                        "schema": "felicia.mementos.v1",
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
                # The object key is content-addressed (issue #75), not the
                # photo's basename, so it must match the file's own hash.
                photo_path = mementos[0]["photos"][0]["path"]
                expected_digest = hashlib.sha256(b"ticket").hexdigest()
                self.assertEqual(f"media/{expected_digest}.jpg", photo_path)
                self.assertEqual(b"ticket", archive.read(photo_path))
                self.assertIn(b"sha256:", archive.read("manifest.yaml"))

    def test_package_rejects_private_and_unsupported_media(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(json.dumps({"schema": "felicia.journey.v1", "id": "0190cbde-f300-7000-8000-111111111111", "journal_id": "0190cbde-f300-7000-8000-000000000000", "slug": "test", "title": "Test", "date_start": "2026-04-01", "date_end": "2026-04-01"}))
            (workspace / "stops.json").write_text(json.dumps({"schema": "felicia.stops.v1", "stops": [{"candidate_key": "stop", "selected": True, "label": "Stop"}]}))
            (workspace / "mementos.json").write_text(json.dumps({"schema": "felicia.mementos.v1", "mementos": [{"id": "0190cbde-f300-7000-8000-a00000000001", "stop_key": "stop", "seq": 1, "kind": "goods", "occurred_at": "2026-04-01T09:00:00Z", "state": "draft", "title": "Video", "kind_data": {}, "media": [{"path": "secret.mp4", "kind": "video"}]}]}))
            (workspace / "route.gpx").write_text("<gpx />")
            (workspace / "secret.mp4").write_bytes(b"video")

            with self.assertRaisesRegex(SystemExit, "unsupported public media kind"):
                build_package(Namespace(workspace=workspace))

    def test_content_addressed_media_key_differs_for_same_basename_different_bytes(self):
        # Issue #75: two trips' IMG_0001.jpg must not collide. Same basename,
        # different content, referenced from two mementos in one package --
        # the object key is derived from content, not from the filename, so
        # both survive under distinct keys instead of one clobbering the other.
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.journey.v1",
                        "id": "0190cbde-f300-7000-8000-111111111111",
                        "journal_id": "0190cbde-f300-7000-8000-000000000000",
                        "slug": "media-collision",
                        "title": "Media collision",
                        "date_start": "2026-04-01",
                        "date_end": "2026-04-01",
                    }
                )
            )
            (workspace / "stops.json").write_text(
                json.dumps({"schema": "felicia.stops.v1", "stops": [{"candidate_key": "a", "selected": True, "label": "A"}, {"candidate_key": "b", "selected": True, "label": "B"}]})
            )
            (workspace / "mementos.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.mementos.v1",
                        "mementos": [
                            {"id": "0190cbde-f300-7000-8000-a00000000001", "stop_key": "a", "seq": 1, "kind": "goods", "occurred_at": "2026-04-01T09:00:00Z", "state": "draft", "title": "First", "kind_data": {}, "media": [{"path": "trip-a/IMG_0001.jpg", "caption": ""}]},
                            {"id": "0190cbde-f300-7000-8000-a00000000002", "stop_key": "b", "seq": 2, "kind": "goods", "occurred_at": "2026-04-01T10:00:00Z", "state": "draft", "title": "Second", "kind_data": {}, "media": [{"path": "trip-b/IMG_0001.jpg", "caption": ""}]},
                        ],
                    }
                )
            )
            (workspace / "route.gpx").write_text("<gpx />")
            (workspace / "trip-a").mkdir()
            (workspace / "trip-b").mkdir()
            (workspace / "trip-a" / "IMG_0001.jpg").write_bytes(b"trip-a-bytes")
            (workspace / "trip-b" / "IMG_0001.jpg").write_bytes(b"trip-b-bytes")

            output = build_package(Namespace(workspace=workspace))

            with zipfile.ZipFile(output) as archive:
                mementos = json.loads(archive.read("mementos.yaml"))
                by_title = {m["title"]: m["photos"][0]["path"] for m in mementos}
                media_names = [name for name in archive.namelist() if name.startswith("media/")]
                self.assertEqual(
                    2, len(set(by_title.values())), f"same-basename photos with different bytes must not collide: {by_title!r}"
                )
                self.assertEqual(sorted(by_title.values()), sorted(media_names))
                for path in by_title.values():
                    self.assertTrue(path.startswith("media/") and path.endswith(".jpg"), path)
                self.assertEqual(b"trip-a-bytes", archive.read(by_title["First"]))
                self.assertEqual(b"trip-b-bytes", archive.read(by_title["Second"]))

    def test_content_addressed_media_key_matches_for_identical_bytes(self):
        # Re-packaging (or two mementos sharing) identical photo bytes must
        # land on the same key -- dedup, and stable across re-runs of the
        # same trip rather than minting a fresh path each time.
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.journey.v1",
                        "id": "0190cbde-f300-7000-8000-111111111111",
                        "journal_id": "0190cbde-f300-7000-8000-000000000000",
                        "slug": "media-dedup",
                        "title": "Media dedup",
                        "date_start": "2026-04-01",
                        "date_end": "2026-04-01",
                    }
                )
            )
            (workspace / "stops.json").write_text(
                json.dumps({"schema": "felicia.stops.v1", "stops": [{"candidate_key": "a", "selected": True, "label": "A"}]})
            )
            (workspace / "mementos.json").write_text(
                json.dumps(
                    {
                        "schema": "felicia.mementos.v1",
                        "mementos": [
                            {
                                "id": "0190cbde-f300-7000-8000-a00000000001",
                                "stop_key": "a",
                                "seq": 1,
                                "kind": "goods",
                                "occurred_at": "2026-04-01T09:00:00Z",
                                "state": "draft",
                                "title": "Same photo twice",
                                "kind_data": {},
                                "media": [
                                    {"path": "first-name.jpg", "caption": "first"},
                                    {"path": "second-name.jpg", "caption": "second"},
                                ],
                            }
                        ],
                    }
                )
            )
            (workspace / "route.gpx").write_text("<gpx />")
            (workspace / "first-name.jpg").write_bytes(b"identical-bytes")
            (workspace / "second-name.jpg").write_bytes(b"identical-bytes")

            output = build_package(Namespace(workspace=workspace))

            with zipfile.ZipFile(output) as archive:
                mementos = json.loads(archive.read("mementos.yaml"))
                photo_paths = [photo["path"] for photo in mementos[0]["photos"]]
                self.assertEqual(1, len(set(photo_paths)), "identical bytes under different filenames must map to one key")
                media_names = [name for name in archive.namelist() if name.startswith("media/")]
                self.assertEqual(1, len(media_names), "identical bytes must only be stored once in the package")


class LocalJourneyIdentityTest(unittest.TestCase):
    """Issue #72: identity must be derived, stable, collision-free, and any
    real collision must fail loudly instead of silently overwriting."""

    def test_derive_journey_identity_is_stable_for_the_same_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            gpx = Path(directory) / "route.gpx"
            gpx.write_text("<gpx>same</gpx>")
            first = derive_journey_identity(gpx)
            second = derive_journey_identity(gpx)
            self.assertEqual(first, second, "re-hashing the same bytes must yield the same identity")

    def test_default_workspace_uses_the_private_workspaces_root(self):
        self.assertEqual(Path(__file__).resolve().parents[1] / ".felicia" / "workspaces", DEFAULT_WORKSPACE_ROOT)

    def test_derive_journey_identity_differs_for_different_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            gpx_a = Path(directory) / "a.gpx"
            gpx_b = Path(directory) / "b.gpx"
            gpx_a.write_text("<gpx>a</gpx>")
            gpx_b.write_text("<gpx>b</gpx>")
            journey_a, slug_a = derive_journey_identity(gpx_a)
            journey_b, slug_b = derive_journey_identity(gpx_b)
            self.assertNotEqual(journey_a, journey_b)
            self.assertNotEqual(slug_a, slug_b)
            self.assertTrue(slug_a.startswith("journey-"))
            self.assertFalse(slug_a.startswith("local-"))

    def test_resolve_identity_fills_in_defaults_from_gpx_when_unset(self):
        with tempfile.TemporaryDirectory() as directory:
            gpx = Path(directory) / "kyoto.gpx"
            gpx.write_text("<gpx>kyoto</gpx>")
            args = Namespace(gpx=gpx, journey=None, slug=None, title=None, workspace=None)
            resolve_identity(args)
            expected_journey, expected_slug = derive_journey_identity(gpx)
            self.assertEqual(expected_journey, args.journey)
            self.assertEqual(expected_slug, args.slug)
            self.assertIn(expected_slug, str(args.workspace))
            self.assertTrue(args.title)

    def test_resolve_identity_respects_explicit_overrides(self):
        with tempfile.TemporaryDirectory() as directory:
            gpx = Path(directory) / "kyoto.gpx"
            gpx.write_text("<gpx>kyoto</gpx>")
            workspace = Path(directory) / "my-workspace"
            args = Namespace(
                gpx=gpx,
                journey="0190cbde-f300-7000-8000-aaaaaaaaaaaa",
                slug="my-slug",
                title="My trip",
                workspace=workspace,
            )
            resolve_identity(args)
            self.assertEqual("0190cbde-f300-7000-8000-aaaaaaaaaaaa", args.journey)
            self.assertEqual("my-slug", args.slug)
            self.assertEqual("My trip", args.title)
            self.assertEqual(workspace, args.workspace)

    def test_guard_workspace_identity_allows_the_same_trip_to_rerun(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(json.dumps({"id": "journey-a", "slug": "slug-a"}))
            guard_workspace_identity(workspace, "journey-a", "slug-a")  # must not raise

    def test_guard_workspace_identity_blocks_a_different_trip(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "journey.json").write_text(json.dumps({"id": "journey-a", "slug": "slug-a"}))
            with self.assertRaisesRegex(SystemExit, "refusing to overwrite"):
                guard_workspace_identity(workspace, "journey-b", "slug-b")

    def test_guard_workspace_identity_is_a_noop_for_a_fresh_workspace(self):
        with tempfile.TemporaryDirectory() as directory:
            guard_workspace_identity(Path(directory), "journey-a", "slug-a")  # must not raise


if __name__ == "__main__":
    unittest.main()
