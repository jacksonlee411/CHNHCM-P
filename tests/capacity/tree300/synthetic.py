#!/usr/bin/env python3
"""Versioned 87,343-node synthetic fixture through the ordinary Go HTTP API.
Includes depth 100 and a 20,000-child branch (covering the R125 100/2,000
hierarchy shapes). It does not claim R125's 50-source background history.
Exact requests and every response remain in a private resumable SQLite ledger.
"""
import argparse
import concurrent.futures
import json
import os
from pathlib import Path
import sqlite3
import time
import uuid

from matrix import ROOT, MeasuredClient, digest

COUNT=87343
VERSION='tree300-synthetic-v1'


def graph():
    for index in range(COUNT):
        parent=None
        if 0<index<100:parent=index-1
        elif index>=164:parent=100 if index<20164 else 101+(index-20164)%63
        yield dict(code=f'T300_SYN_{index:06d}',name='同名部门' if index%10==0 else f'合成部门{index}',parentCode=None if parent is None else f'T300_SYN_{parent:06d}')


def run(args):
    private=args.private.resolve();assert not private.is_relative_to(ROOT) and private.stat().st_mode&0o777==0o700
    assert not args.output.exists(),'preserve each preparation attempt'
    assert 1<=args.workers<=8
    plan=list(graph());assert len(plan)==COUNT
    path=private/'tree300-synthetic.sqlite';db=sqlite3.connect(path);path.chmod(0o600)
    db.executescript('CREATE TABLE IF NOT EXISTS requests(code TEXT PRIMARY KEY,key TEXT NOT NULL,body TEXT NOT NULL,result TEXT);CREATE TABLE IF NOT EXISTS attempts(code TEXT,attempt INTEGER,status INTEGER,response TEXT,ms REAL,PRIMARY KEY(code,attempt));')
    clients=[]
    for _ in range(args.workers):
        c=MeasuredClient(args.base);status,value,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));assert status==200;c.csrf=value['csrf'];clients.append(c)
    result=dict(version=VERSION,completed=False,created=0,expected=COUNT,workers=args.workers,graphSHA256=digest(plan),depth=100,wideChildren=20000,api='ordinary product HTTP only',failedAttempts=[])
    save=lambda:args.output.write_text(json.dumps(result,indent=2)+'\n')
    def job(code,body,client):
        prior=db.execute('SELECT key,body,result FROM requests WHERE code=?',(code,)).fetchone()
        encoded=json.dumps(body,ensure_ascii=False,sort_keys=True)
        if prior:
            assert prior[1]==encoded,'resume request body changed'
            if prior[2]:return json.loads(prior[2]),None
            key=prior[0]
        else:
            key=str(uuid.uuid5(uuid.NAMESPACE_URL,VERSION+'/'+code))
            db.execute('INSERT INTO requests VALUES(?,?,?,NULL)',(code,key,encoded));db.commit()
        attempt=db.execute('SELECT coalesce(max(attempt),0)+1 FROM attempts WHERE code=?',(code,)).fetchone()[0]
        return None,(code,key,body,client,attempt)
    def publish(job,response):
        code,key,body,client,attempt=job;status,value,ms=response
        db.execute('INSERT INTO attempts VALUES(?,?,?,?,?)',(code,attempt,status,json.dumps(value,ensure_ascii=False),ms));db.commit()
        if status!=201:
            result['failedAttempts'].append(dict(index=result['created'],status=status,code=value.get('code'),ms=ms));save();return None
        assert value['outcome']=='accepted';db.execute('UPDATE requests SET result=? WHERE code=?',(json.dumps(value,ensure_ascii=False),code));db.commit();return value
    bu_body=dict(code='T300_SYN_BU',name='部门树合成事业部',effectiveDate='2026-01-01',operationType='CREATE',operationReason='NEW_ORGANIZATION')
    bu,pending=job('T300_SYN_BU',bu_body,clients[0])
    if pending:bu=publish(pending,clients[0].call('POST','/api/v1/business-units',pending[2],pending[1]))
    assert bu;bu_id=bu['buId'];nodes=[];identities={};started=time.perf_counter()
    try:
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
            index=0
            while index<COUNT and not (private/'stop-after-row').exists():
                # Deep prefix is sequential; remaining roots and independent
                # leaves can use separate sessions with acknowledged parents.
                size=1 if index<164 else min(args.workers,COUNT-index)
                batch=[]
                for offset,node in enumerate(plan[index:index+size]):
                    parent=identities[node['parentCode']] if node['parentCode'] else None
                    body=dict(code=node['code'],name=node['name'],managementBuId=bu_id,parentDepartmentId=parent,effectiveDate='2026-01-01',operationType='CREATE',operationReason='NEW_ORGANIZATION')
                    value,pending=job(node['code'],body,clients[offset])
                    future=pool.submit(clients[offset].call,'POST','/api/v1/departments',pending[2],pending[1]) if pending else None
                    batch.append((node,parent,value,pending,future))
                failed=False
                for node,parent,value,pending,future in batch:
                    if pending:value=publish(pending,future.result())
                    if value is None:failed=True;continue
                    identity=value['departmentId'];identities[node['code']]=identity
                    nodes.append(dict(id=identity,code=node['code'],name=node['name'],parentId=parent))
                if failed:raise RuntimeError('failed synthetic requests retained; exact resume required')
                index+=size;result.update(created=index,elapsedSeconds=time.perf_counter()-started)
                if index%100<size:save();print(json.dumps({k:result[k] for k in ['created','expected','elapsedSeconds']}),flush=True)
        result['completed']=len(nodes)==COUNT
        if result['completed']:
            target=private/'tree300-synthetic.json'
            with os.fdopen(os.open(target,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'w') as stream:json.dump(dict(version=VERSION,completed=True,buId=bu_id,nodes=nodes),stream,ensure_ascii=False)
    finally:save();db.close()
    print(json.dumps({k:result[k] for k in ['created','expected','completed']}),flush=True)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('base');p.add_argument('output',type=Path);p.add_argument('--workers',type=int,default=4);run(p.parse_args())
