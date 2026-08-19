#!/usr/bin/env python3
"""Build the checked-in Izu package through the local journey workflow."""

from __future__ import annotations

import json
import shutil
from argparse import Namespace
from pathlib import Path

from local_journey import build_package
from validate_local_authoring import validate_workspace_root


ROOT = Path(__file__).resolve().parent.parent
OUTPUT = ROOT / ".felicia" / "preview.zip"
PACKAGE_DIR = ROOT / ".felicia" / "preview-packages"


def build_from_local_workspace(source: Path, workspace: Path) -> Path:
    workspace.mkdir(parents=True, exist_ok=True)
    for name in ("journey.json", "stops.json", "mementos.json"):
        shutil.copy2(source / name, workspace / name)
    route = source / "route.gpx"
    if route.is_file():
        shutil.copy2(route, workspace / "route.gpx")
    else:
        raise SystemExit(f"route.gpx is required in checked-in journey source: {source}")
    journey = json.loads((source / "journey.json").read_text())
    stops = json.loads((source / "stops.json").read_text())["stops"]
    mementos = json.loads((source / "mementos.json").read_text())["mementos"]
    for memento in mementos:
        for photo in memento.get("media", []):
            relative_path = Path(str(photo.get("path", "")))
            source_media = (source / relative_path).resolve()
            if not source_media.is_file():
                source_media = (ROOT / relative_path).resolve()
            if not source_media.is_file():
                raise SystemExit(f"media file does not exist: {photo.get('path', '')}")
            destination = workspace / relative_path
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source_media, destination)
    print(f"local journey source: {source}")
    print(f"journey: {journey['title']} ({journey['date_start']} → {journey['date_end']})")
    print(f"curated stops: {len([stop for stop in stops if stop.get('selected')])}")
    print(f"authored mementos: {len(mementos)}")
    return build_package(Namespace(workspace=workspace))


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    source_root = ROOT / "examples" / "preview" / "local-journey"
    manifest = source_root / "workspace.json"
    if manifest.is_file():
        validate_workspace_root(source_root)
        entries = json.loads(manifest.read_text(encoding="utf-8"))["journeys"]
        sources = [(source_root / entry["path"]).resolve() for entry in entries]
    else:
        sources = [source_root, *sorted(path for path in source_root.iterdir() if path.is_dir())]
    shutil.rmtree(PACKAGE_DIR, ignore_errors=True)
    PACKAGE_DIR.mkdir(parents=True, exist_ok=True)
    generated = []
    for index, source in enumerate(sources, start=1):
        workspace = ROOT / ".felicia" / "preview-workspace" / source.name
        generated.append(build_from_local_workspace(source, workspace))
        shutil.copy2(generated[-1], PACKAGE_DIR / f"{index:02d}-{source.name}.zip")
    shutil.copy2(generated[0], OUTPUT)
    print(f"preview packages ready: {len(generated)}")


if __name__ == "__main__":
    main()
