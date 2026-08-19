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
    from .local_journey_common import (
        CLI,
        DEFAULT_WORKSPACE_ROOT,
        NAMESPACE,
        ROOT,
        as_coord,
        candidate_key,
        derive_journey_identity,
        ensure_cli,
        read_json,
        run,
        write_json,
    )
    from .local_journey_package import build_package
except ImportError:
    from local_journey_author import interactive_author
    from local_journey_common import (
        CLI,
        DEFAULT_WORKSPACE_ROOT,
        NAMESPACE,
        ROOT,
        as_coord,
        candidate_key,
        derive_journey_identity,
        ensure_cli,
        read_json,
        run,
        write_json,
    )
    from local_journey_package import build_package


def plan_date(value: str | None, fallback: str) -> str:
    """Take the calendar date out of a plan bound, or fall back."""
    if not value:
        return fallback
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).date().isoformat()
    except ValueError:
        return fallback


def workspace_media_path(photo_root: Path, asset: dict) -> str:
    """Point the workspace at a photo the packager can actually resolve.

    Plan media assets come from an untagged Go struct, so their keys are
    PascalCase, and `URI` is relative to the photo root rather than to the
    workspace or the repository. Packaging resolves paths against the
    workspace and then the repository root, so rebase to one of those.
    """
    resolved = (photo_root.resolve() / asset["URI"]).resolve()
    try:
        return resolved.relative_to(ROOT).as_posix()
    except ValueError:
        return str(resolved)


def resolve_identity(args: argparse.Namespace) -> None:
    """Fill in journey/slug/title/workspace from the GPX when the author
    didn't name the trip explicitly (issue #72). Mutates `args` in place so
    every later use of these fields -- including `interactive_author` and
    `preview` in the `run` command -- sees the same resolved values.
    """
    derived_journey, derived_slug = derive_journey_identity(args.gpx.resolve())
    if not args.journey:
        args.journey = derived_journey
    if not args.slug:
        args.slug = derived_slug
    if not args.title:
        args.title = f"Local trip ({args.gpx.stem})"
    if args.workspace is None:
        args.workspace = DEFAULT_WORKSPACE_ROOT / args.slug


def guard_workspace_identity(workspace: Path, journey_id: str, slug: str) -> None:
    """Refuse to reuse a workspace that already describes a different trip.

    Re-running the *same* trip (same derived or explicit journey id) must
    stay idempotent; importing a *different* trip into a workspace that
    already holds one must fail loudly instead of silently overwriting it
    on disk and, later, in the database (issue #72).
    """
    existing_path = workspace / "journey.json"
    if not existing_path.is_file():
        return
    try:
        existing = read_json(existing_path)
    except SystemExit:
        return  # unreadable/malformed workspace file; let the normal write replace it
    existing_id = existing.get("id")
    if existing_id and existing_id != journey_id:
        raise SystemExit(
            f"workspace {workspace} already holds journey {existing_id} "
            f"(slug {existing.get('slug')!r}); refusing to overwrite it with a "
            f"different journey {journey_id} (slug {slug!r}). Pass a distinct "
            "--workspace (and/or --slug) for the new trip."
        )


def preprocess(args: argparse.Namespace) -> None:
    resolve_identity(args)
    workspace = args.workspace.resolve()
    workspace.mkdir(parents=True, exist_ok=True)
    guard_workspace_identity(workspace, args.journey, args.slug)
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
    write_json(workspace / "stops.json", {"schema": "felicia.stops.v1", "stops": stops})

    mementos = []
    for index, source in enumerate(plan.get("mementos", []), start=1):
        mementos.append(
            {
                "id": str(uuid.uuid5(NAMESPACE, f"{args.journey}:memento:{index}")),
                "stop_key": source.get("stop_key", ""),
                "seq": index,
                "kind": source.get("kind") or "goods",
                "occurred_at": source.get("occurred_at", ""),
                # The planner leaves the zone empty until offline resolution
                # lands, and the key is always present -- so default on the
                # value, not on the key, or packaging rejects the workspace.
                "occurred_tz": source.get("occurred_tz") or "UTC",
                "title": source.get("title", ""),
                "place": source.get("place", ""),
                "geom": as_coord(source.get("geom")),
                "state": "draft",
                "kind_data": source.get("kind_data") or {},
                "media": [
                    {"path": workspace_media_path(args.photos, asset), "caption": asset.get("Title", "")}
                    for asset in source.get("media", [])
                    if asset.get("URI")
                ],
                "author_note": "Edit title, kind, kind_data, and media before preview.",
            }
        )
    write_json(workspace / "mementos.json", {"schema": "felicia.mementos.v1", "mementos": mementos})

    # The planner derives the journey's date bounds from the route, visit, and
    # media timestamps; today's date is only the fallback for a plan that could
    # not derive them. Writing `now` unconditionally publishes a trip dated the
    # day it was imported.
    today = datetime.now(timezone.utc).date().isoformat()
    date_start = plan_date(plan.get("date_start"), today)
    date_end = plan_date(plan.get("date_end"), date_start)
    write_json(
        workspace / "journey.json",
        {
            "schema": "felicia.journey.v1",
            "id": args.journey,
            "journal_id": args.journal,
            "slug": args.slug,
            "title": args.title,
            "place": "",
            "country": "",
            "region": "",
            "date_start": date_start,
            "date_end": date_end,
            "source_ref": args.gpx.name,
        },
    )
    shutil.copy2(args.gpx, workspace / "route.gpx")
    print(f"preprocess ready: {workspace}")
    print(f"journey={args.journey} slug={args.slug!r} title={args.title!r}")
    print(f"pass --workspace {workspace} to `package`/`preview` for this trip")
    print("edit journey.json, stops.json, and mementos.json, then run `package` or `preview`")


def preview(args: argparse.Namespace) -> None:
    package_path = build_package(args)
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
    parser.add_argument(
        "--workspace",
        type=Path,
        default=None,
        help="defaults to .felicia/local-journey/<slug> for preprocess/run "
        "(derived from --slug or the GPX content); required for package/preview "
        "when not reusing that default",
    )
    parser.add_argument(
        "--journey",
        default=None,
        help="defaults to a uuid5 derived from the GPX track's own bytes -- "
        "stable across re-runs of the same trip, distinct across trips (issue #72)",
    )
    parser.add_argument("--journal", default="0190cbde-f300-7000-8000-000000000000")
    parser.add_argument("--slug", default=None, help="defaults to a slug derived from the GPX content")
    parser.add_argument("--title", default=None, help="defaults to a title derived from the GPX filename")
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
    elif args.workspace is None:
        raise SystemExit(
            f"{args.command} requires --workspace (preprocess printed the workspace "
            "path to reuse -- see its 'preprocess ready:' line)"
        )
    elif args.command == "package":
        build_package(args)
    else:
        preview(args)


if __name__ == "__main__":
    main()
