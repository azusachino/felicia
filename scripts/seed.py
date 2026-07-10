#!/usr/bin/env python3
"""Load the canonical mock scenario through the admin API."""

import json
import os
import sys
import urllib.error
import urllib.request
import uuid
from pathlib import Path


API_BASE = os.getenv("SEED_API_BASE", "http://localhost:8080")
DATA_PATH = Path(__file__).with_name("data.json")


def post(path: str, payload: dict) -> dict:
    request = urllib.request.Request(
        f"{API_BASE}{path}",
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode()
        raise RuntimeError(f"POST {path} failed ({exc.code}): {detail}") from exc


def localized(value: dict, lang: str) -> str:
    return value.get(lang) or value["ja"]


def seed(data: dict) -> None:
    post("/api/admin/journals", {"id": data["journal_id"]})
    for journey in data["journeys"]:
        journey_id = journey["id"]
        post(
            "/api/admin/journeys",
            {
                "id": journey_id,
                "journal_id": data["journal_id"],
                "slug": journey["slug"],
                "source_ref": f"mock:{journey['slug']}",
                "title": journey["title"]["ja"],
                "place": journey["place"]["ja"],
                "country": journey.get("country"),
                "region": journey.get("region"),
                "date_start": journey["date_start"],
                "date_end": journey["date_end"],
                "gps_route": journey["gps_route"],
                "authored_fields": [],
            },
        )
        for lang in ("en", "zh"):
            post_translation("journey", journey_id, "title", lang, localized(journey["title"], lang))
            post_translation("journey", journey_id, "place", lang, localized(journey["place"], lang))

        for memento in journey["mementos"]:
            memento_id = memento["id"]
            post(
                "/api/admin/mementos",
                {
                    "id": memento_id,
                    "journey_id": journey_id,
                    "kind": memento["kind"],
                    "seq": memento["seq"],
                    "occurred_at": memento["occurred_at"],
                    "occurred_tz": memento.get("occurred_tz", "Asia/Tokyo"),
                    "geom": {"type": "Point", "coordinates": memento["geom"]},
                    "title": memento["title"]["ja"],
                    "place": memento["place"]["ja"],
                    "vendor": memento.get("vendor"),
                    "essay": memento.get("essay", {}).get("ja"),
                    "price_amount": memento.get("price_amount"),
                    "price_currency": memento.get("price_currency", "JPY"),
                    "kind_data": memento["kind_data"],
                    "source_ref": f"mock:{journey['slug']}:{memento_id}",
                    "authored_fields": [],
                },
            )
            for lang in ("en", "zh"):
                post_translation("memento", memento_id, "title", lang, localized(memento["title"], lang))
                post_translation("memento", memento_id, "place", lang, localized(memento["place"], lang))
                if memento.get("essay"):
                    post_translation("memento", memento_id, "essay", lang, localized(memento["essay"], lang))
            for photo in memento.get("photos", []):
                post("/api/admin/photos", {**photo, "memento_id": memento_id})


def post_translation(owner_type: str, owner_id: str, field: str, lang: str, value: str) -> None:
    post(
        "/api/admin/translations",
        {
            "id": str(uuid.uuid5(uuid.NAMESPACE_URL, f"felicia:{owner_type}:{owner_id}:{lang}:{field}")),
            "owner_type": owner_type,
            "owner_id": owner_id,
            "lang": lang,
            "field": field,
            "value": value,
            "provenance": "machine",
        },
    )


def main() -> None:
    try:
        with DATA_PATH.open(encoding="utf-8") as source:
            seed(json.load(source))
    except (OSError, json.JSONDecodeError, RuntimeError, KeyError) as exc:
        print(f"seed failed: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
    print(f"seeded mock data from {DATA_PATH} via {API_BASE}")


if __name__ == "__main__":
    main()
