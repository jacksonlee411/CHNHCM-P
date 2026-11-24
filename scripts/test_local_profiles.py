"""Disposable lifecycle, persistent-data protection and safe inventory behavior."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import local_runtime as runtime

spec = importlib.util.spec_from_file_location("local_bu", Path(__file__).with_name("local-bu.py"))
bu = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bu)


class LocalProfilesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "checkout"
        self.root.mkdir()
        self.state = Path(self.temp.name) / "state" / "test-key"
        self.bin = self.root / ".local/bin/task-test"
        for name, value in dict(ROOT=self.root, PROFILE="task-test", KEY="test-key", PROJECT="chnhcm-bu-test-key",
                STATE=self.state, PROFILE_INFO=self.state / "profile.json", CONFIG=self.state / "runtime.json",
                PORTS=self.state / "ports.json", PROCESSES=self.state / "processes.json", BIN=self.bin).items():
            fixture = patch.object(runtime, name, value)
            fixture.start()
            self.addCleanup(fixture.stop)
            if hasattr(bu, name):
                fixture = patch.object(bu, name, value)
                fixture.start()
                self.addCleanup(fixture.stop)
        fixture = patch.dict(os.environ, {}, clear=True)
        fixture.start()
        self.addCleanup(fixture.stop)

    def register(self):
        with patch.object(runtime, "database_container", return_value=None):
            runtime.register_profile(temporary=True, purpose="测试任务")

    def test_existing_persistent_configuration_cannot_be_marked_disposable(self):
        runtime.write_private(runtime.CONFIG, {"preserve": "business history"})
        with self.assertRaisesRegex(RuntimeError, "cannot become temporary"):
            runtime.register_profile(temporary=True)
        self.assertFalse(runtime.PROFILE_INFO.exists())
        self.assertEqual(runtime.read_private(runtime.CONFIG), {"preserve": "business history"})

    def test_set_default_requires_a_persistent_profile_of_this_checkout(self):
        preference = self.state.parent / "checkout-default" / "preferred-profile.json"
        command = argparse.Namespace(command="set-default")
        with patch.object(runtime, "PREFERENCE", preference):
            with self.assertRaisesRegex(RuntimeError, "registered persistent"):
                bu.execute(command)
            self.register()
            with self.assertRaisesRegex(RuntimeError, "registered persistent"):
                bu.execute(command)
            info = runtime.read_private(runtime.PROFILE_INFO)
            info.update(temporary=False, worktree="/another/checkout")
            runtime.write_private(runtime.PROFILE_INFO, info)
            with self.assertRaisesRegex(RuntimeError, "does not belong"):
                bu.execute(command)
            self.assertFalse(preference.exists())
            info["worktree"] = str(self.root)
            runtime.write_private(runtime.PROFILE_INFO, info)
            bu.execute(command)
            self.assertEqual(runtime.selected_profile(preference), "task-test")

    def test_temporary_marker_survives_regular_start_and_rejects_foreign_identity(self):
        self.register()
        runtime.register_profile()
        self.assertTrue(runtime.temporary_profile())
        info = runtime.read_private(runtime.PROFILE_INFO)
        info["worktree"] = "/another/checkout"
        runtime.write_private(runtime.PROFILE_INFO, info)
        with self.assertRaisesRegex(RuntimeError, "does not belong"):
            runtime.temporary_profile()

    def test_persistent_destroy_rejected_before_stop_or_compose(self):
        runtime.register_profile()
        with patch.object(bu, "stop_processes") as stop, patch.object(bu, "compose") as compose:
            with self.assertRaisesRegex(RuntimeError, "persistent data retained"):
                bu.destroy_temporary()
            stop.assert_not_called()
            compose.assert_not_called()

    def test_orphan_network_without_temporary_marker_is_not_called_cleaned(self):
        def docker(command):
            return "orphan-network" if command[0] == "network" else ""
        with patch.object(runtime, "docker_read", side_effect=docker), patch.object(bu, "compose") as compose:
            with self.assertRaisesRegex(RuntimeError, "persistent data retained"):
                bu.destroy_temporary()
            compose.assert_not_called()

    def test_partial_start_without_credentials_can_be_destroyed_and_retried(self):
        self.register()
        self.bin.mkdir(parents=True)
        (self.bin / "server").write_text("temporary build")
        (self.root / "deploy/local").mkdir(parents=True)
        (self.root / "deploy/local/postgres-image.txt").write_text("postgres:local-test")
        with patch.object(runtime, "docker_read", return_value=""), patch.object(bu, "stop_processes"), patch.object(bu, "compose") as compose:
            bu.destroy_temporary()
            self.assertEqual(compose.call_args.args[1:], ("down", "--volumes"))
            self.assertFalse(runtime.PROFILE_INFO.exists())
            self.assertFalse(self.bin.exists())
            bu.destroy_temporary()
            self.assertEqual(compose.call_count, 1)

    def test_failed_cleanup_retains_marker_for_retry(self):
        self.register()
        (self.root / "deploy/local").mkdir(parents=True)
        (self.root / "deploy/local/postgres-image.txt").write_text("postgres:local-test")
        with patch.object(runtime, "docker_read", return_value=""), patch.object(bu, "stop_processes"), patch.object(bu, "compose", side_effect=RuntimeError("failed")):
            with self.assertRaisesRegex(RuntimeError, "failed"):
                bu.destroy_temporary()
        self.assertTrue(runtime.temporary_profile())

    def test_inventory_lists_stopped_legacy_and_new_profiles_without_credentials(self):
        self.register()
        runtime.CONFIG.write_text("must never be opened by inventory")
        runtime.CONFIG.chmod(0o000)
        def docker(args):
            if args[0] == "ps":
                return json.dumps(dict(Names="chnhcm-bu-legacy-postgres-1", ID="old"))
            if args[0] == "inspect":
                return json.dumps(dict(labels={"com.docker.compose.project": "chnhcm-bu-legacy",
                    "com.docker.compose.project.working_dir": str(self.root / "deploy/local")},
                    status="exited", networks={"chnhcm-bu-legacy_default": {}}, bindings={"5432/tcp": [{"HostPort": "54370"}]}))
            return ""
        with patch.object(runtime, "docker_read", side_effect=docker):
            rows = runtime.profile_inventory()
        self.assertEqual(len(rows), 2)
        self.assertEqual(next(r for r in rows if r["profile"] == "task-test")["purpose"], "测试任务")
        self.assertFalse(next(r for r in rows if r["profile"] == "legacy")["temporary"])
        self.assertEqual(next(r for r in rows if r["profile"] == "legacy")["ports"], {"database": 54370})

    def test_start_and_browser_failure_both_run_cleanup_and_keep_original_error(self):
        for failure in ("up", "browser"):
            calls = []
            def run(command, env):
                calls.append(command[2])
                if failure == "up" and command[2] == "up":
                    raise subprocess.CalledProcessError(17, "up")
            with self.subTest(failure=failure), patch.object(bu, "run", side_effect=run), patch.object(bu.subprocess, "Popen") as process:
                process.return_value.wait.return_value = 19
                process.return_value.poll.return_value = 19
                with self.assertRaises(subprocess.CalledProcessError) as error:
                    bu.smoke(argparse.Namespace(profile="task-test", playwright_args=[]))
                self.assertEqual(error.exception.returncode, 17 if failure == "up" else 19)
                self.assertEqual(calls, ["configure", "up", "destroy"])


if __name__ == "__main__":
    unittest.main()
