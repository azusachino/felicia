#!/usr/bin/env python3
"""Run Felicia's local API workflow for SQLite or PostgreSQL."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
API_BINARY = Path(tempfile.gettempdir()) / f"felicia-api-{os.getpid()}"
# Shared with scripts/admin.py so authoring and serving read one journal. Under
# .felicia/ because the authored journal is what ADR-0025 keeps on the machine,
# and the previous default put it at the repo root where it was committable.
DEFAULT_DATABASE = ROOT / ".felicia" / "felicia.sqlite"


def run(command: list[str], *, env: dict[str, str] | None = None, cwd: Path = ROOT) -> None:
    subprocess.run(command, cwd=cwd, env=env, check=True)


def request_ready(url: str) -> bool:
    request = urllib.request.Request(
        f"{url}/api/admin/journals",
        data=b'{"id":"0190cbde-f300-7000-8000-000000000000"}',
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=2):
            return True
    except (OSError, urllib.error.URLError):
        return False


def wait_ready(url: str) -> None:
    for _ in range(30):
        if request_ready(url):
            return
        time.sleep(1)
    raise RuntimeError(f"API did not become ready: {url}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--driver", choices=("sqlite",), default="sqlite")
    parser.add_argument("--web", action="store_true", help="start the public site alongside the API")
    return parser.parse_args()


def run_sqlite() -> None:
    environment = os.environ.copy()
    database = Path(environment.get("DATABASE_PATH") or DEFAULT_DATABASE)
    database.parent.mkdir(parents=True, exist_ok=True)
    environment.update(
        {
            "DATABASE_DRIVER": "sqlite",
            "DATABASE_PATH": str(database),
            "CACHE_ADDR": environment.get("CACHE_ADDR", ""),
        }
    )
    os.execvpe("go", ["go", "run", "./apps/felicia-server/cmd/api"], environment)


def main() -> int:
    parse_args()
    try:
        run_sqlite()
    except (OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"dev workflow failed: {exc}", file=sys.stderr)
        return 1
    finally:
        if API_BINARY.exists():
            API_BINARY.unlink()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
