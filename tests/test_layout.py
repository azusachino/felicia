import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
READER = ROOT / "packages" / "felicia-reader"
MODEL = ROOT / "packages" / "felicia-model"
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
            ROOT / "apps" / "felicia-public-site",
            READER,
            READER / "src" / "theme-ui" / "atlas" / "Atlas.svelte",
            MODEL / "src" / "themes.ts",
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
            "apps/felicia-web",
            "apps/felicia-providers/postgres",
            "packages/felicia-components",
            "packages/felicia-renderers",
            "packages/felicia-runtime",
            "packages/felicia-reader/src/v1",
            "packages/felicia-reader/src/v2",
            "packages/felicia-reader/src/v3",
            "packages/felicia-reader/src/v4",
            "packages/felicia-reader/src/theme-ui/cabinet",
            "packages/felicia-reader/src/theme-ui/techo",
            "packages/felicia-reader/src/theme-ui/cartography",
        )
        for path in legacy_paths:
            self.assertFalse((ROOT / path).exists(), path)

    def test_frontend_packages_have_no_host_or_transport_dependency(self):
        source = "\n".join(
            path.read_text(encoding="utf-8")
            for package in (READER, MODEL)
            for path in package.rglob("*")
            if path.is_file()
        )
        forbidden_imports = (
            "apps/felicia-public-site",
            "import.meta.env",
            "api/source",
        )
        for forbidden in forbidden_imports:
            self.assertNotIn(forbidden, source, forbidden)

        index_src = (READER / "src" / "index.ts").read_text(encoding="utf-8")
        self.assertIn("Reader", index_src)

        package = json.loads((READER / "package.json").read_text(encoding="utf-8"))
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
