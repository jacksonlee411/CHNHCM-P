#!/usr/bin/env python3
"""Clear only the stopped R6 capacity volume's page cache, then restore its DB.

Never drops host caches, removes a volume, changes credentials, or touches another
profile. The task API is restarted first to clear its immutable fact cache, then reconnects to the same database.
"""
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.request
import local_runtime as runtime


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def main():
    if os.environ.get("CHNHCM_LOCAL_PROFILE") != "r6-capacity-b-1106":
        raise SystemExit("requires the exact isolated r6-capacity-b-1106 profile")
    if not runtime.temporary_profile():
        raise SystemExit("R6 cache resets require an explicitly temporary profile")
    conn = json.loads(run("python3", "scripts/local-bu.py", "connection"))
    ids = run("docker", "ps", "-q", "--filter", f"label=com.docker.compose.project={conn['project']}",
              "--filter", "label=com.docker.compose.service=postgres").split()
    if len(ids) != 1:
        raise SystemExit("expected one running task database")
    labels = json.loads(run("docker", "inspect", "--format", "{{json .Config.Labels}}", ids[0]))
    if labels.get("com.docker.compose.project.working_dir") != str(Path.cwd() / "deploy/local"):
        raise SystemExit("task container checkout mismatch")
    # Only request the mount metadata: full inspection exposes container secrets.
    mounts = json.loads(run("docker", "inspect", "--format", "{{json .Mounts}}", ids[0]))
    volume = conn["project"] + "_database"
    if not any(m["Type"] == "volume" and m["Name"] == volume and
               m["Destination"] == "/var/lib/postgresql" for m in mounts):
        raise SystemExit("task volume ownership mismatch")
    binary = Path(".local/bin/r6-cache").resolve()
    if not binary.is_file():
        raise SystemExit("build CGO_ENABLED=0 go build -o .local/bin/r6-cache ./tests/capacity/cache first")
    image = Path("deploy/local/postgres-image.txt").read_text().strip()
    restarted = json.loads(run("python3", "scripts/r6-api.py", "restart"))
    run("docker", "stop", "--time", "30", ids[0])
    try:
        helper = subprocess.run(["docker", "run", "--rm", "--network", "none", "--read-only", "--user", "postgres",
                  "--mount", f"type=volume,src={volume},dst=/data,readonly",
                  "--mount", f"type=bind,src={binary},dst=/r6-cache,readonly",
                  "--env", "CHNHCM_T07_CACHE=1", "--entrypoint", "/r6-cache", image],
                  text=True, stdout=subprocess.PIPE, check=False)
        try:
            result = json.loads(helper.stdout)
        except ValueError:
            result = {"passed": False, "error": "cache helper did not emit a JSON result"}
        result["helperReturnCode"] = helper.returncode
    finally:
        run("docker", "start", ids[0])
    deadline = time.monotonic() + 30
    while True:
        try:
            with urllib.request.urlopen(conn["apiOrigin"] + "/readyz", timeout=2) as response:
                if response.status == 200:
                    break
        except (OSError, TimeoutError):
            if time.monotonic() >= deadline:
                raise SystemExit("task API did not reconnect after its database restart")
            time.sleep(0.2)
    result.update(profile="r6-capacity-b-1106", volume=volume,
                  apiBinarySHA256=restarted["apiBinarySHA256"], apiRebuilt=False,
                  scope="API fact cache empty; data files cold before DB restart; startup and readiness then warm infrastructure pages")
    print(json.dumps(result))
    if helper.returncode != 0 or result.get("passed") is not True:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
