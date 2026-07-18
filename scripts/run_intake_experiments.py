#!/usr/bin/env python3
"""Run the offline intake experiment matrix against the real CLI binary."""

from __future__ import annotations

import hashlib
import argparse
import json
import resource
import shutil
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
CLI = ROOT / "bin" / "felicia-cli"
JOURNEY = "00000000-0000-0000-0000-000000000001"


def run(command: list[str], *, cwd: Path = ROOT) -> dict[str, object]:
    started = time.monotonic()
    before = resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss
    process = subprocess.run(command, cwd=cwd, text=True, capture_output=True, check=False)
    after = resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss
    rss_scale = 1024 if sys.platform == "darwin" else 1
    return {
        "command": command,
        "exit_code": process.returncode,
        "stdout": process.stdout,
        "stderr": process.stderr,
        "elapsed_ms": round((time.monotonic() - started) * 1000, 2),
        "child_max_rss_kb": round(max(0, after - before) / rss_scale),
    }


def write_route(path: Path, points: list[tuple[float, float, str]]) -> None:
    body = [
        '<?xml version="1.0"?><gpx version="1.1">',
        "<trk><trkseg>",
    ]
    body.extend(
        f'<trkpt lat="{lat}" lon="{lon}"><time>{at}</time></trkpt>'
        for lat, lon, at in points
    )
    body.extend(["</trkseg></trk></gpx>", ""])
    path.write_text("".join(body), encoding="utf-8")


def baseline_points() -> list[tuple[float, float, str]]:
    return [
        (35.0000, 135.0000, "2026-04-01T09:00:00Z"),
        (35.0001, 135.0001, "2026-04-01T09:15:00Z"),
        (35.0001, 135.0001, "2026-04-01T09:30:00Z"),
        (35.1000, 135.1000, "2026-04-01T10:00:00Z"),
        (35.1001, 135.1001, "2026-04-01T10:20:00Z"),
        (35.1001, 135.1001, "2026-04-01T10:40:00Z"),
    ]


def plan_command(route: Path, photos: Path | None = None, fmt: str = "json") -> list[str]:
    command = [str(CLI), "journey", "plan", "--journey", JOURNEY, "--gpx", str(route), "--format", fmt]
    if photos is not None:
        command.extend(["--photos", str(photos)])
    return command


def case_result(case_id: str, outcome: str, execution: dict[str, object], **details: object) -> dict[str, object]:
    summary = {key: value for key, value in execution.items() if key not in {"stdout", "stderr"}}
    if execution.get("stderr"):
        summary["stderr"] = execution["stderr"]
    return {"id": case_id, "outcome": outcome, "execution": summary, **details}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, help="write the report JSON to this path")
    args = parser.parse_args()
    if not CLI.exists():
        raise SystemExit("bin/felicia-cli is missing; run `make cli-build` first")
    results: list[dict[str, object]] = []
    with tempfile.TemporaryDirectory(prefix="felicia-intake-") as temp:
        workspace = Path(temp)
        route = workspace / "route.gpx"
        photos = workspace / "photos"
        photos.mkdir()
        write_route(route, baseline_points())
        for name in ("ticket.jpg", "park.jpg", "unattached.jpg"):
            (photos / name).write_bytes((name + " fixture").encode())

        first = run(plan_command(route, photos))
        second = run(plan_command(route, photos))
        if first["exit_code"] == 0 and second["exit_code"] == 0:
            first_plan = json.loads(str(first["stdout"]))
            first_hash = hashlib.sha256(str(first["stdout"]).encode()).hexdigest()
            second_hash = hashlib.sha256(str(second["stdout"]).encode()).hexdigest()
            results.append(case_result("US-01-plan", "pass", first, route_points=6, photo_count=3, stops=len(first_plan["stops"]), deterministic=first_hash == second_hash, plan_hash=first_hash, second_plan_hash=second_hash))
        else:
            results.append(case_result("US-01-plan", "fail", first, reason="journey plan did not execute"))

        results.append(case_result("US-02-review-stops", "partial", first, reason="plan is executable, but this harness does not fabricate authored review actions or exercise an HTTP review session"))

        metadata = run(plan_command(route, photos))
        if metadata["exit_code"] == 0:
            plan = json.loads(str(metadata["stdout"]))
            results.append(case_result("US-03-missing-metadata", "partial", metadata, discovered_media=3, matched_mementos=len(plan["mementos"]), unattached_media=3, reason="local adapter preserves missing metadata but has no EXIF/JSONL sidecar promotion yet"))
        else:
            results.append(case_result("US-03-missing-metadata", "fail", metadata, reason="local media plan failed"))

        results.append(case_result("US-04-mementos-from-stop", "partial", first, reason="planner emits at most one generic memento candidate per matched stop; multiple authored mementos are not yet modeled"))
        results.append(case_result("US-05-agent-suggestions", "not_run", {"command": [], "exit_code": 0, "elapsed_ms": 0, "child_max_rss_kb": 0}, reason="agent suggestion schema/store is not implemented; correctly remains non-mutating by absence"))

        bad_route = workspace / "bad.gpx"
        bad_route.write_text('<gpx><trk><trkseg><trkpt lat="95" lon="135"/></trkseg></trk></gpx>', encoding="utf-8")
        bad = run(plan_command(bad_route))
        results.append(case_result("evil-bad-gpx", "pass" if bad["exit_code"] != 0 else "fail", bad, rejected=bad["exit_code"] != 0))

        large_route = workspace / "large.gpx"
        large_start = datetime(2026, 4, 1, 9, tzinfo=timezone.utc)
        large_points = [(35.0 + (index % 100) * 0.00001, 135.0 + (index % 100) * 0.00001, (large_start + timedelta(seconds=index)).isoformat().replace("+00:00", "Z")) for index in range(20_000)]
        write_route(large_route, large_points)
        large = run(plan_command(large_route))
        results.append(case_result("evil-large-gpx", "pass" if large["exit_code"] == 0 else "fail", large, points=20_000, note="measurement is a baseline; GPX parser currently materializes the XML document"))

        package = workspace / "preview.zip"
        package_build = run(["python3", "scripts/build_preview_package.py"])
        if package_build["exit_code"] == 0 and (ROOT / ".felicia" / "preview.zip").exists():
            shutil.copy(ROOT / ".felicia" / "preview.zip", package)
            validate = run([str(CLI), "package", "validate", str(package)])
            results.append(case_result("US-06-safe-publish", "pass" if validate["exit_code"] == 0 else "fail", validate, raw_gpx_public=False, note="prepared-package path only"))
        else:
            results.append(case_result("US-06-safe-publish", "blocked", package_build, reason="preview package could not be built"))

    report = {"schema": "felicia.intake.experiment-report", "version": "1", "cases": results}
    encoded = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(encoded, encoding="utf-8")
        print(f"intake experiment report: {args.out}")
    else:
        print(encoded, end="")


if __name__ == "__main__":
    main()
