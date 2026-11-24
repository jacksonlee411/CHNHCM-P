#!/usr/bin/env python3
"""Six separately restarted API/DB first-read observations, never a warm P95.
Only the task-owned disposable database is restarted; host page cache remains.
Each admin/reader x roots/location/scope sample gets its own fresh process.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import subprocess
import time
import urllib.parse

from matrix import ROOT, MeasuredClient, ProcessObservation, fixtures, oracle
from measure import server
from recovery import resolve_originals
from frozen import owned_database as frozen_database


def run(args):
    private = args.private.resolve()
    assert not private.is_relative_to(ROOT)
    args.output.mkdir(exist_ok=False)
    synthetic = args.kind == 'synthetic'
    assert synthetic == (private / 'tree300-synthetic.json').exists()
    frozen = frozen_database(private) if args.kind == 'frozen' else None
    if frozen:
        container, _ = frozen
    else:
        container = ('chnhcm-bu-' + hashlib.sha256((str(ROOT) + ':tree300-synthetic-d').encode()).hexdigest()[:12] + '-postgres-1') if synthetic else 'chnhcm-vanke127'
        labels = json.loads(subprocess.check_output(['docker', 'inspect', '--format', '{{json .Config.Labels}}', container], text=True))
        assert labels.get('io.chnhcm.temporary') == 'true'
        assert labels.get('io.chnhcm.profile') == 'tree300-synthetic-d' if synthetic else labels.get('io.chnhcm.task') == 'vanke127'
    report = dict(completed=False, passed=False, observations=[],
                  definition='API process and PostgreSQL restarted before every sample; authentication/context precede the first business read; host filesystem page cache is retained',
                  p95Gate=None, plannedObservations=6, binarySHA256=hashlib.sha256(args.binary.read_bytes()).hexdigest())
    save = lambda: (args.output / 'summary.json').write_text(json.dumps(report, indent=2) + '\n')
    save()
    for role in ['admin', 'reader']:
        for name in ['roots', 'location-deep', 'scope-large']:
            subprocess.run(['docker', 'restart', container], check=True, stdout=subprocess.DEVNULL)
            for _ in range(60):
                ready = subprocess.run(['docker', 'exec', container, 'pg_isready', '-U', 'postgres'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                if ready.returncode == 0:
                    break
                time.sleep(.5)
            else:
                raise RuntimeError('owned database did not restart')
            label = args.output.name + '-' + role + '-' + name
            with server(private, args.binary.resolve(), args.kind, label) as (process, info, diagnostics):
                client = MeasuredClient(info['url'])
                status, session, _ = client.call('POST', '/api/v1/session/login', dict(login=role, password='admin1234567'))
                assert status == 200
                client.csrf = session['csrf']
                status, context, _ = client.call('GET', '/api/v1/organization/context')
                assert status == 200
                byid, children, path, branch, wide, deep, large, shape = fixtures(oracle(private, context['currentDate']))
                endpoint = {
                    'roots': '/api/v1/departments/tree/nodes',
                    'location-deep': '/api/v1/departments/' + deep['id'] + '/tree-location',
                    'scope-large': '/api/v1/departments?' + urllib.parse.urlencode(dict(scopeDepartmentId=large['id'])),
                }[name]
                observation = ProcessObservation(process.pid)
                observation.start()
                sample = dict(role=role, name=name, at=datetime.now(timezone.utc).isoformat(), passed=False, shape=shape)
                started = time.perf_counter()
                try:
                    status, value, ms = client.call('GET', endpoint)
                    sample.update(status=status, ms=ms, responseBytes=client.response_bytes, capacityRequest=client.capacity_request)
                    if status == 200:
                        assert value['businessAt'] == context['currentDate']
                        expected = children[None][:50] if name == 'roots' else path(deep['id']) if name == 'location-deep' else branch(large['id'])[:20]
                        assert [n['id'] for n in value['items']] == [n['id'] for n in expected]
                        for actual, wanted in zip(value['items'], expected):
                            assert all(actual[k] == wanted[k] for k in ['code', 'name'])
                        if name == 'scope-large':
                            assert value['count'] == len(branch(large['id']))
                        sample['passed'] = True
                    else:
                        sample['error'] = value.get('code')
                except Exception as error:
                    sample.update(error=type(error).__name__)
                    sample.setdefault('status', 0)
                    sample.setdefault('ms', (time.perf_counter() - started) * 1000)
                finally:
                    sample['process'] = observation.finish()
                    report['observations'].append(sample)
                    save()
    with server(private, args.binary.resolve(), args.kind, args.output.name + '-receipt-recovery') as (process, info, diagnostics):
        report['receiptRecovery'] = resolve_originals(private, info['url'], args.journal)
    report['completed'] = True
    report['passed'] = all(row['passed'] for row in report['observations']) and report['receiptRecovery']['passed']
    save()


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('kind', choices=['vanke', 'synthetic', 'frozen'])
    parser.add_argument('private', type=Path)
    parser.add_argument('binary', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--journal', type=Path, help='private acknowledged synthetic mixed-write journal')
    run(parser.parse_args())
