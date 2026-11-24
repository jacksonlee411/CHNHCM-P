#!/usr/bin/env python3
"""One20-record model5 HTTP group on an explicitly restored owned input.

The caller restores the same input per position/round and owns exclusive timing,
full-history comparisons, database restart and recovery. Failures remain samples;
only recovery of the same identifier may resolve an unknown result. No write is
retried. Actual receipts and request identities remain in the private directory.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import sys
import time
import uuid

ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,sql,save


def percentile(samples):
    values=sorted(samples)
    return values[math.ceil(len(values)*.95)-1] if values else None


def run(a):
    assert not a.output.exists() and a.samples>=1
    assert a.private.is_absolute() and ROOT not in a.private.parents
    for key in ('CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE','CHNHCM_CAPACITY_SPLIT_CONSTRAINTS','CHNHCM_QUERY131_STATS'):
        os.environ.pop(key,None)
    meta=json.loads((a.private/'input.json').read_text())
    assert meta['generatedThroughRealApplication']
    record_count=meta['recordCount'];assert record_count==20
    assert a.position in (0,record_count//2,record_count-1)
    member=next(m for m in meta['members'] if m['index']==a.position)
    obj=str(uuid.UUID(meta['object']));name=resource(a.private)
    assert sql(name,'SELECT contract FROM organization.storage_contract')=='m02-compact-5'
    kind=meta.get('kind','bu');assert kind in ('bu','department')
    owner,key,prefix,route=('business_unit','bu_id','bu','business-units') if kind=='bu' else ('department','department_id','department','departments')
    assert sql(name,f"SELECT head_source_id::text FROM organization.{owner} WHERE {key}='{obj}'")==meta['source']
    def storage():
        return json.loads(sql(name,"""SELECT json_build_object('databaseBytes',pg_database_size(current_database()),'wal',pg_current_wal_insert_lsn()::text,
          'relations',(SELECT json_object_agg(n.nspname||'.'||c.relname,json_build_object('tableIncludingToastBytes',pg_table_size(c.oid),'indexBytes',pg_indexes_size(c.oid),'totalBytes',pg_total_relation_size(c.oid)))
          FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification')))"""))
    case.BIN=a.binary.resolve();server=case.Server(a.private,a.private)
    result=dict(completed=False,model='m02-compact-5',objectKind=kind,recordCount=record_count,position=a.position,round=a.round,
                expected=a.samples,targetMS=500,inputSHA256=meta['dumpSHA256'],binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),samples=[],failures=[],singleSession=True)
    receipts=[];attempts=[];stem=a.output.stem
    path='/api/v1/'+route+'/'+obj
    head,revision=meta['source'],member['revision']
    try:
        client=case.Client();client.base=server.start()
        assert server.info['model']=='m02-compact-5' and server.info['apiConnectionLimit']==12
        reply,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));client.csrf=reply['csrf']
        result['before']=storage()
        for i in range(a.samples):
            submission=str(uuid.uuid4());recovery='/api/v1/'+('organization' if kind=='bu' else 'departments')+'/submissions/'+submission+'/resolve'
            body=dict(expectedSourceVersion=head,expectedRecordRevision=revision,changes=dict(name=f'20条普通更正-{a.position}-{i}'))
            attempts.append(dict(index=i,submissionId=submission,source=head,revision=revision,recordId=member['recordId']))
            save(a.private/('attempts-'+stem+'.json'),attempts)
            started=time.perf_counter();saved=None;status=None
            try:
                saved,ms=client.call('POST',path+'/records/'+member['recordId']+'/correct',body,submission)
                status=200
                assert saved['outcome']=='accepted' and saved['sourceVersion']!=head and saved['targetRecordId']==member['recordId'] and saved['targetRevision']!=revision
                sample=dict(index=i,ms=ms,status=200,ok=True)
            except Exception as e:
                sample=dict(index=i,ms=getattr(e,'elapsed_ms',(time.perf_counter()-started)*1000),status=getattr(e,'status',status),code=getattr(e,'code',None),error=type(e).__name__,ok=False)
                result['failures'].append(sample)
                # Recovery is outside the original request timer and does not
                # turn a failed request into a successful capacity sample.
                resolved,_=client.call('POST',recovery)
                assert resolved['outcome'] in ('accepted','not_applied')
                saved=resolved if resolved['outcome']=='accepted' else None
                sample['resolvedOutcome']=resolved['outcome']
            result['samples'].append(sample)
            if saved:
                head,revision=saved['sourceVersion'],saved['targetRevision']
                receipts.append(dict(receipt=saved,route=recovery,path=path,
                    expectedCorrection=dict(originalSource=meta['source'],recordId=member['recordId'],
                        name=body['changes']['name'],records=record_count)))
                save(a.private/('receipts-'+stem+'.json'),receipts)
            save(a.output,result)
            if (i+1)%10==0:print(json.dumps(dict(completed=i+1,expected=a.samples,failures=len(result['failures']))),flush=True)
        result['completed']=True
    finally:
        server.close();result['after']=storage()
        result['WALBytes']=int(sql(name,f"SELECT pg_wal_lsn_diff('{result['after']['wal']}'::pg_lsn,'{result['before']['wal']}'::pg_lsn)::bigint")) if 'before' in result else None
        result['actualSuccessfulReceipts']=len(receipts)
        result['p95AllRequestsMS']=percentile([s['ms'] for s in result['samples']])
        result['p95SuccessfulRequestsMS']=percentile([s['ms'] for s in result['samples'] if s['ok']])
        result['passed']=result['completed'] and not result['failures'] and len(result['samples'])==a.samples and result['p95AllRequestsMS']<=500
        save(a.output,result)
    if not result['passed']:raise SystemExit(1)


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True)
    p.add_argument('--position',type=int,required=True)
    p.add_argument('--samples',type=int,default=100);p.add_argument('--round',type=int,required=True);run(p.parse_args())
