"""Shared seams for the local journey workflow commands."""

from __future__ import annotations

import hashlib
import json
import subprocess
import uuid
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
    if not CLI.exists():
        run(["make", "cli-build"])


def as_coord(value: Any) -> list[float] | None:
    if isinstance(value, list) and len(value) == 2:
        return [float(value[0]), float(value[1])]
    return None


def candidate_key(stop: dict[str, Any]) -> str:
    identity = stop.get("identity", {})
    return str(identity.get("key") or stop.get("id") or "")


def derive_journey_identity(gpx: Path) -> tuple[str, str]:
    """Derive a journey id and slug from the GPX track's own bytes.

    A trip that is re-run (same GPX) lands on the same id and slug every
    time -- idempotent, no duplicate journey. A different trip (different
    GPX content) lands on a different id and slug -- collision-free without
    the author having to invent an identifier (see issue #72: the workspace,
    journey id, journal-scoped slug, and every derived memento id used to be
    hard-coded to one fixed UUID, so a second trip silently overwrote the
    first). This is only the *default*; --journey/--slug/--workspace still
    let an author name a trip explicitly.
    """
    digest = hashlib.sha256(gpx.read_bytes()).hexdigest()
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
