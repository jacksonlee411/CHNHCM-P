#!/usr/bin/env python3
"""Normal runtime HTTP verification after complete historical import."""
import argparse
import concurrent.futures
import json
import math
from pathlib import Path
import signal
import subprocess
import time
import uuid

from load import Client
from runtime import ROOT,BIN,cli,command
from verify import Oracle


def statistics(samples, target, elapsed=None):
    values=sorted(x['ms'] for x in samples if x['ok']);p95=values[max(0,math.ceil(len(values)*.95)-1)] if values else None
    result=dict(requests=len(samples),successful=len(values),failed=len(samples)-len(values),p95MS=p95,maxMS=max(values) if values else None,targetMS=target,passed=len(values)==len(samples) and p95 is not None and p95<=target,samples=samples)
    if elapsed is not None:
        result.update(batchSeconds=elapsed, successfulPerSecond=len(values)/elapsed)
    return result


def measured(client,method,path,body=None,key=None,check=None,want=(200,)):
    start=time.perf_counter()
    try:
        status,result,ms=client.call(method,path,body,key)
        if status not in want:return dict(ok=False,ms=ms,httpStatus=status,code=result.get('code')),None
        if check:check(result)
        return dict(ok=True,ms=ms,httpStatus=status),result
    except Exception as error:
        # Exception strings may include source names or URLs; never persist them.
        return dict(ok=False,ms=(time.perf_counter()-start)*1000,errorType=type(error).__name__),None


class Server:
    def __init__(self,private):self.private=private;self.process=None;self.log=(private/'measurement-server.log').open('a')
    def start(self):
        self.process=subprocess.Popen(command('serve',self.private),cwd=ROOT,stdout=subprocess.PIPE,stderr=self.log,text=True)
        self.info=json.loads(self.process.stdout.readline());self.base=self.info['url']
        return self.base
    def stop(self):
        if self.process and self.process.poll() is None:self.process.send_signal(signal.SIGTERM);self.process.wait(timeout=10)
        self.process=None
    def restart(self,clients):
        self.stop();base=self.start()
        for c in clients:c.base=base


def parallel(jobs):
    with concurrent.futures.ThreadPoolExecutor(max_workers=20) as pool:return list(pool.map(lambda fn:fn(),jobs))


def accepted(value):
    assert value['outcome']=='accepted'


def run(private,out):
    oracle=Oracle(private);result=dict(scope='complete V127 dataset; normal runtime without historical-name adapter',sessions=20,groups={},checks=[],completed=False,cacheScope='cold API process; PostgreSQL and OS caches remain natural')
    save=lambda:out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    result['before']=cli('size',private)
    source_audit=cli('source-audit',private)
    result['sourceAdmission']=source_audit
    assert result['before']['counts']['department']==87343 and result['before']['counts']['departmentSources']==132283
    server=Server(private);clients=[];codes=sorted(c for c in oracle.objects if c!=oracle.root)
    sample_codes=[codes[i*len(codes)//20] for i in range(20)]
    most_history=max(codes,key=lambda c:len(oracle.events[c]));sample_codes[0]=most_history
    leaves=set(codes)
    for code in codes:
        for row in oracle.frames[oracle.heads[code]]['records']:leaves.discard(row[2])
    write_codes=sorted(leaves)[-20:]
    def route(code):return '/api/v1/'+oracle.objects[code]['kind']+'/'+oracle.objects[code]['id']
    def check_list(value,kind):
        expected=[oracle.root] if kind=='business-units' else codes
        assert value['count']==len(expected) and len(value['items'])==min(20,len(expected))
        assert [x['code'] for x in value['items']]==expected[:20]
        for item,code in zip(value['items'],expected):
            row=oracle.record_at(oracle.frames[oracle.heads[code]],value['businessAt'])
            assert item['id']==oracle.objects[code]['id'] and item['name']==row[1]
    try:
        base=server.start()
        for i in range(20):
            c=Client(base);status,login,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));assert status==200;c.csrf=login['csrf'];clients.append(c)
            if i<19:time.sleep(3.1)
        result['beforeReads']=cli('size',private);save()
        for kind in ['business-units','departments']:
            for state in ['cold-api','warm']:
                if state=='cold-api':server.restart(clients)
                begin=time.perf_counter()
                samples=parallel([lambda c=c:measured(c,'GET','/api/v1/'+kind,check=lambda v:check_list(v,kind))[0] for c in clients])
                result['groups'][kind+'-list-'+state]=statistics(samples,300,time.perf_counter()-begin);save();print(kind,state,result['groups'][kind+'-list-'+state]['p95MS'],flush=True)
        for label,old in [('detail',False),('exact-source',True)]:
            begin=time.perf_counter();samples=[]
            for _ in range(3):
                jobs=[]
                for c,code in zip(clients,sample_codes):
                    source=oracle.events[code][0][1] if old else None
                    path=route(code)+('?sourceVersion='+source if source else '')
                    jobs.append(lambda c=c,code=code,source=source,path=path:measured(c,'GET',path,check=lambda v:oracle.check_detail(code,v,source))[0])
                samples+=parallel(jobs)
            result['groups'][label]=statistics(samples,500 if old else 300,time.perf_counter()-begin);save()
        # Original identities are resolved under the importing subject, not invented.
        restore=[]
        for c,code in zip(clients,sample_codes):
            frame=oracle.frames[oracle.events[code][0][1]]
            def check(v,source=frame['source']):assert v['sourceVersion']==source and v['outcome']=='accepted'
            restore.append(measured(c,'POST','/api/v1/departments/submissions/'+frame['submission']+'/resolve',check=check)[0])
        result['groups']['import-result-resolve']=statistics(restore,500)
        no_changes=[]
        for (phase,no),(raw,stored) in oracle.rows.items():
            if stored['outcome']!='no_change':continue
            expected=stored['response']
            def check(v,expected=expected):
                for key in ['submissionId','outcome','observedSourceVersion','selectedRecordId']:
                    assert v.get(key)==expected.get(key)
            no_changes.append(measured(clients[len(no_changes)%20],'POST','/api/v1/departments/submissions/'+stored['requestKey']+'/resolve',check=check)[0])
        assert len(no_changes)==18
        result['groups']['all-no-change-result-resolve']=statistics(no_changes,500)
        special=[]
        for field,numbers in source_audit['invalidRowNumbers'].items():
            phase=field.split('.')[0]
            for no in numbers:
                raw,stored=oracle.rows[phase,no];code=raw['org_code'];source=stored['response']['sourceVersion'];target=stored['response']['targetRecordId']
                path=route(code)+'?sourceVersion='+source+'&selectedRecordId='+target
                special.append(measured(clients[len(special)%20],'GET',path,check=lambda v,code=code,source=source:oracle.check_detail(code,v,source))[0])
        assert len(special)==37
        result['groups']['all-historical-names-exact-source']=statistics(special,500)
        result['afterReads']=cli('size',private)
        result['readSecurityEventGrowth']=result['afterReads']['counts']['securityEvents']-result['beforeReads']['counts']['securityEvents']
        assert result['readSecurityEventGrowth']==0, 'ordinary reads persisted authorization events'
        save()
        write_begin=time.perf_counter();writes=[];results=[];readback=[]
        for round in range(2):
            jobs=[]
            for i,(c,code) in enumerate(zip(clients,write_codes)):
                sample,detail=measured(c,'GET',route(code))
                if detail is None:writes.append(dict(**sample,preparation=True));continue
                row=detail['timeline'][-1];key=str(uuid.uuid4());short='127验证'+str(round)+'-'+str(i)
                body=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=row['revision'],changes=dict(shortName=short))
                jobs.append(lambda c=c,code=code,row=row,key=key,body=body,short=short:(measured(c,'POST',route(code)+'/records/'+row['recordId']+'/correct',body,key,check=accepted),c,code,key,short,row['recordId']))
            for (sample,value),c,code,key,short,record in parallel(jobs):
                writes.append(sample)
                if value:
                    def check(v,code=code,record=record,short=short):
                        expected=next(r for r in oracle.frames[oracle.heads[code]]['records'] if r[3]==record)
                        selected=v['selectedRecord']
                        assert selected['recordId']==record and selected['shortName']==short and selected['name']==expected[1]
                        assert selected['managementBu']['id']==oracle.objects[oracle.root]['id']
                        parent=selected.get('parentDepartment')
                        assert (parent['id'] if parent else None)==(oracle.objects[expected[2]]['id'] if expected[2]!=oracle.root else None)
                    readback.append(measured(c,'GET',route(code)+'?selectedRecordId='+record,check=check)[0])
                    results.append(dict(key=key,source=value['sourceVersion'],code=code))
            save()
        result['groups']['concurrent-short-name-save']=statistics(writes,500,time.perf_counter()-write_begin)
        result['groups']['concurrent-short-name-save']['batchScope']='40 concurrent writes in two rounds, including sequential detail preparation and readback'
        result['groups']['write-readback-and-unchanged-facts']=statistics(readback,300)
        mixed_begin=time.perf_counter();mixed_writes=[];mixed_reads=[]
        for round in range(4):
            jobs=[]
            for i,c in enumerate(clients):
                if i>=10:
                    code=sample_codes[i];source=oracle.events[code][0][1]
                    path=route(code)+'?sourceVersion='+source
                    jobs.append(lambda c=c,code=code,source=source,path=path:('read',measured(c,'GET',path,check=lambda v:oracle.check_detail(code,v,source)),None))
                    continue
                code=write_codes[i];sample,detail=measured(c,'GET',route(code))
                if detail is None:mixed_writes.append(dict(**sample,preparation=True));continue
                row=detail['timeline'][-1];key=str(uuid.uuid4());short='混合资料'+str(round)+'-'+str(i)
                body=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=row['revision'],changes=dict(shortName=short))
                jobs.append(lambda c=c,code=code,row=row,key=key,body=body:('write',measured(c,'POST',route(code)+'/records/'+row['recordId']+'/correct',body,key,check=accepted),dict(key=key,code=code)))
            for kind,(sample,value),identity in parallel(jobs):
                (mixed_reads if kind=='read' else mixed_writes).append(sample)
                if kind=='write' and value:results.append(dict(**identity,source=value['sourceVersion']))
        mixed_elapsed=time.perf_counter()-mixed_begin
        result['mixedBatch']=dict(seconds=mixed_elapsed,requests=len(mixed_writes)+len(mixed_reads),successfulPerSecond=sum(x['ok'] for x in mixed_writes+mixed_reads)/mixed_elapsed,scope='four rounds of ten writes and ten exact-source reads; includes sequential write preparation')
        result['groups']['mixed-ten-writers-save']=statistics(mixed_writes,500)
        result['groups']['mixed-ten-readers-exact-source']=statistics(mixed_reads,500)
        save()
        server.restart(clients)
        restored=[]
        for i,row in enumerate(results):
            def check(v,row=row):assert v['sourceVersion']==row['source'] and v['outcome']=='accepted'
            restored.append(measured(clients[i%20],'POST','/api/v1/departments/submissions/'+row['key']+'/resolve',check=check)[0])
        result['groups']['restart-write-result-resolve']=statistics(restored,500)
        result['afterWrites']=cli('size',private)
        result['onlineMaintenanceAfterWrites']=cli('maintain',private)
        result['afterOnlineMaintenance']=cli('size',private)
        # Preserve only task-local private identities for subsequent browser checks.
        path=private/'measured-writes.json';path.touch(mode=0o600,exist_ok=True);path.write_text(json.dumps(results))
        result['completed']=True;save()
    finally:
        server.stop();server.log.close();save()
    print(json.dumps(dict(completed=result['completed'],groups={k:{x:v[x] for x in ['requests','successful','failed','p95MS','passed']} for k,v in result['groups'].items()})),flush=True)


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('out',type=Path);a=p.parse_args();run(a.private,a.out)
