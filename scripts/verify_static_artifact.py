#!/usr/bin/env python3
"""Verify the complete CLI-backed public artifact."""

from __future__ import annotations

import json
import os
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
DIST = ROOT / "apps" / "felicia-public-site" / "dist"


def main() -> None:
    expected_base = os.environ.get("BASE_PATH", "/")
    if not expected_base.endswith("/"):
        expected_base += "/"
    index = (DIST / "index.html").read_text(encoding="utf-8")
    assert expected_base in index, f"base path {expected_base!r} is missing from index.html"
    journeys = json.loads((DIST / "api/v1/journeys.json").read_text(encoding="utf-8"))
    assert journeys, "static artifact contains no journeys"
    media = set()
    for journey in journeys:
        journey_id = journey["id"]
        detail = json.loads((DIST / "api/v1/journeys" / f"{journey_id}.json").read_text(encoding="utf-8"))
        assert detail["id"] == journey_id
        mementos = json.loads((DIST / "api/v1/journeys" / journey_id / "mementos.json").read_text(encoding="utf-8"))
        for memento in mementos:
            for photo in memento.get("photos", []):
                media.add(photo["object_key"])
    for object_key in media:
        assert (DIST / object_key).is_file(), f"missing published media: {object_key}"
    print(f"static artifact verified: journeys={len(journeys)} media={len(media)} base={expected_base}")


if __name__ == "__main__":
    main()
