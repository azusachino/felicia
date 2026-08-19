#!/usr/bin/env python3
"""Generate the static API projection used by the GitHub Pages design demo."""

from __future__ import annotations

import json
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "scripts" / "data.json"
PUBLIC = ROOT / "apps" / "felicia-public-site" / "public"
API = PUBLIC / "api" / "v1"
MEDIA = ("kyoto_temple.jpg", "tokyo_night.jpg", "osaka_plushie.jpg")


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    source = json.loads(SOURCE.read_text(encoding="utf-8"))
    if API.exists():
        shutil.rmtree(API)

    journey_items = []
    for journey in source["journeys"]:
        journey_id = journey["id"]
        journey_payload = {
            "id": journey_id,
            "journal_id": source["journal_id"],
            "slug": journey["slug"],
            "source_ref": journey.get("route_file"),
            "title": journey["title"],
            "place": journey["place"],
            "country": journey.get("country"),
            "region": journey.get("region"),
            "date_start": journey["date_start"],
            "date_end": journey["date_end"],
            "gps_route": {"type": "MultiLineString", "coordinates": journey["gps_route"]},
            "authored_fields": [],
        }
        mementos = []
        for index, memento in enumerate(journey.get("mementos", [])):
            item = {**memento, "journey_id": journey_id, "occurred_tz": memento.get("occurred_tz", "Asia/Tokyo")}
            item["geom"] = {"type": "Point", "coordinates": memento["geom"]}
            item["authored_fields"] = []
            if memento.get("photos"):
                item["photos"] = [
                    {
                        **photo,
                        "object_key": MEDIA[(index + photo_index) % len(MEDIA)],
                    }
                    for photo_index, photo in enumerate(memento["photos"])
                ]
            mementos.append(item)

        journey_dir = API / "journeys" / journey_id
        write_json(API / "journeys" / f"{journey_id}.json", journey_payload)
        write_json(journey_dir / "mementos.json", mementos)
        journey_items.append(
            {
                "id": journey_id,
                "slug": journey["slug"],
                "title": journey["title"],
                "memento_count": len(mementos),
                "representative_dots": [
                    {"coord": memento["geom"], "label": memento["place"]}
                    for memento in mementos[:3]
                ],
            }
        )

    write_json(API / "journeys.json", journey_items)


if __name__ == "__main__":
    main()
