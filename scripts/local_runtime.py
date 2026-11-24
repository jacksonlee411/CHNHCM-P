"""Local-only configuration and ownership checks shared by runtime/test entrypoints."""
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request
from urllib.parse import urlparse


def read_private(path, default=None):
    if path.is_symlink():
        raise RuntimeError(f"Runtime file must not be a symlink: {path.name}")
    if not path.exists():
        return default
    if not path.is_file() or path.stat().st_mode & 0o077:
        raise RuntimeError(f"Private runtime file must be a regular 0600 file: {path.name}")
    try:
        return json.loads(path.read_text())
    except (ValueError, OSError) as error:
        raise RuntimeError(f"Invalid runtime file: {path.name}") from error


def selected_profile(preference):
    name = os.environ.get("CHNHCM_LOCAL_PROFILE")
    if name is None:
        name = read_private(preference, "default")
    if not isinstance(name, str) or not name.replace("-", "").replace("_", "").isalnum():
        raise RuntimeError("Local profile must contain only letters, digits, - or _")
    return name


ROOT = Path(__file__).resolve().parents[1]
STATE_ROOT = Path.home() / ".local" / "state" / "chnhcm"
PREFERENCE = STATE_ROOT / hashlib.sha256(str(ROOT).encode()).hexdigest()[:12] / "preferred-profile.json"
PROFILE = selected_profile(PREFERENCE)
KEY = hashlib.sha256((str(ROOT) + (":" + PROFILE if PROFILE != "default" else "")).encode()).hexdigest()[:12]
BIN = ROOT / ".local" / "bin"
if PROFILE != "default":
    BIN = BIN / PROFILE
STATE = STATE_ROOT / KEY
CONFIG = STATE / "runtime.json"
PORTS = STATE / "ports.json"
PROCESSES = STATE / "processes.json"
PROFILE_INFO = STATE / "profile.json"
PROJECT = "chnhcm-bu-" + KEY
DEFAULT_PORTS = {"web": 4380, "api": 8380, "database": 54370}


def register_profile(temporary=None, purpose=None):
    info = read_private(PROFILE_INFO, {})
    if temporary:
        if PROFILE == "default":
            raise RuntimeError("Temporary environments require a named profile")
        if not info.get("temporary") and (CONFIG.exists() or database_container()):
            raise RuntimeError("An existing persistent database cannot become temporary")
    info.update(worktree=str(ROOT), profile=PROFILE, project=PROJECT)
    if temporary is not None:
        info["temporary"] = temporary
    info.setdefault("temporary", False)
    info["purpose"] = purpose or info.get("purpose") or ("临时验收" if info["temporary"] else "开发与手工验证")
    write_private(PROFILE_INFO, info)
    return info


def registered_profile():
    info = read_private(PROFILE_INFO, {})
    if info and (info.get("worktree"), info.get("profile"), info.get("project")) != (str(ROOT), PROFILE, PROJECT):
        raise RuntimeError("Profile metadata does not belong to this checkout")
    return info


def temporary_profile():
    return registered_profile().get("temporary") is True and PROFILE != "default"


def profile_inventory():
    # Docker labels discover old profiles; profile.json adds purpose without
    # opening runtime.json, which contains database credentials.
    containers = [json.loads(line) for line in docker_read(["ps", "-a", "--format", "json"]).splitlines()]
    containers = [c for c in containers if c["Names"].startswith("chnhcm-bu-")]
    ids = [c["ID"] for c in containers]
    details = []
    if ids:
        details = [json.loads(line) for line in docker_read(["inspect", "--format",
            '{"labels":{{json .Config.Labels}},"status":{{json .State.Status}},"networks":{{json .NetworkSettings.Networks}},"bindings":{{json .HostConfig.PortBindings}}}', *ids]).splitlines()]
    networks = [json.loads(line) for line in docker_read(["network", "ls", "--format", "json"]).splitlines()]
    network_ids = [n["ID"] for n in networks if n["Name"].startswith("chnhcm-bu-")]
    subnets = {}
    if network_ids:
        for line in docker_read(["network", "inspect", "--format",
            '{"name":{{json .Name}},"ipam":{{json .IPAM.Config}}}', *network_ids]).splitlines():
            n = json.loads(line)
            subnets[n["name"]] = [c.get("Subnet", "") for c in n["ipam"] or []]
    rows = {}
    for c in details:
        labels = c["labels"] or {}
        project = labels.get("com.docker.compose.project", "")
        if not project.startswith("chnhcm-bu-"):
            continue
        worktree = str((Path(labels.get("com.docker.compose.project.working_dir", "")) / "../..").resolve())
        key = project.removeprefix("chnhcm-bu-")
        profile = labels.get("io.chnhcm.profile")
        if not profile:
            candidates = {"default"}
            for folder in (Path(worktree) / ".local/bin", Path(worktree) / ".local/runtime"):
                if folder.is_dir():
                    candidates.update(p.name for p in folder.iterdir() if p.is_dir())
            profile = next((name for name in candidates if hashlib.sha256(
                (worktree + (":" + name if name != "default" else "")).encode()).hexdigest()[:12] == key), key)
        rows[key] = dict(worktree=worktree, profile=profile, project=project,
                         purpose="既有持久环境", temporary=False, status=c["status"],
                         ports={"database": int(c["bindings"]["5432/tcp"][0]["HostPort"])} if c["bindings"].get("5432/tcp") else {},
                         subnets=[s for n in c["networks"] for s in subnets.get(n, [])])
    base = STATE.parent
    if base.is_dir():
        for folder in base.iterdir():
            info = read_private(folder / "profile.json", {})
            if not info or not info.get("project", "").startswith("chnhcm-bu-"):
                continue
            row = rows.setdefault(folder.name, dict(status="未启动/已移除", subnets=[]))
            row.update(info)
    for key, row in rows.items():
        row["ports"] = read_private(base / key / "ports.json", row.get("ports", {}))
        if not row["subnets"]:
            row["subnets"] = subnets.get(row["project"] + "_default", [])
    return sorted(rows.values(), key=lambda row: (row["worktree"], row["profile"]))


def ensure_state(directory=None):
    directory = STATE if directory is None else directory
    if directory.is_symlink() or directory.resolve().is_relative_to(ROOT):
        raise RuntimeError("Runtime state must be outside the checkout and not a symlink")
    directory.mkdir(parents=True, mode=0o700, exist_ok=True)
    os.chmod(directory, 0o700)


def write_private(path, data):
    ensure_state(path.parent)
    if path.is_symlink():
        raise RuntimeError("Refusing to replace a symlink in runtime state")
    fd, temporary = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent)
    try:
        with os.fdopen(fd, "w") as out:
            json.dump(data, out, indent=2)
            out.write("\n")
        os.replace(temporary, path)
    finally:
        Path(temporary).unlink(missing_ok=True)


@contextmanager
def lifecycle_lock():
    ensure_state()
    path = STATE / "runtime.lock"
    fd = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise RuntimeError("Another runtime command owns this checkout's lifecycle lock") from error
        yield
    finally:
        os.close(fd)


def validate_ports(ports):
    if not isinstance(ports, dict) or set(ports) != set(DEFAULT_PORTS):
        raise RuntimeError("Ports must contain exactly web, api and database")
    if any(type(port) is not int or not 1024 <= port <= 65535 for port in ports.values()):
        raise RuntimeError("Each port must be an integer between 1024 and 65535")
    if len(set(ports.values())) != 3:
        raise RuntimeError("Web, API and database ports must be distinct")
    return ports


def ports():
    return validate_ports(read_private(PORTS, DEFAULT_PORTS.copy()))


def connection(selected=None):
    selected = ports() if selected is None else validate_ports(selected)
    result = dict(key=KEY, profile=PROFILE, project=PROJECT, ports=selected,
                  webOrigin=f"http://127.0.0.1:{selected['web']}",
                  apiOrigin=f"http://127.0.0.1:{selected['api']}")
    # Configuration is explicit and persisted once. Environment variables cannot
    # silently redirect just Vite or a test to another checkout.
    expected = {"CHNHCM_WEB_PORT": str(selected["web"]),
                "CHNHCM_POSTGRES_PORT": str(selected["database"]),
                "CHNHCM_API_TARGET": result["apiOrigin"],
                "CHNHCM_ORIGIN": result["webOrigin"],
                "CHNHCM_HTTP_ADDR": f"127.0.0.1:{selected['api']}"}
    for key, value in expected.items():
        if key in os.environ and os.environ[key] != value:
            raise RuntimeError(f"{key} conflicts with this checkout's saved ports; use configure")
    return result


def process_info(pid):
    try:
        fields = Path(f"/proc/{pid}/stat").read_text().rsplit(") ", 1)[1].split()
        if fields[0] == "Z":
            return None
        return dict(start=fields[19], group=int(fields[2]), session=int(fields[3]),
                    cwd=Path(f"/proc/{pid}/cwd").resolve(strict=True))
    except (FileNotFoundError, ProcessLookupError, PermissionError):
        return None


def processes():
    return read_private(PROCESSES, {})


def owned_process(proc):
    info = process_info(proc["pid"])
    if not info or info["start"] != proc["start"]:
        return False
    if info["cwd"] != ROOT or info["group"] != proc["pid"] or info["session"] != proc["pid"]:
        raise RuntimeError("Recorded live process no longer belongs to this checkout; refusing to stop it")
    return True


def group_members(group):
    result = []
    for directory in Path("/proc").iterdir():
        if directory.name.isdigit():
            info = process_info(int(directory.name))
            if info and info["group"] == group:
                result.append(int(directory.name))
    return result


def stop_processes(names=None):
    saved = processes()
    selected = saved if names is None else {name: proc for name, proc in saved.items() if name in names}
    live = [proc for proc in selected.values() if owned_process(proc)]
    if any(not owned_process(proc) and group_members(proc["pid"]) for proc in selected.values()):
        raise RuntimeError("Process leader changed but its group remains; records retained for inspection")
    # Validate every entry before sending any signal.
    for proc in live:
        if owned_process(proc):
            try:
                os.killpg(proc["pid"], signal.SIGTERM)
            except ProcessLookupError:
                pass
    deadline = time.monotonic() + 8
    while any(group_members(proc["pid"]) for proc in live):
        if time.monotonic() > deadline:
            raise RuntimeError("Owned process group did not stop; process records were retained")
        time.sleep(0.1)
    remaining = {name: proc for name, proc in saved.items() if name not in selected}
    if remaining:
        write_private(PROCESSES, remaining)
    else:
        PROCESSES.unlink(missing_ok=True)


def port_available(port):
    with socket.socket() as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(("127.0.0.1", port))
            return True
        except OSError:
            return False


def listener_pids(port):
    # API and Vite explicitly bind IPv4 loopback. Other IPv4 addresses and
    # IPv6-only listeners can share the port without owning that endpoint.
    # An unowned dual-stack listener still fails the IPv4 bind in preflight.
    loopback = f"{int.from_bytes(socket.inet_aton('127.0.0.1'), sys.byteorder):08X}"
    inodes = set()
    for line in Path("/proc/net/tcp").read_text().splitlines()[1:]:
        fields = line.split()
        address, local_port = fields[1].split(":")
        if fields[3] == "0A" and address in (loopback, "00000000") and int(local_port, 16) == port:
            inodes.add(fields[9])
    owners = set()
    for directory in Path("/proc").iterdir():
        if not directory.name.isdigit():
            continue
        try:
            for fd in (directory / "fd").iterdir():
                try:
                    value = os.readlink(fd)
                    if value.startswith("socket:[") and value[8:-1] in inodes:
                        owners.add(int(directory.name))
                except (FileNotFoundError, PermissionError):
                    pass
        except (FileNotFoundError, PermissionError):
            pass
    return owners


def docker_read(args):
    result = subprocess.run(["docker", *args], cwd=ROOT, text=True, capture_output=True)
    if result.returncode:
        raise RuntimeError("Docker ownership inspection failed; no runtime was changed")
    return result.stdout.strip()


def database_container():
    ids = docker_read(["ps", "-aq", "--filter", f"label=com.docker.compose.project={PROJECT}",
                       "--filter", "label=com.docker.compose.service=postgres"]).split()
    if len(ids) > 1:
        raise RuntimeError("Expected at most one database container for this checkout")
    if not ids:
        volumes = docker_read(["volume", "ls", "-q", "--filter", f"name=^{PROJECT}_database$"])
        if volumes:
            raise RuntimeError("Database volume exists without its container; restore its original configuration")
        return None
    # Select only non-sensitive fields; never capture Docker's environment block.
    data = json.loads(docker_read(["inspect", "--format",
        '{"running":{{json .State.Running}},"bindings":{{json .HostConfig.PortBindings}},"mounts":{{json .Mounts}}}', ids[0]]))
    expected = [{"HostIp": "127.0.0.1", "HostPort": str(ports()["database"])}]
    if data["bindings"].get("5432/tcp") != expected:
        raise RuntimeError("Database port binding differs from saved configuration; refusing migration or stop")
    if not any(m.get("Name") == PROJECT + "_database" and m["Destination"] == "/var/lib/postgresql" for m in data["mounts"]):
        raise RuntimeError("Database volume does not belong to this checkout")
    return dict(id=ids[0], running=data["running"])


def preflight(conn, database):
    saved = processes()
    for proc in saved.values():
        owned_process(proc)
    for name in ("web", "api"):
        port = conn["ports"][name]
        if port_available(port):
            continue
        proc = saved.get(name)
        owners = listener_pids(port)
        if not proc or not owned_process(proc) or not owners or any(
                not (info := process_info(pid)) or info["group"] != proc["pid"] for pid in owners):
            raise RuntimeError(f"{name} port {port} is occupied by an unowned service; nothing was stopped")
    if not (database and database["running"]) and not port_available(conn["ports"]["database"]):
        raise RuntimeError("Database port is occupied by another service; nothing was stopped")


def configure(selected):
    connection(selected)
    if selected == ports():
        return
    if processes() or database_container():
        raise RuntimeError("Ports cannot change after runtime initialization; existing processes/data were retained")
    for port in selected.values():
        if not port_available(port):
            raise RuntimeError(f"Requested port {port} is occupied; configuration was not saved")
    write_private(PORTS, selected)


def http_json(address):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with opener.open(address, timeout=2) as result:
        if result.status != 200:
            raise RuntimeError("Local runtime readiness check failed")
        return json.load(result)


def verify_api(conn):
    proc = processes().get("api")
    if not proc or not owned_process(proc) or listener_pids(conn["ports"]["api"]) != {proc["pid"]}:
        raise RuntimeError("API listener is not the recorded process for this checkout")
    if Path(f"/proc/{proc['pid']}/exe").resolve() != BIN / "server":
        raise RuntimeError("API executable is not this checkout's build")
    # Inspect in memory only; /proc environment includes credentials and is never
    # copied to diagnostics. Readiness alone cannot identify the intended instance.
    env = dict(value.split("=", 1) for value in Path(f"/proc/{proc['pid']}/environ").read_text().split("\0") if "=" in value)
    config = read_private(CONFIG)
    database = urlparse(env.get("CHNHCM_DATABASE_URL", ""))
    if (not config or env.get("CHNHCM_INSTANCE_ID") != config["instance"]
            or env.get("CHNHCM_ORIGIN") != conn["webOrigin"]
            or env.get("CHNHCM_HTTP_ADDR") != f"127.0.0.1:{conn['ports']['api']}"
            or database.hostname != "127.0.0.1" or database.port != conn["ports"]["database"]
            or database.path != "/chnhcm" or database.username != "chnhcm_api"
            or database.password != config["api_password"]):
        raise RuntimeError("Owned API instance, Origin or database connection differs from this checkout")
    if http_json(conn["apiOrigin"] + "/readyz") != {"status": "ready"}:
        raise RuntimeError("Owned API is not ready")


def vite_process(pid):
    info = process_info(pid)
    if not info or info["cwd"] not in (ROOT, ROOT / "apps/web"):
        return False
    try:
        args = Path(f"/proc/{pid}/cmdline").read_bytes().decode().split("\0")
        for arg in args:
            if not arg:
                continue
            path = (info["cwd"] / arg).resolve()
            if path.is_relative_to(ROOT / "node_modules") and str(path).endswith("/vite/bin/vite.js"):
                return True
    except (OSError, ValueError):
        pass
    return False


def verify_target(conn, origin=None):
    if origin is not None and origin != conn["webOrigin"]:
        raise RuntimeError("Target origin differs from this checkout's saved Web origin")
    verify_api(conn)
    owners = listener_pids(conn["ports"]["web"])
    if not owners or not all(vite_process(pid) for pid in owners):
        raise RuntimeError("Web listener is not this checkout's Vite process")
    marker = http_json(conn["webOrigin"] + "/__chnhcm_local_runtime")
    expected = {key: conn[key] for key in ("key", "webOrigin", "apiOrigin")}
    if marker != expected or http_json(conn["webOrigin"] + "/readyz") != {"status": "ready"}:
        raise RuntimeError("Web/API identity or proxy readiness mismatch")
