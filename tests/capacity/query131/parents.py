#!/usr/bin/env python3
"""T03 parent facts on the complete preserved Vanke input, never a smaller seed.

Expected identities and labels derive from the independent import workbook log.
This serial stage diagnostic is not T04's 1/20-session capacity matrix.
"""
import argparse
from collections import Counter
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import sys
import threading
import time
from zoneinfo import ZoneInfo

from lists import module, ROOT, case, resource
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import sql


def run(args):
    private=args.private.resolve();assert ROOT not in private.parents and private.stat().st_mode&0o777==0o700
    assert not args.output.exists()
    container=resource(private);case.BIN=args.binary.resolve()
    oracle=module('vanke_parent_oracle',ROOT/'tests/capacity/vanke127/verify.py').Oracle(private)
    day=datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
    departments={c:v for c,v in oracle.objects.items() if v['kind']=='departments'}
    assert len(departments)==87343
    timelines={c:oracle.frames[oracle.heads[c]]['records'] for c in departments}
    referenced={r[2] for records in timelines.values() for r in records}
    target=max((c for c in departments if c not in referenced),key=lambda c:(timelines[c][-1][0],c))
    bu=oracle.objects[oracle.root]['id']
    # The retained independent inputs contain only active CREATE/CHANGE facts,
    # one BU, and no descendant ever refers to this selected leaf. For this
    # shape an eligible parent is precisely an existing object at W's start.
    prefixes=Counter(c[:2] for c in departments)
    search=max(prefixes,key=prefixes.get)
    assert prefixes[search]>20
    result=dict(completed=False,passed=False,shape='full-vanke',objects=87343,concurrency=1,capacityAcceptance=False,
                binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),samples=[],sourceCountBefore=sql(container,'SELECT sum(cardinality(source_refs)) FROM organization.department_history'))
    result['oracleInputSHA256']={n:hashlib.sha256((private/n).read_bytes()).hexdigest() for n in ['import.sqlite','history-order.json']}
    def save():args.output.write_text(json.dumps(result,indent=2)+'\n')
    def expected(mode,needle):
        start=day if mode=='create' else timelines[target][-1][0]
        rows=[]
        for c,obj in departments.items():
            if mode=='maintain' and c==target or timelines[c][0][0]>start:continue
            r=oracle.record_at(oracle.frames[oracle.heads[c]],day)
            if needle and needle.lower() not in (c+' '+r[1]).lower():continue
            rows.append(dict(id=obj['id'],code=c,name=r[1],eligibility='eligible'))
        return sorted(rows,key=lambda r:(r['code'],r['id']))
    stats=private/(args.output.stem+'-work.jsonl');diagnostics=private/(args.output.stem+'-stages');diagnostics.mkdir(mode=0o700)
    os.environ['CHNHCM_QUERY131_STATS']=str(stats);os.environ['CHNHCM_CAPACITY_DIAGNOSTICS']=str(diagnostics)
    server=case.Server(private,private);stop=threading.Event();rss=[]
    def sample():
        while not stop.wait(.05):
            if server.p:
                try:rss.append(int(next(s.split()[1] for s in Path('/proc',str(server.p.pid),'status').read_text().splitlines() if s.startswith('VmRSS:'))))
                except (FileNotFoundError,StopIteration):pass
    monitor=threading.Thread(target=sample);monitor.start();consumed=0
    try:
        server.start();assert server.info['model']=='m02-compact-5';save()
        for role in ['admin','reader']:
            client=case.Client();client.base=server.base;session,_=client.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));client.csrf=session['csrf']
            detail,_=client.call('GET','/api/v1/departments/'+departments[target]['id']);record=detail['timeline'][-1]
            assert record['effectiveDate']==timelines[target][-1][0]
            for mode in ['create','maintain']:
                context=dict(mode=mode,effectiveDate=day,managementBuId=bu) if mode=='create' else dict(mode=mode,departmentId=departments[target]['id'],expectedSourceVersion=detail['headSourceVersion'],action='CORRECT',recordId=record['recordId'],expectedRecordRevision=record['revision'])
                for group,needle,page in [('unsearched-first','',1),('searched-first',search,1),('unsearched-second','',2)]:
                    wanted=[] if mode=='create' and role=='reader' else expected(mode,needle)
                    if not (mode=='create' and role=='reader'):assert len(wanted)>20
                    sampleResult=dict(role=role,mode=mode,group=group,page=page,expectedCount=len(wanted),ok=False)
                    started=time.perf_counter()
                    try:
                        body=dict(context=context,q=needle,pageSize=20)
                        response,_=client.call('POST','/api/v1/departments/references/parents',body)
                        sampleResult['httpRequests']=1
                        if page==2 and response['hasMore']:
                            assert response['items']==wanted[:20]
                            body['cursor']=response['nextCursor']
                            response,_=client.call('POST','/api/v1/departments/references/parents',body)
                            sampleResult['httpRequests']=2
                        ms=(time.perf_counter()-started)*1000
                        sampleResult.update(status=200,ms=ms)
                        assert response['applicable'] and 'count' not in response
                        assert response['hasMore']==(len(wanted)>page*20)
                        assert response['items']==wanted[(page-1)*20:page*20]
                        sampleResult.update(ok=True,rows=len(response['items']))
                    except Exception as error:
                        sampleResult.update(error=type(error).__name__,status=getattr(error,'status',sampleResult.get('status')),code=getattr(error,'code',None),ms=(time.perf_counter()-started)*1000)
                    finally:
                        sampleResult['request']=client.request_id
                        work=[json.loads(line) for line in stats.read_text().splitlines()] if stats.exists() else []
                        sampleResult['work']=work[consumed:];consumed=len(work)
                        result['samples'].append(sampleResult);save()
        result['sourceCountAfter']=sql(container,'SELECT sum(cardinality(source_refs)) FROM organization.department_history')
        assert result['sourceCountAfter']==result['sourceCountBefore']
        result['completed']=True;result['passed']=all(v['ok'] for v in result['samples'])
    finally:
        server.close();stop.set();monitor.join();result['peakRSSKiB']=max(rss,default=None)
        result['observedRequestsHaveTimingInstrumentation']=True
        stageFile=diagnostics/'requests.jsonl'
        if stageFile.exists():
            stages={str(v['Request']):v for v in (json.loads(line) for line in stageFile.read_text().splitlines())}
            for s in result['samples']:
                entries=stages.get(str(s.get('request')),{}).get('Stages',[])
                s['qualifiedTraversals']=sum(x['Name']=='references.qualify' for x in entries)
                s['captureCalls']=sum(x['Name']=='references.capture' for x in entries)
                s['captureMS']=sum(x['DurationMS'] for x in entries if x['Name']=='references.capture')
        save()
    print(json.dumps(dict(passed=result['passed'],samples=len(result['samples']),peakRSSKiB=result['peakRSSKiB'])))
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);run(p.parse_args())
