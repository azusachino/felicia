#!/usr/bin/env python3
"""Start the local admin GUI: the authoring API plus the felicia-admin dev server.

This is the one documented entry point for the authoring surface. It exists to
keep ADR-0025's constraint honest in the tooling rather than only in prose: the
admin is a local process over a local database, and nothing here can publish.

Two deliberate choices:

  * The local stack binds 0.0.0.0 by default so the author can reach it over
    Tailscale. Override `FELICIA_HOST=127.0.0.1` for a host-only session.
  * The database defaults under `.felicia/` (gitignored) instead of the repo
    root. The authored journal is exactly the thing ADR-0025 says never leaves
    the machine, so its default location must not be a committable path. It is
    the same default `scripts/dev.py` uses, so authoring here and then serving
    the public reader with `make dev` reads one journal rather than two.

The GUI talks to the API through Vite's `/api` proxy (VITE_API_PROXY), so the
browser sees one origin and the server needs no CORS wiring — the same shape the
compiled artifact has.
"""

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
WEB_ADMIN = ROOT / "apps" / "felicia-admin"
API_BINARY = Path(tempfile.gettempdir()) / f"felicia-admin-api-{os.getpid()}"
DEFAULT_DATABASE = ROOT / ".felicia" / "felicia.sqlite"
LOCALHOST = "127.0.0.1"


def run(command: list[str], *, env: dict[str, str] | None = None, cwd: Path = ROOT) -> None:
    subprocess.run(command, cwd=cwd, env=env, check=True)


def api_ready(base_url: str) -> bool:
    """The admin journeys list is a plain GET and needs no fixture state."""
    try:
        with urllib.request.urlopen(f"{base_url}/api/admin/journeys", timeout=2):
            return True
    except (OSError, urllib.error.URLError):
        return False


def wait_ready(process: subprocess.Popen, base_url: str, timeout_s: int = 60) -> None:
    deadline = time.monotonic() + timeout_s
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"admin API exited early with code {process.returncode}")
        if api_ready(base_url):
            return
        time.sleep(0.5)
    raise RuntimeError(f"admin API did not become ready within {timeout_s}s: {base_url}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Start the local admin GUI")
    parser.add_argument("--host", default=os.environ.get("FELICIA_HOST", "0.0.0.0"))
    parser.add_argument("--api-port", default=os.environ.get("PORT", "8080"))
    parser.add_argument("--gui-port", default=os.environ.get("ADMIN_GUI_PORT", "5174"))
    parser.add_argument(
        "--db",
        default=os.environ.get("DATABASE_PATH", str(DEFAULT_DATABASE)),
        help=f"SQLite path for authored content (default: {DEFAULT_DATABASE})",
    )
    return parser.parse_args()


def start_api(arguments: argparse.Namespace) -> subprocess.Popen:
    database = Path(arguments.db)
    database.parent.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(API_BINARY), "./apps/felicia-server/cmd/api"])
    environment = os.environ.copy()
    environment.update(
        {
            "DATABASE_DRIVER": "sqlite",
            "DATABASE_PATH": str(database),
            "FELICIA_HOST": arguments.host,
            "PORT": arguments.api_port,
            # An authoring session is single-user; Valkey is a serving-side
            # cache and requiring it here would add a container to the loop.
            "CACHE_ADDR": environment.get("CACHE_ADDR", ""),
        }
    )
    return subprocess.Popen([str(API_BINARY)], cwd=ROOT, env=environment)


def start_gui(arguments: argparse.Namespace) -> None:
    if not (WEB_ADMIN / "node_modules").exists():
        run(["bun", "install"], cwd=ROOT)
    environment = os.environ.copy()
    environment["VITE_API_PROXY"] = f"http://{LOCALHOST}:{arguments.api_port}"
    run(
        [
            "bun",
            "run",
            "dev",
            "--",
            "--host",
            arguments.host,
            "--port",
            arguments.gui_port,
            "--strictPort",
        ],
        cwd=WEB_ADMIN,
        env=environment,
    )


def main() -> int:
    arguments = parse_args()
    api: subprocess.Popen | None = None
    try:
        api = start_api(arguments)
        base_url = f"http://{LOCALHOST}:{arguments.api_port}"
        wait_ready(api, base_url)
        # flush: this banner is the only place the URLs appear, and stdout is
        # block-buffered whenever the caller redirects it to a file or pipe.
        print(
            f"admin GUI:      http://localhost:{arguments.gui_port}/ (bind: {arguments.host})",
            flush=True,
        )
        print(f"admin API:      {base_url}/api/admin", flush=True)
        print(
            f"site preview:   http://localhost:8081/ (bind: {arguments.host}, "
            "after a Build in Site & Deploy)",
            flush=True,
        )
        print(f"database:       {arguments.db}", flush=True)
        print("Ctrl-C to stop both processes.", flush=True)
        start_gui(arguments)
    except KeyboardInterrupt:
        return 0
    except (OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"admin GUI failed to start: {exc}", file=sys.stderr)
        return 1
    finally:
        if api is not None:
            api.terminate()
            try:
                api.wait(timeout=5)
            except subprocess.TimeoutExpired:
                api.kill()
                api.wait()
        API_BINARY.unlink(missing_ok=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
