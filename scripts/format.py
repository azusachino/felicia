#!/usr/bin/env python3
"""Format or check repository source files with the project's native tools."""

from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def go_files() -> list[Path]:
    return sorted(
        path
        for path in ROOT.rglob("*.go")
        if ".git" not in path.parts
        and "node_modules" not in path.parts
        and "vendor" not in path.parts
    )


def markdown_files() -> list[Path]:
    return sorted(
        path
        for path in ROOT.rglob("*.md")
        if ".git" not in path.parts
        and "node_modules" not in path.parts
        and "site" not in path.parts
        and ".venv" not in path.parts
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
    web_apps = []
    for path in sorted((ROOT / "apps").iterdir()):
        package_json = path / "package.json"
        if not package_json.is_file():
            continue
        package = json.loads(package_json.read_text(encoding="utf-8"))
        if "format:check" in package.get("scripts", {}):
            web_apps.append(path)
    if not web_apps:
        return True
    if not all((web / "node_modules").exists() for web in web_apps):
        print("format: frontend dependencies missing; run make web-install")
        return True
    success = True
    for web in web_apps:
        command = ["bun", "run", "format:check" if check else "format"]
        result = run(command, cwd=web)
        print(result.stdout, end="")
        print(result.stderr, end="")
        success = result.returncode == 0 and success
    return success


def format_markdown(check: bool) -> bool:
    mode = "--check" if check else "--write"
    files = [str(path) for path in markdown_files()]
    if not files:
        return True
    # Markdown is repository documentation, not a frontend dependency. Keep this
    # path usable before `make web-install` and avoid loading the Svelte plugin.
    result = run(
        [
            "bun",
            "x",
            "--no-install",
            "prettier",
            "--no-config",
            "--parser",
            "markdown",
            "--prose-wrap",
            "preserve",
            "--print-width",
            "200",
            mode,
            *files,
        ]
    )
    print(result.stdout, end="")
    print(result.stderr, end="")
    return result.returncode == 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    success = format_go(args.check) and format_web(args.check) and format_markdown(args.check)
    return 0 if success else 1


if __name__ == "__main__":
    raise SystemExit(main())
