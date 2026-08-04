#!/usr/bin/env python3
"""Closed-loop admin-GUI E2E pass (ADMIN-01.8).

Drives the real `apps/web-admin` GUI in a real browser (Playwright/chromium)
against the disposable API server: import -> review candidate -> author ->
publish -> compile -> assert the compiled artifact contains the authored
essay.

Stdlib + this repo's existing scripts only (uv-managed environment, no new
Python dependencies):
  - reuses `disposable_server`/`find_free_port`/`workflow_ids` from
    scripts/test_journey_workflow.py for the API server lifecycle and ID
    generation;
  - reuses scripts/local_journey_{common,package}.py to seed a journey the
    same way the local-authoring CLI path does (see
    tests/test_local_journey_mixed_state.py) — a journey package built and
    imported through the compiled felicia-cli binary, not a raw admin API
    POST, so this harness exercises the same path a real author's local
    workspace would.

The seeded journey carries zero local mementos/stops on purpose: the intake
candidate this test reviews comes from scripts/mock_upstream.py's fixed
Dawarich visit fixture, fetched over HTTP by the live API's intake planner
(ADMIN-01.3a) exactly as a real Dawarich-backed deployment would. Both of
mock_upstream's non-declined visits ("明治神宮" confirmed, "道頓堀"
suggested) survive the provider's parseVisits, so the Playwright spec reads
back whichever stop candidate it promotes rather than assuming a specific
one.

Run: make test-admin-e2e
 or: uv run python scripts/e2e_admin_gui.py
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from argparse import Namespace
from contextlib import contextmanager
from pathlib import Path

from local_journey_common import CLI, ensure_cli, run, write_json
from local_journey_package import build_package
from test_journey_workflow import disposable_server, find_free_port, workflow_ids

ROOT = Path(__file__).resolve().parent.parent
WEB_ADMIN_DIR = ROOT / "apps" / "web-admin"
MOCK_UPSTREAM_SCRIPT = ROOT / "scripts" / "mock_upstream.py"

MOCK_API_KEY = "mock-key"

# Sentinel strings the Playwright spec (apps/web-admin/e2e/authoring.spec.ts)
# authors into the memento. Kept identical in both places on purpose: this
# script's filesystem-side assertion against the compiled static artifact
# looks for the *same* strings, so a drift between the two would fail loudly
# rather than the check silently no-op'ing.
JOURNEY_TITLE = "Admin GUI E2E Journey"
MEMENTO_TITLE = "Admin GUI E2E Memento"
ESSAY_SENTINEL = "Felicia admin GUI E2E authored essay -- sentinel 9f3c2b1a"
GOODS_NAME = "E2E Souvenir"

# ADMIN-02 M2: site identity constants, kept identical to the constants of
# the same name in apps/web-admin/e2e/authoring.spec.ts for the same reason
# as the constants above -- this script's filesystem-side check on the
# compiled api/v1/site.json looks for these exact values.
SITE_TITLE = "Admin GUI E2E Site"
SITE_DESIGN = "v4"
SITE_ACCENT = "#336699"

# A minimal two-point timestamped track. The intake planner only needs it to
# satisfy the local-authoring package schema (route.gpx is required
# alongside journey.json/stops.json/mementos.json) — the actual stop
# candidate this test reviews comes from the mock Dawarich *visits* fixture,
# not from deriving visits out of this route (see runtime/intake/planner.go:
# supplied visits take precedence over route-derived ones).
MINIMAL_GPX = """<?xml version="1.0"?>
<gpx version="1.1" creator="felicia admin-gui e2e" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      <trkpt lat="35.6812" lon="139.7671"><time>2026-03-20T00:00:00Z</time></trkpt>
      <trkpt lat="35.6895" lon="139.7003"><time>2026-03-20T01:00:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>
"""


def _wait_ready(
    process: subprocess.Popen, url: str, timeout_s: float, what: str
) -> None:
    deadline = time.monotonic() + timeout_s
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        if process.poll() is not None:
            stderr = process.stderr.read() if process.stderr else ""
            raise RuntimeError(
                f"{what} exited before readiness ({process.returncode}):\n{stderr}"
            )
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                if response.status < 500:
                    return
        except (OSError, urllib.error.URLError) as exc:
            last_error = exc
        time.sleep(0.3)
    raise RuntimeError(
        f"{what} did not become ready within {timeout_s}s: {last_error!r}"
    )


@contextmanager
def mock_upstream(port: int):
    """Runs scripts/mock_upstream.py (fixture-backed Dawarich + Immich) so the
    live API's intake planner has something to fetch from."""
    process = subprocess.Popen(
        [sys.executable, str(MOCK_UPSTREAM_SCRIPT)],
        cwd=ROOT,
        env={**os.environ, "MOCK_PORT": str(port), "MOCK_API_KEY": MOCK_API_KEY},
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
        text=True,
    )
    try:
        _wait_ready(
            process,
            f"http://127.0.0.1:{port}/api/v1/tracks?page=1&api_key={MOCK_API_KEY}",
            20,
            "mock upstream",
        )
        yield
    finally:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


@contextmanager
def vite_dev_server(port: int, api_base: str):
    """Runs `bun run dev` for apps/web-admin with Vite's /api proxy pointed at
    the disposable API server (VITE_API_PROXY, see vite.config.ts): the GUI
    fetches same-origin exactly like the compiled artifact does, so no CORS
    wiring is needed on the server. The Python side owns this process's
    lifecycle entirely — playwright.config.ts intentionally has no
    `webServer` block."""
    env = {**os.environ, "VITE_API_PROXY": api_base}
    process = subprocess.Popen(
        [
            "bun",
            "run",
            "dev",
            "--",
            "--port",
            str(port),
            "--strictPort",
            "--host",
            "127.0.0.1",
        ],
        cwd=WEB_ADMIN_DIR,
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
        text=True,
    )
    try:
        base_url = f"http://127.0.0.1:{port}"
        _wait_ready(process, base_url + "/", 60, "web-admin dev server")
        yield base_url
    finally:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


def seed_journey(db_path: Path, media_root: Path, workspace_root: Path) -> str:
    """Builds a minimal local-authoring journey package and imports it through
    the compiled felicia-cli binary (package validate + import --apply) —
    the same pipeline tests/test_local_journey_mixed_state.py drives, not a
    raw admin API POST.

    The journey carries zero mementos/stops on purpose: the memento this
    test reviews is meant to come from the intake planner (mock Dawarich),
    not from the package itself. Returns the seeded journey's UUID.
    """
    ensure_cli()
    ids = workflow_ids()
    workspace = workspace_root / "journey"
    write_json(
        workspace / "journey.json",
        {
            "schema": "felicia.local.journey.v1",
            "id": ids.journey,
            "journal_id": ids.journal,
            "slug": ids.slug,
            "title": JOURNEY_TITLE,
            "place": "Tokyo",
            "country": "JPN",
            "date_start": "2026-03-20",
            "date_end": "2026-03-20",
            "source_ref": "route.gpx",
        },
    )
    write_json(
        workspace / "stops.json", {"schema": "felicia.local.stops.v1", "stops": []}
    )
    write_json(
        workspace / "mementos.json",
        {"schema": "felicia.local.mementos.v1", "mementos": []},
    )
    (workspace / "route.gpx").write_text(MINIMAL_GPX, encoding="utf-8")

    package = build_package(Namespace(workspace=workspace))
    run([str(CLI), "package", "validate", str(package)])
    run(
        [
            str(CLI),
            "import",
            "--db",
            str(db_path),
            "--media-root",
            str(media_root),
            "--apply",
            str(package),
        ]
    )
    return ids.journey


def assert_preview_server(preview_port: int) -> None:
    """Confirms the server's built-in preview listener (server/api/preview.go)
    is actually serving the compiled artifact on site.preview_port, not just
    that the compile step wrote files to disk. The SPA dist is absent in
    this harness (no `make web-build` here), so this only asserts the
    artifact side of the union file server: api/v1/manifest.json, written by
    the compile the GUI step just triggered.
    """
    url = f"http://127.0.0.1:{preview_port}/api/v1/manifest.json"
    with urllib.request.urlopen(url, timeout=5) as response:
        if response.status != 200:
            raise AssertionError(
                f"preview server at {url} returned status {response.status}, expected 200"
            )
        payload = json.loads(response.read().decode("utf-8"))
    if not isinstance(payload, dict):
        raise AssertionError(
            f"preview server at {url} did not return a JSON object: {payload!r}"
        )
    print("preview server serves the compiled manifest -- preview-port check passed")


def assert_compiled_artifact(out_dir: Path, journey_id: str) -> None:
    """Defense-in-depth: re-reads the artifact the GUI's compile step wrote
    directly off disk (mirrors run_static_parity_check's read_static_file in
    scripts/test_journey_workflow.py) rather than trusting only the browser
    assertions in the Playwright spec.
    """
    mementos_path = out_dir / "api" / "v1" / "journeys" / journey_id / "mementos.json"
    if not mementos_path.is_file():
        raise AssertionError(
            f"compiled artifact is missing the journey's mementos file: {mementos_path}"
        )
    mementos = json.loads(mementos_path.read_text(encoding="utf-8"))
    matching = [m for m in mementos if m.get("essay") == ESSAY_SENTINEL]
    if not matching:
        raise AssertionError(
            f"compiled artifact at {mementos_path} does not contain the authored essay sentinel {ESSAY_SENTINEL!r}: {mementos!r}"
        )
    if matching[0].get("title") != MEMENTO_TITLE:
        raise AssertionError(
            f"authored memento title mismatch in compiled artifact: {matching[0]!r}"
        )
    print("compiled artifact contains the authored essay -- filesystem check passed")


def assert_site_settings(out_dir: Path) -> None:
    """Re-reads the compiled api/v1/site.json directly off disk (ADMIN-02 M2):
    the Playwright spec already asserts the live /api/v1/site endpoint after
    saving and rebuilding, so this is the same live/static-parity
    double-check assert_compiled_artifact does for the journey/memento data.
    """
    site_path = out_dir / "api" / "v1" / "site.json"
    if not site_path.is_file():
        raise AssertionError(f"compiled artifact is missing site.json: {site_path}")
    site = json.loads(site_path.read_text(encoding="utf-8"))
    if site.get("title") != SITE_TITLE:
        raise AssertionError(f"compiled site.json title mismatch: {site!r}")
    if site.get("design") != SITE_DESIGN:
        raise AssertionError(f"compiled site.json design mismatch: {site!r}")
    if site.get("accent") != SITE_ACCENT:
        raise AssertionError(f"compiled site.json accent mismatch: {site!r}")
    print("compiled artifact reflects the saved site identity -- filesystem check passed")


def run_admin_gui_e2e() -> None:
    with tempfile.TemporaryDirectory(prefix="felicia-admin-gui-e2e-") as root_dir:
        root = Path(root_dir)
        db_path = root / "felicia.db"
        media_root = root / "media"
        out_dir = root / "site"

        journey_id = seed_journey(db_path, media_root, root / "workspace")

        mock_port = find_free_port()
        api_port = find_free_port()
        gui_port = find_free_port()
        preview_port = find_free_port()
        mock_base = f"http://127.0.0.1:{mock_port}"
        api_base = f"http://127.0.0.1:{api_port}"

        extra_env = {
            "DAWARICH_URL": mock_base,
            "DAWARICH_API_KEY": MOCK_API_KEY,
            "IMMICH_URL": mock_base,
            "IMMICH_API_KEY": MOCK_API_KEY,
            "MEDIA_ROOT": str(media_root),
            # Playwright drives the GUI at machine speed; the default
            # 1 req/s (burst 20) limiter is tuned for humans and would 429
            # mid-flow, so the disposable server gets a test-only allowance.
            "RATE_PER_SECOND": "50",
            "RATE_BURST": "200",
            # The server now compiles into its configured site output (the
            # GUI's Build button omits out_dir), so out_dir is set here
            # instead of passed in the compile request — kept equal to the
            # `out_dir` this script already reads back off disk below.
            "SITE_OUT_DIR": str(out_dir),
            "SITE_PREVIEW_PORT": str(preview_port),
        }

        with mock_upstream(mock_port):
            with disposable_server(
                api_port, "sqlite", str(db_path), "", "", extra_env=extra_env
            ):
                with vite_dev_server(gui_port, api_base) as gui_base:
                    env = {
                        **os.environ,
                        "E2E_BASE_URL": gui_base,
                        "E2E_API_BASE": api_base,
                        "E2E_JOURNEY_ID": journey_id,
                        "E2E_OUT_DIR": str(out_dir),
                    }
                    result = subprocess.run(
                        ["bunx", "playwright", "test"], cwd=WEB_ADMIN_DIR, env=env
                    )
                    if result.returncode != 0:
                        raise RuntimeError(
                            f"playwright test failed with exit code {result.returncode}"
                        )

                # The Playwright spec's own "Build site" click already
                # compiled into SITE_OUT_DIR; the disposable API server (and
                # therefore its preview listener) is still up at this point,
                # so this checks the live preview port before the server
                # process is torn down.
                assert_preview_server(preview_port)

        assert_compiled_artifact(out_dir, journey_id)
        assert_site_settings(out_dir)

    print("admin GUI E2E passed")


if __name__ == "__main__":
    try:
        run_admin_gui_e2e()
    except (
        AssertionError,
        OSError,
        RuntimeError,
        urllib.error.URLError,
        subprocess.CalledProcessError,
    ) as exc:
        print(f"admin GUI E2E failed: {exc!r}", file=sys.stderr)
        raise SystemExit(1) from exc
