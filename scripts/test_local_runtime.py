"""Exercise isolation boundaries with task-owned sockets/processes and external state."""
import os
import json
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import local_runtime as runtime


class LocalRuntimeTest(unittest.TestCase):
    def test_named_profile_owns_separate_state_and_binary(self):
        script = "import json,runpy,sys; r=runpy.run_path(sys.argv[1]); print(json.dumps([r['KEY'],str(r['CONFIG']),r['PROJECT'],str(r['BIN'])]))"
        source = str(Path(runtime.__file__).resolve())
        def profile(name):
            env = dict(os.environ, CHNHCM_LOCAL_PROFILE=name)
            return json.loads(subprocess.check_output([sys.executable, '-B', '-c', script, source], env=env, text=True))
        default, first, second = profile('default'), profile('test-first'), profile('test-second')
        for index in range(4):
            self.assertEqual(len({default[index], first[index], second[index]}), 3)
        self.assertTrue(first[3].endswith('/.local/bin/test-first'))
        self.assertTrue(second[3].endswith('/.local/bin/test-second'))

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="chnhcm-runtime-test-")
        self.addCleanup(self.temp.cleanup)
        base = Path(self.temp.name)
        self.root = base / "checkout"
        self.root.mkdir()
        self.state = base / "private"
        overrides = dict(ROOT=self.root, STATE=self.state, CONFIG=self.state / "runtime.json",
                         PORTS=self.state / "ports.json", PROCESSES=self.state / "processes.json")
        for name, value in overrides.items():
            fixture = patch.object(runtime, name, value)
            fixture.start()
            self.addCleanup(fixture.stop)
        fixture = patch.dict(os.environ, {}, clear=True)
        fixture.start()
        self.addCleanup(fixture.stop)

    def listener(self, host="127.0.0.1"):
        sock = socket.socket()
        sock.bind((host, 0))
        sock.listen()
        self.addCleanup(sock.close)
        return sock.getsockname()[1]

    def child(self, cwd=None):
        process = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"],
                                   cwd=cwd or self.root, start_new_session=True)
        def cleanup():
            if process.poll() is None:
                process.terminate()
            process.wait(timeout=5)
        self.addCleanup(cleanup)
        return process, dict(pid=process.pid, start=runtime.process_info(process.pid)["start"])

    def test_default_and_configured_connection_never_read_credentials(self):
        self.assertEqual(runtime.connection()["ports"], runtime.DEFAULT_PORTS)
        selected = dict(web=44381, api=44382, database=44383)
        runtime.write_private(runtime.PORTS, selected)
        # Even a deliberately invalid secret file cannot affect connection discovery.
        runtime.CONFIG.write_text("not json and not a credential")
        self.assertEqual(runtime.connection()["webOrigin"], "http://127.0.0.1:44381")
        self.assertEqual(runtime.connection()["apiOrigin"], "http://127.0.0.1:44382")
        self.assertEqual(runtime.PORTS.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.state.stat().st_mode & 0o777, 0o700)

    def test_checkout_default_is_local_and_explicit_profile_takes_precedence(self):
        preference = self.state / "chosen-checkout" / "preferred-profile.json"
        other = self.state / "other-checkout" / "preferred-profile.json"
        self.assertEqual(runtime.selected_profile(preference), "default")
        runtime.write_private(preference, "acceptance-d")
        self.assertEqual(runtime.selected_profile(preference), "acceptance-d")
        self.assertEqual(runtime.selected_profile(other), "default")
        with patch.dict(os.environ, {"CHNHCM_LOCAL_PROFILE": "smoke-test"}):
            self.assertEqual(runtime.selected_profile(preference), "smoke-test")
        runtime.write_private(preference, "../other")
        with self.assertRaises(RuntimeError):
            runtime.selected_profile(preference)

    def test_stopping_only_api_retains_live_web_and_its_record(self):
        api, api_record = self.child()
        web, web_record = self.child()
        runtime.write_private(runtime.PROCESSES, dict(api=api_record, web=web_record))
        runtime.stop_processes(['api'])
        api.wait(timeout=5)
        self.assertIsNotNone(api.poll())
        self.assertIsNone(web.poll())
        self.assertEqual(runtime.processes(), dict(web=web_record))

    def test_invalid_or_duplicate_ports_fail_without_writing_state(self):
        for selected in [dict(web=80, api=8380, database=54370),
                         dict(web=True, api=8380, database=54370),
                         dict(web=8380, api=8380, database=54370),
                         dict(web=4380, api=8380),
                         dict(web=4380, api=8380, database=65536)]:
            with self.subTest(selected=selected), self.assertRaises(RuntimeError):
                runtime.configure(selected)
        self.assertFalse(self.state.exists())

    def test_environment_cannot_redirect_one_consumer(self):
        with patch.dict(os.environ, {"CHNHCM_API_TARGET": "http://127.0.0.1:1"}):
            with self.assertRaisesRegex(RuntimeError, "conflicts"):
                runtime.connection()

    def test_occupied_port_leaves_configuration_and_service_untouched(self):
        port = self.listener()
        with patch.object(runtime, "database_container", return_value=None):
            with self.assertRaisesRegex(RuntimeError, "occupied"):
                runtime.configure(dict(web=port, api=1025, database=1026))
        self.assertFalse(runtime.PORTS.exists())
        self.assertIn(os.getpid(), runtime.listener_pids(port))

    def test_initialized_database_blocks_port_change(self):
        with patch.object(runtime, "database_container", return_value={"running": False}):
            with self.assertRaisesRegex(RuntimeError, "initialization"):
                runtime.configure(dict(web=44381, api=44382, database=44383))
        self.assertFalse(runtime.PORTS.exists())

    def test_preflight_rejects_foreign_web_and_api_without_signalling(self):
        for name in ("web", "api"):
            conn = runtime.connection()
            conn["ports"][name] = port = self.listener()
            with self.subTest(name=name), self.assertRaisesRegex(RuntimeError, "unowned"):
                runtime.preflight(conn, None)
            self.assertIn(os.getpid(), runtime.listener_pids(port))
        self.assertFalse(runtime.CONFIG.exists())

    def test_target_mismatch_fails_before_http(self):
        with patch.object(runtime, "http_json") as request:
            with self.assertRaisesRegex(RuntimeError, "origin differs"):
                runtime.verify_target(runtime.connection(), "http://127.0.0.1:1")
            request.assert_not_called()

    def test_unowned_api_cannot_be_accepted_by_healthy_http(self):
        conn = runtime.connection()
        conn["ports"]["api"] = self.listener()
        with patch.object(runtime, "http_json", return_value={"status": "ready"}) as request:
            with self.assertRaisesRegex(RuntimeError, "recorded process"):
                runtime.verify_api(conn)
            request.assert_not_called()

    def test_listener_ownership_ignores_same_port_on_another_ipv4_address(self):
        port = self.listener()
        child = subprocess.Popen([sys.executable, "-u", "-c",
            "import socket,sys,time; s=socket.socket(); "
            "s.bind(('127.0.0.2',int(sys.argv[1]))); s.listen(); "
            "print('ready',flush=True); time.sleep(60)", str(port)],
            cwd=self.root, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
            text=True, start_new_session=True)
        def cleanup():
            if child.poll() is None:
                child.terminate()
            child.wait(timeout=5)
            child.stdout.close()
        self.addCleanup(cleanup)
        self.assertEqual(child.stdout.readline().strip(), "ready")
        self.assertEqual(runtime.listener_pids(port), {os.getpid()})
        self.assertIsNone(child.poll())

    def test_ipv4_wildcard_conflict_is_still_identified_and_rejected(self):
        port = self.listener("0.0.0.0")
        self.assertIn(os.getpid(), runtime.listener_pids(port))
        conn = runtime.connection()
        conn["ports"]["web"] = port
        with self.assertRaisesRegex(RuntimeError, "unowned"):
            runtime.preflight(conn, None)

    def ipv6_listener(self, v6only):
        if not socket.has_ipv6:
            self.skipTest("IPv6 unavailable")
        sock = socket.socket(socket.AF_INET6)
        self.addCleanup(sock.close)
        sock.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, v6only)
        try:
            sock.bind(("::", 0))
        except OSError:
            self.skipTest("IPv6 loopback unavailable")
        sock.listen()
        return sock.getsockname()[1]

    def test_ipv6_only_listener_does_not_own_ipv4_endpoint(self):
        port = self.ipv6_listener(1)
        self.assertTrue(runtime.port_available(port))
        self.assertEqual(runtime.listener_pids(port), set())

    def test_dual_stack_conflict_is_rejected_before_start(self):
        port = self.ipv6_listener(0)
        self.assertFalse(runtime.port_available(port))
        conn = runtime.connection()
        conn["ports"]["web"] = port
        with self.assertRaisesRegex(RuntimeError, "unowned"):
            runtime.preflight(conn, None)

    def test_stop_owns_only_recorded_process_and_waits(self):
        owned, record = self.child()
        foreign, _ = self.child()
        runtime.write_private(runtime.PROCESSES, {"api": record})
        runtime.stop_processes()
        self.assertIsNotNone(owned.poll())
        self.assertIsNone(foreign.poll())
        self.assertFalse(runtime.PROCESSES.exists())

    def test_foreign_cwd_prevents_all_signals_and_retains_records(self):
        owned, record = self.child()
        foreign, wrong = self.child(cwd=self.root.parent)
        runtime.write_private(runtime.PROCESSES, {"api": record, "web": wrong})
        before = runtime.PROCESSES.read_bytes()
        with self.assertRaisesRegex(RuntimeError, "no longer belongs"):
            runtime.stop_processes()
        self.assertIsNone(owned.poll())
        self.assertIsNone(foreign.poll())
        self.assertEqual(runtime.PROCESSES.read_bytes(), before)

    def test_changed_start_time_never_signals_reused_pid(self):
        foreign, record = self.child()
        record["start"] = "0"
        runtime.write_private(runtime.PROCESSES, {"api": record})
        with self.assertRaisesRegex(RuntimeError, "leader changed"):
            runtime.stop_processes()
        self.assertIsNone(foreign.poll())
        self.assertTrue(runtime.PROCESSES.exists())

    def test_lifecycle_lock_rejects_overlapping_mutation(self):
        with runtime.lifecycle_lock():
            with self.assertRaisesRegex(RuntimeError, "lifecycle lock"):
                with runtime.lifecycle_lock():
                    self.fail("overlapping lifecycle commands were accepted")

    def test_external_files_reject_symlinks_and_public_permissions(self):
        runtime.write_private(runtime.PORTS, runtime.DEFAULT_PORTS)
        runtime.PORTS.chmod(0o644)
        with self.assertRaisesRegex(RuntimeError, "0600"):
            runtime.connection()
        runtime.PORTS.unlink()
        target = self.root / "must-not-change.json"
        target.write_text("preserved")
        runtime.PORTS.symlink_to(target)
        with self.assertRaisesRegex(RuntimeError, "symlink"):
            runtime.write_private(runtime.PORTS, {})
        self.assertEqual(target.read_text(), "preserved")


if __name__ == "__main__":
    unittest.main()
