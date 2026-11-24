#!/usr/bin/env python3
"""T01 list semantics on a disposable database of the current model5.

Expected values come from retained independent input bundles, never HTTP or the
candidate's query columns. Public evidence contains no labels, cookies or DSNs.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import threading
from collections import Counter
from datetime import date, timedelta
import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import time
from urllib.parse import urlencode
from zoneinfo import ZoneInfo
from datetime import datetime

ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case, resource


def module(name,path):
    spec=importlib.util.spec_from_file_location(name,path)
    value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value


def expected(shape,source,day):
    rows={}
    if shape=='synthetic':
        oracle=case.Oracle(source)
        current={};byid={}
        for code,(kind,owner) in oracle.bycode.items():
            background=re.fullmatch('R6_[BD]_[0-9]{5}',code) is not None
            timeline=(oracle.proto[kind]['Frames'][-1] if background else oracle.frame(code))['State']['Timeline']
            live=next((r for r in timeline['Records'] if r['Date']<=day and (not r['End'] or day<r['End'])),None)
            r=live or timeline['Records'][0]
            oid=owner['bu_id' if kind=='bu' else 'department_id'];byid[oid]=code
            current[code]=dict(id=oid,name=r['Name'] if kind=='bu' else r['Data']['Name'],effectiveDate=r['Date'],status=r['State'] if live else 'not_effective')
            if kind=='dep':
                bu=case.mapped(f'R6_B_{int(code[-5:])%2000:05d}:',oracle.proto['bu']['Owner']['bu_id']) if background else r['Data']['BU']
                current[code].update(bu=bu,parent=r['Data']['Parent'],shortName=r['Data'].get('ShortName'))
        path_cache={}
        def path(code,at):
            if (code,at) in path_cache:return path_cache[code,at]
            kind,_=oracle.bycode[code]
            frame=(oracle.proto[kind]['Frames'][-1] if re.fullmatch('R6_[BD]_[0-9]{5}',code) else oracle.frame(code))['State']['Timeline']
            r=next((r for r in reversed(frame['Records']) if r['Date']<=at),frame['Records'][0])
            parent=r['Data']['Parent']
            value=(path(byid[parent],at)+' / ' if parent else '')+r['Data']['Name'];path_cache[code,at]=value;return value
        for code,r in current.items():
            if 'bu' not in r:continue
            path_day=day if r['status']=='active' else r['effectiveDate'] if r['status']=='not_effective' else (date.fromisoformat(r['effectiveDate'])-timedelta(days=1)).isoformat()
            bu=current[byid[r['bu']]]
            rows[code]=dict(**{k:v for k,v in r.items() if k not in ['bu','parent']},code=code,managementBu=dict(id=bu['id'],code=byid[r['bu']],name=bu['name']),path=path(code,path_day),pathBusinessAt=path_day,pathMode={'active':'active','inactive':'inactive_before_stop','not_effective':'future_first'}[r['status']])
        shape_facts=dict(maxFanout=max(Counter(r.get('parent') for r in current.values() if r.get('parent')).values()),maxPathNodes=max(v['path'].count(' / ')+1 for v in rows.values()))
    else:
        mod=module('vanke_oracle',ROOT/'tests/capacity/vanke127/verify.py');oracle=mod.Oracle(source)
        root=oracle.root;bu=oracle.record_at(oracle.frames[oracle.heads[root]],day)
        for code,obj in oracle.objects.items():
            if obj['kind']!='departments':continue
            source_id=oracle.heads[code];r=oracle.record_at(oracle.frames[source_id],day)
            rows[code]=dict(id=obj['id'],code=code,name=r[1],shortName=None,effectiveDate=r[0],status='active',managementBu=dict(id=oracle.objects[root]['id'],code=root,name=bu[1]),path=' / '.join(n[3] for n in oracle.path(code,source_id,day,current=True)),pathMode='active',pathBusinessAt=day)
        shape_facts=dict(objects=len(rows))
    return rows,shape_facts


def run(args):
    private=args.private.resolve();source=args.source.resolve();output=args.output
    assert ROOT not in private.parents and private.stat().st_mode&0o777==0o700
    assert ROOT not in source.parents and not output.exists()
    resource(private);case.BIN=args.binary.resolve()
    oracle_files=['organization.enterprise.jsonl.gz']+(['owners.json','bu-prototype.json','dep-prototype.json','bu-typed.jsonl.gz','dep-typed.jsonl.gz'] if args.shape=='synthetic' else ['import.sqlite','history-order.json'])
    def input_hashes():
        result={}
        for name in oracle_files:
            with (source/name).open('rb') as stream:result[name]=hashlib.file_digest(stream,'sha256').hexdigest()
        return result
    inputs=input_hashes()
    day=datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
    rows,shape_facts=expected(args.shape,source,day)
    ordered={sort:sorted(rows.values(),key=lambda v:v['id']) for sort in ['code','name','name_desc']}
    for sort,values in ordered.items():values.sort(key=lambda v:v['code' if sort=='code' else 'name'],reverse=sort=='name_desc')
    result=dict(shape=args.shape,driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),oracleInputSHA256=inputs,fullTraversalWorkers=args.full_workers,expectedObjects=len(rows),inputShape=shape_facts,businessAt=day,binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),samples=[],complete=False,passed=False,capacityAcceptance=False)
    lock=threading.Lock();stop=threading.Event()
    def save():output.write_text(json.dumps(result,indent=2)+'\n')
    def record(sample):
        with lock:
            result['samples'].append(sample);save()
    server=case.Server(private,private);save()
    def check(client,role,page,size,sort,extra=None):
        extra=extra or {}
        sample=dict(role=role,page=page,pageSize=size,sort=sort,ok=False,queryFields=sorted(extra))
        candidates=ordered[sort]
        if extra.get('q'):
            needle=extra['q'].lower()
            candidates=[r for r in candidates if needle in r['code'].lower() or needle in r['name'].lower()]
        if extra.get('filterField')=='departmentPath':
            needle=extra['filterValue'].lower()
            candidates=[r for r in candidates if needle in r['path'].lower()]
        try:
            v,ms=client.call('GET','/api/v1/departments?'+urlencode(dict(page=page,pageSize=size,sort=sort,**extra)))
            sample['ms']=ms
            assert v['businessAt']==day and v['page']==page and v['pageSize']==size
            if extra.get('filterField')=='departmentPath':
                assert 'count' not in v, 'path list uses the current cursor contract'
                assert bool(v.get('hasMore'))==(page*size<len(candidates))
                assert bool(v.get('nextCursor'))==bool(v.get('hasMore'))
            else:
                assert v['count']==len(candidates)
            assert 'selectionVersion' not in v and len(v['queryKey'])==64
            wanted=candidates[(page-1)*size:page*size]
            assert len(v['items'])==len(wanted)
            for actual,want in zip(v['items'],wanted,strict=True):
                for field in ['id','code','name','shortName','effectiveDate','status','managementBu']:assert actual.get(field)==want[field],field
                path=actual['path']
                for field in ['departmentPath','pathMode','pathBusinessAt']:assert path.get(field)==want['path' if field=='departmentPath' else field],field
                assert len(path['pathVersion'])==64 and not path.get('pathSegments')
            sample['rows']=len(wanted);sample['ok']=True
        except Exception as e:
            sample.update(error=type(e).__name__,status=getattr(e,'status',None),code=getattr(e,'code',None),ms=getattr(e,'elapsed_ms',sample.get('ms')))
            stop.set();record(sample);raise
        record(sample)
        return v.get('nextCursor')
    try:
        base=server.start();assert server.info['model']=='m02-compact-5'
        for role in ['admin','reader']:
            client=case.Client();client.base=base;login,_=client.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));client.csrf=login['csrf']
            for size in [20,50,100]:
                last=(len(rows)+size-1)//size
                for sort in ordered:
                    for page in sorted({1,(last+1)//2,last,last+1}):check(client,role,page,size,sort)
            if args.shape=='synthetic':
                fan=[r for code,r in rows.items() if code.startswith('R6_FAN_')]
                assert len(fan)==2001
                root_path=min((r['path'] for r in fan),key=len)
                cursor=None
                for page in range(1,(len(fan)+99)//100+1):
                    query=dict(q='R6_FAN_',filterField='departmentPath',filterOperator='contains',filterValue=root_path)
                    if cursor:query['cursor']=cursor
                    cursor=check(client,role,page,100,'code',query)
                assert not cursor, 'full path traversal must end without another cursor'
                check(client,role,1,100,'code',dict(q='R6_DEPTH_'))
            if role=='admin' and args.full:
                def traverse(worker):
                    actor=case.Client();actor.base=base
                    session,_=actor.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));actor.csrf=session['csrf']
                    for page in range(worker+1,(len(rows)+99)//100+1,args.full_workers):
                        if stop.is_set():return
                        check(actor,role,page,100,'code')
                # Independent read-only pages and sessions; timings are semantic
                # diagnostics, never the fixed 1/20-session capacity matrix.
                with ThreadPoolExecutor(max_workers=args.full_workers) as workers:
                    jobs=[workers.submit(traverse,i) for i in range(args.full_workers)]
                    try:
                        for job in jobs:job.result()
                    except BaseException:
                        stop.set();raise
        assert input_hashes()==inputs,'independent expected inputs changed during verification'
        result['complete']=True;result['passed']=all(v['ok'] for v in result['samples'])
    finally:
        server.close();save()
    print(json.dumps({k:result[k] for k in ['shape','expectedObjects','complete','passed']}))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('shape',choices=['synthetic','vanke']);p.add_argument('private',type=Path);p.add_argument('source',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--full',action='store_true');p.add_argument('--full-workers',type=int,choices=[1,2],default=1);run(p.parse_args())
