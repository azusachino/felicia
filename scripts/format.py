#!/usr/bin/env python3
"""Format or check repository source files with the project's native tools."""

from __future__ import annotations

import argparse
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent


def go_files() -> list[Path]:
    return sorted(
        path
        for path in ROOT.rglob("*.go")
        if ".git" not in path.parts and "node_modules" not in path.parts
    )


def run(command: list[str], *, cwd: Path = ROOT) -> subprocess.CompletedProcess[str]:
    return subprocess.run(command, cwd=cwd, check=False, text=True, capture_output=True)


def format_go(check: bool) -> bool:
    files = go_files()
    if not files:
        return True
    command = ["gofmt", "-l"] if check else ["gofmt", "-w"]
    result = run(command + [str(path.relative_to(ROOT)) for path in files])
    if result.returncode != 0:
        print(result.stderr, end="")
        return False
    if check and result.stdout:
        print("Unformatted Go files:")
        print(result.stdout, end="")
        return False
    return True


def format_web(check: bool) -> bool:
    web = ROOT / "apps" / "web-public"
    if not (web / "node_modules").exists():
        print("format: frontend dependencies missing; run make web-install")
        return True
    command = ["bun", "run", "format:check" if check else "format"]
    result = run(command, cwd=web)
    print(result.stdout, end="")
    print(result.stderr, end="")
    return result.returncode == 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    return 0 if format_go(args.check) and format_web(args.check) else 1


if __name__ == "__main__":
    raise SystemExit(main())
