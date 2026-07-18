#!/usr/bin/env python3
"""Simulate the offline Felicia journey authoring loop.

The files in the workspace are deliberately boring JSON so an author or a
local agent can edit them with any tool.  The package command turns those
files into the same portable package consumed by felicia-cli.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import uuid
import zipfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
CLI = ROOT / "bin" / "felicia-cli"
NAMESPACE = uuid.UUID("0190cbde-f300-7000-8000-999999999999")


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def read_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text())
    except FileNotFoundError as exc:
        raise SystemExit(f"missing {path}; run preprocess first") from exc
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON in {path}: {exc}") from exc


def run(command: list[str]) -> None:
    print("$", " ".join(command))
    subprocess.run(command, cwd=ROOT, check=True)


def ensure_cli() -> None:
    if not CLI.exists():
        run(["make", "cli-build"])


def as_coord(value: Any) -> list[float] | None:
    if isinstance(value, list) and len(value) == 2:
        return [float(value[0]), float(value[1])]
    return None


def candidate_key(stop: dict[str, Any]) -> str:
    identity = stop.get("identity", {})
    return str(identity.get("key") or stop.get("id") or "")


def preprocess(args: argparse.Namespace) -> None:
    workspace = args.workspace.resolve()
    workspace.mkdir(parents=True, exist_ok=True)
    ensure_cli()
    command = [
        str(CLI),
        "journey",
        "plan",
        "--journey",
        args.journey,
        "--gpx",
        str(args.gpx.resolve()),
        "--photos",
        str(args.photos.resolve()),
        "--format",
        "json",
    ]
    if args.sidecar:
        command.extend(["--sidecar", str(args.sidecar.resolve())])
    result = subprocess.run(command, cwd=ROOT, check=True, capture_output=True, text=True)
    plan = json.loads(result.stdout)
    write_json(workspace / "plan.json", plan)

    stops = []
    for index, source in enumerate(plan.get("stops", []), start=1):
        key = candidate_key(source)
        stops.append(
            {
                "candidate_key": key,
                "selected": True,
                "label": source.get("label", ""),
                "coord": as_coord(source.get("coord")),
                "arrive": source.get("arrive", ""),
                "depart": source.get("depart", ""),
                "confidence": source.get("confidence", 0),
                "source_index": index,
                "review_note": "Edit label/selection; this is the author's stop decision.",
            }
        )
    write_json(workspace / "stops.json", {"schema": "felicia.local.stops.v1", "stops": stops})

    mementos = []
    for index, source in enumerate(plan.get("mementos", []), start=1):
        mementos.append(
            {
                "id": str(uuid.uuid5(NAMESPACE, f"{args.journey}:memento:{index}")),
                "stop_key": source.get("stop_key", ""),
                "seq": index,
                "kind": source.get("kind") or "note",
                "occurred_at": source.get("occurred_at", ""),
                "occurred_tz": source.get("occurred_tz", "UTC"),
                "title": source.get("title", ""),
                "place": source.get("place", ""),
                "geom": as_coord(source.get("geom")),
                "state": "draft",
                "kind_data": source.get("kind_data") or {},
                "media": [
                    {"path": asset.get("uri", ""), "caption": asset.get("title", "")}
                    for asset in source.get("media", [])
                    if asset.get("uri")
                ],
                "author_note": "Edit title, kind, kind_data, and media before preview.",
            }
        )
    write_json(workspace / "mementos.json", {"schema": "felicia.local.mementos.v1", "mementos": mementos})

    now = datetime.now(timezone.utc).date().isoformat()
    write_json(
        workspace / "journey.json",
        {
            "schema": "felicia.local.journey.v1",
            "id": args.journey,
            "journal_id": str(uuid.uuid5(NAMESPACE, "journal")),
            "slug": args.slug,
            "title": args.title,
            "place": "",
            "country": "",
            "region": "",
            "date_start": now,
            "date_end": now,
            "source_ref": args.gpx.name,
        },
    )
    shutil.copy2(args.gpx, workspace / "route.gpx")
    print(f"preprocess ready: {workspace}")
    print("edit journey.json, stops.json, and mementos.json, then run `package` or `preview`")


def safe_media_path(workspace: Path, source: str) -> Path:
    path = Path(source)
    if path.is_absolute():
        resolved = path.resolve()
    else:
        resolved = (workspace / path).resolve()
        if not resolved.exists():
            resolved = (ROOT / path).resolve()
    if not resolved.is_file():
        raise SystemExit(f"media file does not exist: {source}")
    return resolved


def package(args: argparse.Namespace) -> Path:
    workspace = args.workspace.resolve()
    journey = read_json(workspace / "journey.json")
    stop_data = read_json(workspace / "stops.json")
    memento_data = read_json(workspace / "mementos.json")
    selected = {
        stop["candidate_key"]
        for stop in stop_data.get("stops", [])
        if stop.get("selected", stop.get("state", "kept") == "kept")
        and stop.get("state", "kept") != "ignored"
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
                for key in (
                    "id",
                    "seq",
                    "kind",
                    "occurred_at",
                    "occurred_tz",
                    "title",
                    "place",
                    "geom",
                    "kind_data",
                    "state",
                )
                if key in memento
            }
            | {"photos": photos}
        )
    files["mementos.yaml"] = (json.dumps(package_mementos, ensure_ascii=False) + "\n").encode()
    files["route.gpx"] = (workspace / "route.gpx").read_bytes()
    files.update(copied_media)

    manifest_lines = [
        'schema_version: "1"',
        f'package_id: "local-{journey["id"]}"',
        'source: "felicia local journey workflow"',
        "files:",
    ]
    for name, data in files.items():
        manifest_lines.extend(
            [
                f"  - path: {name}",
                "    kind: local-workflow",
                f"    bytes: {len(data)}",
                f"    sha256: {hashlib.sha256(data).hexdigest()}",
            ]
        )
    files["manifest.yaml"] = ("\n".join(manifest_lines) + "\n").encode()
    output = workspace / "journey.zip"
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for name, data in files.items():
            archive.writestr(name, data)
    print(f"package ready: {output} (stops={len(selected)}, mementos={len(package_mementos)}, media={len(copied_media)})")
    return output


def preview(args: argparse.Namespace) -> None:
    package_path = package(args)
    workspace = args.workspace.resolve()
    database = workspace / "felicia.sqlite"
    media_root = workspace / "media"
    site = workspace / "site"
    ensure_cli()
    run([str(CLI), "package", "validate", str(package_path)])
    run([str(CLI), "import", "--db", str(database), "--media-root", str(media_root), "--apply", str(package_path)])
    run([str(CLI), "static", "compile", "--db", str(database), "--media-root", str(media_root), "--out", str(site)])
    print(f"preview ready: {site}")


def parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("preprocess", "package", "preview"))
    parser.add_argument("--workspace", type=Path, default=Path(".felicia/local-journey"))
    parser.add_argument("--journey", default="0190cbde-f300-7000-8000-111111111111")
    parser.add_argument("--slug", default="local-journey")
    parser.add_argument("--title", default="Local journey draft")
    parser.add_argument("--gpx", type=Path)
    parser.add_argument("--photos", type=Path)
    parser.add_argument("--sidecar", type=Path)
    return parser


def main() -> None:
    args = parser().parse_args()
    if args.command == "preprocess":
        if not args.gpx or not args.photos:
            raise SystemExit("preprocess requires --gpx and --photos")
        preprocess(args)
    elif args.command == "package":
        package(args)
    else:
        preview(args)


if __name__ == "__main__":
    main()
