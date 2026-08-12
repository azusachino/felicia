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
DEFAULT_DATABASE = ROOT / ".felicia" / "local.sqlite"


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
    parser.add_argument("--driver", choices=("sqlite", "postgres"), required=True)
    parser.add_argument("--web", action="store_true", help="seed PostgreSQL and start the web app")
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
    os.execvpe("go", ["go", "run", "./server/cmd/api"], environment)


def run_postgres(start_web: bool) -> None:
    environment = os.environ.copy()
    dsn = environment.get("DATABASE_DSN", "")
    if not dsn:
        raise RuntimeError("DATABASE_DSN is required for the PostgreSQL development workflow")

    run(["make", "db-up"])
    run(["make", "migrate"], env=environment)
    run(["go", "build", "-o", str(API_BINARY), "./server/cmd/api"])

    api_environment = environment | {
        "DATABASE_DRIVER": "postgres",
        "DATABASE_DSN": dsn,
        "CACHE_ADDR": environment.get("CACHE_ADDR", "localhost:6379"),
    }
    api = subprocess.Popen([str(API_BINARY)], cwd=ROOT, env=api_environment)
    try:
        port = environment.get("PORT", "8080")
        base_url = f"http://localhost:{port}"
        wait_ready(base_url)
        if not start_web:
            api.wait()
            return

        seed_environment = environment | {"SEED_API_BASE": base_url}
        run([sys.executable, "scripts/seed.py"], env=seed_environment)
        web_dir = ROOT / "apps" / "web-public"
        if not (web_dir / "node_modules").exists():
            run(["bun", "install"], cwd=web_dir)
        run(["bun", "run", "dev"], cwd=web_dir)
    finally:
        api.terminate()
        try:
            api.wait(timeout=5)
        except subprocess.TimeoutExpired:
            api.kill()
            api.wait()
        API_BINARY.unlink(missing_ok=True)


def main() -> int:
    arguments = parse_args()
    try:
        if arguments.driver == "sqlite":
            run_sqlite()
        else:
            run_postgres(arguments.web)
    except (OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"dev workflow failed: {exc}", file=sys.stderr)
        return 1
    finally:
        if API_BINARY.exists():
            API_BINARY.unlink()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
