#!/usr/bin/env python3
"""Build the Pages demo package through the local journey workflow."""

from __future__ import annotations

import shutil
from argparse import Namespace
from pathlib import Path

from local_journey import package


ROOT = Path(__file__).resolve().parent.parent
OUTPUT = ROOT / ".felicia" / "preview.zip"


def build_from_local_workspace() -> Path:
    source = ROOT / "examples" / "preview" / "local-journey"
    workspace = ROOT / ".felicia" / "preview-workspace"
    workspace.mkdir(parents=True, exist_ok=True)
    for name in ("journey.json", "stops.json", "mementos.json"):
        shutil.copy2(source / name, workspace / name)
    shutil.copy2(ROOT / "scripts" / "tracks" / "narita-express.gpx", workspace / "route.gpx")
    return package(Namespace(workspace=workspace))


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    generated = build_from_local_workspace()
    shutil.copy2(generated, OUTPUT)
    print(f"preview package ready: {OUTPUT}")


if __name__ == "__main__":
    main()
