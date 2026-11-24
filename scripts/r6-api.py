#!/usr/bin/env python3
"""Restart the exact measured API build in the explicitly disposable R6 profile.

Cold resets must not rebuild the current checkout and silently replace a frozen
baseline binary. Normal local-bu development commands retain their build step.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil

import local_runtime as runtime


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("stop", "start", "restart"))
    parser.add_argument("--binary", type=Path)
    parser.add_argument("--sha256")
    args = parser.parse_args()
    if runtime.PROFILE != "r6-capacity-b-1106" or not runtime.temporary_profile():
        raise SystemExit("requires this checkout's explicitly temporary R6 profile")
    if (args.binary is None) != (args.sha256 is None) or (args.command == "stop" and args.binary):
        raise SystemExit("select a frozen binary with both --binary and --sha256")
    spec = importlib.util.spec_from_file_location("r6_local_bu", runtime.ROOT / "scripts/local-bu.py")
    local = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(local)
    with runtime.lifecycle_lock():
        conn = runtime.connection()
        binary = runtime.BIN / "server"
        selected = args.binary.resolve() if args.binary else binary
        if not selected.is_relative_to(runtime.ROOT / ".local/bin") or not selected.is_file():
            raise SystemExit("measured binary must be inside this checkout's .local/bin")
        sha = digest(selected)
        if args.sha256 and sha != args.sha256:
            raise SystemExit("measured binary digest mismatch; running process retained")
        proc = runtime.processes().get("api")
        running = proc and runtime.owned_process(proc)
        if running:
            runtime.verify_api(conn)
            if args.command == "start":
                raise SystemExit("API already running; select restart explicitly")
        elif args.command == "restart":
            raise SystemExit("API is not running; select start explicitly")
        if running:
            runtime.stop_processes(["api"])
        if args.command != "stop":
            if not runtime.port_available(conn["ports"]["api"]):
                raise SystemExit("task API port is occupied")
            if selected != binary:
                temporary = binary.with_name("server.r6-selected")
                if temporary.exists() or temporary.is_symlink():
                    raise SystemExit("selected binary staging path already exists")
                shutil.copyfile(selected, temporary)
                temporary.chmod(0o755)
                os.replace(temporary, binary)
            env = local.environment(local.configuration(), conn)
            env = {k: v for k, v in env.items() if k not in
                   ("CHNHCM_ADMIN_DATABASE_URL", "CHNHCM_POSTGRES_PASSWORD")}
            local.start("api", [str(binary)], env, runtime.processes())
            local.wait_ready(conn["apiOrigin"] + "/readyz")
            runtime.verify_api(conn)
            if digest(binary) != sha:
                raise RuntimeError("API build changed during restart")
        print(json.dumps(dict(profile=runtime.PROFILE, command=args.command,
                              apiBinarySHA256=sha, rebuilt=False)))


if __name__ == "__main__":
    main()
