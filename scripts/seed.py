#!/usr/bin/env python3
"""Load the canonical mock scenario through the admin API."""

import json
import os
import sys
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET
from datetime import date, datetime
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


def get(path: str) -> object:
    request = urllib.request.Request(f"{API_BASE}{path}", method="GET")
    try:
        with urllib.request.urlopen(request) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode()
        raise RuntimeError(f"GET {path} failed ({exc.code}): {detail}") from exc


def read_gpx(relative_path: str) -> list[list[list[float]]]:
    path = DATA_PATH.parent / relative_path
    try:
        root = ET.parse(path).getroot()
    except (ET.ParseError, OSError) as exc:
        raise RuntimeError(f"read GPX {relative_path} failed: {exc}") from exc

    segments: list[list[list[float]]] = []
    for segment in root.findall(".//{*}trkseg"):
        points = [
            [float(point.attrib["lon"]), float(point.attrib["lat"])]
            for point in segment.findall("{*}trkpt")
            if "lon" in point.attrib and "lat" in point.attrib
        ]
        if len(points) >= 2:
            segments.append(points)
    if not segments:
        raise RuntimeError(f"GPX {relative_path} contains no track segment with two points")
    return segments


def validate_data(data: dict) -> None:
    required_kinds = {"transit", "live", "goods", "receipt", "stamp"}
    journey_ids: set[str] = set()
    memento_ids: set[str] = set()
    for journey in data.get("journeys", []):
        if journey["id"] in journey_ids:
            raise RuntimeError(f"duplicate journey id {journey['id']}")
        journey_ids.add(journey["id"])
        country = journey.get("country")
        if not isinstance(country, str) or len(country) != 3 or country != country.upper():
            raise RuntimeError(f"journey {journey['slug']} must declare one ISO country code")
        start = date.fromisoformat(journey["date_start"])
        end = date.fromisoformat(journey["date_end"])
        previous = None
        kinds = set()
        for memento in journey["mementos"]:
            if memento["id"] in memento_ids:
                raise RuntimeError(f"duplicate memento id {memento['id']}")
            memento_ids.add(memento["id"])
            occurred = datetime.fromisoformat(memento["occurred_at"])
            if not start <= occurred.date() <= end:
                raise RuntimeError(f"memento {memento['id']} falls outside {journey['slug']} dates")
            if previous and occurred < previous:
                raise RuntimeError(f"mementos are not chronological in {journey['slug']}")
            previous = occurred
            kinds.add(memento["kind"])
        if kinds != required_kinds:
            raise RuntimeError(f"journey {journey['slug']} does not cover all memento kinds")


def seed(data: dict) -> None:
    validate_data(data)
    post(f"/api/admin/journals/{data['journal_id']}/reset-mock", {})
    post("/api/admin/journals", {"id": data["journal_id"]})
    existing_journeys = get("/api/admin/journeys")
    if not isinstance(existing_journeys, list):
        raise RuntimeError("GET /api/admin/journeys returned an invalid response")
    journey_ids = {
        item["slug"]: item["id"]
        for item in existing_journeys
        if isinstance(item, dict) and isinstance(item.get("slug"), str) and item.get("id")
    }

    for journey in data["journeys"]:
        journey_id = journey_ids.get(journey["slug"], journey["id"])
        route = read_gpx(journey["route_file"]) if journey.get("route_file") else journey["gps_route"]
        post(
            "/api/admin/journeys",
            {
                "id": journey_id,
                "journal_id": data["journal_id"],
                "slug": journey["slug"],
                "source_ref": f"mock:{journey['slug']}",
                "title": journey["title"],
                "place": journey["place"],
                "country": journey.get("country"),
                "region": journey.get("region"),
                "date_start": journey["date_start"],
                "date_end": journey["date_end"],
                "gps_route": route,
                "authored_fields": [],
            },
        )
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
                    "title": memento["title"],
                    "place": memento["place"],
                    "vendor": memento.get("vendor"),
                    "essay": memento.get("essay"),
                    "price_amount": memento.get("price_amount"),
                    "price_currency": memento.get("price_currency", "JPY"),
                    "kind_data": memento["kind_data"],
                    "source_ref": f"mock:{journey['slug']}:{memento_id}",
                    "authored_fields": [],
                },
            )
            for photo in memento.get("photos", []):
                post("/api/admin/photos", {**photo, "memento_id": memento_id})


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
