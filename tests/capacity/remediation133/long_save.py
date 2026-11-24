#!/usr/bin/env python3
"""Three ordinary HTTP corrections from one restored real20-record model5 input.

Restoration, VACUUM/ANALYZE, full-history checks and restart recovery belong to
the caller. This runner preserves one sample per position; it computes no P95.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/storage130'))
from growth import case, resource, sql


def run(a):
    assert not a.output.exists()
    for key in ['CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE', 'CHNHCM_CAPACITY_SPLIT_CONSTRAINTS']:
        os.environ.pop(key, None)
    meta = json.loads((a.private / 'input.json').read_text())
    assert meta['generatedThroughRealApplication'] and meta['recordCount'] == 20 and len(meta['members']) == 3
    assert [m['index'] for m in meta['members']] == [0, 10, 19]
    obj = str(uuid.UUID(meta['object']))
    name = resource(a.private)
    assert sql(name, 'SELECT contract FROM organization.storage_contract') == 'm02-compact-5'
    assert not a.native_cost or a.diagnostic
    assert sql(name, f"SELECT head_source_id::text FROM organization.business_unit WHERE bu_id='{obj}'") == meta['source']
    assert int(sql(name, f"SELECT count(*) FROM organization.bu_read_segment WHERE object_id='{obj}'")) == 20
    stages = a.private / (a.output.stem + '-stages')
    if a.diagnostic:
        stages.mkdir(mode=0o700)
        os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(stages)
        if a.native_cost:
            os.environ['CHNHCM_CAPACITY_SPLIT_CONSTRAINTS'] = '1'
    case.BIN = a.binary
    server = case.Server(a.private, a.private)
    result = dict(completed=False, diagnosticOnly=a.diagnostic, inputSHA256=meta['dumpSHA256'],
                  binarySHA256=hashlib.sha256(a.binary.read_bytes()).hexdigest(), targetMS=500,
                  samplesPerPosition=1, p95=None, samples=[], failures=[])
    receipts = []
    attempted = []
    requests = []
    index = None
    def storage():
        return json.loads(sql(name, """SELECT json_object_agg(c.relname,json_build_object(
          'tableIncludingToastBytes',pg_table_size(c.oid),'indexBytes',pg_indexes_size(c.oid),
          'totalBytes',pg_total_relation_size(c.oid))) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='organization' AND c.relname IN
          ('business_unit','bu_history','bu_dictionary','bu_shape','bu_read_segment','bu_terminal')"""))
    def functions():
        return json.loads(sql(name, """SELECT coalesce(json_object_agg(funcname,json_build_object(
          'calls',calls,'totalMS',total_time,'selfMS',self_time)),'{}'::json)
          FROM pg_stat_user_functions WHERE schemaname='organization'"""))
    if a.native_cost:
        # Caller enables tracking only in this disposable database before API
        # startup. Native function times are diagnostic, never save samples.
        assert sql(name, 'SHOW track_functions') == 'all'
        result['nativeCostScope'] = 'actual function counter deltas; total times nest; C constraints and inlined SQL functions are not separately measured'
    try:
        c = case.Client()
        c.base = server.start()
        v, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
        c.csrf = v['csrf']
        path = '/api/v1/business-units/' + obj
        head = meta['source']
        for member in meta['members']:
            index = member['index']
            target = '单条更正' + (str(index) if index else '')
            if a.diagnostic:
                sql(name, f"CREATE TABLE public.long133_before AS SELECT * FROM organization.bu_read_segment WHERE object_id='{obj}'")
                wal = sql(name, 'SELECT pg_current_wal_insert_lsn()::text')
            if a.native_cost:
                functions_before, storage_before = functions(), storage()
            key = str(uuid.uuid4())
            body = dict(expectedSourceVersion=head, expectedRecordRevision=member['revision'], changes=dict(name=target))
            attempted.append(index)
            requests.append(dict(position=index, submissionId=key, expectedSource=head,
                                 recordId=member['recordId']))
            (a.private / ('attempts-' + a.output.stem + '.json')).write_text(json.dumps(requests, indent=2)+'\n')
            reply, ms = c.call('POST', path + '/records/' + member['recordId'] + '/correct', body, key)
            sample = dict(position=index, ms=ms, request=c.request_id, status=200, accepted=reply['outcome']=='accepted')
            result['samples'].append(sample)
            if sample['accepted']:
                # Keep actual receipts even if a later observer/check fails.
                receipts.append(dict(receipt=reply, route='/api/v1/organization/submissions/'+key+'/resolve', path=path))
            assert sample['accepted'] and reply['sourceVersion'] != head
            if a.diagnostic:
                sample['walBytes'] = int(sql(name, f"SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),'{wal}'::pg_lsn)::bigint"))
                sample['bindings'] = json.loads(sql(name, f"SELECT json_build_object('members',count(*),'sourceChanged',count(*) FILTER(WHERE a.source_id<>b.source_id),'dictionaryChanged',count(*) FILTER(WHERE a.dictionary_id<>b.dictionary_id),'uuidDirectoryChanged',count(*) FILTER(WHERE a.ids<>b.ids),'shapeChanged',count(*) FILTER(WHERE a.shape_id<>b.shape_id)) FROM organization.bu_read_segment a JOIN public.long133_before b USING(instance_id,object_id,slot) WHERE a.object_id='{obj}'"))
                sql(name, 'DROP TABLE public.long133_before')
            if a.native_cost:
                start = time.monotonic()
                while True:
                    functions_after = functions()
                    # Both real BU head UPDATE checks must have flushed after
                    # COMMIT before reading the completed backend's counters.
                    checks = functions_after.get('bu_head_guard',{}).get('calls',0) - functions_before.get('bu_head_guard',{}).get('calls',0)
                    if checks == 2:
                        break
                    assert checks < 2 and time.monotonic()-start < 10, 'native statistics did not flush completely'
                    time.sleep(.1)
                sample['functionStatsWaitMS'] = (time.monotonic()-start)*1000
                sample['nativeFunctions'] = {key: {metric:value-functions_before.get(key,{}).get(metric,0) for metric,value in counters.items()}
                                             for key,counters in functions_after.items() if counters['calls']>functions_before.get(key,{}).get('calls',0)}
                sample['storageBefore'],sample['storageAfter'] = storage_before,storage()
            head = reply['sourceVersion']
            view, _ = c.call('GET', path+'?sourceVersion='+head+'&selectedRecordId='+member['recordId'])
            assert view['resolvedSourceVersion']==head and view['selectedRecord']['name']==target and view['readOnly']
            if a.diagnostic:
                # A rolled-back bulk UPDATE still creates WAL/dead tuples and
                # changes later samples. Use the actual request's native
                # counters and separated deferred checks instead.
                sample['constraintProbe'] = 'not run; actual request counters and deferred checks only'
        result['completed'] = True
    except Exception as e:
        result['failures'].append(dict(position=index, type=type(e).__name__, status=getattr(e,'status',None), code=getattr(e,'code',None)))
        raise
    finally:
        server.close()
        os.environ.pop('CHNHCM_CAPACITY_DIAGNOSTICS', None)
        os.environ.pop('CHNHCM_CAPACITY_SPLIT_CONSTRAINTS', None)
        if a.diagnostic and (stages/'requests.jsonl').exists():
            result['observations'] = [json.loads(line) for line in (stages/'requests.jsonl').read_text().splitlines()]
        result['singleSamplesWithinTarget'] = result['completed'] and all(s['ms']<=500 for s in result['samples'])
        result['capacityP95Accepted'] = False
        result['unattemptedPositions'] = [m['index'] for m in meta['members'] if m['index'] not in attempted]
        result['attemptedRequests'] = requests
        a.output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        (a.private/('receipts-'+a.output.stem+'.json')).write_text(json.dumps(receipts,ensure_ascii=False,indent=2)+'\n')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private',type=Path)
    p.add_argument('output',type=Path)
    p.add_argument('--binary',required=True,type=Path)
    p.add_argument('--diagnostic',action='store_true')
    p.add_argument('--native-cost',action='store_true')
    a = p.parse_args()
    a.private,a.binary = a.private.resolve(),a.binary.resolve()
    run(a)
