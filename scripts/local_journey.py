#!/usr/bin/env python3
"""Orchestrate the offline Felicia journey authoring loop."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import uuid
from datetime import datetime, timezone
from pathlib import Path

try:
    from .local_journey_author import interactive_author
    from .local_journey_common import CLI, NAMESPACE, ROOT, as_coord, candidate_key, ensure_cli, run, write_json
    from .local_journey_package import package
except ImportError:
    from local_journey_author import interactive_author
    from local_journey_common import CLI, NAMESPACE, ROOT, as_coord, candidate_key, ensure_cli, run, write_json
    from local_journey_package import package


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
        stops.append(
            {
                "candidate_key": candidate_key(source),
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
            "journal_id": args.journal,
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
    parser.add_argument("command", choices=("run", "preprocess", "package", "preview"))
    parser.add_argument("--workspace", type=Path, default=Path(".felicia/local-journey"))
    parser.add_argument("--journey", default="0190cbde-f300-7000-8000-111111111111")
    parser.add_argument("--journal", default="0190cbde-f300-7000-8000-000000000000")
    parser.add_argument("--slug", default="local-journey")
    parser.add_argument("--title", default="Local journey draft")
    parser.add_argument("--gpx", type=Path)
    parser.add_argument("--photos", type=Path)
    parser.add_argument("--sidecar", type=Path)
    return parser


def main() -> None:
    args = parser().parse_args()
    if args.command in {"run", "preprocess"}:
        if not args.gpx or not args.photos:
            raise SystemExit(f"{args.command} requires --gpx and --photos")
        preprocess(args)
        if args.command == "run":
            interactive_author(args)
            preview(args)
    elif args.command == "package":
        package(args)
    else:
        preview(args)


if __name__ == "__main__":
    main()
