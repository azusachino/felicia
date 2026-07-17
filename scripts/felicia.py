#!/usr/bin/env python3
"""Agent-friendly commands for building Felicia's v0.1 static publication."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
WEB = ROOT / "apps" / "web-public"
CLI = ROOT / "bin" / "felicia-cli"
DEFAULT_DB = ROOT / ".felicia" / "felicia.sqlite"
DEFAULT_MEDIA_ROOT = ROOT / ".felicia" / "media"
DEFAULT_DIST = WEB / "dist"


def run(command: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    print("$", " ".join(command))
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
    database = Path(os.environ.get("PAGES_DB", DEFAULT_DB))
    media_root = Path(os.environ.get("PAGES_MEDIA_ROOT", DEFAULT_MEDIA_ROOT))
    output = DEFAULT_DIST
    if not database.is_file():
        raise SystemExit(f"SQLite database not found: {database} (import a package first)")
    media_root.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(CLI), "./cli/cmd/felicia"])
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
    for required in (output / "index.html", output / "api" / "v1" / "journeys.json"):
        if not required.is_file():
            raise SystemExit(f"preview artifact missing: {required}")
    print(f"preview ready: {output}")


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
