#!/usr/bin/env python3
"""Exercise the complete authoring-to-public journey workflow over HTTP."""

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from contextlib import contextmanager, nullcontext
from dataclasses import dataclass
from pathlib import Path
from uuid import NAMESPACE_URL, uuid4, uuid5


BASE_URL = os.getenv("API_BASE", "http://localhost:8080")
REPO_ROOT = Path(__file__).resolve().parent.parent
# A real JPEG to stand in for ingested media (see run_static_parity_check).
DEMO_PHOTO = REPO_ROOT / "apps" / "web-public" / "public" / "kyoto_temple.jpg"


@dataclass(frozen=True)
class WorkflowIDs:
    journal: str
    journey: str
    memento: str
    photo: str
    slug: str


@dataclass(frozen=True)
class ServerContext:
    """What the disposable server can hand back to the driver script.

    ``database_path`` is only populated for the sqlite driver — the static
    compiler CLI (``felicia-cli static compile``) only knows how to open a
    SQLite file (see cli/cmd/felicia/main.go's compileCommand, which calls
    sqlite.Open unconditionally), so the static/live parity check below only
    runs when this is set.
    """

    ids: WorkflowIDs
    driver: str
    database_path: str


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
    # Backs off on 429: the API's client rate limiter (1 req/s, burst 20)
    # is easily exhausted by the later parity/regression stages, and every
    # write this harness makes is an idempotent upsert, so replaying is safe.
    body = None if payload is None else json.dumps(payload).encode()
    for _ in range(60):
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
            if exc.code == 429:
                time.sleep(1.1)
                continue
            detail = exc.read().decode()
            try:
                detail = json.loads(detail)
            except json.JSONDecodeError:
                pass
            raise AssertionError(f"{method} {path} failed ({exc.code}): {detail}") from exc
    raise AssertionError(f"{method} {path} still rate-limited after 60 attempts")


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

    # Advance one legal step at a time (docs/contracts/memento-lifecycle.md §3):
    # draft→authored, then authored→published. A direct draft→published jump is
    # rejected by the lifecycle guard (422 invalid_transition).
    authored = {
        **draft,
        **published_memento_fields(),
        "state": "authored",
        "expected_revision": 1,
    }
    post("/api/admin/mementos", authored)

    published = {
        **draft,
        **published_memento_fields(),
        "state": "published",
        "expected_revision": 2,
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


RATE_LIMIT_RETRIES = 60


def published_memento_fields() -> dict:
    """The authored field set of the workflow's published memento.

    Shared between the publish step and the stale-artifact regression's
    unpublish step, which must send a complete upsert body (the admin GET
    response serializes geometry in orb's array form, not the GeoJSON object
    the upsert endpoint expects, so replaying a GET body is not an option).
    """
    return {
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


def fetch_response(path: str) -> tuple[int, bytes]:
    """GET a path, waiting out the API's rate limiter (1 req/s, burst 20).

    The parity and negative-case checks below issue more requests than the
    server's default burst allows; a well-behaved client backs off on 429
    instead of failing, so the limiter's defaults stay untouched.
    """
    for _ in range(RATE_LIMIT_RETRIES):
        req = urllib.request.Request(f"{BASE_URL}{path}")
        try:
            with urllib.request.urlopen(req) as response:
                return response.status, response.read()
        except urllib.error.HTTPError as exc:
            if exc.code == 429:
                time.sleep(1.1)
                continue
            return exc.code, exc.read()
    raise RuntimeError(f"GET {path} still rate-limited after {RATE_LIMIT_RETRIES} attempts")


def fetch_raw(path: str) -> bytes:
    """Fetch the raw response body for a live public endpoint (no JSON decode)."""
    status, body = fetch_response(path)
    expect(status == 200, f"GET {path} expected 200, got {status}")
    return body


def fetch_status(path: str) -> int:
    """GET a path and return its status code without raising on non-2xx.

    Unlike request() above (which treats any non-2xx response as a hard
    test failure — see its HTTPError branch), this is for asserting an
    *expected* error status, such as the 404 a journey with no published
    mementos should return.
    """
    status, _ = fetch_response(path)
    return status


def read_static_file(out_dir: str, relative_path: str) -> bytes:
    with open(os.path.join(out_dir, *relative_path.split("/")), "rb") as handle:
        return handle.read()


def canonical_json(raw: bytes) -> str:
    """Canonicalize JSON bytes for cross-surface comparison.

    The live server encodes responses with encoding/json's Encoder — compact,
    no indentation, one trailing newline (server/api/server.go respondJSON).
    The static compiler writes files with json.MarshalIndent at a 2-space
    indent plus a manually appended trailing newline
    (cli/cmd/felicia/main.go fileArtifactWriter.WriteJSON). Both sides
    project through the same publication package types (PublishedMementos,
    NewStaticJourney, NewStaticMemento, NewStaticPhoto), so the *values* are
    expected to be identical, but the raw bytes never will be — only the
    whitespace differs, by construction of the two writers. Decoding and
    re-serializing through the same canonical form (sorted keys, compact
    separators) makes the comparison target content, not formatting.
    """
    return json.dumps(json.loads(raw.decode("utf-8")), sort_keys=True, separators=(",", ":"))


def expect_json_parity(label: str, live_raw: bytes, static_raw: bytes) -> None:
    live_canonical = canonical_json(live_raw)
    static_canonical = canonical_json(static_raw)
    expect(
        live_canonical == static_canonical,
        f"{label}: live/static public JSON diverged\n  live:   {live_canonical}\n  static: {static_canonical}",
    )


def expect_aliases_match(path_a: str, path_b: str) -> None:
    """The extensionless route and its ".json" alias share one handler
    (server/api/server.go) — they must return byte-identical bodies."""
    expect(
        fetch_raw(path_a) == fetch_raw(path_b),
        f"{path_a} and {path_b} should be byte-identical (they share a handler)",
    )


def run_static_parity_check(context: ServerContext) -> None:
    """Compile the static artifact from the same SQLite database the live
    server is reading, then prove the two publication surfaces agree.

    Both surfaces already share the same projection code — this only guards
    that the two entry points (server/api/server.go's public handlers and
    cli/cmd/felicia/main.go's `static compile`) keep calling that shared
    code the same way, for the same data.
    """
    ids = context.ids
    with tempfile.TemporaryDirectory(prefix="felicia-workflow-static-") as static_dir:
        cli_bin = os.path.join(static_dir, "felicia-cli")
        media_root = os.path.join(static_dir, "media")
        out_dir = os.path.join(static_dir, "site")

        # The workflow above only exercises the admin API's metadata calls —
        # it registers a photo's object_key but never uploads real bytes.
        # Plant a real JPEG at that object key so the compiler's media source
        # (which reads real files off disk) can open it, the way a real ingest
        # would have placed it there. It has to be a genuinely decodable image:
        # the compiler resizes and EXIF-strips every published derivative, and
        # fails the compile rather than emit an original it cannot sanitize.
        photo_path = os.path.join(media_root, "workflow", "live.jpg")
        os.makedirs(os.path.dirname(photo_path), exist_ok=True)
        shutil.copyfile(DEMO_PHOTO, photo_path)

        subprocess.run(["go", "build", "-o", cli_bin, "./cli/cmd/felicia"], check=True)

        def compile_static() -> dict:
            result = subprocess.run(
                [
                    cli_bin,
                    "static",
                    "compile",
                    "--db",
                    context.database_path,
                    "--media-root",
                    media_root,
                    "--out",
                    out_dir,
                ],
                check=True,
                capture_output=True,
                text=True,
            )
            return json.loads(result.stdout)

        report = compile_static()
        expect(report.get("Journeys") == 1, f"expected compiler to publish 1 journey, got {report!r}")

        journey_path = f"/api/v1/journeys/{ids.journey}"
        mementos_path = f"/api/v1/journeys/{ids.journey}/mementos"

        expect_aliases_match("/api/v1/journeys", "/api/v1/journeys.json")
        expect_aliases_match(journey_path, journey_path + ".json")
        expect_aliases_match(mementos_path, mementos_path + ".json")
        expect_aliases_match("/api/v1/site", "/api/v1/site.json")

        expect_json_parity(
            "journeys index",
            fetch_raw("/api/v1/journeys"),
            read_static_file(out_dir, "api/v1/journeys.json"),
        )
        expect_json_parity(
            "site settings",
            fetch_raw("/api/v1/site"),
            read_static_file(out_dir, "api/v1/site.json"),
        )
        expect_json_parity(
            "journey detail",
            fetch_raw(journey_path),
            read_static_file(out_dir, f"api/v1/journeys/{ids.journey}.json"),
        )
        expect_json_parity(
            "journey mementos",
            fetch_raw(mementos_path),
            read_static_file(out_dir, f"api/v1/journeys/{ids.journey}/mementos.json"),
        )
        print("live/static public JSON parity passed")

        # The published media must be the sanitized derivative, never the
        # original bytes: public images are resized and EXIF-stripped
        # (docs/direction.md, ADR-0026). The object key is unchanged, so the
        # JSON projection above still resolves.
        published_photo = os.path.join(out_dir, "workflow", "live.jpg")
        expect(os.path.exists(published_photo), f"compiler did not publish {published_photo}")
        with open(published_photo, "rb") as handle:
            published_bytes = handle.read()
        expect(
            published_bytes != DEMO_PHOTO.read_bytes(),
            "compiler published the original media bytes verbatim instead of a sanitized derivative",
        )
        expect(
            b"Exif\x00\x00" not in published_bytes,
            "published media still carries an EXIF segment",
        )
        print("published media is a sanitized derivative -- EXIF-strip check passed")

        # Negative case: a journey with only a draft memento has no public
        # projection anywhere. The shared PublishedMementos gate hides it
        # from the live API (404 on detail/mementos, absent from the index)
        # and the compiler must not emit any file for it.
        unpublished_journey_uuid = uuid4()
        unpublished_journey = str(unpublished_journey_uuid)
        unpublished_memento = str(uuid4())
        post(
            "/api/admin/journeys",
            {
                "id": unpublished_journey,
                "journal_id": ids.journal,
                "slug": f"workflow-unpublished-{unpublished_journey_uuid.hex[:12]}",
                "title": "Unpublished workflow journey",
                "place": "Kyoto",
                "country": "JPN",
                "date_start": "2026-03-25",
                "date_end": "2026-03-25",
                "gps_route": [[[135.7, 35.0], [135.8, 35.1]]],
                "authored_fields": [],
            },
        )
        post(
            "/api/admin/mementos",
            {
                "id": unpublished_memento,
                "journey_id": unpublished_journey,
                "kind": "live",
                "seq": 1,
                "state": "draft",
                "kind_data": {"artist": "test"},
            },
        )

        report = compile_static()
        expect(
            report.get("Journeys") == 1,
            f"expected compiler to still publish only 1 journey after adding a draft-only journey, got {report!r}",
        )

        status = fetch_status(f"/api/v1/journeys/{unpublished_journey}")
        expect(status == 404, f"expected 404 for unpublished journey detail, got {status}")
        status = fetch_status(f"/api/v1/journeys/{unpublished_journey}/mementos")
        expect(status == 404, f"expected 404 for unpublished journey mementos, got {status}")

        live_index = json.loads(fetch_raw("/api/v1/journeys"))
        expect(
            all(item.get("id") != unpublished_journey for item in live_index),
            f"unpublished journey leaked into the live journeys index: {live_index!r}",
        )
        static_index = json.loads(read_static_file(out_dir, "api/v1/journeys.json"))
        expect(
            all(item.get("id") != unpublished_journey for item in static_index),
            f"unpublished journey leaked into the static journeys index: {static_index!r}",
        )
        expect(
            not os.path.exists(os.path.join(out_dir, "api", "v1", "journeys", f"{unpublished_journey}.json")),
            "compiler wrote a detail file for an unpublished journey",
        )
        expect(
            not os.path.exists(os.path.join(out_dir, "api", "v1", "journeys", unpublished_journey, "mementos.json")),
            "compiler wrote a mementos file for an unpublished journey",
        )
        print("unpublished-journey exclusion passed")

        # Stale-artifact regression: content published in an earlier compile
        # and later unpublished must disappear from a REUSED output
        # directory. Unpublish the memento (published→authored, the only legal
        # backward step per docs/contracts/memento-lifecycle.md §3; authored is
        # non-public, same as draft for compilation), recompile into the same
        # out_dir, and assert its old JSON and media are gone (the compiler
        # reconciles via the artifact manifest).
        _, current = request(f"/api/admin/mementos/{ids.memento}")
        unpublish = {
            "id": ids.memento,
            "journey_id": ids.journey,
            "kind": "live",
            "seq": 1,
            **published_memento_fields(),
            "state": "authored",
            "expected_revision": current["revision"],
        }
        post("/api/admin/mementos", unpublish)

        report = compile_static()
        expect(
            report.get("Journeys") == 0,
            f"expected no published journeys after unpublish, got {report!r}",
        )
        expect(
            report.get("Removed", 0) >= 3,
            f"expected the stale detail/mementos/media artifacts to be removed, got {report!r}",
        )
        expect(
            not os.path.exists(os.path.join(out_dir, "api", "v1", "journeys", f"{ids.journey}.json")),
            "stale journey detail JSON survived recompilation into the same out dir",
        )
        expect(
            not os.path.exists(os.path.join(out_dir, "api", "v1", "journeys", ids.journey, "mementos.json")),
            "stale journey mementos JSON survived recompilation into the same out dir",
        )
        expect(
            not os.path.exists(os.path.join(out_dir, "workflow", "live.jpg")),
            "stale published media survived recompilation into the same out dir",
        )
        static_index = json.loads(read_static_file(out_dir, "api/v1/journeys.json"))
        expect(
            static_index == [],
            f"unpublished journey still listed in the static index: {static_index!r}",
        )
        print("stale-artifact cleanup passed")


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
def disposable_server(
    port: int,
    driver: str,
    database_path: str,
    database_dsn: str,
    postgres_admin_dsn: str,
    extra_env: dict[str, str] | None = None,
):
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
                # Lets a caller wire optional ingest sources (DAWARICH_URL/
                # IMMICH_URL/etc. — see server/config/config.go) or override
                # MEDIA_ROOT without this function needing to know about
                # every one of them. Unused by the workflow test itself.
                **(extra_env or {}),
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
                yield ServerContext(
                    ids=ids,
                    driver=driver,
                    database_path=database_path if driver == "sqlite" else "",
                )
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
            ) as context:
                run_workflow(context.ids)
                if context.database_path:
                    run_static_parity_check(context)
                else:
                    print("static/live parity check skipped (requires --database-driver sqlite)")
        else:
            run_workflow(workflow_ids())
    except (
        AssertionError,
        OSError,
        RuntimeError,
        urllib.error.URLError,
        subprocess.CalledProcessError,
    ) as exc:
        print(f"workflow failed: {exc!r}", file=sys.stderr)
        raise SystemExit(1) from exc
