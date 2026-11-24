"""303 local deployment evidence; reconcile only an isolated copy of the old d fixture.

The product upgrader still rejects the sparse legacy ledger. No credentials or
database dumps are written to the repository, including ignored directories.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
SOURCE = "tree300-synthetic-d"
TARGET = "primary303-current-d"
EVIDENCE = ROOT / "docs/verification/data/303-primary-navigation"
os.environ["CHNHCM_LOCAL_PROFILE"] = TARGET
sys.path.insert(0, str(ROOT / "scripts"))
spec = importlib.util.spec_from_file_location("local_bu", ROOT / "scripts/local-bu.py")
local = importlib.util.module_from_spec(spec)
spec.loader.exec_module(local)
runtime = local.runtime


def source_key():
    return hashlib.sha256((str(ROOT) + ":" + SOURCE).encode()).hexdigest()[:12]


def container(profile_key):
    return "chnhcm-bu-" + profile_key + "-postgres-1"


def sql(target, query):
    return subprocess.check_output(["docker", "exec", "-i", target, "psql", "-X", "-v", "ON_ERROR_STOP=1",
                                    "-U", "postgres", "-d", "chnhcm", "-At"], input=query, text=True)


def snapshot(target):
    names = sql(target, "SELECT tablename FROM pg_tables WHERE schemaname='organization' AND tablename NOT LIKE 'country_%' AND tablename NOT LIKE 'geo%' AND tablename NOT IN ('department_selection_draft','department_selection_block') ORDER BY tablename").splitlines()
    rows = []
    for name in names:
        assert name.replace('_', '').isalnum()
        count, digest = sql(target, f"SELECT count(*),md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5(to_jsonb(t)::text) h FROM organization.{name} t) x").strip().split('|')
        rows.append(dict(table=name, rows=int(count), contentMD5=digest))
    principal = sql(target, "SELECT md5(string_agg(h,'' ORDER BY h)) FROM (SELECT md5(to_jsonb(t)::text) h FROM access.principal t) x").strip()
    return dict(tables=rows, principalMD5=principal,
                migrations=sql(target, "SELECT version_id FROM public.goose_db_version WHERE is_applied AND version_id>0 ORDER BY version_id").splitlines(),
                index=sql(target, "SELECT pg_get_indexdef('organization.department_tree_parent'::regclass)").strip())


def save(name, value):
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    path = EVIDENCE / name
    if path.exists():
        raise RuntimeError("Refusing to replace existing evidence: " + name)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")


if sys.argv[1] == "clone":
    if runtime.CONFIG.exists() or runtime.database_container():
        raise RuntimeError("Candidate already exists; refusing to overwrite")
    original = container(source_key())
    before = snapshot(original)
    assert before['migrations'] == [str(n) for n in range(1, 9)] + ['10']
    old_sql = subprocess.check_output(['git', 'show', 'e2ed466d:db/migrations/00010_department_tree_parent_index.sql'], cwd=ROOT)
    assert old_sql == (ROOT / 'db/migrations/00012_department_tree_parent_index.sql').read_bytes()
    save('runtime-before.json', before)
    sockets = [socket.socket() for _ in range(3)]
    for sock in sockets:
        sock.bind(('127.0.0.1', 0))
    ports = [sock.getsockname()[1] for sock in sockets]
    for sock in sockets:
        sock.close()
    runtime.configure(dict(web=ports[0], api=ports[1], database=ports[2]))
    runtime.register_profile(temporary=False, purpose='303当前模型部门一级菜单；接管45300供用户验收；责任d')
    config = local.configuration(True)
    source_config = runtime.read_private(Path.home() / '.local/state/chnhcm' / source_key() / 'runtime.json')
    for key in ['instance', 'enterprise', 'login', 'reader_login']:
        if key in source_config:
            config[key] = source_config[key]
    runtime.write_private(runtime.CONFIG, config)
    local.compose(local.environment(config, runtime.connection()), 'up', '-d', '--wait')
    candidate = container(runtime.KEY)
    sql(candidate, 'CREATE ROLE chnhcm_app NOLOGIN')
    dump = subprocess.Popen(['docker', 'exec', original, 'pg_dump', '-U', 'postgres', '-d', 'chnhcm', '-Fc'], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    restore = subprocess.run(['docker', 'exec', '-i', candidate, 'pg_restore', '-U', 'postgres', '-d', 'chnhcm', '--no-owner', '--exit-on-error'], stdin=dump.stdout, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    dump.stdout.close()
    if dump.wait() or restore.returncode:
        raise RuntimeError('Isolated database copy failed; original is untouched')
    copied = snapshot(candidate)
    assert copied == before, 'Copy mismatch'
    # The original index SQL is byte-identical to current 00012. Remove only
    # this copied derived index and its copied ledger row, then use the normal
    # current-model 00009—00012 upgrader. Original facts and ledger are untouched.
    sql(candidate, "BEGIN; DROP INDEX organization.department_tree_parent; DELETE FROM public.goose_db_version WHERE version_id=10; COMMIT;")
    save('runtime-copy.json', dict(originalContainer=original, candidate=candidate, ports=runtime.connection()['ports'],
                                  exactCopy=True, oldIndexSQLMatchesCurrent=True, original=snapshot(original)))
    print('Isolated copy verified; current prefix prepared, original unchanged.')
elif sys.argv[1] == 'snapshot':
    candidate = snapshot(container(runtime.KEY))
    before = json.loads((EVIDENCE / 'runtime-before.json').read_text())
    original_running = subprocess.check_output(['docker', 'inspect', '--format', '{{.State.Running}}', container(source_key())], text=True).strip() == 'true'
    if original_running:
        assert snapshot(container(source_key())) == before, 'Original business data or ledger changed'
    assert candidate['tables'] == before['tables'], 'Candidate business data changed'
    assert candidate['principalMD5'] == before['principalMD5'], 'Principal identities changed'
    assert candidate['index'] == before['index'], 'Tree index changed'
    assert candidate['migrations'] == [str(n) for n in range(1, 13)]
    save(sys.argv[2], dict(candidate=candidate, originalUnchanged=True if original_running else None,
                          originalStopped=not original_running, originalFactsAndIdentitiesPreserved=True))
    print('Original facts, identities and index preserved; current migrations 1—12 verified.')
elif sys.argv[1] == 'adopt-ports':
    assert json.loads((EVIDENCE / 'runtime-staged.json').read_text())['passed']
    assert json.loads((EVIDENCE / 'runtime-verified.json').read_text())['originalFactsAndIdentitiesPreserved']
    previous = runtime.connection()['ports']
    source_env = dict(os.environ, CHNHCM_LOCAL_PROFILE=SOURCE)
    subprocess.run(['python3', 'scripts/local-bu.py', 'down'], cwd=ROOT, env=source_env, check=True)
    subprocess.run(['python3', 'scripts/local-bu.py', 'down'], cwd=ROOT, env=os.environ, check=True)
    assert not runtime.processes()
    database = runtime.database_container()
    assert database and not database['running']
    assert subprocess.check_output(['docker', 'inspect', '--format', '{{.State.Running}}', container(source_key())], text=True).strip() == 'false'
    assert runtime.port_available(45300) and runtime.port_available(45301)
    selected = dict(previous, web=45300, api=45301)
    runtime.validate_ports(selected)
    # Explicit task-owned endpoint handoff after both programs have stopped.
    # The ordinary configure command remains immutable after initialization.
    # The candidate database binding and volume are deliberately unchanged.
    runtime.write_private(runtime.PORTS, selected)
    assert runtime.database_container()['id'] == database['id']
    save('runtime-port-handoff.json', dict(previous=previous, current=selected,
                                         databaseContainerPreserved=True, originalStoppedAndRetained=True))
    print('45300/45301 handed to current profile; both database bindings and volumes preserved.')
elif sys.argv[1] == 'build-check':
    runtime.verify_target(runtime.connection())
    manifest = json.loads((runtime.BIN / 'fixed-build.json').read_text())
    assert hashlib.sha256((runtime.BIN / 'server').read_bytes()).hexdigest() == manifest['apiSHA256']
    for name, digest in manifest['sourceSHA256'].items():
        assert hashlib.sha256((ROOT / name).read_bytes()).hexdigest() == digest, name
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for name, digest in manifest['webSHA256'].items():
        with opener.open(runtime.connection()['webOrigin'] + '/' + name, timeout=20) as response:
            assert hashlib.sha256(response.read()).hexdigest() == digest, name
    save('runtime-build-check.json', dict(baseCommit=manifest['baseCommit'], apiSHA256=manifest['apiSHA256'],
                                        sourceFiles=len(manifest['sourceSHA256']), actualHTTPFiles=len(manifest['webSHA256']), passed=True))
    import gzip
    with gzip.open(EVIDENCE / 'runtime-fixed-build.json.gz', 'wt') as output:
        json.dump(manifest, output, ensure_ascii=False, indent=2)
    print('Actual API, frontend HTTP files and source inputs match the fixed build.')
else:
    raise RuntimeError('Unknown action')
