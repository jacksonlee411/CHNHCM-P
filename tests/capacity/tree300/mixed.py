#!/usr/bin/env python3
"""Two real corrections concurrent with tree reads on the owned synthetic case.
No timing gate: preserve all outcomes and verify each captured path against
independent before/after graphs. Successful changes are restored by new sources.
"""
import argparse
import concurrent.futures
import json
import os
from pathlib import Path
import threading
import time
import uuid
import subprocess

from matrix import ROOT, MeasuredClient, fixtures, oracle
from history import owned_database


def storage_observation(private):
    container, database = owned_database(private)
    statement = "SELECT json_build_object('walLSN',pg_current_wal_lsn()::text,'parentIndexBytes',pg_relation_size('organization.department_tree_parent'),'departmentSources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history))"
    return json.loads(subprocess.check_output(['docker', 'exec', container, 'psql', '-U', 'postgres', '-d', database, '-X', '-Atq', '-v', 'ON_ERROR_STOP=1', '-c', statement], text=True))


def lsn_number(value):
    high, low = value.split('/')
    return (int(high, 16) << 32) + int(low, 16)


def run(args):
    private=args.private.resolve()
    assert not private.is_relative_to(ROOT) and (private/'tree300-synthetic.json').exists()
    assert not args.output.exists()
    clients=[]
    for _ in range(12):
        c=MeasuredClient(args.base)
        status,session,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'))
        assert status==200;c.csrf=session['csrf'];clients.append(c)
    status,context,_=clients[0].call('GET','/api/v1/organization/context');assert status==200
    byid,children,path,branch,wide,deep,large,shape=fixtures(oracle(private,context['currentDate']))
    moved=children[wide][-1];destination=next(n for n in children[None] if n['id']!=wide)
    rename=deep;new_name=rename['name']+'并发核验'
    before_paths={n['id']:path(n['id']) for n in [rename,moved]}
    after_paths={rename['id']:[dict(n,name=new_name) if n['id']==rename['id'] else n for n in before_paths[rename['id']]],moved['id']:path(destination['id'])+[dict(moved,parentId=destination['id'])]}
    result=dict(completed=False,shape=shape,reads=[],writes=[],restorations=[],restoredReads=[],planned=dict(readSessions=10,readAttempts=200,concurrentCorrections=2,restorationCorrections=2,restoredReads=2),passed=False)
    save=lambda:args.output.write_text(json.dumps(result,indent=2)+'\n')
    fd=os.open(private/(args.output.stem+'-requests.jsonl'),os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    journal=os.fdopen(fd,'w');lock=threading.Lock()
    def write(node,changes,client,phase):
        status,detail,_=client.call('GET','/api/v1/departments/'+node['id'])
        assert status==200
        record=detail['selectedRecord'];key=str(uuid.uuid4())
        endpoint='/api/v1/departments/'+node['id']+'/records/'+record['recordId']+'/correct'
        body=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=record['revision'],changes=changes)
        with lock:
            journal.write(json.dumps(dict(phase=phase,path=endpoint,body=body,key=key))+'\n');journal.flush();os.fsync(journal.fileno())
        status,value,ms=client.call('POST',endpoint,body,key)
        with lock:
            journal.write(json.dumps(dict(phase=phase,status=status,response=value,key=key))+'\n');journal.flush();os.fsync(journal.fileno())
        return dict(status=status,ms=ms,passed=status==200 and value.get('outcome')=='accepted',error=value.get('code'),capacityRequest=client.capacity_request)
    gate=threading.Barrier(12)
    def writer(node,changes,client):
        gate.wait();started=time.perf_counter()
        try:return write(node,changes,client,'change')
        except Exception as error:return dict(status=0,ms=(time.perf_counter()-started)*1000,passed=False,error=type(error).__name__,receipt='private journal; no automatic new request')
    def reads(client,index):
        gate.wait();out=[]
        for attempt in range(20):
            node=[rename,moved][(index+attempt)%2];started=time.perf_counter()
            try:status,value,ms=client.call('GET','/api/v1/departments/'+node['id']+'/tree-location')
            except Exception as error:
                out.append(dict(status=0,ms=(time.perf_counter()-started)*1000,passed=False,error=type(error).__name__));continue
            metric=dict(status=status,ms=ms,passed=False,capacityRequest=client.capacity_request)
            if status==200:
                actual=[{k:n.get(k) for k in ['id','code','name','parentId']} for n in value['items']]
                metric['passed']=value['businessAt']==context['currentDate'] and actual in [before_paths[node['id']],after_paths[node['id']]]
                if not metric['passed']:metric['error']='mixed_path'
            else:metric['error']=value.get('code')
            out.append(metric)
        return out
    try:
        result['storageBefore'] = storage_observation(private)
        save()
        with concurrent.futures.ThreadPoolExecutor(max_workers=12) as pool:
            a=pool.submit(writer,rename,dict(name=new_name),clients[0]);b=pool.submit(writer,moved,dict(parentDepartmentId=destination['id']),clients[1])
            reading=[pool.submit(reads,client,i) for i,client in enumerate(clients[2:])]
            result['writes']=[a.result(),b.result()]
            result['reads']=[row for future in reading for row in future.result()];save()
        # Never allocate a fresh request to conceal a failed/unknown write.
        # Its durable request remains for explicit diagnosis and exact recovery.
        if all(row['passed'] for row in result['writes']):
            result['restorations']=[write(rename,dict(name=rename['name']),clients[0],'restore'),write(moved,dict(parentDepartmentId=moved['parentId']),clients[1],'restore')]
            for node in [rename,moved]:
                status,value,ms=clients[0].call('GET','/api/v1/departments/'+node['id']+'/tree-location')
                actual=[{k:n.get(k) for k in ['id','code','name','parentId']} for n in value.get('items',[])]
                result['restoredReads'].append(dict(status=status,ms=ms,passed=status==200 and actual==before_paths[node['id']],capacityRequest=clients[0].capacity_request))
        result['completed']=True
        result['passed']=len(result['reads'])==200 and len(result['restorations'])==2 and len(result['restoredReads'])==2 and all(row['passed'] for group in ['reads','writes','restorations','restoredReads'] for row in result[group])
        result['storageAfter'] = storage_observation(private)
        result['workloadWALBytes'] = lsn_number(result['storageAfter']['walLSN']) - lsn_number(result['storageBefore']['walLSN'])
        result['walScope'] = 'entire mixed workload including session/security activity; not index-only write amplification'
        result['businessSourceDelta'] = result['storageAfter']['departmentSources'] - result['storageBefore']['departmentSources']
        result['passed'] = result['passed'] and result['businessSourceDelta'] == 4
        save()
    finally:journal.close()


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('private',type=Path);parser.add_argument('base');parser.add_argument('output',type=Path);run(parser.parse_args())
