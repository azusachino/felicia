"""Package the curated local journey workspace for the real Felicia importer."""

from __future__ import annotations

import hashlib
import json
import uuid
import zipfile
from argparse import Namespace
from pathlib import Path

try:
    from .local_journey_common import NAMESPACE, read_json, safe_media_path
except ImportError:
    from local_journey_common import NAMESPACE, read_json, safe_media_path


def build_package(args: Namespace) -> Path:
    workspace = args.workspace.resolve()
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
            name = f"media/{source.name}"
            data = source.read_bytes()
            copied_media[name] = data
            photos.append(
                {
                    "id": str(uuid.uuid5(NAMESPACE, f"{memento['id']}:photo:{photo_index}")),
                    "path": name,
                    "content_hash": "sha256:" + hashlib.sha256(data).hexdigest(),
                    "caption": photo.get("caption", ""),
                    "seq": photo_index,
                }
            )
        package_mementos.append(
            {
                key: memento[key]
                for key in ("id", "seq", "kind", "occurred_at", "occurred_tz", "title", "place", "geom", "kind_data", "state")
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
