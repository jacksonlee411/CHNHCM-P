#!/usr/bin/env python3
"""Read-only 20-session observations on exactly the new retained candidate.
No API/DB restart or OS cache eviction; this is not the 125 cold matrix.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
from measure import case,measured,parallel,stats,save,ROOT

def run(output):
    assert os.environ.get('CHNHCM_LOCAL_PROFILE')=='m02-current-vanke-b' and not output.exists()
    sys.path.insert(0,str(ROOT/'scripts'));import local_runtime as runtime
    conn=runtime.connection();runtime.verify_target(conn)
    private=runtime.STATE/'125-vanke';sys.path.insert(0,str(ROOT/'tests/capacity/vanke127'));from verify import Oracle
    oracle=Oracle(private);codes=sorted(c for c in oracle.objects if c!=oracle.root)
    result=dict(completed=False,profile=runtime.PROFILE,url=conn['webOrigin'],cacheScope='retained API/DB and natural OS caches; warm list after one explicit preparation',groups={},sessions=20,apiConnections=12)
    clients=[]
    container=runtime.database_container()['id']
    def security():
        return int(subprocess.check_output(['docker','exec',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1','-c','SELECT count(*) FROM access.security_event'],text=True))
    def check_list(v,kind):
        expected=codes if kind=='departments' else [oracle.root]
        assert v['count']==len(expected) and [r['code'] for r in v['items']]==expected[:20]
        for item,code in zip(v['items'],expected):
            row=oracle.record_at(oracle.frames[oracle.heads[code]],v['businessAt']);assert item['id']==oracle.objects[code]['id'] and item['name']==row[1] and item['effectiveDate']==row[0] and item['status']=='active'
    try:
        for index in range(20):
            c=case.Client();c.base=conn['webOrigin'];v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];clients.append(c)
            if index<19:time.sleep(3.1)
        baseline=security()
        for kind in ['business-units','departments']:
            preparation,_=measured(clients[0],'GET','/api/v1/'+kind,check=lambda v:check_list(v,kind));result.setdefault('preparation',[]).append(preparation)
            values=[]
            for _ in range(3):values+=parallel([lambda c=c:measured(c,'GET','/api/v1/'+kind,check=lambda v:check_list(v,kind))[0] for c in clients])
            result['groups'][kind+'-warm-list']=stats(values,300);save(output,result)
        context,_=clients[0].call('GET','/api/v1/departments/context')
        body=dict(context=dict(mode='create',effectiveDate=context['currentDate'],managementBuId=oracle.objects[oracle.root]['id']),page=1,pageSize=20,q='',selectedId=oracle.objects[codes[0]]['id'])
        def check(v):
            assert v['count']==87343 and [r['code'] for r in v['items']]==codes[:20]
            assert v['selectedReference']['id']==oracle.objects[codes[0]]['id'] and all(r['eligibility']=='eligible' for r in v['items'])
        values=parallel([lambda c=c:measured(c,'POST','/api/v1/departments/references/parents',body,check=check)[0] for c in clients])
        result['groups']['parents-create-no-search']=stats(values,300)
        result['readSecurityEventGrowth']=security()-baseline;assert result['readSecurityEventGrowth']==0
        result['completed']=True
    finally:
        result['passed']=result['completed'] and all(v['ok'] for v in result.get('preparation',[])) and all(v['passed'] for v in result['groups'].values());save(output,result)
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('output',type=Path);a=p.parse_args();run(a.output)
