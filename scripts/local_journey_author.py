"""Interactive authoring over the local journey workspace JSON files."""

from __future__ import annotations

import uuid
from argparse import Namespace

try:
    from .local_journey_common import NAMESPACE, read_json, write_json
except ImportError:
    from local_journey_common import NAMESPACE, read_json, write_json


def ask(prompt: str, default: str = "") -> str:
    suffix = f" [{default}]" if default else ""
    answer = input(f"{prompt}{suffix}: ").strip()
    return answer or default


def parse_coord(value: str) -> list[float] | None:
    if not value:
        return None
    try:
        longitude, latitude = (float(part.strip()) for part in value.split(",", 1))
    except (ValueError, StopIteration):
        raise SystemExit("coordinate must be longitude,latitude")
    if not -180 <= longitude <= 180 or not -90 <= latitude <= 90:
        raise SystemExit("coordinate is out of range")
    return [longitude, latitude]


def parse_price(value: str) -> int | None:
    if not value:
        return None
    try:
        return int(value)
    except ValueError as error:
        raise SystemExit("price amount must be an integer") from error


def interactive_author(args: Namespace) -> None:
    """Let a person curate generated evidence before it becomes authored data."""
    workspace = args.workspace.resolve()
    journey = read_json(workspace / "journey.json")
    stop_data = read_json(workspace / "stops.json")
    memento_data = read_json(workspace / "mementos.json")

    print("\nJourney metadata (press Enter to keep the default)")
    for field in ("title", "place", "country", "region", "date_start", "date_end"):
        journey[field] = ask(field, str(journey.get(field, "")))

    print("\nStop curation: keep the places that should anchor the journey.")
    curated_stops = []
    for stop in stop_data.get("stops", []):
        key = stop["candidate_key"]
        label = stop.get("label") or key
        keep = ask(f"Keep {key} ({label})? y/N", "y" if stop.get("selected", True) else "n").lower()
        stop["selected"] = keep in {"y", "yes"}
        if stop["selected"]:
            stop["label"] = ask("  label", label)
            curated_stops.append(stop)

    next_manual = len(stop_data.get("stops", [])) + 1
    while ask("Add a manual stop? y/N", "n").lower() in {"y", "yes"}:
        key = f"manual-{next_manual:03d}"
        stop = {
            "candidate_key": key,
            "selected": True,
            "label": ask("  label"),
            "coord": parse_coord(ask("  coordinate longitude,latitude")),
            "arrive": ask("  arrive RFC3339"),
            "depart": ask("  depart RFC3339"),
            "confidence": 1.0,
            "review_note": "Manually selected by author.",
        }
        stop_data.setdefault("stops", []).append(stop)
        curated_stops.append(stop)
        next_manual += 1

    by_stop = {stop["candidate_key"] for stop in curated_stops}
    mementos = [m for m in memento_data.get("mementos", []) if m.get("stop_key") in by_stop]
    for memento in mementos:
        print(f"\nEdit memento {memento.get('id')}")
        memento["title"] = ask("  title", memento.get("title", ""))
        memento["kind"] = ask("  kind", memento.get("kind", "goods"))
        memento["vendor"] = ask("  vendor", memento.get("vendor", ""))
        memento["essay"] = ask("  essay", memento.get("essay", ""))
        memento["price_amount"] = parse_price(ask("  price amount", str(memento.get("price_amount", "") or "")))
        memento["price_currency"] = ask("  price currency", memento.get("price_currency", "JPY"))
        memento["authored_fields"] = ["title", "kind", "vendor", "essay", "price_amount", "price_currency"]
        memento["state"] = ask("  state (draft/published)", "published")
        media_default = ",".join(item.get("path", "") for item in memento.get("media", []))
        media_paths = ask("  media paths (comma-separated)", media_default)
        memento["media"] = [{"path": path.strip(), "caption": ""} for path in media_paths.split(",") if path.strip()]

    for stop in curated_stops:
        while ask(f"Add a memento at {stop['label']}? y/N", "n").lower() in {"y", "yes"}:
            index = len(mementos) + 1
            mementos.append(
                {
                    "id": str(uuid.uuid5(NAMESPACE, f"{args.journey}:memento:{index}")),
                    "stop_key": stop["candidate_key"],
                    "seq": index,
                    "kind": ask("  kind", "goods"),
                    "occurred_at": ask("  occurred_at RFC3339", stop.get("arrive", "")),
                    "occurred_tz": ask("  timezone", "UTC"),
                    "title": ask("  title"),
                    "place": stop["label"],
                    "geom": stop.get("coord"),
                    "state": ask("  state (draft/published)", "published"),
                    "kind_data": {},
                    "vendor": ask("  vendor"),
                    "essay": ask("  essay"),
                    "price_amount": parse_price(ask("  price amount")),
                    "price_currency": ask("  price currency", "JPY"),
                    "authored_fields": ["title", "kind", "vendor", "essay", "price_amount", "price_currency"],
                    "media": [
                        {"path": path.strip(), "caption": ""}
                        for path in ask("  media paths (comma-separated)").split(",")
                        if path.strip()
                    ],
                }
            )
    write_json(workspace / "journey.json", journey)
    write_json(workspace / "stops.json", stop_data)
    write_json(workspace / "mementos.json", {"schema": "felicia.mementos.v1", "mementos": mementos})
    print(f"authoring saved: stops={len(curated_stops)}, mementos={len(mementos)}")
