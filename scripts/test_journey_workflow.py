#!/usr/bin/env python3
"""Exercise the complete authoring-to-public journey workflow over HTTP."""

import argparse
import json
import os
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from contextlib import contextmanager


BASE_URL = os.getenv("API_BASE", "http://localhost:8080")
JOURNAL_ID = "0190cbde-f300-7000-8000-d11111111111"
JOURNEY_ID = "0190cbde-f300-7000-8000-d22222222222"
MEMENTO_ID = "0190cbde-f300-7000-8000-d33333333333"
PHOTO_ID = "0190cbde-f300-7000-8000-d44444444444"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--start-server",
        action="store_true",
        help="build and run the API against a disposable SQLite database",
    )
    parser.add_argument("--port", type=int, default=18080)
    return parser.parse_args()


def request(path: str, method: str = "GET", payload: dict | None = None):
    body = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        f"{BASE_URL}{path}",
        data=body,
        headers={"Content-Type": "application/json"},
        method=method,
    )
    try:
        with urllib.request.urlopen(req) as response:
            raw = response.read().decode()
            return response.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode()
        try:
            detail = json.loads(detail)
        except json.JSONDecodeError:
            pass
        raise AssertionError(f"{method} {path} failed ({exc.code}): {detail}") from exc


def post(path: str, payload: dict):
    status, body = request(path, "POST", payload)
    assert status == 200, f"expected 200 from {path}, got {status}: {body}"
    return body


def run_workflow() -> None:
    post(f"/api/admin/journals", {"id": JOURNAL_ID})
    post(
        "/api/admin/journeys",
        {
            "id": JOURNEY_ID,
            "journal_id": JOURNAL_ID,
            "slug": "workflow-journey",
            "title": "Workflow journey",
            "place": "Tokyo",
            "country": "JPN",
            "date_start": "2026-03-20",
            "date_end": "2026-03-22",
            "gps_route": [[[139.7, 35.6], [139.8, 35.7]]],
            "authored_fields": [],
        },
    )

    draft = {
        "id": MEMENTO_ID,
        "journey_id": JOURNEY_ID,
        "kind": "live",
        "seq": 1,
        "state": "draft",
        "kind_data": {"artist": "羊文学"},
    }
    post("/api/admin/mementos", draft)
    status, saved = request(f"/api/admin/mementos/{MEMENTO_ID}")
    assert status == 200 and saved["state"] == "draft" and saved["revision"] == 1

    published = {
        **draft,
        "state": "published",
        "expected_revision": 1,
        "occurred_at": "2026-03-21T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "geom": {"type": "Point", "coordinates": [139.75, 35.69]},
        "title": "Live show",
        "place": "Tokyo",
        "kind_data": {
            "artist": "羊文学",
            "venue": {"name": "日本武道館", "coords": [139.75, 35.69]},
            "date": "2026-03-21T18:30:00+09:00",
        },
    }
    post("/api/admin/mementos", published)

    post(
        "/api/admin/photos",
        {
            "id": PHOTO_ID,
            "memento_id": MEMENTO_ID,
            "object_key": "workflow/live.jpg",
            "content_hash": "workflow-photo-hash",
            "seq": 1,
        },
    )
    status, mementos = request(f"/api/admin/journeys/{JOURNEY_ID}/mementos")
    assert status == 200 and len(mementos) == 1 and mementos[0]["state"] == "published"
    status, public = request(f"/api/v1/journeys/{JOURNEY_ID}/mementos")
    assert status == 200 and len(public) == 1 and len(public[0]["photos"]) == 1
    assert public[0]["title"] == "Live show"
    print("full journey workflow passed")


@contextmanager
def disposable_server(port: int):
    global BASE_URL
    with tempfile.TemporaryDirectory(prefix="felicia-workflow-") as temp_dir:
        api_bin = os.path.join(temp_dir, "felicia-api")
        database_path = os.path.join(temp_dir, "felicia.sqlite")
        subprocess.run(["go", "build", "-o", api_bin, "./cmd/api"], check=True)
        environment = {
            **os.environ,
            "DATABASE_DRIVER": "sqlite",
            "DATABASE_PATH": database_path,
            "CACHE_ADDR": "",
            "PORT": str(port),
        }
        server = subprocess.Popen(
            [api_bin],
            env=environment,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            BASE_URL = f"http://localhost:{port}"
            for _ in range(30):
                try:
                    request(
                        "/api/admin/journals",
                        "POST",
                        {"id": JOURNAL_ID},
                    )
                    break
                except (OSError, urllib.error.URLError):
                    time.sleep(1)
            else:
                stderr = server.stderr.read() if server.stderr else ""
                raise RuntimeError(f"API did not become ready:\n{stderr}")
            yield
        finally:
            server.terminate()
            try:
                server.wait(timeout=5)
            except subprocess.TimeoutExpired:
                server.kill()
                server.wait()


if __name__ == "__main__":
    try:
        arguments = parse_args()
        if arguments.start_server:
            with disposable_server(arguments.port):
                run_workflow()
        else:
            run_workflow()
    except (AssertionError, OSError, RuntimeError, urllib.error.URLError) as exc:
        print(f"workflow failed: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
