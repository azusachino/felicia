#!/usr/bin/env python3
"""Exercise the complete authoring-to-public journey workflow over HTTP."""

import argparse
import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from contextlib import contextmanager, nullcontext
from dataclasses import dataclass
from uuid import NAMESPACE_URL, uuid4, uuid5


BASE_URL = os.getenv("API_BASE", "http://localhost:8080")


@dataclass(frozen=True)
class WorkflowIDs:
    journal: str
    journey: str
    memento: str
    photo: str
    slug: str


def workflow_ids() -> WorkflowIDs:
    run_id = uuid4()
    prefix = f"felicia-workflow:{run_id}"
    return WorkflowIDs(
        journal=str(uuid5(NAMESPACE_URL, f"{prefix}:journal")),
        journey=str(uuid5(NAMESPACE_URL, f"{prefix}:journey")),
        memento=str(uuid5(NAMESPACE_URL, f"{prefix}:memento")),
        photo=str(uuid5(NAMESPACE_URL, f"{prefix}:photo")),
        slug=f"workflow-{run_id.hex[:12]}",
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--start-server",
        action="store_true",
        help="build and run the API against the selected database",
    )
    parser.add_argument("--port", type=int, default=0, help="server port; 0 reserves a free port")
    parser.add_argument("--database-driver", choices=("sqlite", "postgres"), default="sqlite")
    parser.add_argument("--database-path", default="", help="SQLite database path")
    parser.add_argument("--database-dsn", default="", help="PostgreSQL test database DSN")
    parser.add_argument(
        "--postgres-admin-dsn",
        default="",
        help="create and drop a disposable PostgreSQL database using this admin DSN",
    )
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
    if status != 200:
        raise AssertionError(f"expected 200 from {path}, got {status}: {body!r}")
    return body


def expect(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def run_workflow(ids: WorkflowIDs) -> None:
    post("/api/admin/journals", {"id": ids.journal})
    post(
        "/api/admin/journeys",
        {
            "id": ids.journey,
            "journal_id": ids.journal,
            "slug": ids.slug,
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
        "id": ids.memento,
        "journey_id": ids.journey,
        "kind": "live",
        "seq": 1,
        "state": "draft",
        "kind_data": {"artist": "羊文学"},
    }
    post("/api/admin/mementos", draft)
    status, saved = request(f"/api/admin/mementos/{ids.memento}")
    expect(
        status == 200 and saved.get("state") == "draft" and saved.get("revision") == 1,
        f"draft memento mismatch: status={status} body={saved!r}",
    )

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
            "id": ids.photo,
            "memento_id": ids.memento,
            "object_key": "workflow/live.jpg",
            "content_hash": "workflow-photo-hash",
            "seq": 1,
        },
    )
    status, mementos = request(f"/api/admin/journeys/{ids.journey}/mementos")
    expect(
        status == 200 and len(mementos) == 1 and mementos[0].get("state") == "published",
        f"admin memento mismatch: status={status} body={mementos!r}",
    )
    status, public = request(f"/api/v1/journeys/{ids.journey}/mementos")
    expect(
        status == 200 and len(public) == 1 and len(public[0].get("photos", [])) == 1,
        f"public memento mismatch: status={status} body={public!r}",
    )
    expect(public[0].get("title") == "Live show", f"public title mismatch: {public!r}")
    print("full journey workflow passed")


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def create_postgres_database(admin_dsn: str, database_name: str) -> str:
    import psycopg
    from psycopg import sql
    from psycopg.conninfo import make_conninfo

    with psycopg.connect(admin_dsn, autocommit=True) as connection:
        connection.execute(sql.SQL("CREATE DATABASE {} ").format(sql.Identifier(database_name)))
    return make_conninfo(admin_dsn, dbname=database_name)


def drop_postgres_database(admin_dsn: str, database_name: str) -> None:
    import psycopg
    from psycopg import sql

    with psycopg.connect(admin_dsn, autocommit=True) as connection:
        connection.execute(
            sql.SQL("DROP DATABASE IF EXISTS {} WITH (FORCE)").format(sql.Identifier(database_name))
        )


@contextmanager
def disposable_postgres_database(admin_dsn: str):
    database_name = f"felicia_workflow_{uuid4().hex[:16]}"
    database_dsn = create_postgres_database(admin_dsn, database_name)
    try:
        yield database_dsn
    finally:
        drop_postgres_database(admin_dsn, database_name)


@contextmanager
def disposable_server(port: int, driver: str, database_path: str, database_dsn: str, postgres_admin_dsn: str):
    global BASE_URL
    ids = workflow_ids()
    with tempfile.TemporaryDirectory(prefix="felicia-workflow-") as temp_dir:
        api_bin = os.path.join(temp_dir, "felicia-api")
        if driver == "sqlite":
            database_path = database_path or os.path.join(temp_dir, "felicia.db")
        elif not database_dsn and not postgres_admin_dsn:
            raise RuntimeError(
                "--database-dsn, FELICIA_TEST_DATABASE_DSN, or "
                "--postgres-admin-dsn is required for postgres"
            )
        database_context = (
            disposable_postgres_database(postgres_admin_dsn)
            if driver == "postgres" and postgres_admin_dsn
            else nullcontext(database_dsn)
        )
        with database_context as selected_database_dsn:
            subprocess.run(["go", "build", "-o", api_bin, "./server/cmd/api"], check=True)
            if driver == "postgres" and postgres_admin_dsn:
                subprocess.run(
                    ["make", "migrate"],
                    check=True,
                    env={**os.environ, "DATABASE_DSN": selected_database_dsn},
                )
            selected_port = port or find_free_port()
            environment = {
                **os.environ,
                "DATABASE_DRIVER": driver,
                "CACHE_ADDR": "",
                "PORT": str(selected_port),
            }
            if driver == "sqlite":
                environment["DATABASE_PATH"] = database_path
            else:
                environment["DATABASE_DSN"] = selected_database_dsn
            server = subprocess.Popen(
                [api_bin],
                env=environment,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.PIPE,
                text=True,
            )
            try:
                BASE_URL = f"http://127.0.0.1:{selected_port}"
                for _ in range(30):
                    if server.poll() is not None:
                        stderr = server.stderr.read() if server.stderr else ""
                        raise RuntimeError(f"API exited before readiness ({server.returncode}):\n{stderr}")
                    try:
                        status, body = request("/readyz")
                        if status != 200:
                            raise RuntimeError(f"API readiness failed ({status}): {body!r}")
                        break
                    except (OSError, urllib.error.URLError, ConnectionError):
                        time.sleep(0.2)
                else:
                    stderr = server.stderr.read() if server.stderr else ""
                    raise RuntimeError(f"API did not become ready:\n{stderr}")
                yield ids
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
            database_dsn = arguments.database_dsn or os.getenv("FELICIA_TEST_DATABASE_DSN", "")
            with disposable_server(
                arguments.port,
                arguments.database_driver,
                arguments.database_path,
                database_dsn,
                arguments.postgres_admin_dsn or os.getenv("FELICIA_TEST_POSTGRES_ADMIN_DSN", ""),
            ) as ids:
                run_workflow(ids)
        else:
            run_workflow(workflow_ids())
    except (AssertionError, OSError, RuntimeError, urllib.error.URLError) as exc:
        print(f"workflow failed: {exc!r}", file=sys.stderr)
        raise SystemExit(1) from exc
