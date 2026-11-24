#!/usr/bin/env python3
"""After a fresh API start, resolve every retained real capacity receipt.
No mutation or synthetic terminal is constructed; full identities stay private.
"""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path
from measure import Lane,case,measured,parallel,stats,ROOT,save

def run(private,output,binary):
    case.BIN=binary
    server=case.Server(private,private)
    result=dict(completed=False,files=[],checks=[],binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest())
    try:
        server.start();clients=[]
        assert server.info['apiConnectionLimit']==12
        for _ in range(20):
            c=case.Client();c.base=server.base;login,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=login['csrf'];clients.append(c)
        result['before']=case.cli('size',private)
        container=server.info['container']
        allowed={'chnhcm-storage126-product':'storage126-product','chnhcm-storage130-b':'storage130'}
        assert container in allowed
        labels=json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Config.Labels}}',container],text=True))
        assert labels.get('io.chnhcm.task')==allowed[container] and labels.get('io.chnhcm.temporary')=='true'
        if container=='chnhcm-storage130-b':assert labels.get('io.chnhcm.parent-task')=='storage130'
        heads=json.loads(subprocess.check_output(['docker','exec',container,'psql','-U','postgres','-d','storage126','-X','-Atq','-v','ON_ERROR_STOP=1','-c',"SELECT json_build_object('bu',(SELECT json_object_agg(bu_id,head_source_id) FROM organization.business_unit),'dep',(SELECT json_object_agg(department_id,head_source_id) FROM organization.department))"],text=True))
        requests=[]
        for path in sorted(private.glob('125-receipts-*.json')):
            values=json.loads(path.read_text())
            if not values:continue
            result['files'].append(dict(name=path.name,receipts=len(values),sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
            requests+=values
        samples=[]
        for start in range(0,len(requests),20):
            jobs=[]
            for client,row in zip(clients,requests[start:start+20]):
                def check(v,row=row):
                    Lane.exact_receipt(v,row['receipt'])
                    receipt=row['receipt'];kind='bu' if receipt.get('buId') else 'dep';identity=receipt['buId' if kind=='bu' else 'departmentId']
                    assert v.get('currentChanged')==(heads[kind][identity]!=receipt['sourceVersion'])
                jobs.append(lambda client=client,row=row,check=check:measured(client,'POST',row['route'],check=check)[0])
            samples+=parallel(jobs)
        result['requests']=stats(samples,500)
        result['after']=case.cli('size',private)
        for key in ['buSources','departmentSources','securityEvents']:
            assert result['before']['counts'][key]==result['after']['counts'][key]
        result['businessSourcesWritten']=0;result['securityEventGrowth']=0;result['currentChangedValidatedAgainstPinnedHeads']=True;result['completed']=True
    finally:
        server.close();result['passed']=result['completed'] and result['requests']['passed'];save(output,result)
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);a=p.parse_args()
    assert not a.output.exists() and a.private.is_absolute() and ROOT not in a.private.parents
    run(a.private,a.output,a.binary.resolve())
