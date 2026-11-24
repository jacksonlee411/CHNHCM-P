#!/usr/bin/env python3
"""Verify imported identity allocators through actual new BU/department saves."""
import argparse
import json
import subprocess
import time
import uuid
from pathlib import Path

from growth import case, resource, sql, save


def run(private,binary,output):
    assert not output.exists()
    case.BIN=binary
    name=resource(private)
    server=case.Server(private,private);client=case.Client()
    result=dict(completed=False,createdBU=0,createdDepartment=0,restoredOriginalReceipts=0)
    receipts=[]
    try:
        client.base=server.start()
        login,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'))
        client.csrf=login['csrf'];bus=[]
        for kind in ['bu','department']:
            for index in range(2):
                body=dict(code=f'S130_FINAL_{kind.upper()}_{index}',name=f'130分配器核验{kind}{index}',
                          effectiveDate='2026-01-01',operationType='CREATE',operationReason='NEW_ORGANIZATION')
                route='/api/v1/'+('business-units' if kind=='bu' else 'departments')
                if kind=='department':body.update(managementBuId=bus[index],parentDepartmentId=None)
                key=str(uuid.uuid4());reply,_=client.call('POST',route,body,key,want=201)
                assert reply['outcome']=='accepted' and reply['submissionId']==key
                oid=reply['buId' if kind=='bu' else 'departmentId']
                detail,_=client.call('GET',route+'/'+oid)
                assert detail['selectedRecord']['name']==body['name'] and detail['headSourceVersion']==reply['sourceVersion']
                if kind=='bu':bus.append(oid);result['createdBU']+=1
                else:result['createdDepartment']+=1
                receipts.append(dict(kind=kind,object=oid,route=route,body=body,receipt=reply))
        server.stop()
        assert sql(name,"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND backend_type='client backend' AND pid<>pg_backend_pid()")=='0'
        subprocess.run(['docker','stop','--time','30',name],stdout=subprocess.DEVNULL,check=True)
        subprocess.run(['docker','start',name],stdout=subprocess.DEVNULL,check=True)
        for _ in range(150):
            if subprocess.run(['docker','exec',name,'pg_isready','-h','127.0.0.1','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:break
            time.sleep(.2)
        else:raise RuntimeError('owned database did not resume')
        client.base=server.start()
        for value in receipts:
            prefix='/api/v1/organization' if value['kind']=='bu' else '/api/v1/departments'
            reply,_=client.call('POST',prefix+'/submissions/'+value['receipt']['submissionId']+'/resolve')
            assert all(reply.get(k)==v for k,v in value['receipt'].items() if k!='currentChanged')
            detail,_=client.call('GET',value['route']+'/'+value['object']+'?sourceVersion='+value['receipt']['sourceVersion'])
            assert detail['readOnly'] and detail['selectedRecord']['name']==value['body']['name']
            result['restoredOriginalReceipts']+=1
        result.update(completed=True,passed=True,normalDatabaseAndAPIRestart=True,allNewAllocationsSucceeded=True)
    finally:
        server.close();save(output,result);save(private/'130-new-object-receipts.json',receipts)


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True)
    a=p.parse_args();run(a.private,a.binary.resolve(),a.output)
