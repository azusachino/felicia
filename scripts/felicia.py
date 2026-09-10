#!/usr/bin/env python3
"""Build the production publication catalog into the public Pages artifact."""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
WEB = ROOT / "apps" / "felicia-public-site"
CLI = ROOT / "bin" / "felicia-cli"
DEFAULT_DB = ROOT / ".felicia" / "pages-preview.sqlite"
# The preview owns a disposable media workspace of its own. It must never
# default to the authoring media root (`.felicia/media`, per
# apps/felicia-server/config/config.go and `felicia-cli import`), which holds
# the only copy of an author's originals: a public artifact is deliberately
# lossy and providers are not a backup, so nothing there is reconstructible.
DEFAULT_MEDIA_ROOT = ROOT / ".felicia" / "pages-preview-media"
DEFAULT_PUBLICATION_PACKAGE = ROOT / ".felicia" / "publication.zip"
DEFAULT_DIST = WEB / "dist"
# Written into a directory this script has emptied, so a later run can tell a
# workspace it owns from one it was merely pointed at.
WORKSPACE_MARKER = ".felicia-publication-workspace"


def run(command: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    print("$", " ".join(command), flush=True)
    subprocess.run(command, cwd=cwd, env=env, check=True)


def reset_disposable(path: Path) -> None:
    """Empty a directory this build can prove it owns, then mark it as owned.

    A missing or empty directory is adopted, and one already carrying the marker
    is reset. Anything else is refused with nothing deleted -- the preview is
    never worth destroying data whose provenance it cannot establish.
    """
    if path.exists() and any(path.iterdir()) and not (path / WORKSPACE_MARKER).exists():
        raise SystemExit(
            f"refusing to reset {path}: it is not a publication workspace "
            f"(no {WORKSPACE_MARKER}) and may hold original media. Point the "
            f"preview at a disposable directory, or empty this one yourself."
        )
    shutil.rmtree(path, ignore_errors=True)
    path.mkdir(parents=True, exist_ok=True)
    (path / WORKSPACE_MARKER).touch()


def publish(base_path: str) -> None:
    configured_database = os.environ.get("PAGES_DB")
    database = Path(configured_database) if configured_database else DEFAULT_DB
    configured_media_root = os.environ.get("PAGES_MEDIA_ROOT")
    media_root = Path(configured_media_root) if configured_media_root else DEFAULT_MEDIA_ROOT
    output = DEFAULT_DIST
    # Reset only what this build owns, and decide that per root rather than from
    # one shared flag: keying the media reset on PAGES_DB is what let an
    # explicitly configured PAGES_MEDIA_ROOT be deleted.
    if not configured_database:
        for suffix in ("", "-wal", "-shm"):
            database.with_name(database.name + suffix).unlink(missing_ok=True)
    if configured_media_root:
        # Used as given. The caller may have pointed it at real media, and a
        # stale file here cannot reach the artifact: the compiler resolves media
        # from photo rows, never by walking this directory.
        media_root.mkdir(parents=True, exist_ok=True)
    else:
        reset_disposable(media_root)
    shutil.rmtree(output, ignore_errors=True)
    # The API tree is produced by the Go compiler, not by Vite's public source.
    # Remove an older generated copy before Vite copies its public directory.
    shutil.rmtree(WEB / "public" / "api", ignore_errors=True)
    # `go build -o` does not create missing parent directories (bin/ is
    # gitignored, so a fresh clone/CI runner never has it yet).
    CLI.parent.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(CLI), "./apps/felicia-cli/cmd/felicia"])
    run([sys.executable, "scripts/build_publication_package.py"])
    packages = sorted((ROOT / ".felicia" / "publication-packages").glob("*.zip"))
    if not packages:
        packages = [DEFAULT_PUBLICATION_PACKAGE]
    for package in packages:
        run(
            [
                str(CLI),
                "import",
                "--db",
                str(database),
                "--media-root",
                str(media_root),
                "--apply",
                str(package),
            ]
        )
    environment = os.environ.copy()
    environment["BASE_PATH"] = base_path
    run(["bun", "run", "build"], cwd=WEB, env=environment)
    run(
        [
            str(CLI),
            "static",
            "compile",
            "--db",
            str(database),
            "--media-root",
            str(media_root),
            "--out",
            str(output),
        ]
    )
    print(f"publication ready: {output} (packages imported: {len(packages)})")


def parser() -> argparse.ArgumentParser:
    command_parser = argparse.ArgumentParser(description=__doc__)
    command_parser.add_argument("command", choices=("publish",))
    command_parser.add_argument("--base-path", default=os.environ.get("BASE_PATH", "/"))
    return command_parser


def main() -> None:
    args = parser().parse_args()
    if not args.base_path.endswith("/"):
        args.base_path += "/"
    publish(args.base_path)


if __name__ == "__main__":
    main()
