#!/usr/bin/env python3
"""Prepare the owned synthetic database with a fixed current-model binary.
Run only after other preparation ends; this is never a timing result.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

from matrix import ROOT


def ensure_capacity_shared_memory(private):
    """Apply the adopted temporary-fixture budget without replacing its volume."""
    import runpy

    private = private.resolve()
    assert not private.is_relative_to(ROOT) and private.stat().st_mode & 0o777 == 0o700
    profile = 'tree300-synthetic-d'
    project = 'chnhcm-bu-' + hashlib.sha256((str(ROOT) + ':' + profile).encode()).hexdigest()[:12]
    container = project + '-postgres-1'
    inspect = lambda: json.loads(subprocess.check_output(['docker', 'inspect', container], text=True))[0]
    before = inspect()
    labels = before['Config']['Labels']
    assert labels.get('io.chnhcm.temporary') == 'true' and labels.get('io.chnhcm.profile') == profile
    assert labels.get('com.docker.compose.project') == project
    assert any(m.get('Name') == project + '_database' for m in before['Mounts'])
    def history_fingerprint():
        tables = ['business_unit', 'department', 'bu_history', 'department_history', 'bu_dictionary', 'department_dictionary', 'bu_shape', 'department_shape']
        statement = ' UNION ALL '.join("SELECT json_build_array('" + table + "',count(*),md5(coalesce(string_agg(fingerprint,'' ORDER BY fingerprint),''))) FROM (SELECT md5(to_jsonb(h)::text) fingerprint FROM organization." + table + ' h) items' for table in tables)
        return subprocess.check_output(['docker', 'exec', container, 'psql', '-U', 'postgres', '-d', 'chnhcm', '-X', '-Atq', '-v', 'ON_ERROR_STOP=1', '-c', statement], text=True)
    report = dict(containerBefore=before['Id'], sharedMemoryBefore=before['HostConfig']['ShmSize'], changed=False)
    if before['HostConfig']['ShmSize'] == 512 * 1024 * 1024:
        return report
    # A local profile starts with Docker's 64 MiB default. The adopted capacity
    # fixture uses 512 MiB; do not change persistent profiles or global defaults.
    assert before['HostConfig']['ShmSize'] == 64 * 1024 * 1024
    subprocess.run(['docker', 'start', container], check=True, stdout=subprocess.DEVNULL)
    for _ in range(120):
        if subprocess.run(['docker', 'exec', container, 'pg_isready', '-U', 'postgres'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
            break
        time.sleep(.25)
    else:
        raise RuntimeError('owned capacity database not ready')
    prior = history_fingerprint()
    log_path = private / 'postgres-before-shm.log'
    log_path.write_bytes(subprocess.check_output(['docker', 'logs', container], stderr=subprocess.STDOUT))
    log_path.chmod(0o600)
    override = private / 'capacity-compose.yaml'
    override.write_text('services:\n  postgres:\n    shm_size: 512m\n')
    override.chmod(0o600)
    previous_profile = os.environ.get('CHNHCM_LOCAL_PROFILE')
    os.environ['CHNHCM_LOCAL_PROFILE'] = profile
    sys.path.insert(0, str(ROOT / 'scripts'))
    try:
        local = runpy.run_path(str(ROOT / 'scripts/local-bu.py'), run_name='capacity_local_runtime')
        assert local['runtime'].PROFILE == profile and local['runtime'].temporary_profile()
        env = local['environment'](local['configuration'](), local['runtime'].connection())
        subprocess.run(['docker', 'compose', '-p', project, '-f', str(ROOT / 'deploy/local/compose.yaml'), '-f', str(override), 'up', '-d', '--wait', 'postgres'], cwd=ROOT, env=env, check=True)
    finally:
        if previous_profile is None:
            os.environ.pop('CHNHCM_LOCAL_PROFILE', None)
        else:
            os.environ['CHNHCM_LOCAL_PROFILE'] = previous_profile
        sys.path.pop(0)
    after = inspect()
    assert after['HostConfig']['ShmSize'] == 512 * 1024 * 1024
    assert before['Mounts'] == after['Mounts'] and before['Image'] == after['Image']
    assert before['Config']['Cmd'] == after['Config']['Cmd']
    assert before['HostConfig']['PortBindings'] == after['HostConfig']['PortBindings']
    assert prior == history_fingerprint()
    report.update(changed=True, containerAfter=after['Id'], sharedMemoryAfter=after['HostConfig']['ShmSize'], historyTablesUnchanged=True, originalVolumeRetained=True)
    return report


def run(args):
    private = args.private.resolve()
    assert not private.is_relative_to(ROOT) and private.stat().st_mode & 0o777 == 0o700
    args.output.mkdir(exist_ok=False)
    binary = args.binary.resolve()
    profile = 'tree300-synthetic-d'
    container = 'chnhcm-bu-' + hashlib.sha256((str(ROOT) + ':' + profile).encode()).hexdigest()[:12] + '-postgres-1'
    labels = json.loads(subprocess.check_output(['docker', 'inspect', '--format', '{{json .Config.Labels}}', container], text=True))
    assert labels.get('io.chnhcm.profile') == profile and labels.get('io.chnhcm.temporary') == 'true'
    report = dict(completed=False, steps=[], binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest())
    save = lambda: (args.output / 'summary.json').write_text(json.dumps(report, indent=2) + '\n')
    def step(name, command, env=None):
        started = time.monotonic()
        with (args.output / (name + '.log')).open('w') as log:
            code = subprocess.run(command, cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT).returncode
        report['steps'].append(dict(name=name, exitCode=code, seconds=time.monotonic() - started))
        save()
        if code:
            raise RuntimeError(name + ' failed; preserve exact requests and original log')
    save()
    step('stop-profile-api', ['python3', 'scripts/local-bu.py', 'down'], dict(os.environ, CHNHCM_LOCAL_PROFILE=profile))
    report['capacityResources'] = ensure_capacity_shared_memory(private)
    save()
    subprocess.run(['docker', 'start', container], check=True, stdout=subprocess.DEVNULL)
    for _ in range(60):
        if subprocess.run(['docker', 'exec', container, 'pg_isready', '-U', 'postgres'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
            break
        time.sleep(.5)
    else:
        raise RuntimeError('owned database did not start')
    with os.fdopen(os.open(private / (args.output.name + '-server.log'), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as log:
        env = dict(os.environ)
        env.pop('CHNHCM_QUERY131_STATS', None)
        env.pop('CHNHCM_CAPACITY_DIAGNOSTICS', None)
        process = subprocess.Popen([str(binary)], cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=log, text=True)
        try:
            info = json.loads(process.stdout.readline())
            assert info['container'] == container
            report['apiConnectionLimit'] = info['apiConnectionLimit']
            save()
            step('create', ['python3', 'tests/capacity/tree300/synthetic.py', str(private), info['url'], str(args.output / 'create.json'), '--workers', str(args.workers)])
            created = json.loads((args.output / 'create.json').read_text())
            if not created['completed'] and (private / 'stop-after-row').exists() and not created['failedAttempts']:
                report['pausedAtAcknowledgedBoundary'] = created['created']
                save()
                return
            assert created['completed']
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                process.wait(timeout=15)
    dump = private / 'product-history-before.jsonl'
    step('dump', [str(binary), '-mode', 'dump', '-output', str(dump)])
    step('history', ['python3', 'tests/capacity/tree300/history.py', str(private), str(dump), str(private / 'source-fingerprints-before.sqlite'), str(args.output / 'history.json')])
    report['completed'] = True
    save()


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path)
    parser.add_argument('binary', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--workers', type=int, choices=range(1, 9), default=4)
    run(parser.parse_args())
