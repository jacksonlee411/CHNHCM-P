#!/usr/bin/env python3
"""Own only this checkout's BU runtime; credentials are stored outside the repository."""
import argparse
import hashlib
import json
import os
import secrets
import shutil
import signal
import socket
import subprocess
import sys
import time
import urllib.request
import uuid

import local_runtime as runtime
from local_runtime import (ROOT, CONFIG, PROCESSES, PROJECT, read_private,
                           write_private, processes, stop_processes)

BIN = runtime.BIN
LOG = ROOT / ".local" / "runtime" / runtime.PROFILE
DEVELOPMENT_LOGIN_PASSWORD = "admin1234567"


def configuration(create=False):
    if CONFIG.exists():
        return read_private(CONFIG)
    if not create:
        raise RuntimeError("runtime not initialized; run up first")
    config = dict(instance=str(uuid.uuid4()), database_password=secrets.token_urlsafe(32),
                  api_password=secrets.token_urlsafe(32),
                  login="admin", password=DEVELOPMENT_LOGIN_PASSWORD,
                  reader_login="reader", reader_password=DEVELOPMENT_LOGIN_PASSWORD)
    write_private(CONFIG, config)
    return config


def environment(config, conn):
    env = os.environ.copy()
    env.update(CHNHCM_POSTGRES_IMAGE=(ROOT / "deploy/local/postgres-image.txt").read_text().strip(),
               CHNHCM_LOCAL_PROFILE=runtime.PROFILE,
               CHNHCM_TEMPORARY=str(runtime.temporary_profile()).lower(),
               CHNHCM_POSTGRES_SHM_SIZE=str(config.get("database_shm_size", "64m")),
               CHNHCM_POSTGRES_PASSWORD=config["database_password"], CHNHCM_POSTGRES_PORT=str(conn["ports"]["database"]),
               CHNHCM_INSTANCE_ID=config["instance"], CHNHCM_ORIGIN=conn["webOrigin"],
               CHNHCM_INSECURE_LOCAL="1", CHNHCM_HTTP_ADDR=f"127.0.0.1:{conn['ports']['api']}")
    for key, user, password in [("CHNHCM_ADMIN_DATABASE_URL", "postgres", "database_password"),
                               ("CHNHCM_DATABASE_URL", "chnhcm_api", "api_password")]:
        env[key] = f"postgres://{user}:{config[password]}@127.0.0.1:{conn['ports']['database']}/chnhcm?sslmode=disable"
    return env


def run(args, env, data=None):
    # Never interpolate credentials into shell commands or command-line arguments.
    subprocess.run(args, cwd=ROOT, env=env, input=json.dumps(data) if data else None,
                   text=True, check=True)


def compose(env, *args):
    run(["docker", "compose", "-p", PROJECT, "-f", "deploy/local/compose.yaml", *args], env)


def synchronize_login_passwords(config, env):
    for login_key, password_key in [("login", "password"), ("reader_login", "reader_password")]:
        if config[password_key] == DEVELOPMENT_LOGIN_PASSWORD:
            continue
        run([str(BIN / "upgrader"), "-mode", "local-password"], env,
            dict(Instance=config["instance"], Login=config[login_key],
                 PreviousPassword=config[password_key], Password=DEVELOPMENT_LOGIN_PASSWORD))
        config[password_key] = DEVELOPMENT_LOGIN_PASSWORD
        write_private(CONFIG, config)


def start_existing_database(env):
    args = ["docker", "compose", "-p", PROJECT, "-f", "deploy/local/compose.yaml"]
    existing = subprocess.run(args + ["ps", "--all", "--quiet", "postgres"],
                              cwd=ROOT, env=env, text=True, capture_output=True, check=True)
    if existing.stdout.strip():
        # start retains the existing container's WAL setting; up may recreate it.
        ids = existing.stdout.split()
        if len(ids) != 1:
            raise RuntimeError("Expected one database container for this checkout")
        compose(env, "start", "postgres")
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            state = subprocess.run(["docker", "inspect", "--format",
                                    "{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}", ids[0]],
                                   env=env, text=True, capture_output=True, check=True).stdout.strip()
            if state == "running healthy":
                return True
            if state.startswith(("exited", "dead")) or state == "running unhealthy":
                raise RuntimeError("Existing database did not start with its original configuration")
            time.sleep(0.2)
        raise RuntimeError("Existing database readiness timeout; its configuration was retained")
    volume = subprocess.run(["docker", "volume", "inspect", PROJECT + "_database"],
                            env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if volume.returncode == 0:
        raise RuntimeError("Database volume exists without its container; restore the original database configuration before migrating. No volume was changed.")
    return False


def start_stamp(pid):
    info = runtime.process_info(pid)
    return info["start"] if info else None


def start(name, args, env, owned):
    LOG.mkdir(parents=True, exist_ok=True)
    with (LOG / f"{name}.log").open("ab") as out:
        p = subprocess.Popen(args, cwd=ROOT, env=env, stdout=out, stderr=subprocess.STDOUT,
                             start_new_session=True, stdin=subprocess.DEVNULL)
    owned[name] = dict(pid=p.pid, start=start_stamp(p.pid))
    write_private(PROCESSES, owned)


def wait_ready(address):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        try:
            with opener.open(address, timeout=1) as result:
                if result.status == 200:
                    return
        except OSError:
            pass
        time.sleep(0.2)
    raise RuntimeError("runtime readiness timeout; inspect sanitized logs in .local/runtime")


def destroy_temporary():
    resource_commands = (["ps", "-aq", "--filter", f"label=com.docker.compose.project={PROJECT}"],
                         ["network", "ls", "-q", "--filter", f"label=com.docker.compose.project={PROJECT}"],
                         ["volume", "ls", "-q", "--filter", f"label=com.docker.compose.project={PROJECT}"])
    if not runtime.temporary_profile():
        # An absent profile is an idempotent cleanup; an old database is not.
        if not runtime.PROFILE_INFO.exists() and not CONFIG.exists() and not any(
                runtime.docker_read(command) for command in resource_commands):
            print("Temporary profile is already absent.")
            return
        raise RuntimeError("Only an explicitly temporary profile can be destroyed; persistent data retained")
    ids = runtime.docker_read(["ps", "-aq", "--filter", f"label=com.docker.compose.project={PROJECT}"]).split()
    if ids:
        for line in runtime.docker_read(["inspect", "--format", "{{json .Config.Labels}}", *ids]).splitlines():
            labels = json.loads(line)
            if labels.get("com.docker.compose.project.working_dir") != str(ROOT / "deploy/local"):
                raise RuntimeError("Compose container belongs to another checkout; nothing was removed")
    stop_processes()
    config = read_private(CONFIG)
    if config:
        env = environment(config, runtime.connection())
    else:
        env = dict(os.environ, CHNHCM_POSTGRES_IMAGE=(ROOT / "deploy/local/postgres-image.txt").read_text().strip(),
                   CHNHCM_POSTGRES_PASSWORD="unused", CHNHCM_POSTGRES_PORT=str(runtime.ports()["database"]))
    compose(env, "down", "--volumes")
    for command in resource_commands:
        if runtime.docker_read(command):
            raise RuntimeError("Temporary resources remain; profile metadata retained for retry")
    if BIN.exists():
        shutil.rmtree(BIN)
    for path in (CONFIG, runtime.PORTS, runtime.PROFILE_INFO):
        path.unlink(missing_ok=True)
    print("Removed this temporary profile's containers, network, volume and private configuration; sanitized logs retained.")


def install_smoke_policy_fixture(command, env):
    connection = json.loads(subprocess.check_output(command + ["connection"], cwd=ROOT, env=env))
    with (ROOT / "tests/fixtures/access/policy.sql").open() as fixture:
        subprocess.run(["docker", "exec", "-i", connection["project"] + "-postgres-1",
                        "psql", "-Xq", "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "chnhcm"],
                       cwd=ROOT, env=env, stdin=fixture, check=True)


def smoke(args):
    profile = args.profile or "smoke-" + uuid.uuid4().hex[:12]
    env = dict(os.environ, CHNHCM_LOCAL_PROFILE=profile)
    command = ["python3", "scripts/local-bu.py"]
    # Bind together so the three selected ports cannot coincide. Preflight still
    # detects any competing allocation after these sockets are released.
    sockets = [socket.socket() for _ in range(3)]
    try:
        for sock in sockets:
            sock.bind(("127.0.0.1", 0))
        ports = [sock.getsockname()[1] for sock in sockets]
    finally:
        for sock in sockets:
            sock.close()
    configured = False
    primary_error = None
    try:
        run(command + ["configure", "--temporary", "--purpose", "自动浏览器验收",
                       "--web-port", str(ports[0]), "--api-port", str(ports[1]), "--db-port", str(ports[2])], env)
        configured = True
        print(f"Temporary Chromium profile: {profile}; Web=http://127.0.0.1:{ports[0]}; API={ports[1]}; DB={ports[2]}", flush=True)
        run(command + ["up", "--first-hire", "--sensitive-profile"] + (["--fixed-web"] if getattr(args, "fixed_web", False) else []), env)
        install_smoke_policy_fixture(command, env)
        extra = args.playwright_args
        if extra[:1] == ["--"]:
            extra = extra[1:]
        client = subprocess.Popen(["corepack", "pnpm", "test:smoke", *extra], cwd=ROOT, env=env,
                                  start_new_session=True)
        try:
            result = client.wait()
            if result:
                raise subprocess.CalledProcessError(result, "Chromium business flows")
        finally:
            if client.poll() is None:
                os.killpg(client.pid, signal.SIGTERM)
                try:
                    client.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    os.killpg(client.pid, signal.SIGKILL)
                    client.wait()
    except BaseException as error:
        primary_error = error
        raise
    finally:
        if configured:
            try:
                run(command + ["destroy"], env)
            except (OSError, subprocess.CalledProcessError):
                print(f"Temporary cleanup failed; retry with CHNHCM_LOCAL_PROFILE={profile} python3 scripts/local-bu.py destroy", flush=True)
                if primary_error is None:
                    raise


def execute(args):
    if args.command == "list":
        print(json.dumps(runtime.profile_inventory(), ensure_ascii=False, indent=2))
        return
    if args.command == "set-default":
        if runtime.PROFILE != "default" and runtime.registered_profile().get("temporary") is not False:
            raise RuntimeError("Select a registered persistent profile of this checkout")
        write_private(runtime.PREFERENCE, runtime.PROFILE)
        print("Checkout default profile:", runtime.PROFILE)
        return
    if args.command == "destroy":
        destroy_temporary()
        return
    if args.command == "configure":
        if args.temporary:
            runtime.register_profile(temporary=True, purpose=args.purpose)
        runtime.configure(dict(web=args.web_port, api=args.api_port, database=args.db_port))
        runtime.register_profile(purpose=args.purpose)
        print("Saved checkout ports:", json.dumps(runtime.connection()["ports"]))
        return
    conn = runtime.connection()
    if args.command == "connection":
        print(json.dumps(conn))
        return
    if args.command == "verify":
        if args.api_only:
            runtime.verify_api(conn)
        else:
            runtime.verify_target(conn, args.origin)
        print("Verified this checkout's " + ("API" if args.api_only else "Web/API") + " ownership and readiness.")
        return
    if args.command == "credentials-path":
        configuration()
        print(CONFIG)
        return
    database = runtime.database_container()
    if args.command in ("up", "migrate"):
        # Inspect ownership and collisions before credentials, signals or DB writes.
        runtime.preflight(conn, database)
    if args.command == "up":
        runtime.register_profile(temporary=True if args.temporary else None, purpose=args.purpose)
    config = configuration(args.command == "up")
    env = environment(config, conn)
    if args.command == "access-admin":
        # Recovery remains available while either application listener is down.
        # database_container() verified the profile, bound port and retained volume.
        if not database or not database["running"]:
            raise RuntimeError("Access recovery requires this profile's running database")
        request = json.load(sys.stdin)
        request.setdefault("instanceId", config["instance"])
        run([str(BIN / "upgrader"), "-mode", "access-admin"], env, request)
        return
    if args.command == "login-passwords":
        runtime.verify_api(conn)
        run(["go", "build", "-o", str(BIN / "upgrader"), "./cmd/upgrader"], env)
        synchronize_login_passwords(config, env)
        print("Applied the development login password; database credentials and business data retained.")
        return
    if args.command == "restart-api":
        runtime.verify_api(conn)
        runtime.stop_processes(["api"])
        owned = processes()
        run(["go", "build", "-o", str(BIN / "server"), "./cmd/server"], env)
        api_env = {k: v for k, v in env.items() if k not in ["CHNHCM_ADMIN_DATABASE_URL", "CHNHCM_POSTGRES_PASSWORD"]}
        start("api", [str(BIN / "server")], api_env, owned)
        wait_ready(conn["apiOrigin"] + "/readyz")
        runtime.verify_api(conn)
        print("Restarted this profile's API; frontend and database retained.")
        return
    if args.command == "migrate":
        stop_processes()
        BIN.mkdir(parents=True, exist_ok=True)
        run(["go", "build", "-o", str(BIN / "upgrader"), "./cmd/upgrader"], env)
        if not start_existing_database(env):
            raise RuntimeError("No existing database to migrate; use up for a fresh installation")
        run([str(BIN / "upgrader"), "-mode", "migrate"], env)
        write_private(CONFIG, config)
        print("Migration complete using the running database; run up to apply its current configuration.")
        return
    if args.command == "down":
        stop_processes()
        compose(env, "stop")
        print("Stopped this checkout's runtime; database volume retained.")
        return
    if args.command == "status":
        compose(env, "ps")
        for name, proc in processes().items():
            print(name, "running" if start_stamp(proc["pid"]) == proc["start"] else "stopped")
        return
    stop_processes()
    BIN.mkdir(parents=True, exist_ok=True)
    for name in ["server", "upgrader"]:
        run(["go", "build", "-o", str(BIN / name), "./cmd/" + name], env)
    if start_existing_database(env):
        run([str(BIN / "upgrader"), "-mode", "migrate"], env)
    compose(env, "up", "-d", "--wait")
    run([str(BIN / "upgrader"), "-mode", "migrate"], env)
    synchronize_login_passwords(config, env)
    env = environment(config, conn)
    run([str(BIN / "upgrader"), "-mode", "runtime-roles"], env,
        dict(API=config["api_password"]))
    enable_sensitive = args.sensitive_profile and not config.get("sensitive_profile")
    if enable_sensitive:
        config["sensitive_profile"] = True
    enable_hire = args.first_hire and not config.get("first_hire")
    if enable_hire:
        config["first_hire"] = True
    for login, password, read_only in [(config["login"], config["password"], False),
                                        (config["reader_login"], config["reader_password"], True)]:
        enterprise = config.get("enterprise", dict(EnterpriseCode="LOCAL", EnterpriseName="本地学习企业"))
        temporary = runtime.temporary_profile()
        declaration = dict(Instance=config["instance"], **enterprise, Login=login, Password=password,
                           ReadOnly=read_only, DevelopmentFixture=temporary)
        if temporary:
            declaration["NewAccountFixture"] = dict(PersonnelProfile="reader" if read_only else "registrar",
                                                    SensitiveProfile=bool(config.get("sensitive_profile") and not read_only),
                                                    FirstHire=bool(config.get("first_hire") and not read_only))
        if enable_sensitive and not read_only:
            declaration["SensitiveProfileFixture"] = True
        if enable_hire and not read_only:
            declaration.update(HireIdentityFixture=True, EmploymentFixture=True)
        if config.get("first_hire") and not read_only:
            declaration["FirstHireRule"] = dict(Version="T02-employee-1", From="2027-04-01", HistoricalFixtureFrom="2026-01-01")
        run([str(BIN / "upgrader"), "-mode", "initialize"], env, declaration)
    for catalog in ("country-catalog", "geoarea-catalog"):
        run([str(BIN / "upgrader"), "-mode", catalog, "-skip-unauthorized-catalog"], env,
            dict(login=config["login"], password=config["password"]))
    write_private(CONFIG, config)
    # Each process receives only the credential class it needs.
    owned = {}
    api_env = {k: v for k, v in env.items() if k not in ["CHNHCM_ADMIN_DATABASE_URL", "CHNHCM_POSTGRES_PASSWORD"]}
    web_env = {k: v for k, v in env.items() if not k.startswith("CHNHCM_")}
    web_env["CHNHCM_LOCAL_PROFILE"] = runtime.PROFILE
    start("api", [str(BIN / "server")], api_env, owned)
    wait_ready(conn["apiOrigin"] + "/readyz")
    runtime.verify_api(conn)
    if not args.no_web:
        info = read_private(runtime.PROFILE_INFO, {})
        fixed_web = args.fixed_web or info.get("fixedWeb", False)
        if fixed_web:
            # A retained environment serves its own build, so later source edits
            # cannot silently change its frontend while its API stays frozen.
            dist = BIN / "web"
            run(["corepack", "pnpm", "--filter", "@chnhcm/web", "exec", "vite", "build", "--outDir", str(dist)], web_env)
            sources = subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "--",
                "cmd", "internal", "db", "api", "apps/web", "packages", "go.mod", "go.sum", "pnpm-lock.yaml"], cwd=ROOT, text=True).splitlines()
            manifest = dict(baseCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                sourceSHA256={name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest() for name in sorted(set(sources)) if (ROOT / name).is_file()},
                apiSHA256=hashlib.sha256((BIN / "server").read_bytes()).hexdigest(),
                webSHA256={str(p.relative_to(dist)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(dist.rglob("*")) if p.is_file()})
            (BIN / "fixed-build.json").write_text(json.dumps(manifest, indent=2)+"\n")
            info["fixedWeb"] = True
            write_private(runtime.PROFILE_INFO, info)
            command = ["corepack", "pnpm", "--filter", "@chnhcm/web", "exec", "vite", "preview", "--outDir", str(dist)]
        else:
            command = ["corepack", "pnpm", "dev"]
        start("web", command, web_env, owned)
        wait_ready(conn["webOrigin"] + "/__chnhcm_local_runtime")
        runtime.verify_target(conn)
    print(f"BU API ready: {conn['apiOrigin']}" if args.no_web else
          f"BU runtime ready: {conn['webOrigin']} (API: {conn['apiOrigin']})")
    print("Local credentials are stored outside the repository:", CONFIG)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    up = commands.add_parser("up")
    up.add_argument("--no-web", action="store_true", help="start database and API only")
    up.add_argument("--first-hire", action="store_true", help="enable the synthetic Employee hire rule and local operator grants")
    up.add_argument("--sensitive-profile", action="store_true", help="enable synthetic bank and health purposes for the local registrar")
    up.add_argument("--fixed-web", action="store_true", help="serve a profile-owned fixed frontend build with this API")
    up.add_argument("--temporary", action="store_true", help="mark a new named profile as disposable")
    up.add_argument("--purpose", help="short local environment purpose")
    for command in ("down", "status", "credentials-path", "migrate", "connection", "restart-api", "login-passwords", "access-admin"):
        commands.add_parser(command)
    configure = commands.add_parser("configure")
    configure.add_argument("--temporary", action="store_true")
    configure.add_argument("--purpose")
    for name in ("web", "api", "db"):
        configure.add_argument(f"--{name}-port", type=int, required=True)
    verify = commands.add_parser("verify")
    verify.add_argument("--origin", help="exact expected Web origin; checked before contacting services")
    verify.add_argument("--api-only", action="store_true")
    commands.add_parser("list", help="list CHNHCM profiles without reading credentials")
    commands.add_parser("set-default", help="use the selected persistent profile as this checkout's default")
    commands.add_parser("destroy", help="remove only the selected explicitly temporary profile")
    browser = commands.add_parser("smoke", help="run Chromium in an automatically reclaimed temporary profile")
    browser.add_argument("--profile", help="optional named temporary profile, for CI cleanup")
    browser.add_argument("--fixed-web", action="store_true", help="verify a profile-owned production frontend build")
    browser.add_argument("playwright_args", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.command == "smoke":
        smoke(args)
    elif args.command in ("configure", "set-default", "up", "down", "destroy", "migrate", "restart-api", "login-passwords", "access-admin"):
        with runtime.lifecycle_lock():
            execute(args)
    else:
        execute(args)


if __name__ == "__main__":
    def cancel(signum, frame):
        raise KeyboardInterrupt
    signal.signal(signal.SIGTERM, cancel)
    try:
        main()
    except KeyboardInterrupt:
        raise SystemExit(130)
    except (RuntimeError, subprocess.CalledProcessError) as error:
        raise SystemExit(str(error))
    except (OSError, ValueError, KeyError, TypeError):
        raise SystemExit("Invalid local runtime state or unavailable service; no credential values are displayed.")
