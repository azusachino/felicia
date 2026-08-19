import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SHARED = ROOT / "packages" / "felicia-shared"
GO_IMPORT = re.compile(r"github\.com/azusachino/felicia/apps/felicia-([a-z-]+)")


class LayoutBoundaryTests(unittest.TestCase):
    def test_target_tree_and_contract_authority_exist(self):
        for path in (
            ROOT / "apps" / "felicia-core",
            ROOT / "apps" / "felicia-runtime",
            ROOT / "apps" / "felicia-providers",
            ROOT / "apps" / "felicia-publication",
            ROOT / "apps" / "felicia-server",
            ROOT / "apps" / "felicia-cli",
            ROOT / "apps" / "felicia-admin",
            ROOT / "apps" / "felicia-web",
            ROOT / "apps" / "felicia-public-site",
            SHARED,
            SHARED / "src" / "theme-ui" / "registry.ts",
            SHARED / "src" / "theme-ui" / "themes.ts",
            ROOT / "ops",
            ROOT / "contracts" / "canonical" / "v1" / "schema.json",
            ROOT / "publication" / "journeys" / "catalog.json",
        ):
            self.assertTrue(path.exists(), path)

    def test_legacy_roots_are_absent(self):
        legacy_paths = (
            "core",
            "runtime",
            "providers",
            "server",
            "cli",
            "deploy",
            "apps/web-admin",
            "apps/web-public",
            "packages/felicia-shared/src/v1",
            "packages/felicia-shared/src/v2",
            "packages/felicia-shared/src/v3",
            "packages/felicia-shared/src/v4",
        )
        for path in legacy_paths:
            self.assertFalse((ROOT / path).exists(), path)

    def test_shared_package_has_no_host_or_transport_dependency(self):
        source = "\n".join(
            path.read_text(encoding="utf-8") for path in SHARED.rglob("*") if path.is_file()
        )
        forbidden_imports = (
            "apps/felicia-web",
            "apps/felicia-public-site",
            "import.meta.env",
            "api/source",
        )
        for forbidden in forbidden_imports:
            self.assertNotIn(forbidden, source, forbidden)

        registry = (SHARED / "src" / "theme-ui" / "registry.ts").read_text(encoding="utf-8")
        for old_id in ("v1", "v2", "v3", "v4"):
            self.assertNotIn(f'id: "{old_id}"', registry, old_id)

        package = json.loads((SHARED / "package.json").read_text(encoding="utf-8"))
        self.assertIn(".", package["exports"])
        self.assertIn("./public.css", package["exports"])

    def test_go_dependencies_point_inward(self):
        forbidden = {
            "felicia-core": {"runtime", "providers", "publication", "server", "cli"},
            "felicia-runtime": {"providers", "server", "cli"},
            "felicia-publication": {"runtime", "providers", "server", "cli"},
        }
        for app, disallowed in forbidden.items():
            for path in (ROOT / "apps" / app).rglob("*.go"):
                imports = set(GO_IMPORT.findall(path.read_text(encoding="utf-8")))
                self.assertFalse(imports & disallowed, f"{path}: {imports & disallowed}")


if __name__ == "__main__":
    unittest.main()
