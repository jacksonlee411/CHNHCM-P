#!/usr/bin/env python3
"""Read-only final-binary Vanke checks; independent full-import oracle.
Performance groups are separate from diagnostic sampling and CSV single runs.
"""
import argparse
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import sys
import time
from types import SimpleNamespace

ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/vanke127'))

def run(private,output,binary,owned_cold=False):
    os.environ['CHNHCM_VANKE_BINARY']=str(binary)
    from load import Client
    from verify import Oracle
    from measure import Server
    from runtime import cli
    # Import the 125 statistics function without confusing the two measure.py
    # modules; this file's independent HTTP adapter keeps error classes apart.
    import importlib.util
    spec=importlib.util.spec_from_file_location('remediation125',ROOT/'tests/capacity/remediation125/measure.py')
    capacity=importlib.util.module_from_spec(spec);spec.loader.exec_module(capacity)
    result=dict(completed=False,binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),sessions=20,apiConnections=12,groups={},checks=[],coldResets=[],cacheScope='owned API, database and data-file caches' if owned_cold else 'cold API only; full OS/DB cold acceptance belongs to the separate 125 matrix')
    save=lambda:output.write_text(json.dumps(result,indent=2)+'\n')
    assert not output.exists()
    oracle=Oracle(private);server=Server(private);clients=[]
    codes=sorted(c for c in oracle.objects if c!=oracle.root)
    assert len(codes)==87343
    class Checked:
        def __init__(self,client):self.client=client
        def call(self,*args):
            status,value,elapsed=self.client.call(*args)
            if status!=200:
                error=RuntimeError('HTTP failure');error.status=status;error.code=value.get('code');error.elapsed_ms=elapsed;raise error
            return value,elapsed
    def measure(client,method,path,body=None,check=None):
        return capacity.measured(Checked(client),method,path,body,check=check)[0]
    def parallel(jobs):
        with concurrent.futures.ThreadPoolExecutor(max_workers=20) as pool:return list(pool.map(lambda f:f(),jobs))
    def group(name,values,target):
        result['groups'][name]=capacity.stats(values,target);save()
        print(json.dumps(dict(group=name,n=len(values),p95=result['groups'][name]['p95MS'],errors=result['groups'][name]['failures'])),flush=True)
    def cold(tag):
        if owned_cold:
            state=SimpleNamespace(server=server,result=result,output=output,clients={'admin':clients})
            capacity.Lane.cold(state,tag)
        else:
            server.restart(clients)
    def list_check(kind,value):
        expected=codes if kind=='departments' else [oracle.root]
        assert value['count']==len(expected) and len(value['items'])==min(20,len(expected))
        assert [v['code'] for v in value['items']]==expected[:20]
        for item,code in zip(value['items'],expected):
            row=oracle.record_at(oracle.frames[oracle.heads[code]],value['businessAt'])
            assert item['id']==oracle.objects[code]['id'] and item['name']==row[1] and item['effectiveDate']==row[0] and item['status']=='active'
            if kind=='departments':
                path=' / '.join(['组织历史验证企业']+[n[3] for n in oracle.path(code,oracle.heads[code],value['businessAt'],current=True)])
                assert item['path']['departmentPath']==path and item['managementBu']['id']==oracle.objects[oracle.root]['id']
    try:
        base=server.start()
        for _ in range(20):
            c=Client(base);status,login,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'))
            assert status==200;c.csrf=login['csrf'];clients.append(c)
        result['beforeReads']=cli('size',private)
        for kind in ['business-units','departments']:
            cold_label='cold-owned' if owned_cold else 'cold-api'
            for state in [cold_label,'warm']:
                if state==cold_label:cold('vanke-'+kind+'-list')
                values=[]
                for _ in range(1 if state==cold_label else 3):
                    values+=parallel([lambda c=c:measure(c,'GET','/api/v1/'+kind,check=lambda v:list_check(kind,v)) for c in clients])
                group(kind+'-list-'+state,values,300)
        # The growth lane validates all of its changed timelines and receipts.
        # Keep this separate workbook-based detail oracle on unchanged heads;
        # full list counts and CSV still include every growth target.
        exclusions_path=private/'detail-read-exclusions.json'
        exclusions=set(json.loads(exclusions_path.read_text())['objects']) if exclusions_path.exists() else set()
        read_codes=[code for code in codes if oracle.objects[code]['id'] not in exclusions]
        assert len(read_codes)>=20
        result['detailOracleExcludedChangedHeads']=len(exclusions)
        result['detailOracleScope']='unchanged workbook heads; growth targets separately checked after every command and all original sources fully decoded; list and CSV retain all objects'
        sampled=[read_codes[i*len(read_codes)//20] for i in range(20)]
        original_receipts=[]
        for label,exact in [('current-detail',False),('exact-source',True)]:
            values=[]
            for _ in range(3):
                jobs=[]
                for c,code in zip(clients,sampled):
                    source=oracle.events[code][0][1] if exact else None
                    path='/api/v1/departments/'+oracle.objects[code]['id']+('?sourceVersion='+source if source else '')
                    jobs.append(lambda c=c,code=code,source=source,path=path:measure(c,'GET',path,check=lambda v:oracle.check_detail(code,v,source)))
                values+=parallel(jobs)
            group(label,values,500 if exact else 300)
        for c,code in zip(clients,sampled):
            frame=oracle.frames[oracle.events[code][0][1]]
            def check(v,source=frame['source']):assert v['outcome']=='accepted' and v['sourceVersion']==source
            path='/api/v1/departments/submissions/'+frame['submission']+'/resolve'
            original_receipts.append((c,path,check))
        for state in ['before-restart','after-restart']:
            if state=='after-restart':server.restart(clients)
            group('original-result-'+state,[measure(c,'POST',path,check=check) for c,path,check in original_receipts],500)
        no_changes=[]
        for (phase,no),(raw,stored) in oracle.rows.items():
            if stored['outcome']!='no_change':continue
            expected=stored['response']
            def check(v,expected=expected):
                for key in ['submissionId','outcome','observedSourceVersion','selectedRecordId']:assert v.get(key)==expected.get(key)
            no_changes.append(measure(clients[len(no_changes)%20],'POST','/api/v1/departments/submissions/'+stored['requestKey']+'/resolve',check=check))
        assert len(no_changes)==18;group('all-no-change-original-results',no_changes,500)
        # The imported data has one BU and all departments active through today.
        # A new current-day root target can therefore select all 87,343 parents.
        status,context,_=clients[0].call('GET','/api/v1/departments/context')
        assert status==200
        selected=codes[0]
        body=dict(context=dict(mode='create',effectiveDate=context['currentDate'],managementBuId=oracle.objects[oracle.root]['id']),page=1,pageSize=20,q='',selectedId=oracle.objects[selected]['id'])
        for search,page in [('',1),(selected,1),('',2)]:
            if owned_cold and not search and page==1:
                cold('vanke-first-no-search-parent-candidate')
            request=dict(body,q=search,page=page)
            expected=[code for code in codes if not search or search in code or search in oracle.record_at(oracle.frames[oracle.heads[code]],context['currentDate'])[1]]
            def check(v,expected=expected,page=page):
                assert v['applicable'] and v['count']==len(expected)
                assert [item['code'] for item in v['items']]==expected[(page-1)*20:page*20]
                assert v['selectedReference']['id']==oracle.objects[selected]['id']
                assert all(item['eligibility']=='eligible' for item in v['items'])
            group(f'parents-create-search{bool(search)}-page{page}',parallel([lambda c=c:measure(c,'POST','/api/v1/departments/references/parents',request,check=check) for c in clients]),300)
        result['afterReads']=cli('size',private)
        result['readSecurityEventGrowth']={k:result['afterReads']['counts'][k]-result['beforeReads']['counts'][k] for k in ['securityEvents','authenticationEvents','securityChangeEvents']}
        assert all(v==0 for v in result['readSecurityEventGrowth'].values())
        result['completed']=True
    finally:
        server.stop();server.log.close()
        result['passed']=result['completed'] and all(v['passed'] for v in result['groups'].values());save()
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--owned-cold',action='store_true');a=p.parse_args()
    assert a.private.is_absolute() and ROOT not in a.private.parents and a.private.stat().st_mode&0o777==0o700
    run(a.private,a.output,a.binary.resolve(),a.owned_cold)
