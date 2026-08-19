#!/usr/bin/env python3
"""Build and serve Felicia behind the local compose stack and a quick tunnel."""

from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent


def compose_command() -> list[str]:
    configured = os.getenv("COMPOSE", "")
    if configured:
        return shlex.split(configured)
    if shutil.which("podman-compose"):
        return ["podman-compose"]
    if shutil.which("docker"):
        return ["docker", "compose"]
    raise RuntimeError("no podman-compose or docker compose command found")


def run(command: list[str], *, env: dict[str, str] | None = None, cwd: Path = ROOT) -> None:
    subprocess.run(command, cwd=cwd, env=env, check=True)


def compose(compose_base: list[str], *args: str) -> list[str]:
    return [*compose_base, "-f", "ops/compose.yaml", *args]


def wait_for_db(command: list[str]) -> None:
    for _ in range(30):
        result = subprocess.run(
            compose(command, "exec", "-T", "db", "pg_isready", "-U", "postgres", "-d", "felicia"),
            cwd=ROOT,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        if result.returncode == 0:
            return
        time.sleep(1)
    raise RuntimeError("database did not become ready")


def wait_for_api() -> None:
    port = os.getenv("PORT", "8080")
    request = urllib.request.Request(
        f"http://localhost:{port}/api/admin/journals",
        data=b'{"id":"0190cbde-f300-7000-8000-000000000000"}',
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    for _ in range(60):
        try:
            with urllib.request.urlopen(request, timeout=2):
                return
        except (OSError, urllib.error.URLError):
            time.sleep(1)
    raise RuntimeError("API did not become ready")


def tunnel_url(command: list[str]) -> str:
    for _ in range(30):
        result = subprocess.run(
            compose(command, "logs", "cloudflared"),
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
        for line in result.stdout.splitlines():
            for word in line.split():
                if word.startswith("https://") and word.endswith(".trycloudflare.com"):
                    return word
        time.sleep(1)
    return ""


def main() -> int:
    try:
        compose_base = compose_command()
        environment = os.environ.copy()
        dsn = environment.get("DATABASE_DSN", "postgres://postgres:password@localhost:5432/felicia?sslmode=disable")

        run(["bun", "run", "build"], cwd=ROOT / "apps" / "web-public")
        run(compose(compose_base, "up", "-d", "db", "cache"))
        wait_for_db(compose_base)
        run(["make", "migrate"], env=environment | {"DATABASE_DSN": dsn})
        run(compose(compose_base, "up", "-d", "--build", "api"))
        wait_for_api()
        run(
            [sys.executable, "scripts/seed.py"],
            env=environment | {"DATABASE_DSN": dsn, "SEED_API_BASE": "http://localhost:8080"},
        )
        run(compose(compose_base, "up", "-d", "web", "cloudflared"))
        url = tunnel_url(compose_base)
        print(f"share URL: {url or 'not ready; inspect compose logs cloudflared'}")
        print("local preview: http://localhost:8081")
        print("stop sharing: make share-down")
    except (OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"share workflow failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
