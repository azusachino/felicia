"""Shared seams for the local journey workflow commands."""

from __future__ import annotations

import hashlib
import json
import subprocess
import uuid
import xml.etree.ElementTree as ET
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
CLI = ROOT / "bin" / "felicia-cli"
NAMESPACE = uuid.UUID("0190cbde-f300-7000-8000-999999999999")

# The workspace a bare `make journey-local` writes to when no --workspace is
# given: one directory per derived slug under this root, never a single
# shared path (see derive_journey_identity -- issue #72).
DEFAULT_WORKSPACE_ROOT = ROOT / ".felicia" / "workspaces"


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def read_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text())
    except FileNotFoundError as exc:
        raise SystemExit(f"missing {path}; run preprocess first") from exc
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid JSON in {path}: {exc}") from exc


def run(command: list[str]) -> None:
    print("$", " ".join(command))
    subprocess.run(command, cwd=ROOT, check=True)


def ensure_cli() -> None:
    # An existing binary can embed an older SQLite schema than the current server.
    CLI.parent.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(CLI), "./apps/felicia-cli/cmd/felicia"])


def as_coord(value: Any) -> list[float] | None:
    if isinstance(value, list) and len(value) == 2:
        return [float(value[0]), float(value[1])]
    return None


def candidate_key(stop: dict[str, Any]) -> str:
    identity = stop.get("identity", {})
    return str(identity.get("key") or stop.get("id") or "")


def derive_journey_identity(gpx: Path) -> tuple[str, str]:
    """Derive the default identity from normalized track points, not GPX bytes.

    Exporter metadata and XML formatting do not affect identity. Track
    coordinates are quantized to ~11m and timestamps to minutes, so small
    export noise is ignored while route shape and timing keep trips distinct.
    """
    root = ET.parse(gpx).getroot()
    segments: list[list[tuple[int, int, str]]] = []
    for element in root.iter():
        if element.tag.rsplit("}", 1)[-1] != "trkseg":
            continue
        segment = []
        for point in element:
            if point.tag.rsplit("}", 1)[-1] != "trkpt":
                continue
            lat, lon = float(point.attrib["lat"]), float(point.attrib["lon"])
            timestamp = next(
                (child.text for child in point if child.tag.rsplit("}", 1)[-1] == "time"),
                "",
            )
            if timestamp:
                parsed = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                timestamp = parsed.astimezone(timezone.utc).replace(second=0, microsecond=0).isoformat()
            segment.append((round(lat * 10_000), round(lon * 10_000), timestamp))
        if segment:
            segments.append(segment)
    if not segments:
        raise ValueError(f"{gpx} contains no GPX track points")

    canonical = json.dumps(segments, separators=(",", ":"), ensure_ascii=True)
    digest = hashlib.sha256(canonical.encode()).hexdigest()
    journey_id = str(uuid.uuid5(NAMESPACE, f"journey:{digest}"))
    slug = f"journey-{digest[:12]}"
    return journey_id, slug


def safe_media_path(workspace: Path, source: str) -> Path:
    path = Path(source)
    if path.is_absolute():
        resolved = path.resolve()
    else:
        resolved = (workspace / path).resolve()
        if not resolved.exists():
            resolved = (ROOT / path).resolve()
    if not resolved.is_file():
        raise SystemExit(f"media file does not exist: {source}")
    return resolved
