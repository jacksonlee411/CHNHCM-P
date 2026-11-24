#!/usr/bin/env python3
"""Clear only the stopped T07 capacity volume's page cache, then restore its DB.

Never drops host caches, removes a volume, changes credentials, or touches another
profile. The API stays running and reconnects to the same current database.
"""
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.request


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def main():
    if os.environ.get("CHNHCM_LOCAL_PROFILE") != "m02-t07-capacity":
        raise SystemExit("requires the exact isolated m02-t07-capacity profile")
    conn = json.loads(run("python3", "scripts/local-bu.py", "connection"))
    ids = run("docker", "ps", "-q", "--filter", f"label=com.docker.compose.project={conn['project']}",
              "--filter", "label=com.docker.compose.service=postgres").split()
    if len(ids) != 1:
        raise SystemExit("expected one running task database")
    # Only request the mount metadata: full inspection exposes container secrets.
    mounts = json.loads(run("docker", "inspect", "--format", "{{json .Mounts}}", ids[0]))
    volume = conn["project"] + "_database"
    if not any(m["Type"] == "volume" and m["Name"] == volume and
               m["Destination"] == "/var/lib/postgresql" for m in mounts):
        raise SystemExit("task volume ownership mismatch")
    binary = Path(".local/bin/t07-cache").resolve()
    if not binary.is_file():
        raise SystemExit("build CGO_ENABLED=0 go build -o .local/bin/t07-cache ./tests/capacity/cache first")
    image = Path("deploy/local/postgres-image.txt").read_text().strip()
    run("docker", "stop", "--time", "30", ids[0])
    try:
        raw = run("docker", "run", "--rm", "--network", "none", "--read-only", "--user", "postgres",
                  "--mount", f"type=volume,src={volume},dst=/data,readonly",
                  "--mount", f"type=bind,src={binary},dst=/t07-cache,readonly",
                  "--env", "CHNHCM_T07_CACHE=1", "--entrypoint", "/t07-cache", image)
        result = json.loads(raw)
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
    result.update(profile="m02-t07-capacity", volume=volume,
                  scope="data files cold before DB restart; startup and readiness then warm infrastructure pages")
    print(json.dumps(result))


if __name__ == "__main__":
    main()
