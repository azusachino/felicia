import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts import format as formatting
from scripts import local_checks as checks


class LocalChecksTest(unittest.TestCase):
    def test_go_format_group_does_not_run_frontend_formatters(self):
        with patch.object(checks.sys, "argv", ["format.py", "--check", "--only", "go"]), \
             patch.object(formatting, "format_go", return_value=True) as go, \
             patch.object(formatting, "format_web", return_value=True) as web:
            self.assertEqual(formatting.main(), 0)
            go.assert_called_once_with(True)
            web.assert_not_called()

    def test_boundaries_invalidate_dependents(self):
        for path, groups in {
            "apps/felicia-server/api/x.go": {"go", "scripts"},
            "apps/felicia-admin/src/App.svelte": {"admin", "scripts"},
            "apps/felicia-public-site/src/App.svelte": {"public", "scripts"},
            "packages/felicia-reader/src/x.ts": {"admin", "public", "scripts"},
            "scripts/local_journey_common.py": {"scripts"},
            "tests/test_local_journey.py": {"scripts"},
            "docs/roadmap.md": {"docs"},
            "scripts/go_tasks.py": set(checks.GROUPS),
            "mise.toml": set(checks.GROUPS),
            "unknown/config": set(checks.GROUPS),
        }.items():
            with self.subTest(path=path):
                self.assertEqual(checks.groups_for(path), groups)
                before = {path: b"before"}
                after = {path: b"after"}
                for group in checks.GROUPS:
                    self.assertEqual(
                        checks.fingerprint(group, before, {}) != checks.fingerprint(group, after, {}),
                        group in groups,
                    )

    def test_make_selectors_do_not_poison_next_hit(self):
        plain = {"PATH": "/tools", "MAKEFLAGS": "", "MAKELEVEL": "1"}
        forced = {**plain, "MAKEFLAGS": " -- ARGS=--force", "MAKEOVERRIDES": "ARGS=--force", "ARGS": "--force"}
        self.assertEqual(checks.environment_key(plain), checks.environment_key(forced))
        self.assertNotEqual(checks.environment_key(plain), checks.environment_key({**plain, "GOFLAGS": "-race"}))
        self.assertNotIn("MAKEFLAGS", checks.child_environment(forced))
        self.assertNotIn("ARGS", checks.child_environment(forced))

    def test_environment_and_commands_invalidate(self):
        self.assertNotEqual(checks.fingerprint("go", {}, {"environment": "a"}),
                            checks.fingerprint("go", {}, {"environment": "b"}))
        self.assertNotEqual(checks.fingerprint("go", {}, {"go": "1"}),
                            checks.fingerprint("go", {}, {"go": "2"}))

    def test_hit_force_failure_and_changed_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(checks, "snapshot", return_value={"go.work": b"a"}) as snapshot, \
                 patch.object(checks.subprocess, "run") as run:
                run.return_value.returncode = 0
                self.assertEqual(checks.run_cached(root, "go", {}), 0)
                self.assertEqual(checks.run_cached(root, "go", {}), 0)
                self.assertEqual(run.call_count, 1)
                self.assertEqual(checks.run_cached(root, "go", {}, force=True), 0)
                self.assertEqual(run.call_count, 2)
                run.return_value.returncode = 1
                self.assertEqual(checks.run_cached(root, "go", {}, force=True), 1)
                self.assertEqual(checks.run_cached(root, "go", {}), 1)
                self.assertEqual(run.call_count, 4)
                self.assertFalse((root / ".cache/local-checks/go.json").exists())
                run.return_value.returncode = 0
                snapshot.side_effect = [{"go.work": b"a"}, {"go.work": b"b"}]
                with self.assertRaisesRegex(RuntimeError, "inputs changed"):
                    checks.run_cached(root, "go", {})

    def test_corrupt_record_runs_again(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            record = root / ".cache/local-checks/go.json"
            record.parent.mkdir(parents=True)
            record.write_text("not json")
            with patch.object(checks, "snapshot", return_value={}), patch.object(checks.subprocess, "run") as run:
                run.return_value.returncode = 0
                self.assertEqual(checks.run_cached(root, "go", {}), 0)
                run.assert_called_once()
                self.assertIn("key", json.loads(record.read_text()))

    def test_version_mismatch_and_missing_tool_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "mise.toml").write_text('[tools]\ngo = "latest"\n')
            with patch.object(checks.shutil, "which", return_value="/bin/go"), \
                 patch.object(checks.subprocess, "check_output", side_effect=[
                     json.dumps({"go": [{"version": "1.27.0", "source": {"path": str(root / "mise.toml")}}]}),
                     "go version go1.26.7 darwin/arm64",
                 ]):
                with self.assertRaisesRegex(RuntimeError, "does not match pin"):
                    checks.verify_tools(root)
            with patch.object(checks.shutil, "which", return_value=None):
                with self.assertRaisesRegex(RuntimeError, "missing"):
                    checks.verify_tools(root)

    def test_matching_tools_allow_direct_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "mise.toml").write_text(
                '[tools]\ngo="latest"\nuv="latest"\nnode="lts"\nbun="latest"\ngolangci-lint="latest"\n'
            )
            selected = {name: [{"version": version, "source": {"path": str(root / "mise.toml")}}] for name, version in {
                "go": "1.27.0", "uv": "0.12.10", "node": "24.21.0",
                "bun": "1.4.2", "golangci-lint": "2.14.0",
            }.items()}
            # An ancestor's selection must not override this project's Bun.
            selected["bun"].append({"version": "99.0.0", "source": {"path": str(root.parent / "mise.toml")}})
            with patch.object(checks.shutil, "which", return_value="/bin/tool"), \
                 patch.object(checks.subprocess, "check_output", side_effect=[
                     json.dumps(selected), "go version go1.27.0 darwin/arm64", "uv 0.12.10",
                     "v24.21.0", "1.4.2", "version 2.14.0", "make version", "rumdl version",
                 ]) as output:
                identities = checks.verify_tools(root)
                self.assertIn(":1.27.0", identities["go"])
                self.assertIn(":24.21.0", identities["node"])
                self.assertIn(":1.4.2", identities["bun"])
                output.assert_any_call(["mise", "ls", "--local", "--current", "--json"], cwd=root, text=True)
                self.assertIn("make version", identities["make"])

    def test_ignored_dotenv_changes_invalidate_frontend(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            dotenv = root / "apps/felicia-admin/.env.local"
            dotenv.parent.mkdir(parents=True)
            dotenv.write_text("PUBLIC_MODE=one")
            with patch.object(checks.subprocess, "check_output", return_value=b""):
                before = checks.snapshot(root)
                dotenv.write_text("PUBLIC_MODE=two")
                self.assertNotEqual(checks.fingerprint("admin", before, {}), checks.fingerprint("admin", checks.snapshot(root), {}))

    def test_preflight_runs_even_when_group_is_cached(self):
        with patch.dict(checks.os.environ, {}, clear=True), \
             patch.object(checks.sys, "argv", ["local_checks.py", "--groups", "docs"]), \
             patch.object(checks, "verify_tools", return_value={}), \
             patch.object(checks.platform, "platform", return_value="test-host"), \
             patch.object(checks, "run_cached", return_value=0) as cached, \
             patch.object(checks.subprocess, "run") as run:
            self.assertEqual(checks.main(), 0)
            self.assertEqual(checks.main(), 0)
            self.assertEqual(cached.call_count, 2)
            self.assertEqual(run.call_count, 4)
            self.assertEqual(run.call_args_list[0].args[0], ["bun", "install", "--frozen-lockfile"])
            self.assertEqual(run.call_args_list[1].args[0], ["make", "MISE_RUN=", "layout-check"])

    def test_ci_refuses_local_cache(self):
        with patch.dict(checks.os.environ, {"CI": "1"}), patch.object(checks.sys, "argv", ["local_checks.py"]):
            with self.assertRaises(SystemExit) as error:
                checks.main()
            self.assertEqual(error.exception.code, 2)

    def test_dangling_symlink_target_changes_invalidate(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "apps/felicia-admin/src/link.ts"
            source.parent.mkdir(parents=True)
            source.symlink_to("missing-one.ts")
            with patch.object(checks.subprocess, "check_output", return_value=b"apps/felicia-admin/src/link.ts\0"):
                before = checks.snapshot(root)
                source.unlink()
                source.symlink_to("missing-two.ts")
                self.assertNotEqual(checks.fingerprint("admin", before, {}),
                                    checks.fingerprint("admin", checks.snapshot(root), {}))

    def test_snapshot_includes_untracked_deleted_and_symlink_targets(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "new.ts"
            source.write_text("a")
            with patch.object(checks.subprocess, "check_output", return_value=b"new.ts\0gone.ts\0"):
                before = checks.snapshot(root)
                self.assertIn("new.ts", before)
                self.assertEqual(before["gone.ts"], b"deleted")
                source.write_text("b")
                self.assertNotEqual(before, checks.snapshot(root))
                source.unlink()
                target = root / "target.ts"
                target.write_text("first")
                source.symlink_to("target.ts")
                before = checks.snapshot(root)
                target.write_text("second")
                self.assertNotEqual(before, checks.snapshot(root))
                source.unlink()
                source.symlink_to("/tmp")
                with self.assertRaisesRegex(RuntimeError, "escapes checkout"):
                    checks.snapshot(root)


if __name__ == "__main__":
    unittest.main()
