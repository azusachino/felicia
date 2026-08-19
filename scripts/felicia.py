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
DEFAULT_MEDIA_ROOT = ROOT / ".felicia" / "media"
DEFAULT_PUBLICATION_PACKAGE = ROOT / ".felicia" / "publication.zip"
DEFAULT_DIST = WEB / "dist"


def run(command: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    print("$", " ".join(command), flush=True)
    subprocess.run(command, cwd=cwd, env=env, check=True)


def publish(base_path: str) -> None:
    configured_database = os.environ.get("PAGES_DB")
    database = Path(configured_database) if configured_database else DEFAULT_DB
    media_root = Path(os.environ.get("PAGES_MEDIA_ROOT", DEFAULT_MEDIA_ROOT))
    output = DEFAULT_DIST
    media_root.mkdir(parents=True, exist_ok=True)
    if not configured_database:
        for suffix in ("", "-wal", "-shm"):
            database.with_name(database.name + suffix).unlink(missing_ok=True)
        shutil.rmtree(media_root)
        media_root.mkdir(parents=True, exist_ok=True)
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
