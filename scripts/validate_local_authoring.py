"""Validate local authoring JSON documents against the committed v1 schema."""

from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker


ROOT = Path(__file__).resolve().parent.parent
SCHEMA_PATH = ROOT / "schemas" / "journey-v1.schema.json"
DEFINITION_FOR_FILE = {
    "workspace.json": "workspace",
    "catalog.json": "catalog",
    "journey.json": "journey",
    "stops.json": "stops_file",
    "mementos.json": "mementos_file",
    "plan.json": "plan",
}


def validate_workspace_root(root: Path) -> None:
    manifest_path = root / "catalog.json"
    if not manifest_path.is_file():
        manifest_path = root / "workspace.json"
    validate_document(manifest_path)
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    root_resolved = root.resolve()
    seen_paths: set[Path] = set()
    seen_ids: set[str] = set()
    for entry in manifest["journeys"]:
        journey_path = (root / entry["path"]).resolve()
        try:
            journey_path.relative_to(root_resolved)
        except ValueError as error:
            raise ValueError(f"catalog journey escapes root: {entry['path']}") from error
        if journey_path in seen_paths:
            raise ValueError(f"catalog journey path is duplicated: {entry['path']}")
        if entry["id"] in seen_ids:
            raise ValueError(f"catalog journey ID is duplicated: {entry['id']}")
        seen_paths.add(journey_path)
        seen_ids.add(entry["id"])
    for entry in manifest["journeys"]:
        journey_path = (root / entry["path"]).resolve()
        if not journey_path.is_dir():
            raise ValueError(f"catalog journey directory is missing: {entry['path']}")
        validate_workspace(journey_path)
        journey = json.loads((journey_path / "journey.json").read_text(encoding="utf-8"))
        if journey["id"] != entry["id"]:
            raise ValueError(f"catalog journey ID mismatch: {entry['path']}")
        if journey["journal_id"] != manifest["journal_id"]:
            raise ValueError(f"catalog journal ID mismatch: {entry['path']}")
        if entry.get("slug") and journey.get("slug") != entry["slug"]:
            raise ValueError(f"catalog journey slug mismatch: {entry['path']}")


def validate_document(path: Path) -> None:
    try:
        definition = DEFINITION_FOR_FILE[path.name]
    except KeyError as error:
        raise ValueError(f"unsupported journey contract file: {path.name}") from error
    document = json.loads(path.read_text(encoding="utf-8"))
    schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))
    selected = {"$schema": schema["$schema"], "$defs": schema["$defs"], "$ref": f"#/$defs/{definition}"}
    validator = Draft202012Validator(selected, format_checker=FormatChecker())
    errors = sorted(validator.iter_errors(document), key=lambda error: list(error.path))
    if errors:
        location = ".".join(str(part) for part in errors[0].path) or "document"
        raise ValueError(f"{path.name}:{location}: {errors[0].message}")


def validate_workspace(workspace: Path) -> None:
    for filename in ("journey.json", "stops.json", "mementos.json"):
        path = workspace / filename
        if not path.is_file():
            raise ValueError(f"{filename} is required in {workspace}")
        validate_document(path)
    plan = workspace / "plan.json"
    if plan.is_file():
        validate_document(plan)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("workspace", type=Path)
    args = parser.parse_args()
    workspace = args.workspace.resolve()
    if (workspace / "catalog.json").is_file() or (workspace / "workspace.json").is_file():
        validate_workspace_root(workspace)
        print(f"publication catalog valid: {args.workspace}")
    else:
        validate_workspace(workspace)
        print(f"local journey valid: {args.workspace}")
