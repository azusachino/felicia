"""Package the curated local journey workspace for the real Felicia importer."""

from __future__ import annotations

import hashlib
import json
import mimetypes
import uuid
import zipfile
from argparse import Namespace
from pathlib import Path

try:
    from .local_journey_common import NAMESPACE, read_json, safe_media_path
    from .validate_local_authoring import validate_workspace
except ImportError:
    from local_journey_common import NAMESPACE, read_json, safe_media_path
    from validate_local_authoring import validate_workspace


def build_package(args: Namespace) -> Path:
    workspace = args.workspace.resolve()
    try:
        validate_workspace(workspace)
    except ValueError as error:
        raise SystemExit(str(error)) from error
    journey = read_json(workspace / "journey.json")
    stop_data = read_json(workspace / "stops.json")
    memento_data = read_json(workspace / "mementos.json")
    selected = {
        stop["candidate_key"]
        for stop in stop_data.get("stops", [])
        if stop.get("selected", stop.get("state", "kept") == "kept") and stop.get("state", "kept") != "ignored"
    }
    if not journey.get("id") or not journey.get("journal_id"):
        raise SystemExit("journey.json requires id and journal_id")
    if not (workspace / "route.gpx").is_file():
        raise SystemExit("route.gpx is missing; run preprocess first")

    files: dict[str, bytes] = {}
    journey_yaml = {key: value for key, value in journey.items() if key != "schema"}
    files["journey.yaml"] = (json.dumps(journey_yaml, ensure_ascii=False) + "\n").encode()
    package_stops = []
    for stop in stop_data.get("stops", []):
        if not stop.get("selected", stop.get("state", "kept") == "kept") or stop.get("state", "kept") == "ignored":
            continue
        package_stops.append(
            {
                "derivation_version": "gpx-stops-v1",
                "key": stop.get("candidate_key", ""),
                "label": stop.get("label", ""),
                "coord": stop.get("coord"),
                "arrive": stop.get("arrive", ""),
                "depart": stop.get("depart", ""),
                "confidence": stop.get("confidence", 0),
            }
        )
    files["stops.yaml"] = (json.dumps(package_stops, ensure_ascii=False) + "\n").encode()
    package_mementos = []
    copied_media: dict[str, bytes] = {}
    for memento in memento_data.get("mementos", []):
        if memento.get("stop_key") not in selected or memento.get("state") == "archived":
            continue
        if not memento.get("id") or not memento.get("occurred_at") or not memento.get("kind"):
            raise SystemExit("each included memento requires id, kind, and occurred_at")
        photos = []
        for photo_index, photo in enumerate(memento.get("media", []), start=1):
            source = safe_media_path(workspace, str(photo.get("path", "")))
            validate_public_image(photo, source)
            data = source.read_bytes()
            digest = hashlib.sha256(data).hexdigest()
            # Content-addressed, not basename-addressed (ADR-0026 / issue #75):
            # two trips' IMG_0001.jpg no longer collide in the shared media
            # root or in dist/, because the key IS the content. Re-packaging
            # identical bytes reproduces the same key (dedup, cache-safe);
            # the extension is kept so MIME/type detection downstream
            # (validate_public_image above, publication.SanitizePublicImage)
            # keeps working.
            name = f"media/{digest}{source.suffix.lower()}"
            if name in copied_media and copied_media[name] != data:
                raise SystemExit(f"content-addressed media key collided with different bytes: {name}")
            copied_media[name] = data
            photos.append(
                {
                    "id": str(uuid.uuid5(NAMESPACE, f"{memento['id']}:photo:{photo_index}")),
                    "path": name,
                    "content_hash": "sha256:" + digest,
                    "caption": photo.get("caption", ""),
                    "seq": photo_index,
                }
            )
        package_mementos.append(
            {
                key: memento[key]
                for key in (
                    "id", "seq", "kind", "occurred_at", "occurred_tz", "title", "place", "geom",
                    "vendor", "essay", "price_amount", "price_currency", "authored_fields", "kind_data", "state"
                )
                if key in memento
            }
            | {"photos": photos}
        )
    files["mementos.yaml"] = (json.dumps(package_mementos, ensure_ascii=False) + "\n").encode()
    files["route.gpx"] = (workspace / "route.gpx").read_bytes()
    files.update(copied_media)

    manifest_lines = ['schema_version: "1"', f'package_id: "local-{journey["id"]}"', 'source: "felicia local journey workflow"', "files:"]
    for name, data in files.items():
        manifest_lines.extend([f"  - path: {name}", "    kind: local-workflow", f"    bytes: {len(data)}", f"    sha256: {hashlib.sha256(data).hexdigest()}"])
    files["manifest.yaml"] = ("\n".join(manifest_lines) + "\n").encode()
    output = workspace / "journey.zip"
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for name, data in files.items():
            archive.writestr(name, data)
    print(f"package ready: {output} (stops={len(selected)}, mementos={len(package_mementos)}, media={len(copied_media)})")
    return output


def validate_public_image(attachment: dict, source: Path) -> None:
    """Enforce the v1 public-package media boundary."""
    visibility = attachment.get("visibility", "public")
    if visibility != "public":
        raise SystemExit(f"private media cannot enter a public package: {attachment.get('path', '')}")
    kind = attachment.get("kind", "image")
    if kind != "image":
        raise SystemExit(f"unsupported public media kind {kind!r}: {attachment.get('path', '')}")
    extension = source.suffix.lower()
    if extension not in {".jpg", ".jpeg", ".png", ".webp"}:
        raise SystemExit(f"only JPEG, PNG, and WebP images are supported in public packages: {source.name}")
    declared_mime = attachment.get("mime")
    detected_mime, _ = mimetypes.guess_type(source.name)
    allowed_mime = {"image/jpeg", "image/png", "image/webp"}
    if declared_mime and declared_mime not in allowed_mime:
        raise SystemExit(f"unsupported public image MIME {declared_mime!r}: {source.name}")
    if detected_mime and detected_mime not in allowed_mime:
        raise SystemExit(f"unsupported public image MIME {detected_mime!r}: {source.name}")
