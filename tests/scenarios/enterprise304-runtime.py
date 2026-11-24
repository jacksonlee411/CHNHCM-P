"""304 authorized default-runtime update checks; dumps stay outside the repository."""
import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
os.environ['CHNHCM_LOCAL_PROFILE'] = 'primary303-current-d'
sys.path.insert(0, str(ROOT / 'scripts'))
import local_runtime as runtime

EVIDENCE = ROOT / 'docs/verification/data/304-publication'


def sql(query):
    return subprocess.check_output(['docker', 'exec', '-i', runtime.database_container()['id'],
                                   'psql', '-X', '-At', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'chnhcm'], input=query, text=True).strip()


def save(name, value):
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    p = EVIDENCE / name
    if p.exists():
        raise RuntimeError('Evidence already exists: ' + name)
    p.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def snapshot():
    tables = {}
    for name in sql("SELECT tablename FROM pg_tables WHERE schemaname='organization' ORDER BY tablename").splitlines():
        assert name.replace('_', '').isalnum()
        count, digest = sql(f"SELECT count(*),md5(coalesce(string_agg(h,'' ORDER BY h),'')) FROM (SELECT md5(to_jsonb(t)::text) h FROM organization.{name} t) x").split('|')
        tables[name] = dict(count=int(count), contentMD5=digest)
    return dict(tables=tables, container=runtime.database_container()['id'],
                principalMD5=sql("SELECT md5(string_agg(h,'' ORDER BY h)) FROM (SELECT md5(to_jsonb(t)::text) h FROM access.principal t) x"),
                departmentSources=int(sql('SELECT sum(cardinality(source_refs)) FROM organization.department_history')),
                buSources=int(sql('SELECT sum(cardinality(source_refs)) FROM organization.bu_history')),
                migrations=sql('SELECT version_id FROM goose_db_version WHERE is_applied AND version_id>0 ORDER BY version_id').splitlines())


if sys.argv[1] == 'backup':
    runtime.verify_target(runtime.connection())
    runtime.stop_processes()
    before = snapshot()
    assert before['migrations'] == [str(n) for n in range(1, 13)]
    backup = runtime.STATE / 'backups' / ('enterprise304-' + datetime.datetime.now().strftime('%Y%m%d-%H%M%S'))
    backup.mkdir(parents=True, mode=0o700)
    dump = backup / 'database.dump'
    with dump.open('xb') as f:
        subprocess.run(['docker', 'exec', runtime.database_container()['id'], 'pg_dump', '-U', 'postgres', '-d', 'chnhcm', '-Fc'], stdout=f, check=True)
    dump.chmod(0o600)
    with dump.open('rb') as f:
        subprocess.run(['docker', 'exec', '-i', runtime.database_container()['id'], 'pg_restore', '--list'], stdin=f, stdout=subprocess.DEVNULL, check=True)
    shutil.copytree(runtime.BIN, backup / 'program')
    assert snapshot() == before
    save('before.json', before)
    save('backup.json', dict(directory=str(backup), dumpBytes=dump.stat().st_size,
                             dumpSHA256=hashlib.sha256(dump.read_bytes()).hexdigest(),
                             archiveReadable=True, oldProgramRetained=True, stoppedBeforeSnapshot=True))
    print('Original database and fixed program backed up outside repository; exact before snapshot retained.')
elif sys.argv[1] == 'verify':
    runtime.verify_target(runtime.connection())
    after = snapshot()
    before = json.loads((EVIDENCE / 'before.json').read_text())
    assert after['container'] == before['container']
    assert after['principalMD5'] == before['principalMD5']
    assert all(after['tables'][name] == value for name, value in before['tables'].items()), 'Original business table changed'
    assert after['migrations'] == [str(n) for n in range(1, 15)]
    assert after['tables']['enterprise_profile']['count'] == 1
    assert sql('SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history') == '1'
    assert sql("SELECT initialization_key=encode(initialization_digest,'hex') FROM organization.enterprise_profile p JOIN organization.enterprise e USING(instance_id,enterprise_id)") == 't'
    save(sys.argv[2], dict(after=after, originalTablesPreserved=len(before['tables']), identitiesPreserved=True, initializedSources=1, passed=True))
    print('Every original business table and identity preserved; migration1—14 and one enterprise source verified.')
elif sys.argv[1] == 'build':
    runtime.verify_target(runtime.connection())
    manifest = json.loads((runtime.BIN / 'fixed-build.json').read_text())
    assert hashlib.sha256((runtime.BIN / 'server').read_bytes()).hexdigest() == manifest['apiSHA256']
    changed_tests = []
    for name, digest in manifest['sourceSHA256'].items():
        if hashlib.sha256((ROOT / name).read_bytes()).hexdigest() != digest:
            # Test fixtures are not linked into server/upgrader or the web build.
            assert name.endswith('_test.go'), name
            changed_tests.append(name)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for name, digest in manifest['webSHA256'].items():
        with opener.open(runtime.connection()['webOrigin'] + '/' + name, timeout=20) as response:
            assert hashlib.sha256(response.read()).hexdigest() == digest, name
    save('build-check.json', dict(baseCommit=manifest['baseCommit'], sourceFiles=len(manifest['sourceSHA256']), changedTestFixtures=changed_tests,
                                  apiSHA256=manifest['apiSHA256'], httpFiles=len(manifest['webSHA256']), passed=True))
    with (EVIDENCE / 'fixed-build.json.gz').open('xb') as f:
        f.write(gzip.compress(json.dumps(manifest, indent=2).encode(), mtime=0))
    print('Actual API and HTTP frontend match all current product sources.')
else:
    raise RuntimeError('Unknown action')
