#!/usr/bin/env python3
"""Build the small raw package used by the no-setup local preview."""

from __future__ import annotations

import hashlib
import zipfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
OUTPUT = ROOT / ".felicia" / "preview.zip"


def package_files() -> dict[str, Path]:
    return {
        "journey.yaml": ROOT / "examples" / "preview" / "journey.yaml",
        "mementos.yaml": ROOT / "examples" / "preview" / "mementos.yaml",
        "route.gpx": ROOT / "scripts" / "tracks" / "narita-express.gpx",
        "media/kyoto_temple.jpg": ROOT / "apps" / "web-public" / "public" / "kyoto_temple.jpg",
    }


def manifest(files: dict[str, bytes]) -> bytes:
    lines = [
        'schema_version: "1"',
        'package_id: "preview-sample-v1"',
        'source: "felicia preview fixture"',
        "files:",
    ]
    for name, data in files.items():
        digest = hashlib.sha256(data).hexdigest()
        lines.extend(
            [
                f"  - path: {name}",
                "    kind: preview",
                f"    bytes: {len(data)}",
                f"    sha256: {digest}",
            ]
        )
    return ("\n".join(lines) + "\n").encode()


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    files = {name: path.read_bytes() for name, path in package_files().items()}
    with zipfile.ZipFile(OUTPUT, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("manifest.yaml", manifest(files))
        for name, data in files.items():
            archive.writestr(name, data)
    print(f"preview package ready: {OUTPUT}")


if __name__ == "__main__":
    main()
