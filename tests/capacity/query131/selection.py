#!/usr/bin/env python3
"""Full frozen-input selection/CSV checks against retained independent inputs.

Only an explicitly disposable case of the current model5 is admitted. Public output contains
counts, hashes and timings; original data, CSV and session values stay private.
This serial stage measurement does not stand in for T04's capacity matrix.
"""
import argparse
from collections import defaultdict
import csv
from datetime import datetime
import hashlib
import io
import json
import math
import os
from pathlib import Path
import sys
import time
import uuid
from zoneinfo import ZoneInfo

from lists import expected, case, resource, ROOT
sys.path.insert(0, str(ROOT/'tests/capacity/storage130'))
from growth import sql
sys.path.insert(0, str(ROOT/'tests/capacity/remediation125'))
from vanke_csv import raw_export, protected_text


def run(args):
    private = args.private.resolve()
    assert private.stat().st_mode & 0o777 == 0o700 and ROOT not in private.parents
    assert not args.output.exists()
    assert not (args.continue_http_errors and args.observe), 'keep exhaustive HTTP results separate from detailed diagnosis'
    container = resource(private)
    case.BIN = args.binary.resolve()
    day = datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
    oracle, _ = expected(args.shape, private, day)
    total = 87343 if args.shape == 'vanke' else 22104
    assert len(oracle) == total
    ordered = {}
    for sort in ['code', 'name', 'name_desc']:
        values = sorted(oracle.values(), key=lambda r: r['id'])
        values.sort(key=lambda r: r['code' if sort == 'code' else 'name'], reverse=sort == 'name_desc')
        ordered[sort] = values
    inputs = {name: hashlib.file_digest((private/name).open('rb'), 'sha256').hexdigest()
              for name in ['organization.enterprise.jsonl.gz'] + (['import.sqlite', 'history-order.json'] if args.shape == 'vanke' else ['owners.json', 'bu-prototype.json', 'dep-prototype.json', 'bu-typed.jsonl.gz', 'dep-typed.jsonl.gz'])}
    result = dict(completed=False, passed=False, expectedObjects=total, businessAt=day,
                  driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                  binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),
                  oracleInputSHA256=inputs, concurrency=1, capacityAcceptance=False,
                  continueHTTPErrors=args.continue_http_errors, expectedCSVAttempts=18, samples=[], csv=[])
    def save():
        args.output.write_text(json.dumps(result, indent=2)+'\n')
    def count_sources():
        return sql(container, 'SELECT sum(cardinality(source_refs)) FROM organization.department_history')
    before = count_sources()
    server = case.Server(private, private)
    stats = private/f'{args.output.stem}-work.jsonl'
    if args.observe:
        assert not stats.exists()
        os.environ['CHNHCM_QUERY131_STATS'] = str(stats)
    else:
        os.environ.pop('CHNHCM_QUERY131_STATS', None)
    captured = 0
    def work():
        nonlocal captured
        if not args.observe:
            return None
        rows = [json.loads(line) for line in stats.read_text().splitlines()]
        new = rows[captured:]
        captured = len(rows)
        return new
    def record(sample):
        observation_error = None
        try:
            sample['work'] = work()
        except Exception as error:
            # An early HTTP failure can precede the first observation file.
            # Retain that request and its original error, rather than replace
            # it with the observer error or silently count missing work as zero.
            observation_error = error
            sample['work'] = None
            sample['workObservationErrorType'] = type(error).__name__
        result['samples'].append(sample)
        save()
        if observation_error is not None and sample['ok']:
            raise RuntimeError('request retained; work observation unavailable') from observation_error
    def call(client, role, phase, path, body, want=200):
        sample = dict(role=role, phase=phase, ok=False)
        try:
            value, ms = client.call('POST', path, body, want=want)
            sample.update(ok=True, status=want, ms=ms)
            return value
        except Exception as error:
            sample.update(status=getattr(error, 'status', None), code=getattr(error, 'code', None),
                          ms=getattr(error, 'elapsed_ms', None), error=type(error).__name__)
            raise
        finally:
            record(sample)
    def export(client, role, body, mode='', sort='', want=200):
        sample = dict(role=role, phase='export' if want == 200 else 'export-denied', mode=mode, sort=sort, ok=False)
        started = time.perf_counter()
        try:
            status, headers, raw, metric = raw_export(client, body, server.p.pid)
            sample.update(metric, ok=status == want,
                          fileHeaderPresent=bool(headers.get('Content-Disposition')),
                          csvPrefixPresent=raw.startswith(b'\xef\xbb\xbf'))
            if status >= 400:
                sample['code'] = json.loads(raw).get('code')
                assert not sample['fileHeaderPresent'] and not sample['csvPrefixPresent'], 'failed export emitted file material'
            return status, headers, raw
        except Exception as error:
            sample.update(error=type(error).__name__, ms=(time.perf_counter()-started)*1000)
            raise
        finally:
            record(sample)
    def check_page(value, sort, page, size, count):
        p, s = value['page'], value['summary']
        assert p['count'] == total and s['selectedCount'] == count
        assert p['queryKey'] == s['queryKey'] and p['businessAt'] == day and 'selectionVersion' not in p
        wanted = ordered[sort][(page-1)*size:page*size]
        assert [r['id'] for r in p['items']] == [r['id'] for r in wanted]
        for actual, original in zip(p['items'], wanted, strict=True):
            for field in ['code', 'name', 'shortName', 'managementBu', 'effectiveDate', 'status']:
                assert actual.get(field) == original[field]
            assert actual['path']['departmentPath'] == original['path']
    try:
        base = server.start()
        assert server.info['model'] == 'm02-compact-5'
        for role in ['admin', 'reader']:
            client = case.Client(); client.base = base
            login, _ = client.call('POST', '/api/v1/session/login', dict(login=role, password='admin1234567'))
            client.csrf = login['csrf']
            instance, subject = str(uuid.UUID(login['instanceId'])), str(uuid.UUID(login['subjectId']))
            where = f"instance_id='{instance}' AND principal_id='{subject}'"
            def mutate(statement):
                sql(container, f'BEGIN;SELECT policy_no FROM access.policy_guard WHERE {where} FOR UPDATE;{statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE {where};COMMIT;')
            selected_ids = [ordered['code'][0]['id'], ordered['code'][-1]['id']]
            excluded_ids = [ordered['code'][20]['id'], ordered['code'][-1]['id']]
            modes = [('explicit', dict(mode='explicit', ids=selected_ids), 2),
                     ('all', dict(mode='allMatching', excludedIds=[]), total),
                     ('excluded', dict(mode='allMatching', excludedIds=excluded_ids), total-2)]
            initial = call(client, role, 'start-all', '/api/v1/departments/selection/start',
                           dict(query={}, selection=modes[1][1], page=1, pageSize=20))
            check_page(initial, 'code', 1, 20, total)
            draft = initial['summary']
            grant = None
            if role == 'reader':
                status, headers, raw = export(client, role, dict(query={}, draftId=draft['draftId'], expectedRevision=draft['revision']), want=403)
                assert status == 403 and not headers.get('Content-Disposition') and not raw.startswith(b'\xef\xbb\xbf')
                grant = str(uuid.uuid4())
                mutate(f"INSERT INTO access.subject_grant SELECT instance_id,'{grant}'::uuid,'{subject}'::uuid,'enterprise',enterprise_id,'department.export',ARRAY[]::text[],ARRAY[]::text[] FROM organization.enterprise WHERE instance_id='{instance}'")
            try:
                for mode, choice, count in modes:
                    established = call(client, role, 'start-explicit' if mode == 'explicit' else 'start-all',
                                       '/api/v1/departments/selection/start', dict(query={}, selection=choice if mode!='excluded' else dict(mode='allMatching',excludedIds=[]), page=1, pageSize=20))
                    draft = established['summary']
                    if mode=='excluded':
                        established=call(client,role,'exclude','/api/v1/departments/selection/resolve',dict(query={},draftId=draft['draftId'],expectedRevision=draft['revision'],removeIds=excluded_ids,page=1,pageSize=20))
                        draft=established['summary']
                    check_page(established, 'code', 1, 20, count)
                    assert draft['canExport']
                    for sort in ordered:
                        for size in [20, 50, 100]:
                            page = (total+size-1)//size
                            resolved = call(client, role, 'resolve', '/api/v1/departments/selection/resolve',
                                            dict(query={}, draftId=draft['draftId'], expectedRevision=draft['revision'], page=page, pageSize=size, sort=sort))
                            check_page(resolved, sort, page, size, count)
                            assert resolved['summary']['draftId']==draft['draftId'] and resolved['summary']['revision']==draft['revision']
                        status, headers, raw = export(client, role, dict(query={}, draftId=draft['draftId'], expectedRevision=draft['revision'], sort=sort), mode, sort)
                        if status != 200 and args.continue_http_errors:
                            result['csv'].append(dict(role=role, mode=mode, sort=sort, expectedRows=count,
                                rows=0, comparedCells=0, passed=False, status=status, code=json.loads(raw).get('code')))
                            save()
                            continue
                        assert status == 200 and int(headers['X-Selected-Count']) == count
                        rows = list(csv.reader(io.StringIO(raw.decode('utf-8-sig'))))
                        assert rows[0] == ['编码', '名称', '简称', 'BU编码', 'BU名称', '部门路径', '状态', '生效日期']
                        wanted = [r for r in ordered[sort] if mode == 'all' or mode == 'explicit' and r['id'] in selected_ids or mode == 'excluded' and r['id'] not in excluded_ids]
                        assert len(rows) == count+1 == len(wanted)+1
                        for actual, original in zip(rows[1:], wanted, strict=True):
                            values = [original['code'], original['name'], original['shortName'] or '', original['managementBu']['code'], original['managementBu']['name'], original['path'], {'active':'启用','inactive':'停用','not_effective':'未生效'}[original['status']], original['effectiveDate']]
                            assert actual == [protected_text(value) for value in values]
                        file = private/f'{args.output.stem}-{role}-{mode}-{sort}.csv'; file.write_bytes(raw); file.chmod(0o600)
                        result['csv'].append(dict(role=role, mode=mode, sort=sort, rows=count, comparedCells=count*8, passed=True,
                                                  sha256=hashlib.sha256(raw).hexdigest(),
                                                  orderedIdentitySHA256=hashlib.sha256(json.dumps([r['id'] for r in wanted]).encode()).hexdigest()))
                        save()
            finally:
                if grant: mutate(f"DELETE FROM access.subject_grant WHERE {where} AND grant_id='{grant}'")
        if args.observe:
            for sample in result['samples']:
                work=sample['work'];assert work
                if sample['status']<400: assert all(not o.get('Failed',False) for o in work)
                else: assert sample['phase']=='export-denied' and sample['ok'] and not sample['fileHeaderPresent']
                decoded=sum(o.get('Stats',{}).get('DecodedDepartments',0) for o in work)
                if sample['phase']=='export' and sample['status']==200: assert decoded>=(len(selected_ids) if sample.get('mode')=='explicit' else total-len(excluded_ids) if sample.get('mode')=='excluded' else total)
                elif sample['phase'] in ['start-all','start-explicit','resolve','exclude']: assert decoded<total,'identity/page operation projected the complete collection'
            result['boundedSelectionObserved']=True
        assert count_sources() == before
        result['businessSourcesUnchanged'] = True
        assert all(hashlib.file_digest((private/name).open('rb'), 'sha256').hexdigest() == digest for name, digest in inputs.items())
        result['completed'] = len(result['csv']) == result['expectedCSVAttempts']
        result['passed'] = result['completed'] and all(s['ok'] for s in result['samples']) and all(v['passed'] for v in result['csv'])
    finally:
        server.close()
        result['workObservationErrors'] = sum('workObservationErrorType' in sample for sample in result['samples'])
        groups = defaultdict(list)
        for sample in result['samples']: groups[sample['phase']].append(sample)
        result['groups'] = {}
        for phase, samples in groups.items():
            durations = sorted(s['ms'] for s in samples if s.get('ms') is not None)
            result['groups'][phase] = dict(requests=len(samples), failed=sum(not s['ok'] for s in samples),
                                          errorRate=sum(not s['ok'] for s in samples)/len(samples),
                                          httpErrors=sum(s.get('status', 0) is not None and s.get('status', 0) >= 400 for s in samples),
                                          httpErrorRate=sum(s.get('status', 0) is not None and s.get('status', 0) >= 400 for s in samples)/len(samples),
                                          p95MS=durations[math.ceil(len(durations)*.95)-1] if durations else None,
                                          maxMS=max(durations, default=None))
        save()
    print(json.dumps(dict(completed=result['completed'], passed=result['passed'], csvAttempts=len(result['csv']), csvFiles=sum(v['passed'] for v in result['csv']))))
    if not result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path); parser.add_argument('output', type=Path)
    parser.add_argument('--shape', choices=['synthetic', 'vanke'], default='vanke')
    parser.add_argument('--binary', type=Path, required=True); parser.add_argument('--observe', action='store_true')
    parser.add_argument('--continue-http-errors', action='store_true', help='retain each failed complete export and attempt the remaining independent exports; never count failed files as verified')
    run(parser.parse_args())
