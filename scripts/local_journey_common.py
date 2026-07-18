"""Shared seams for the local journey workflow commands."""

from __future__ import annotations

import json
import subprocess
import uuid
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
CLI = ROOT / "bin" / "felicia-cli"
NAMESPACE = uuid.UUID("0190cbde-f300-7000-8000-999999999999")


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
