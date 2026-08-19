#!/usr/bin/env python3
"""Agent-friendly commands for building Felicia's v0.1 static publication."""

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
DEFAULT_INBOX = ROOT / ".felicia" / "inbox"
DEFAULT_MEDIA_ROOT = ROOT / ".felicia" / "media"
DEFAULT_PREVIEW_PACKAGE = ROOT / ".felicia" / "preview.zip"
DEFAULT_DIST = WEB / "dist"


def run(command: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    print("$", " ".join(command), flush=True)
    subprocess.run(command, cwd=cwd, env=env, check=True)


def build(base_path: str) -> None:
    environment = os.environ.copy()
    environment["BASE_PATH"] = base_path
    run([sys.executable, "scripts/build_static_demo.py"], env=environment)
    run(["bun", "run", "build"], cwd=WEB, env=environment)


def validate(base_path: str) -> None:
    environment = os.environ.copy()
    environment["BASE_PATH"] = base_path
    run([sys.executable, "scripts/verify_static_demo.py"], env=environment)


def publish(base_path: str, dry_run: bool) -> None:
    build(base_path)
    validate(base_path)
    dist = WEB / "dist"
    files = sorted(path.relative_to(dist).as_posix() for path in dist.rglob("*") if path.is_file())
    if dry_run:
        print(f"publish dry-run: {len(files)} files arranged under {dist}")
        for path in files:
            print(path)
        return
    print(f"publish ready: {len(files)} files arranged under {dist}")
    print("Commit and push this artifact through the repository's normal review workflow.")


def preview(base_path: str) -> None:
    configured_database = os.environ.get("PAGES_DB")
    database = Path(configured_database) if configured_database else DEFAULT_DB
    inbox = Path(os.environ.get("PAGES_INBOX", DEFAULT_INBOX))
    media_root = Path(os.environ.get("PAGES_MEDIA_ROOT", DEFAULT_MEDIA_ROOT))
    output = DEFAULT_DIST
    inbox.mkdir(parents=True, exist_ok=True)
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
    packages = sorted(inbox.glob("*.zip"))
    if not packages:
        run([sys.executable, "scripts/build_preview_package.py"])
        packages = sorted((ROOT / ".felicia" / "preview-packages").glob("*.zip"))
        if not packages:
            packages = [DEFAULT_PREVIEW_PACKAGE]
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
    print(f"preview ready: {output} (packages imported: {len(packages)})")


def parser() -> argparse.ArgumentParser:
    command_parser = argparse.ArgumentParser(description=__doc__)
    command_parser.add_argument("command", choices=("build", "validate", "publish", "preview"))
    command_parser.add_argument("--base-path", default=os.environ.get("BASE_PATH", "/"))
    command_parser.add_argument("--dry-run", action="store_true", help="show the publish manifest without suggesting a push")
    return command_parser


def main() -> None:
    args = parser().parse_args()
    if not args.base_path.endswith("/"):
        args.base_path += "/"
    if args.command == "build":
        build(args.base_path)
    elif args.command == "validate":
        validate(args.base_path)
    elif args.command == "publish":
        publish(args.base_path, args.dry_run)
    else:
        preview(args.base_path)


if __name__ == "__main__":
    main()
