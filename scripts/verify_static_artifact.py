#!/usr/bin/env python3
"""Verify the complete CLI-backed public artifact."""

from __future__ import annotations

import json
import os
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent.parent


class AssetParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.assets: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        url = values.get("src") if tag == "script" else values.get("href") if tag == "link" else None
        if url and not urlsplit(url).netloc and (tag == "script" or values.get("rel") in {"stylesheet", "modulepreload"}):
            self.assets.append(url)


def main() -> None:
    dist = Path(os.environ.get("SITE_DIST", "apps/felicia-public-site/dist"))
    if not dist.is_absolute():
        dist = ROOT / dist
    expected_base = os.environ.get("BASE_PATH", "/")
    if not expected_base.endswith("/"):
        expected_base += "/"
    index = (dist / "index.html").read_text(encoding="utf-8")
    parser = AssetParser()
    parser.feed(index)
    assert parser.assets, "index.html contains no script or stylesheet assets"
    for url in parser.assets:
        assert url.startswith(expected_base + "assets/"), f"asset {url!r} does not match base path {expected_base!r}"
    journeys = json.loads((dist / "api/v1/journeys.json").read_text(encoding="utf-8"))
    assert journeys, "static artifact contains no journeys"
    media = set()
    for journey in journeys:
        journey_id = journey["id"]
        detail = json.loads((dist / "api/v1/journeys" / f"{journey_id}.json").read_text(encoding="utf-8"))
        assert detail["id"] == journey_id
        mementos = json.loads((dist / "api/v1/journeys" / journey_id / "mementos.json").read_text(encoding="utf-8"))
        for memento in mementos:
            for photo in memento.get("photos", []):
                media.add(photo["object_key"])
    for object_key in media:
        assert (dist / object_key).is_file(), f"missing published media: {object_key}"
    print(f"static artifact verified: journeys={len(journeys)} media={len(media)} base={expected_base}")


if __name__ == "__main__":
    main()
