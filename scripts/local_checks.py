"""Content-keyed local checks; never used by full acceptance gates."""

import argparse
import hashlib
import json
import os
import platform
import re
import shutil
import subprocess
import sys
import time
import tomllib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
# Adapted from Iroha's local_checks.py; full gates never read these records.
GROUPS = {
    "go": ("fmt-go-check", "vet", "lint", "test"),
    "scripts": ("test-features",),
    "admin": ("admin-check",),
    "public": ("web-check",),
    "docs": ("fmt-docs-check",),
}
TOOLS = {
    "go": (["go", "version"], r"go version go(\d+\.\d+\.\d+)"),
    "uv": (["uv", "--version"], r"uv (\d+\.\d+\.\d+)"),
    "bun": (["bun", "--version"], r"^(\d+\.\d+\.\d+)"),
    "golangci-lint": (["golangci-lint", "--version"], r"version (\d+\.\d+\.\d+)"),
}
# Terminal/session metadata cannot affect these commands; all other environment
# values participate, including PATH, Go flags, timezone and frontend variables.
DISPLAY_ENV = {"PWD", "OLDPWD", "SHLVL", "_", "TERM", "TERM_PROGRAM", "TERM_PROGRAM_VERSION"}
# Never pass outer make control flags/CLI selectors into the fixed group commands.
MAKE_ENV = {"MAKEFLAGS", "MAKELEVEL", "MAKEOVERRIDES", "MFLAGS", "ARGS"}


def child_environment(environment: dict[str, str]) -> dict[str, str]:
    return {key: value for key, value in environment.items() if key not in MAKE_ENV}


def environment_key(environment: dict[str, str]) -> str:
    return hashlib.sha256(json.dumps({
        key: value for key, value in child_environment(environment).items()
        if key not in DISPLAY_ENV and not key.startswith(("__MISE_", "HERDR_", "PI_"))
    }, sort_keys=True).encode()).hexdigest()


def verify_tools(root: Path) -> dict[str, str]:
    pins = tomllib.loads((root / "mise.toml").read_text())["tools"]
    identities = {}
    for name, (command, pattern) in TOOLS.items():
        executable = shutil.which(command[0])
        if not executable:
            raise RuntimeError(f"{name} missing: run make local-check (pinned mise environment)")
        output = subprocess.check_output(command, text=True)
        match = re.search(pattern, output)
        pin = pins[name].removeprefix("v")
        if not match or not (match[1] == pin or match[1].startswith(pin + ".")):
            raise RuntimeError(f"{name} does not match pin {pin}; use make local-check")
        identities[name] = f"{Path(executable).resolve()}:{match[1]}"
    # These host executables are not pinned by mise, but changes invalidate hits.
    for name in ("node", "make", "rumdl"):
        executable = shutil.which(name)
        identities[name] = (
            f"{Path(executable).resolve()}:{subprocess.check_output([name, '--version'], text=True)}"
            if executable else "missing"
        )
    return identities


def groups_for(path: str) -> set[str]:
    if path.endswith(".md"):
        # Docs also contain the native feature/acceptance contract.
        return {"docs"}
    if path.startswith("apps/felicia-admin/"):
        return {"admin", "scripts"}
    if path.startswith("apps/felicia-public-site/"):
        return {"public", "scripts"}
    if path.startswith("packages/"):
        return {"admin", "public", "scripts"}
    if path.startswith("apps/") or path.startswith("go.work"):
        # Python feature tests seed/compile through the Go CLI.
        return {"go", "scripts"}
    if path.startswith(("tests/", "scripts/local_journey", "scripts/validate_local_authoring")):
        return {"scripts"}
    if path.startswith("docs/"):
        return {"docs"}
    # Runners, contracts, publication fixtures, root config and unknown paths
    # conservatively invalidate all groups rather than guess independent inputs.
    return set(GROUPS)


def input_digest(path: Path, root: Path, parents: frozenset[Path] = frozenset()) -> bytes:
    resolved = path.resolve()
    if not resolved.is_relative_to(root.resolve()):
        raise RuntimeError(f"source symlink escapes checkout: {path}")
    if resolved in parents:
        raise RuntimeError(f"source symlink cycle: {path}")
    if not path.exists():
        return b"deleted\0" + os.readlink(path).encode() if path.is_symlink() else b"deleted"
    digest = hashlib.sha256()
    if path.is_symlink():
        digest.update(os.readlink(path).encode())
    if path.is_dir():
        for child in sorted(path.iterdir()):
            digest.update(child.name.encode() + b"\0")
            digest.update(input_digest(child, root, parents | {resolved}))
    else:
        digest.update(path.read_bytes())
    return digest.digest()


def snapshot(root: Path) -> dict[str, bytes]:
    names = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=root,
    ).decode().split("\0")
    # Vite/SvelteKit load ignored local dotenv files too. Hash their bytes without
    # retaining contents in records or logs. Node_modules is restored separately.
    for pattern in (".env*", "apps/*/.env*", "packages/*/.env*"):
        names.extend(str(path.relative_to(root)) for path in root.glob(pattern) if path.is_file())
    result = {}
    for name in sorted(set(names) - {""}):
        path = root / name
        result[name] = input_digest(path, root)
    return result


def fingerprint(group: str, files: dict[str, bytes], identity: dict[str, str]) -> str:
    digest = hashlib.sha256(json.dumps(identity, sort_keys=True).encode())
    digest.update(json.dumps(GROUPS[group]).encode())
    for name, content in sorted(files.items()):
        if group in groups_for(name):
            digest.update(name.encode() + b"\0" + content)
    return digest.hexdigest()


def run_cached(root: Path, group: str, identity: dict[str, str], force: bool = False) -> int:
    cache = root / ".cache" / "local-checks"
    cache.mkdir(parents=True, exist_ok=True)
    record = cache / f"{group}.json"
    key = fingerprint(group, snapshot(root), identity)
    try:
        previous = json.loads(record.read_text())
    except (OSError, ValueError):
        previous = {}
    if not force and isinstance(previous, dict) and previous.get("key") == key:
        print(f"HIT  {group} ({', '.join(GROUPS[group])})", flush=True)
        return 0
    # Remove earlier success before attempting a check (including interruption).
    record.unlink(missing_ok=True)
    print(f"RUN  {group} ({', '.join(GROUPS[group])})", flush=True)
    started = time.monotonic()
    result = subprocess.run(
        ["make", "MISE_RUN=", *GROUPS[group]], cwd=root,
        env=child_environment(dict(os.environ)),
    )
    if result.returncode:
        return result.returncode
    if fingerprint(group, snapshot(root), identity) != key:
        raise RuntimeError(f"{group} inputs changed during checks; result not cached")
    temporary = record.with_suffix(".tmp")
    temporary.write_text(json.dumps({"key": key, "seconds": time.monotonic() - started}))
    temporary.replace(record)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--force", action="store_true", help="rerun every local group")
    parser.add_argument("--groups", nargs="+", choices=GROUPS, default=list(GROUPS))
    args = parser.parse_args()
    if os.environ.get("CI"):
        parser.error("local-check is unavailable in CI; use make check / make validate")
    try:
        identity = verify_tools(ROOT)
        identity["python"] = sys.version
        identity["platform"] = platform.platform()
        identity["environment"] = environment_key(dict(os.environ))
        environment = child_environment(dict(os.environ))
        # Restore locked dependencies even on hits; cache results, not installs.
        subprocess.run(["bun", "install", "--frozen-lockfile"], cwd=ROOT, env=environment, check=True)
        # Legacy ignored directories are absent from the content snapshot, so
        # the cheap layout guard must always run, including on all-hit runs.
        subprocess.run(["make", "MISE_RUN=", "layout-check"], cwd=ROOT, env=environment, check=True)
        for group in args.groups:
            code = run_cached(ROOT, group, identity, args.force)
            if code:
                return code
        return 0
    except (RuntimeError, OSError, subprocess.CalledProcessError) as error:
        print(f"local-check: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
