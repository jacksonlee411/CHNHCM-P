#!/usr/bin/env python3
"""Full Vanke CSV and failure-zero-file checks on the disposable rebuilt case.
All labels, CSV bytes, cookies and policy snapshots stay in its private directory.
Public evidence contains counts, hashes, status, time and memory only.
"""
import argparse
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import threading
import time
import unicodedata
import urllib.error
import urllib.request
import uuid

ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/vanke127'))


def protected_text(value):
    first=value
    while first and (first[0].isspace() or unicodedata.category(first[0])=='Cc'):
        first=first[1:]
    return "'"+value if value.startswith(('\t','\r','\n')) or first.startswith(('=','+','-','@')) else value


def raw_export(client,body,pid):
    peak=[];stop=threading.Event()
    def memory():
        while not stop.is_set():
            try:
                fields={line.split(':')[0]:int(line.split()[1]) for line in Path(f'/proc/{pid}/status').read_text().splitlines() if line.startswith(('VmRSS:','VmHWM:','VmSwap:'))}
                peak.append(fields)
            except (OSError,ValueError):pass
            stop.wait(.02)
    watcher=threading.Thread(target=memory);watcher.start()
    req=urllib.request.Request(client.base+'/api/v1/departments/export',method='POST',data=json.dumps(body).encode(),headers={'Content-Type':'application/json','Origin':client.base,'X-CSRF-Token':client.csrf})
    start=time.perf_counter()
    try:
        try:
            with client.http.open(req,timeout=25) as response: status,headers,data=response.status,response.headers,response.read()
        except urllib.error.HTTPError as response: status,headers,data=response.code,response.headers,response.read()
        elapsed=(time.perf_counter()-start)*1000
        return status,headers,data,dict(ms=elapsed,bytes=len(data),status=status,peakRSSKB=max((m.get('VmRSS',0) for m in peak),default=0),peakHWMKB=max((m.get('VmHWM',0) for m in peak),default=0),peakSwapKB=max((m.get('VmSwap',0) for m in peak),default=0),singleSample=True)
    finally:
        stop.set();watcher.join()


def run(private,output,binary):
    os.environ['CHNHCM_VANKE_BINARY']=str(binary)
    from load import Client
    from verify import Oracle
    from measure import Server
    from runtime import owned_resource
    container,database=owned_resource(private)
    oracle=Oracle(private)
    server=Server(private);result=dict(completed=False,rows=87343,successes=[],failures=[],checks=[])
    def sql(statement):
        value=subprocess.run(['docker','exec','-i',container,'psql','-U','postgres','-d',database,'-X','-Atq','-v','ON_ERROR_STOP=1'],input=statement,text=True,capture_output=True)
        if value.returncode:raise RuntimeError('owned SQL fixture failed; raw output intentionally omitted')
        return value.stdout.strip()
    # The caller verified ownership of this disposable benchmark database.
    if sql("SELECT to_regclass('access.subject_grant') IS NULL") == "t":
        sql((ROOT / "tests/fixtures/access/policy.sql").read_text())
    def policy_mutation(instance,subject,statement):
        where=f"instance_id='{str(uuid.UUID(instance))}' AND principal_id='{str(uuid.UUID(subject))}'"
        sql(f'BEGIN;SELECT principal_id FROM access.policy_guard WHERE {where} FOR UPDATE;{statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE {where};COMMIT;')
    try:
        base=server.start();admin=None;choice=None
        codes=sorted(code for code,value in oracle.objects.items() if value['kind']=='departments')
        for role in ['admin','reader']:
            client=Client(base);status,login,_=client.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));assert status==200;client.csrf=login['csrf']
            if role=='admin':admin=client
            status,listing,_=client.call('GET','/api/v1/departments');assert status==200 and listing['count']==87343
            assert 'selectionVersion' not in listing
            selection=dict(mode='allMatching',excludedIds=[])
            status,established,metric=client.call('POST','/api/v1/departments/selection/start',dict(query={},selection=selection,page=1,pageSize=20))
            assert status==200
            resolved=established['summary']
            assert resolved['selectedCount']==87343
            result['checks'].append(dict(kind='selection-start',role=role,passed=True,ms=metric))
            body=dict(query={},draftId=resolved['draftId'],expectedRevision=resolved['revision'],sort='code')
            if role=='admin':choice=body
            fixture=None
            if role=='reader':
                status,headers,data,metric=raw_export(client,body,server.process.pid)
                result['failures'].append(dict(kind='default-reader-has-no-export',passed=status==403 and 'Content-Disposition' not in headers,csvFileCreated=False,**metric))
                fixture=str(uuid.uuid4());instance=str(uuid.UUID(login['instanceId']));subject=str(uuid.UUID(login['subjectId']))
                policy_mutation(instance,subject,f"INSERT INTO access.subject_grant SELECT instance_id,'{fixture}'::uuid,'{subject}'::uuid,'enterprise',enterprise_id,'department.export',ARRAY[]::text[],ARRAY[]::text[] FROM organization.enterprise WHERE instance_id='{instance}'")
                result['readerExportFixture']='temporary explicit export grant; default reader denial checked first; removed afterward'
            try:
                status,headers,data,metric=raw_export(client,body,server.process.pid)
            finally:
                if fixture:policy_mutation(instance,subject,f"DELETE FROM access.subject_grant WHERE instance_id='{instance}' AND grant_id='{fixture}'")
            metric['role']=role
            if status!=200:
                metric.update(passed=False,csvFileCreated=False,code=json.loads(data).get('code'))
                result['successes'].append(metric);continue
            assert 'text/csv' in headers.get('Content-Type','') and 'attachment' in headers.get('Content-Disposition','')
            rows=csv.reader(io.StringIO(data.decode('utf-8-sig')))
            assert next(rows)==['编码','名称','简称','BU编码','BU名称','部门路径','状态','生效日期']
            seen=set();count=0;day=listing['businessAt']
            root=oracle.root;bu=oracle.record_at(oracle.frames[oracle.heads[root]],day)
            for values,code in zip(rows,codes,strict=True):
                frame=oracle.frames[oracle.heads[code]];record=oracle.record_at(frame,day)
                path=' / '.join(['组织历史验证企业']+[n[3] for n in oracle.path(code,oracle.heads[code],day,current=True)])
                expected=[code,record[1],'',root,bu[1],path,'启用',record[0]]
                assert values==[protected_text(v) for v in expected]
                assert code not in seen;seen.add(code);count+=1
            assert count==87343 and seen==set(codes)
            path=private/f'125-full-{role}.csv'
            with os.fdopen(os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'wb') as f:f.write(data)
            metric.update(passed=True,rows=count,columns=8,duplicates=0,missing=0,allFieldsCompared=True,sha256=hashlib.sha256(data).hexdigest())
            result['successes'].append(metric)
        # Revoke only export authority: the old readable selection remains valid.
        status,session,_=admin.call('GET','/api/v1/session');assert status==200
        instance,subject=session['instanceId'],session['subjectId'];uuid.UUID(instance);uuid.UUID(subject)
        where=f"instance_id='{instance}' AND principal_id='{subject}'"
        snapshot=sql("SELECT json_agg(g) FROM access.subject_grant g WHERE "+where+" AND action='department.export'")
        snapshot_path=private/'125-export-grants.json';snapshot_path.write_text(snapshot);snapshot_path.chmod(0o600)
        def mutate(statement):
            sql(f'BEGIN;SELECT principal_id FROM access.policy_guard WHERE {where} FOR UPDATE;{statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE {where};COMMIT;')
        try:
            mutate(f"DELETE FROM access.subject_grant WHERE {where} AND action='department.export'")
            status,headers,data,metric=raw_export(admin,choice,server.process.pid)
            metric.update(kind='revoked-export',passed=status==403 and 'Content-Disposition' not in headers and 'application/json' in headers.get('Content-Type',''),csvFileCreated=False)
            result['failures'].append(metric)
        finally:
            escaped=snapshot.replace("'","''")
            mutate("INSERT INTO access.subject_grant SELECT * FROM json_populate_recordset(NULL::access.subject_grant,'"+escaped+"')")
        # A real source change preserves fixed identities and exports current facts;
        # restoring the name is another immutable correction.
        parents={row[2] for frame in oracle.frames.values() for row in frame['records'] if row[2]}
        selected=next(code for code in codes[10000:] if code not in parents and all(c.isalnum() or c==' ' for c in oracle.record_at(oracle.frames[oracle.heads[code]],day)[1]))
        path='/api/v1/departments/'+oracle.objects[selected]['id']
        status,detail,_=admin.call('GET',path);assert status==200
        record=detail['selectedRecord'];original=record['name'];changed=None
        try:
            request=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=record['revision'],changes=dict(name='125完整导出来源变化核验'))
            status,changed,_=admin.call('POST',path+'/records/'+record['recordId']+'/correct',request,str(uuid.uuid4()));assert status==200 and changed['outcome']=='accepted'
            status,headers,data,metric=raw_export(admin,choice,server.process.pid)
            rows=list(csv.reader(io.StringIO(data.decode('utf-8-sig')))) if status==200 else []
            expected_rows=[]
            for code in codes:
                frame=oracle.frames[oracle.heads[code]];item=oracle.record_at(frame,day)
                full_path=' / '.join(['组织历史验证企业']+[n[3] for n in oracle.path(code,oracle.heads[code],day,current=True)])
                name=item[1]
                if code==selected:
                    name='125完整导出来源变化核验';full_path=full_path.rsplit(' / ',1)[0]+' / '+name
                expected_rows.append([protected_text(v) for v in [code,name,'',root,bu[1],full_path,'启用',item[0]]])
            metric.update(kind='changed-source-current-export',passed=status==200 and len(rows)==87344 and rows[1:]==expected_rows,comparedCells=len(expected_rows)*8)
            result['checks'].append(metric)
        finally:
            if changed:
                request=dict(expectedSourceVersion=changed['sourceVersion'],expectedRecordRevision=changed['targetRevision'],changes=dict(name=original))
                status,restored,_=admin.call('POST',path+'/records/'+record['recordId']+'/correct',request,str(uuid.uuid4()))
                result['checks'].append(dict(kind='restore-name-new-source',passed=status==200 and restored['outcome']=='accepted'))
        result['completed']=True
    finally:
        server.stop();server.log.close()
        result['passed']=result['completed'] and len(result['successes'])==2 and all(m['passed'] for m in result['successes']+result['failures']+result['checks'])
        output.write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(dict(passed=result['passed'],completed=result['completed'])))


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('private',type=Path);parser.add_argument('output',type=Path);parser.add_argument('--binary',type=Path,required=True);args=parser.parse_args()
    assert args.private.is_absolute() and ROOT not in args.private.parents and args.private.stat().st_mode&0o777==0o700
    assert not args.output.exists()
    run(args.private,args.output,args.binary.resolve())
