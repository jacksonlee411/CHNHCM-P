#!/usr/bin/env python3
"""Ten real ordinary corrections for serial growth cost, never a P95 verdict."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(ROOT/'tests/capacity/storage130'))
from growth import case, resource, sql


def run(a):
    os.umask(0o077)
    assert not a.output.exists()
    for key in ['CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_SPLIT_CONSTRAINTS',
                'CHNHCM_CAPACITY_PROFILE', 'CHNHCM_QUERY131_STATS']:
        os.environ.pop(key, None)
    name = resource(a.private)
    assert name == 'chnhcm-storage126-product'
    meta = json.loads((a.private/'input.json').read_text())
    obj = str(uuid.UUID(meta['object']))
    path = '/api/v1/business-units/'+obj
    result = dict(completed=False, serialCostOnly=True, capacityAcceptance=False,
                  p95=None, targetMS=500, samples=[], failures=[], attempts=[],
                  inputSHA256=meta['dumpSHA256'],
                  binarySHA256=hashlib.sha256(a.binary.read_bytes()).hexdigest())
    receipts = []
    case.BIN = a.binary
    server = case.Server(a.private, a.private)

    def storage():
        return json.loads(sql(name, """SELECT json_object_agg(c.relname,json_build_object(
          'tableBytes',pg_table_size(c.oid),'indexBytes',pg_indexes_size(c.oid),
          'totalBytes',pg_total_relation_size(c.oid))) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='organization' AND c.relkind='r'"""))

    try:
        c = case.Client()
        c.base = server.start()
        v, _ = c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'))
        c.csrf = v['csrf']
        result['before'] = storage()
        for i in range(10):
            member = meta['members'][i % 3]
            view, _ = c.call('GET',path+'?selectedRecordId='+member['recordId'])
            record = next(r for r in view['timeline'] if r['recordId']==member['recordId'])
            key = str(uuid.uuid4())
            request = dict(expectedSourceVersion=view['headSourceVersion'],
                           expectedRecordRevision=record['revision'],changes=dict(name='真实来源增长'+str(i)))
            result['attempts'].append(dict(submissionId=key,position=member['index']))
            wal = sql(name,'SELECT pg_current_wal_insert_lsn()::text')
            reply, ms = c.call('POST',path+'/records/'+member['recordId']+'/correct',request,key)
            assert reply['outcome']=='accepted'
            receipts.append(dict(receipt=reply,route='/api/v1/organization/submissions/'+key+'/resolve',path=path))
            result['samples'].append(dict(position=member['index'],ms=ms,
                walBytes=int(sql(name,f"SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),'{wal}'::pg_lsn)::bigint"))))
        result['afterWrites'] = storage()
        start = time.monotonic()
        sql(name,'VACUUM (ANALYZE)')
        result['ordinaryMaintenanceMS'] = (time.monotonic()-start)*1000
        result['afterMaintenance'] = storage()
        if a.prototype:
            result['reachability'] = json.loads(sql(name,"""WITH RECURSIVE live(instance_id,object_id,hash) AS (
              SELECT instance_id,object_id,root FROM organization.proof133_source
              UNION SELECT n.instance_id,n.object_id,c.hash FROM live l
              JOIN organization.proof133_node n USING(instance_id,object_id,hash)
              CROSS JOIN LATERAL (VALUES(n.left_hash),(n.right_hash)) c(hash) WHERE c.hash IS NOT NULL)
              SELECT json_build_object('nodes',(SELECT count(*) FROM organization.proof133_node),
               'sourceRoots',(SELECT count(*) FROM organization.proof133_source),
               'unreferencedNodes',(SELECT count(*) FROM organization.proof133_node n WHERE NOT EXISTS(
                 SELECT FROM live l WHERE(l.instance_id,l.object_id,l.hash)=(n.instance_id,n.object_id,n.hash))))"""))
            assert result['reachability']['unreferencedNodes']==0
            result['reclamationBoundary']='all nodes are protected by immutable real history; no live-history deletion or net reclaim claim'
        result['completed'] = True
    except Exception as e:
        result['failures'].append(dict(type=type(e).__name__,status=getattr(e,'status',None),code=getattr(e,'code',None)))
        raise
    finally:
        server.close()
        a.output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        (a.private/('receipts-'+a.output.stem+'.json')).write_text(json.dumps(receipts,ensure_ascii=False,indent=2)+'\n')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private',type=Path)
    p.add_argument('output',type=Path)
    p.add_argument('--binary',required=True,type=Path)
    p.add_argument('--prototype',action='store_true')
    a = p.parse_args()
    a.private,a.binary = a.private.resolve(),a.binary.resolve()
    run(a)
