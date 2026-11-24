#!/usr/bin/env python3
"""Complete enterprise/subtree/small CSV comparison on the owned Vanke case.
The oracle derives names, dates and paths from the original workbook ledger.
CSV, identities, policy fixtures and session material stay outside the repository.
"""
import argparse
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import uuid

from matrix import ROOT, MeasuredClient, fixtures, oracle
from frozen import owned_database as frozen_database
sys.path.insert(0,str(ROOT/'tests/capacity/vanke127'))
from verify import Oracle
from runtime import owned_resource
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from vanke_csv import protected_text, raw_export


def run(args):
    private=args.private.resolve()
    assert not private.is_relative_to(ROOT) and private.stat().st_mode&0o777==0o700
    assert not args.output.exists()
    csv_directory=private/(args.output.parent.name+'-'+args.output.stem+'-files')
    csv_directory.mkdir(mode=0o700,exist_ok=False)
    container,database=frozen_database(private) or owned_resource(private)
    def sql(statement):
        value=subprocess.run(['docker','exec','-i',container,'psql','-U','postgres','-d',database,'-X','-Atq','-v','ON_ERROR_STOP=1'],input=statement,text=True,capture_output=True)
        if value.returncode:raise RuntimeError('owned fixture SQL failed; raw output intentionally withheld')
        return value.stdout.strip()
    business_digest=lambda:sql("SELECT md5(string_agg(md5(to_jsonb(h)::text),'' ORDER BY md5(to_jsonb(h)::text))) FROM organization.department_history h")
    before=business_digest();source=Oracle(private)
    result=dict(completed=False,passed=False,cases=[],readerDefaultExportDenied=None,businessHistoryUnchanged=None)
    save=lambda:args.output.write_text(json.dumps(result,indent=2)+'\n')
    for role in ['admin','reader']:
        client=MeasuredClient(args.base);status,session,_=client.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));assert status==200;client.csrf=session['csrf']
        status,context,_=client.call('GET','/api/v1/organization/context');assert status==200;day=context['currentDate']
        byid,_,_,branch,_,_,large,shape=fixtures(oracle(private,day))
        selected=branch(large['id']);small=list(dict.fromkeys([selected[0]['id'],selected[-1]['id']]))
        cases=[('enterprise',{},dict(mode='allMatching',excludedIds=[]),sorted(byid.values(),key=lambda n:n['code'])),('subtree',dict(scopeDepartmentId=large['id']),dict(mode='allMatching',excludedIds=[]),selected),('small',dict(scopeDepartmentId=large['id']),dict(mode='explicit',ids=small),[n for n in selected if n['id'] in small])]
        result['shape']=shape
        args.output.with_suffix('.matrix.json').write_text(json.dumps(dict(roles=['admin','reader'],cases=[dict(name=n,expectedCount=len(rows),formalAttempts=1) for n,q,c,rows in cases],comparedColumns=8,timing='observed only; no invented CSV P95 target'),indent=2)+'\n')
        grant=None
        instance,subject=str(uuid.UUID(session['instanceId'])),str(uuid.UUID(session['subjectId']))
        where=f"instance_id='{instance}' AND principal_id='{subject}'"
        def mutate(statement):sql(f'BEGIN;SELECT principal_id FROM access.policy_guard WHERE {where} FOR UPDATE;{statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE {where};COMMIT;')
        try:
            for label,query,selection,wanted in cases:
                sample=dict(role=role,case=label,expectedCount=len(wanted),passed=False)
                result['cases'].append(sample);save()
                status,value,ms=client.call('POST','/api/v1/departments/selection/start',dict(query=query,selection=selection,page=1,pageSize=20))
                sample.update(startStatus=status,startMS=ms)
                if status!=200:sample['error']=value.get('code');save();continue
                summary=value['summary'];assert summary['selectedCount']==len(wanted)
                body=dict(query=query,draftId=summary['draftId'],expectedRevision=summary['revision'],sort='code')
                if role=='reader' and grant is None:
                    status,headers,data,metric=raw_export(client,body,args.server_pid)
                    result['readerDefaultExportDenied']=dict(passed=status==403 and not headers.get('Content-Disposition') and not data.startswith(b'\xef\xbb\xbf'),**metric)
                    grant=str(uuid.uuid4())
                    mutate(f"INSERT INTO access.subject_grant SELECT instance_id,'{grant}'::uuid,'{subject}'::uuid,'enterprise',enterprise_id,'department.export',ARRAY[]::text[],ARRAY[]::text[] FROM organization.enterprise WHERE instance_id='{instance}'")
                status,headers,data,metric=raw_export(client,body,args.server_pid);sample.update(metric)
                if status!=200:
                    sample.update(error=json.loads(data).get('code'),zeroFile=not headers.get('Content-Disposition') and not data.startswith(b'\xef\xbb\xbf'));save();continue
                try:
                    assert headers.get('Content-Disposition') and 'text/csv' in headers.get('Content-Type','')
                    rows=list(csv.reader(io.StringIO(data.decode('utf-8-sig'))))
                    assert rows.pop(0)==['编码','名称','简称','BU编码','BU名称','部门路径','状态','生效日期']
                    bu=source.record_at(source.frames[source.heads[source.root]],day)
                    assert len(rows)==len(wanted)
                    for actual,node in zip(rows,wanted,strict=True):
                        code=node['code'];record=source.record_at(source.frames[source.heads[code]],day)
                        path=' / '.join(n[3] for n in source.path(code,source.heads[code],day,current=True))
                        assert actual==[protected_text(v) for v in [code,record[1],'',source.root,bu[1],path,'启用',record[0]]]
                    sample.update(passed=True,comparedCells=len(rows)*8,sha256=hashlib.sha256(data).hexdigest())
                    with os.fdopen(os.open(csv_directory/f'{role}-{label}.csv',os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'wb') as output:output.write(data)
                except AssertionError:sample['error']='semantic_mismatch'
                save()
        finally:
            if grant:mutate(f"DELETE FROM access.subject_grant WHERE {where} AND grant_id='{grant}'")
    result['businessHistoryUnchanged']=before==business_digest()
    result['completed']=True;result['passed']=all(c['passed'] for c in result['cases']) and bool(result['readerDefaultExportDenied'] and result['readerDefaultExportDenied']['passed']) and result['businessHistoryUnchanged'];save()

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('base');p.add_argument('output',type=Path);p.add_argument('--server-pid',type=int,required=True);run(p.parse_args())
