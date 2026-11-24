#!/usr/bin/env python3
"""Small, separately recorded coverage-preview diagnosis on an owned full case.

The unchanged 1200-request acceptance matrix remains in m3_coverage_preview.py.
This collector observes the original 12-connection/5-second path, including
request stages and container filesystem pressure, without changing DB settings.
Raw PostgreSQL output stays in the private case; only error lines are published.
"""
import argparse
from collections import Counter
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import threading
import time

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/storage130'))
from growth import case, resource, sql, save
sys.path.insert(0, str(ROOT / 'tests/capacity/remediation125'))
from measure import parallel


def run(a):
    assert not a.output.exists()
    case.BIN = a.binary.resolve()
    container = resource(a.private)
    diagnostic = a.private / (a.output.stem + '-stages')
    diagnostic.mkdir(mode=0o700)
    os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(diagnostic)
    server = case.Server(a.private, a.private)
    before = case.cli('size', a.private)
    start = datetime.now(timezone.utc).isoformat()
    samples, resources = [], []
    done = threading.Event()

    def observe():
        while not done.is_set():
            reply = subprocess.run(['docker', 'exec', container, 'df', '-P', '-B', '1',
                                    '/dev/shm', '/var/lib/postgresql'],
                                   capture_output=True, text=True)
            values = [list(map(int, row.split()[1:4])) for row in reply.stdout.splitlines()[1:]] if reply.returncode == 0 else []
            resources.append(dict(atNS=time.monotonic_ns(), exitCode=reply.returncode, filesystems=values))
            done.wait(.1)

    monitor = threading.Thread(target=observe)
    try:
        server.start()
        assert server.info['apiConnectionLimit'] == 12
        clients = []
        for _ in range(20):
            c = case.Client(); c.base = server.base
            v, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
            c.csrf = v['csrf']; clients.append(c)
        oid = sql(container, 'SELECT bu_id FROM organization.business_unit ORDER BY code LIMIT 1')
        path = '/api/v1/business-units/' + oid
        detail, _ = clients[0].call('GET', path)
        last = detail['timeline'][-1]
        assert last['state'] == 'active' and last['effectiveDate'] < '2057-01-01'
        body = dict(action='DEACTIVATE', expectedSourceVersion=detail['headSourceVersion'],
                    anchorRecordId=last['recordId'], effectiveDate='2057-01-01',
                    operationType='DEACTIVATE', operationReason='ORG_RESTRUCTURING')

        def request(client):
            began = time.monotonic_ns()
            try:
                v, elapsed = client.call('POST', path + '/preview', body, want=409)
                result = dict(status=409, code=v.get('code'), ms=elapsed)
            except Exception as error:
                result = dict(status=getattr(error, 'status', None), code=getattr(error, 'code', None),
                              ms=getattr(error, 'elapsed_ms', (time.monotonic_ns()-began)/1e6),
                              errorType=type(error).__name__)
            result.update(request=client.request_id, startedNS=began, endedNS=time.monotonic_ns())
            return result

        monitor.start()
        for sessions in [1, 20]:
            for wave in range(3):
                results = parallel([lambda c=c: request(c) for c in clients[:sessions]])
                for value in results:
                    value.update(sessions=sessions, wave=wave)
                samples.extend(results)
                print(json.dumps(dict(sessions=sessions, wave=wave, codes=dict(Counter(v['code'] for v in results)))), flush=True)
    finally:
        done.set()
        if monitor.is_alive():
            monitor.join()
        server.close()
        os.environ.pop('CHNHCM_CAPACITY_DIAGNOSTICS', None)
        logged = subprocess.run(['docker', 'logs', '--since', start, container], capture_output=True)
        raw = logged.stdout + logged.stderr
        log = a.private / (a.output.stem + '-postgres.log')
        log.write_bytes(raw); log.chmod(0o600)
        errors = [line.split('ERROR:', 1)[1].strip() for line in raw.decode(errors='replace').splitlines() if 'ERROR:' in line]
        observed = {}
        stage_path = diagnostic / 'requests.jsonl'
        if stage_path.exists():
            observed = {str(v['Request']): v for v in map(json.loads, stage_path.read_text().splitlines())}
        for sample in samples:
            sample['observation'] = observed.get(str(sample['request']))
        after = case.cli('size', a.private)
        save(a.output, dict(diagnosticOnly=True, completed=len(samples)==63, binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),
                            samples=samples, resources=resources, postgresErrors=dict(Counter(errors)),
                            before=before, after=after, newBusinessSources={k:after['counts'][k]-before['counts'][k] for k in ['buSources', 'departmentSources']}))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--binary', type=Path, required=True)
    run(parser.parse_args())
