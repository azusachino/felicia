"""Guards against the backend kind registry and the public frontend's kind
lists silently drifting apart again (ADMIN-01.7).

Text-level parsing only, kept dependency-light: the registry's `kind:` line
and the frontend's TypeScript literals are both grep-able without a YAML or
TypeScript parser.
"""

import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
KINDS_DIR = ROOT / "core" / "kinds"
DATA_TS = ROOT / "apps" / "web-public" / "src" / "data.ts"
STUBS_TS = ROOT / "apps" / "web-public" / "src" / "v4" / "stubs.ts"
AUTHORING_SCHEMA = ROOT / "schemas" / "local-authoring-v1.schema.json"

KIND_LINE = re.compile(r"^kind:\s*(\S+)\s*$", re.MULTILINE)
MEMENTO_KIND_UNION = re.compile(r'export type MementoKind\s*=\s*(.+)')
QUOTED = re.compile(r'"([^"]+)"')
STUB_TEMPLATES_START = re.compile(r"export const stubTemplates:.*=\s*\{")
STUB_KEY = re.compile(r"^\s*([A-Za-z_][A-Za-z0-9_]*):\s*\{")


def registry_kinds() -> set[str]:
    kinds: set[str] = set()
    for path in sorted(KINDS_DIR.glob("*.yaml")):
        text = path.read_text(encoding="utf-8")
        match = KIND_LINE.search(text)
        if not match:
            raise AssertionError(f"{path}: no top-level 'kind:' line found")
        kinds.add(match.group(1))
    return kinds


def data_ts_kinds() -> set[str]:
    text = DATA_TS.read_text(encoding="utf-8")
    match = MEMENTO_KIND_UNION.search(text)
    if not match:
        raise AssertionError(f"{DATA_TS}: no 'export type MementoKind =' line found")
    return set(QUOTED.findall(match.group(1)))


def stubs_ts_kinds() -> set[str]:
    text = STUBS_TS.read_text(encoding="utf-8")
    start = STUB_TEMPLATES_START.search(text)
    if not start:
        raise AssertionError(f"{STUBS_TS}: no 'stubTemplates' object literal found")
    body_start = start.end()
    close = text.index("\n}", body_start)
    body = text[body_start:close]
    keys = {m.group(1) for line in body.splitlines() for m in [STUB_KEY.match(line)] if m}
    if not keys:
        raise AssertionError(f"{STUBS_TS}: found the stubTemplates literal but parsed no keys")
    return keys


def authoring_schema_kinds() -> set[str]:
    schema = json.loads(AUTHORING_SCHEMA.read_text(encoding="utf-8"))
    return set(schema["$defs"]["memento"]["properties"]["kind"]["enum"])


class KindRegistryDriftTest(unittest.TestCase):
    """core/kinds/*.yaml is the source of truth (D8 soft enum); every kind it
    declares must also be a memento kind on the public frontend, and vice
    versa — a one-sided kind is exactly the drift this test exists to catch.
    """

    def test_frontend_data_ts_matches_registry_kinds(self):
        backend = registry_kinds()
        frontend = data_ts_kinds()
        self.assertEqual(
            backend,
            frontend,
            f"core/kinds/*.yaml vs data.ts MementoKind drift: "
            f"only in registry={backend - frontend}, only in data.ts={frontend - backend}",
        )

    def test_local_authoring_schema_matches_registry_kinds(self):
        # The workspace schema is the first gate a hand-authored memento passes,
        # and the import boundary now validates the kind against this same
        # registry (issue #77). A kind the registry declares but the schema
        # forbids cannot be authored locally at all; one the schema allows but
        # the registry does not know fails later, at import.
        backend = registry_kinds()
        authoring = authoring_schema_kinds()
        self.assertEqual(
            backend,
            authoring,
            f"core/kinds/*.yaml vs local-authoring-v1 memento kind drift: "
            f"only in registry={backend - authoring}, only in schema={authoring - backend}",
        )

    def test_frontend_stub_registry_matches_registry_kinds(self):
        backend = registry_kinds()
        frontend = stubs_ts_kinds()
        self.assertEqual(
            backend,
            frontend,
            f"core/kinds/*.yaml vs v4/stubs.ts stubTemplates drift: "
            f"only in registry={backend - frontend}, only in stubs.ts={frontend - backend}",
        )


if __name__ == "__main__":
    unittest.main()
