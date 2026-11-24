#!/usr/bin/env python3
"""Prepare the explicitly retained Vanke profile through the normal local runtime."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
PROFILE = 'm02-vanke-validation-b'
os.environ['CHNHCM_LOCAL_PROFILE'] = PROFILE
sys.path.insert(0, str(ROOT / 'scripts'))
import local_runtime as runtime

spec = importlib.util.spec_from_file_location('vanke_local_bu', ROOT / 'scripts/local-bu.py')
local = importlib.util.module_from_spec(spec)
spec.loader.exec_module(local)


def prepare():
    with runtime.lifecycle_lock():
        assert not runtime.CONFIG.exists() and not runtime.database_container(), 'existing profile retained; do not recreate it'
        runtime.configure(dict(web=4388, api=8388, database=54388))
        runtime.register_profile(temporary=False, purpose='万科全量组织及历史固定验证；用户明确要求保留')
        config = local.configuration(create=True)
        config['enterprise'] = dict(EnterpriseCode='VANKE127', EnterpriseName='组织历史验证企业', Start='1984-01-01')
        runtime.write_private(runtime.CONFIG, config)
    subprocess.run([sys.executable, str(ROOT / 'scripts/local-bu.py'), 'up', '--no-web'], cwd=ROOT, check=True)
    with runtime.lifecycle_lock():
        conn = runtime.connection()
        runtime.verify_api(conn)
        runtime.stop_processes(['api'])
        private = runtime.STATE / 'vanke127'
        private.mkdir(mode=0o700)
        env = local.environment(config, conn)
        target = dict(AdminDSN=env['CHNHCM_ADMIN_DATABASE_URL'], APIDSN=env['CHNHCM_DATABASE_URL'], Instance=config['instance'], Profile=PROFILE)
        path = private / 'database-private.json'
        with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as f:
            json.dump(target, f)
    binary = ROOT / '.local/vanke-127/vanke127'
    subprocess.run([str(binary), '-mode', 'initialize', '-private', str(private)], cwd=ROOT, check=True)
    summary = dict(profile=PROFILE, project=runtime.PROJECT, temporary=False, privateDirectory=str(private), ports=conn['ports'], ordinaryAPIStoppedDuringImport=True)
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    prepare()
