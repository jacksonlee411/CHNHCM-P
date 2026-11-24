#!/usr/bin/env python3
"""Supplemental accepted coverage reductions in a full owned database.

One BU is first empty, then owns a Department that ends exactly at the proposed
BU stop. These 400 warm requests supplement, not replace, the original matrix.
Only three explicitly recorded fixture writes occur; no preview writes.
"""
import argparse
import hashlib
import sys
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/remediation125'))
from measure import parallel, measured, stats
sys.path.insert(0, str(ROOT / 'tests/capacity/storage130'))
from growth import case, resource, save


def run(a):
    assert not a.output.exists()
    case.BIN = a.binary.resolve()
    resource(a.private)
    server = case.Server(a.private, a.private)
    result = dict(completed=False, supplementalWarmOnly=True, groups={},
                  binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),
                  before=case.cli('size', a.private), fixtureWrites=0)
    receipts = []
    try:
        server.start()
        assert server.info['apiConnectionLimit'] == 12
        clients = []
        for _ in range(20):
            c = case.Client(); c.base = server.base
            v, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
            c.csrf = v['csrf']; clients.append(c)
        c = clients[0]

        def write(path, body):
            v, _ = c.call('POST', path, body, str(uuid.uuid4()), want=201)
            receipts.append(v); result['fixtureWrites'] += 1
            return v

        code = 'ZZ_C133_' + uuid.uuid4().hex[:8]
        bu = write('/api/v1/business-units', dict(code=code, name='预览期间验证', effectiveDate='2026-01-01', operationType='CREATE', operationReason='NEW_ORGANIZATION'))
        path = '/api/v1/business-units/' + bu['buId']
        detail, _ = c.call('GET', path)
        body = dict(action='DEACTIVATE', expectedSourceVersion=detail['headSourceVersion'],
                    anchorRecordId=detail['timeline'][-1]['recordId'], effectiveDate='2035-01-01',
                    operationType='DEACTIVATE', operationReason='ORG_RESTRUCTURING')

        def check(v):
            assert v['outcome'] == 'accepted' and v['expectedSourceVersion'] == detail['headSourceVersion']

        for state in ['empty', 'finite']:
            if state == 'finite':
                dep = write('/api/v1/departments', dict(code=code+'_D', name='有截止日的引用', managementBuId=bu['buId'], effectiveDate='2026-01-01', operationType='CREATE', operationReason='NEW_ORGANIZATION'))
                dp = '/api/v1/departments/' + dep['departmentId']
                d, _ = c.call('GET', dp)
                write(dp+'/deactivate', dict(expectedSourceVersion=d['headSourceVersion'], anchorRecordId=d['timeline'][-1]['recordId'], effectiveDate='2035-01-01', operationType='DEACTIVATE', operationReason='ORG_RESTRUCTURING'))
                denied, _ = c.call('POST', path+'/preview', {**body, 'effectiveDate':'2034-12-31'}, want=409)
                assert denied['code'] == 'M02_DEPARTMENT_REFERENCE_CONFLICT'
            before = case.cli('size', a.private)['counts']
            for sessions in [1, 20]:
                samples = []
                for _ in range(100//sessions):
                    samples += parallel([lambda c=c: measured(c, 'POST', path+'/preview', body, check=check)[0] for c in clients[:sessions]])
                result['groups'][f'{state}-{sessions}'] = stats(samples, 300)
                save(a.output, result)
            after = case.cli('size', a.private)['counts']
            assert all(before[k] == after[k] for k in ['buSources', 'departmentSources'])
        after, _ = c.call('GET', path)
        assert after['headSourceVersion'] == detail['headSourceVersion'] and after['timeline'] == detail['timeline']
        result['completed'] = True
    finally:
        server.close()
        save(a.private/(a.output.stem+'-receipts.json'), receipts)
        result['after'] = case.cli('size', a.private)
        result['newSources'] = {k:result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources', 'departmentSources']}
        result['passed'] = result['completed'] and all(g['passed'] for g in result['groups'].values()) and result['newSources'] == dict(buSources=1, departmentSources=2)
        save(a.output, result)
    if not result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path); p.add_argument('output', type=Path)
    p.add_argument('--binary', type=Path, required=True)
    run(p.parse_args())
