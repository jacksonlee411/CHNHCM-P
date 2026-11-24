#!/usr/bin/env python3
"""Complete-preview timing on the full synthetic case; never saves a command."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('capacity125', ROOT/'tests/capacity/remediation125/measure.py')
lane125 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lane125)


def run(args):
    assert not args.output.exists()
    assert args.samples % args.sessions == 0
    case = lane125.case
    case.BIN = args.binary.resolve()
    oracle = case.Oracle(args.private)
    path = oracle.path(args.code)
    server = case.Server(args.private, args.private)
    result = dict(completed=False, caseCode=args.code, action=args.action, sessions=args.sessions, expectedRequests=args.samples,
                  diagnosticOnly=args.samples != 200, binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),
                  before=case.cli('size', args.private), samples=[])
    try:
        server.start()
        clients = []
        for _ in range(args.sessions):
            c = case.Client(); c.base = server.base
            v, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
            c.csrf = v['csrf']; clients.append(c)
        detail, _ = clients[0].call('GET', path)
        record = detail['timeline'][-1] if args.action == 'ADD' else detail['timeline'][0]
        body = dict(action=args.action, expectedSourceVersion=detail['headSourceVersion'],
                    changes=dict(name='M2预览只读核验'))
        if args.action == 'ADD':
            assert record['effectiveDate'] < '2055-01-01'
            body.update(anchorRecordId=record['recordId'], effectiveDate='2055-01-01',
                        operationType='CHANGE', operationReason='NAME_ADJUSTMENT')
        else:
            body.update(recordId=record['recordId'], expectedRecordRevision=record['revision'])
        def check(value):
            assert value['valid'] is True and value['noChange'] is False
            assert 'name' in value['readFields'] and value['impact']['dataAffectedRanges']
        for _ in range(args.samples//args.sessions):
            result['samples'] += lane125.parallel([
                lambda c=c: lane125.measured(c, 'POST', path+'/preview', body, check=check)[0]
                for c in clients])
        after, _ = clients[0].call('GET', path)
        assert after['headSourceVersion'] == detail['headSourceVersion'] and after['timeline'] == detail['timeline']
        result['completed'] = True
    finally:
        server.close()
        result['after'] = case.cli('size', args.private)
        result['businessSourceGrowth'] = {k: result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources', 'departmentSources']}
        result['group'] = lane125.stats(result['samples'], 300)
        result['passed'] = result['completed'] and result['group']['passed'] and not any(result['businessSourceGrowth'].values())
        args.output.write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps({k:result['group'][k] for k in ['requests','failures','p95MS','targetMS']}))


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--sessions', type=int, choices=[1,20], required=True)
    p.add_argument('--samples', type=int, default=200)
    p.add_argument('--code', choices=['R6_D_00000', 'R6_FAN_ROOT', 'R6_DEPTH_099', 'R6_LONG_HISTORY'], default='R6_D_00000')
    p.add_argument('--action', choices=['ADD', 'CORRECT'], default='CORRECT')
    run(p.parse_args())
