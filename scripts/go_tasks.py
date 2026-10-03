#!/usr/bin/env python3
"""Run one Go task across every module declared by go.work."""

from __future__ import annotations

import argparse
import os
import plistlib
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GO_WORK = ROOT / "go.work"
TASKS = {
    "vet": ["go", "vet", "./..."],
    "lint": ["golangci-lint", "run", "./..."],
    "test": ["go", "test", "-race", "-cover", "./..."],
    "build": ["go", "build", "./..."],
    "desktop-build": ["go", "build", "-tags", "production", "-o", "../../bin/felicia-desktop", "."],
    "desktop-e2e-build": ["go", "build", "-tags", "production,e2e", "-o", "../../bin/felicia-desktop-e2e", "."],
    "desktop-e2e-test": ["go", "test", "-tags", "e2e", "."],
}


def discover_modules(go_work: Path = GO_WORK) -> list[str]:
    """Return Go module directories from a go.work use block."""
    modules: list[str] = []
    in_block = False
    for raw in go_work.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("//"):
            continue
        if line.startswith("use ("):
            in_block = True
            continue
        if in_block:
            if line == ")":
                in_block = False
            else:
                modules.append(line.lstrip("./"))
        elif line.startswith("use "):
            modules.append(line[len("use ") :].strip().lstrip("./"))
    return modules


def should_skip_module(module: str) -> bool:
    """Check if a module requires platform-specific toolchains that are missing."""
    if module == "apps/felicia-desktop" and sys.platform.startswith("linux"):
        check = subprocess.run(
            ["pkg-config", "--exists", "gtk4", "webkitgtk-6.0"],
            check=False,
            capture_output=True,
        )
        if check.returncode != 0:
            print(
                "go_tasks: skipping apps/felicia-desktop on Linux (missing gtk4 / webkitgtk-6.0 dev packages)",
                file=sys.stderr,
            )
            return True
    return False


def module_environment(module: str) -> dict[str, str]:
    environment = dict(os.environ)
    if (
        module == "apps/felicia-desktop"
        and sys.platform == "darwin"
        and environment.get("GOOS", "darwin") == "darwin"
    ):
        info = plistlib.loads((ROOT / module / "Info.plist").read_bytes())
        minimum = info["LSMinimumSystemVersion"]
        environment["MACOSX_DEPLOYMENT_TARGET"] = minimum
        for flag in ("CGO_CFLAGS", "CGO_LDFLAGS"):
            environment[flag] = environment.get(flag, "-O2 -g") + f" -mmacosx-version-min={minimum}"
    return environment


def run_task(task: str, modules: list[str]) -> int:
    """Run one task in each discovered module, stopping at the first failure."""
    for module in modules:
        if should_skip_module(module):
            continue
        result = subprocess.run(TASKS[task], cwd=ROOT / module, env=module_environment(module), check=False)
        if result.returncode:
            print(
                f"go_tasks: {task} failed in {module} (exit {result.returncode})",
                file=sys.stderr,
            )
            return result.returncode
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("task", choices=sorted(TASKS))
    args = parser.parse_args()
    modules = ["apps/felicia-desktop"] if args.task.startswith("desktop-") else discover_modules()
    if not modules:
        print(f"go_tasks: no modules found in {GO_WORK}", file=sys.stderr)
        return 1
    return run_task(args.task, modules)


if __name__ == "__main__":
    raise SystemExit(main())
