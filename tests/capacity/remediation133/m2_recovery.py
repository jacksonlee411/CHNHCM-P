#!/usr/bin/env python3
"""Recover every selected growth receipt after a real owned DB/API restart.

No write is replayed. Receipt contents remain in the external private case.
This is correctness evidence, not the concurrent capacity denominator.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'tests/capacity/storage130'))
from growth import case, resource, sql


def run(args):
    assert not args.output.exists()
    assert args.private.is_absolute() and args.private.parent.name in {
        'remediation133-m2-20261112', 'remediation133-long-save-20261112', 'remediation133-m3-20261112'}
    assert args.private.stat().st_mode & 0o777 == 0o700
    case.BIN = args.binary.resolve()
    name = resource(args.private)
    receipts = []
    for cycle in args.cycles:
        values = json.loads((args.private/f'growth-{cycle}-receipts.json').read_text())
        assert len(values) == 1000
        receipts.extend(values)
    assert len({r['key'] for r in receipts}) == len(receipts)
    assert sql(name, "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND backend_type='client backend' AND pid<>pg_backend_pid()") == '0'
    def state():
        return json.loads(sql(name, "SELECT json_build_object("+
            "'sources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history)+(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history),"+
            "'terminals',(SELECT count(*) FROM organization.bu_terminal)+(SELECT count(*) FROM organization.department_terminal),"+
            "'heads',(SELECT json_object_agg(department_id,head_source_id) FROM organization.department))"))
    before = state()
    result = dict(completed=False, passed=False, databaseRestarted=False, recovered=0,
                  expectedReceipts=len(receipts), exactSources=0, failures=[],
                  cycles=args.cycles, replayedBusinessCommands=0, capacityAcceptance=False,
                  binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest())
    server = case.Server(args.private, args.private)
    stable = lambda v: {k:x for k,x in v.items() if k != 'currentChanged'}
    try:
        subprocess.run(['docker', 'restart', name], check=True, stdout=subprocess.DEVNULL)
        for _ in range(100):
            ready = subprocess.run(['docker', 'exec', name, 'pg_isready', '-U', 'postgres', '-d', 'storage126'],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if ready.returncode == 0:
                break
            time.sleep(.1)
        assert ready.returncode == 0
        result['databaseRestarted'] = True
        server.start()
        client = case.Client(); client.base = server.base
        login, _ = client.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
        client.csrf = login['csrf']
        for index, value in enumerate(receipts):
            try:
                reply, _ = client.call('POST', '/api/v1/departments/submissions/'+value['key']+'/resolve')
                assert stable(reply) == stable(value['receipt'])
                assert reply['currentChanged'] == (before['heads'][value['object']] != value['receipt']['sourceVersion'])
                result['recovered'] += 1
                exact, _ = client.call('GET', '/api/v1/departments/'+value['object']+'?sourceVersion='+value['receipt']['sourceVersion'])
                assert exact['readOnly'] and exact['resolvedSourceVersion'] == value['receipt']['sourceVersion']
                result['exactSources'] += 1
            except Exception as error:
                result['failures'].append(dict(index=index, errorType=type(error).__name__,
                    status=getattr(error, 'status', None), code=getattr(error, 'code', None)))
        result['completed'] = True
        result['currentChangedValidatedAgainstPinnedHeads'] = not result['failures']
    finally:
        server.close()
        after = state()
        result['newBusinessSources'] = after['sources'] - before['sources']
        result['newTerminalResults'] = after['terminals'] - before['terminals']
        result['passed'] = (result['completed'] and not result['failures'] and
                           result['recovered'] == result['exactSources'] == len(receipts) and
                           result['newBusinessSources'] == result['newTerminalResults'] == 0 and
                           after['heads'] == before['heads'])
        args.output.write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps(result))
    if not result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--cycles', nargs='+', type=int, required=True)
    run(p.parse_args())
