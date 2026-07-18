#!/usr/bin/env python3
"""Build the Pages demo package through the local journey workflow."""

from __future__ import annotations

import json
import shutil
from argparse import Namespace
from pathlib import Path

from local_journey import package


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
    elif source.name == "kansai-ramble":
        (workspace / "route.gpx").write_text(
            '<?xml version="1.0"?><gpx version="1.1" creator="felicia demo" '
            'xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg>'
            '<trkpt lat="34.6687" lon="135.5016"><time>2026-04-01T01:00:00Z</time></trkpt>'
            '<trkpt lat="34.6794" lon="135.1878"><time>2026-04-02T04:00:00Z</time></trkpt>'
            '<trkpt lat="34.6851" lon="135.8398"><time>2026-04-03T05:00:00Z</time></trkpt>'
            '</trkseg></trk></gpx>\n'
        )
    else:
        shutil.copy2(ROOT / "scripts" / "tracks" / "narita-express.gpx", workspace / "route.gpx")
    journey = json.loads((source / "journey.json").read_text())
    stops = json.loads((source / "stops.json").read_text())["stops"]
    mementos = json.loads((source / "mementos.json").read_text())["mementos"]
    print(f"local journey source: {source}")
    print(f"journey: {journey['title']} ({journey['date_start']} → {journey['date_end']})")
    print(f"curated stops: {len([stop for stop in stops if stop.get('selected')])}")
    print(f"authored mementos: {len(mementos)}")
    return package(Namespace(workspace=workspace))


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    source_root = ROOT / "examples" / "preview" / "local-journey"
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
