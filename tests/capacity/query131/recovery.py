#!/usr/bin/env python3
"""Two real corrections and exact result recovery on an owned T01 candidate.

This is semantic restart protection, not either of T04's 1000-command cycles.
All labels, object IDs and receipts remain in the external private directory.
"""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import time
import uuid

ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,save


def run(args):
    private=args.private.resolve();source=args.source.resolve();assert not args.output.exists()
    assert ROOT not in private.parents and private.stat().st_mode&0o777==0o700
    case.BIN=args.binary.resolve();name=resource(private)
    if args.shape=='synthetic':
        oracle=case.Oracle(source);oid=oracle.bycode['R6_D_00000'][1]['department_id']
        first=oracle.frame('R6_D_00000',1)['Operation'];old_source=first['Source']['ID'];old_key=first['Source']['Submission']
    else:
        spec=importlib.util.spec_from_file_location('vanke',ROOT/'tests/capacity/vanke127/verify.py');mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod)
        oracle=mod.Oracle(source);code=sorted(c for c,v in oracle.objects.items() if v['kind']=='departments')[-1]
        oid=oracle.objects[code]['id'];old=oracle.frames[oracle.events[code][0][1]];old_source=old['source'];old_key=old['submission']
    path='/api/v1/departments/'+oid;server=case.Server(private,private);client=case.Client()
    result=dict(shape=args.shape,completed=False,passed=False,accepted=0,oldReceiptRecovered=False,newReceiptsRecovered=0,capacityAcceptance=False,binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest())
    receipts=[]
    def public():args.output.write_text(json.dumps(result,indent=2)+'\n')
    def login():
        client.base=server.base;v,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));client.csrf=v['csrf']
    def stable(value):
        return {k:value.get(k) for k in ['resolvedSourceVersion','timeline','selectedRecord','path']}
    try:
        server.start();login();public()
        before,_=client.call('GET',path);record=before['timeline'][0];original_name=record['name']
        old_detail,_=client.call('GET',path+'?sourceVersion='+old_source)
        old_receipt,_=client.call('POST','/api/v1/departments/submissions/'+old_key+'/resolve')
        assert old_receipt['outcome']=='accepted' and old_receipt['sourceVersion']==old_source
        for label in ['T01恢复验证'+str(uuid.uuid4()),original_name]:
            detail,_=client.call('GET',path);row=next(r for r in detail['timeline'] if r['recordId']==record['recordId']);key=str(uuid.uuid4())
            response,_=client.call('POST',path+'/records/'+row['recordId']+'/correct',dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=row['revision'],changes=dict(name=label)),key)
            assert response['outcome']=='accepted' and response['submissionId']==key
            after,_=client.call('GET',path+'?selectedRecordId='+row['recordId']);assert after['selectedRecord']['name']==label
            receipts.append(dict(key=key,receipt=response));result['accepted']+=1;public()
        restored,_=client.call('GET',path)
        assert len(restored['timeline'])==len(before['timeline'])
        for a,b in zip(restored['timeline'],before['timeline'],strict=True):
            for field in ['recordId','name','shortName','effectiveDate','state','managementBuId','parentDepartmentId','endExclusive']:assert a.get(field)==b.get(field),field
        save(private/'t01-recovery-private.json',dict(oldSource=old_source,oldKey=old_key,oldReceipt=old_receipt,receipts=receipts))
        server.stop();resource(private)
        subprocess.run(['docker','restart',name],check=True,stdout=subprocess.DEVNULL)
        for _ in range(100):
            if subprocess.run(['docker','exec',name,'pg_isready','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:break
            time.sleep(.1)
        server.start();login()
        got,_=client.call('POST','/api/v1/departments/submissions/'+old_key+'/resolve');assert {k:v for k,v in got.items() if k!='currentChanged'}=={k:v for k,v in old_receipt.items() if k!='currentChanged'} and got['currentChanged'];result['oldReceiptRecovered']=True
        current_old,_=client.call('GET',path+'?sourceVersion='+old_source)
        assert stable(current_old)==stable(old_detail);result['oldSourceAndFrozenPathRecovered']=True
        for index,value in enumerate(receipts):
            got,_=client.call('POST','/api/v1/departments/submissions/'+value['key']+'/resolve');assert {k:v for k,v in got.items() if k!='currentChanged'}=={k:v for k,v in value['receipt'].items() if k!='currentChanged'};assert got['currentChanged']==(index<len(receipts)-1);result['newReceiptsRecovered']+=1
        result['completed']=True;result['passed']=True
    except Exception as error:
        result.update(errorType=type(error).__name__,status=getattr(error,'status',None),code=getattr(error,'code',None));raise
    finally:server.close();public()
    print(json.dumps(result))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('shape',choices=['synthetic','vanke']);p.add_argument('private',type=Path);p.add_argument('source',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);run(p.parse_args())
