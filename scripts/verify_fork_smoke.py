#!/usr/bin/env python3
"""Build a clean checkout from another filesystem path as a fork smoke test."""

from __future__ import annotations

import io
import os
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
BASE_PATH = "/forked-felicia/"


def run(command: list[str], *, cwd: Path, env: dict[str, str] | None = None) -> None:
    print("$", " ".join(command))
    subprocess.run(command, cwd=cwd, env=env, check=True)


def archive_checkout(destination: Path) -> None:
    archive = subprocess.run(
        ["git", "archive", "--format=tar", "HEAD"],
        cwd=ROOT,
        check=True,
        capture_output=True,
    ).stdout
    with tarfile.open(fileobj=io.BytesIO(archive), mode="r:") as tar:
        tar.extractall(destination)


def main() -> None:
    with tempfile.TemporaryDirectory(prefix="felicia-fork-smoke-") as temporary:
        checkout = Path(temporary) / "clone with a different path"
        checkout.mkdir()
        archive_checkout(checkout)

        environment = os.environ.copy()
        environment["BASE_PATH"] = BASE_PATH
        run(["bun", "install", "--frozen-lockfile"], cwd=checkout / "apps" / "web-public", env=environment)
        run(
            [sys.executable, "scripts/felicia.py", "publish", "--base-path", BASE_PATH],
            cwd=checkout,
            env=environment,
        )

        dist = checkout / "apps" / "web-public" / "dist"
        assert (dist / "index.html").is_file()
        assert (dist / "api" / "v1" / "journeys.json").is_file()
        print(f"fork smoke verified: checkout={checkout} base={BASE_PATH}")


if __name__ == "__main__":
    main()
