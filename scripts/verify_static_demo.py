#!/usr/bin/env python3
"""Verify the files produced by the GitHub Pages design-demo build."""

from __future__ import annotations

import json
import os
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
DIST = ROOT / "apps" / "web-public" / "dist"
REQUIRED_MEDIA = {"kyoto_temple.jpg", "tokyo_night.jpg", "osaka_plushie.jpg"}


def main() -> None:
    expected_base = os.environ.get("BASE_PATH", "/")
    if not expected_base.endswith("/"):
        expected_base += "/"

    index = (DIST / "index.html").read_text(encoding="utf-8")
    journeys = json.loads((DIST / "api/v1/journeys.json").read_text(encoding="utf-8"))
    journey_files = list((DIST / "api/v1/journeys").glob("*.json"))
    memento_files = []
    for journey in journeys:
        journey_id = journey["id"]
        detail_path = DIST / "api/v1" / "journeys" / f"{journey_id}.json"
        mementos_path = DIST / "api/v1" / "journeys" / journey_id / "mementos.json"
        detail = json.loads(detail_path.read_text(encoding="utf-8"))
        json.loads(mementos_path.read_text(encoding="utf-8"))
        assert detail["id"] == journey_id, f"journey detail id mismatch: {journey_id}"
        memento_files.append(mementos_path)
    media = {path.name for path in DIST.iterdir() if path.suffix in {".jpg", ".jpeg", ".png", ".webp"}}

    assert expected_base in index, f"base path {expected_base!r} is missing from index.html"
    assert len(journeys) == len(journey_files) == len(memento_files), "journey projections disagree"
    assert REQUIRED_MEDIA <= media, f"missing demo media: {REQUIRED_MEDIA - media}"

    print(
        "static demo verified: "
        f"base={expected_base} journeys={len(journeys)} "
        f"journey_files={len(journey_files)} memento_files={len(memento_files)} media={len(media)}"
    )


if __name__ == "__main__":
    main()
