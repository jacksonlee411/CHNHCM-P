#!/usr/bin/env python3
"""Tree300 deterministic matrix and independent workbook/synthetic graph oracle.
Private identities, session data and fixture ledgers never enter the report.
Every formal attempt, including HTTP/semantic errors, is retained.
"""
import argparse
import atexit
import concurrent.futures
from collections import Counter
import csv
import hashlib
import io
import json
import math
import os
from pathlib import Path
import sqlite3
import sys
import subprocess
import time
import threading
import urllib.error
import urllib.request
import urllib.parse

ROOT=Path(__file__).resolve().parents[3]
sys.path.append(str(ROOT/'tests/capacity'))
import path_targets
import tree_scope_targets

sys.path.append(str(ROOT/'tests/capacity/vanke127'))
from load import Client
from policy import sparse_reads
from frozen import owned_database as frozen_database, nodes as frozen_nodes


class MeasuredClient(Client):
    def call(self, method, path, body=None, key=None):
        headers={'Content-Type':'application/json','Origin':self.base,'X-CSRF-Token':self.csrf}
        if key:headers['Idempotency-Key']=key
        request=urllib.request.Request(self.base+path,method=method,headers=headers,data=None if body is None else json.dumps(body,ensure_ascii=False).encode())
        started=time.perf_counter()
        self.response_bytes=0;self.capacity_request=None
        try:
            response=self.http.open(request,timeout=25)
        except urllib.error.HTTPError as error:response=error
        with response:
            raw=response.read();self.response_bytes=len(raw);self.capacity_request=response.headers.get("X-Capacity-Request")
            return response.status,json.loads(raw),(time.perf_counter()-started)*1000


class ProcessObservation:
    def __init__(self,pid):
        self.pid=pid;self.stop=threading.Event();self.samples=[]
    def read(self):
        if self.pid is None:return None
        try:
            stats=Path('/proc/'+str(self.pid)+'/stat').read_text().split(') ',1)[1].split()
            status=Path('/proc/'+str(self.pid)+'/status').read_text().splitlines()
            return dict(cpuSeconds=(int(stats[11])+int(stats[12]))/os.sysconf('SC_CLK_TCK'),rssKB=int(next(x for x in status if x.startswith('VmRSS:')).split()[1]))
        except (OSError,ValueError,StopIteration):return None
    def start(self):
        def sample():
            while not self.stop.is_set():
                value=self.read()
                if value:self.samples.append(value)
                self.stop.wait(.1)
        self.thread=threading.Thread(target=sample,daemon=True);self.thread.start()
    def finish(self):
        self.stop.set();self.thread.join()
        if not self.samples:return dict(available=False)
        return dict(available=True,cpuSeconds=self.samples[-1]['cpuSeconds']-self.samples[0]['cpuSeconds'],peakRSSKB=max(s['rssKB'] for s in self.samples),samples=len(self.samples))


def digest(value):
    return hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode()).hexdigest()


def target_for(name,sessions,fixed):
    if name in tree_scope_targets.CASES:return tree_scope_targets.limits(name,sessions,phase='warm')['stageTargetMS']
    if fixed is not None:return fixed
    return (1000 if sessions==1 else 5000) if name.startswith(('start-','resolve-')) else (300 if sessions==1 else 1000)


def oracle(private, day):
    if (private/'tree300-frozen.json').exists():
        frozen_database(private)
        return frozen_nodes(private, day)
    synthetic=private/'tree300-synthetic.json'
    if synthetic.exists():
        data=json.loads(synthetic.read_text())
        assert data['completed'] and len(data['nodes'])==87343
        return data['nodes']
    db=sqlite3.connect(private/'import.sqlite')
    objects={code:(oid,kind) for code,oid,kind in db.execute('SELECT code,object_id,kind FROM objects')}
    assert len(objects)==87344
    names,parents,starts={}, {}, {}
    for phase,raw,result in db.execute('SELECT phase,payload,result FROM rows ORDER BY phase,row_no'):
        assert result is not None,'full original replay is required before capacity'
        r=json.loads(raw);code=r['org_code']
        if phase=='create':
            names[code]={r['effective_date']:r['name']};parents[code]={r['effective_date']:r.get('parent_org_code','')};starts[code]=r['effective_date']
    for phase,raw in db.execute("SELECT phase,payload FROM rows WHERE phase<>'create' ORDER BY CASE phase WHEN 'version' THEN 0 ELSE 1 END,row_no"):
        r=json.loads(raw);code=r['org_code']
        if phase=='version' and r.get('core.name'):names[code][r['effective_date']]=r['core.name']
        elif phase=='parent-change':parents[code][r.get('target_effective_date') or r['effective_date']]=r['new_parent_org_code']
    def current(events):return events[max(d for d in events if d<=day)]
    nodes=[]
    for code,(identity,kind) in objects.items():
        if kind!='departments':continue
        parent=current(parents[code]);p=objects[parent] if parent else ('','business-units')
        nodes.append(dict(id=identity,code=code,name=current(names[code]),parentId=p[0] if p[1]=='departments' else None))
    db.close();assert len(nodes)==87343
    return nodes


def fixtures(nodes):
    byid={n['id']:n for n in nodes};children={None:[]};depth={}
    for n in nodes:children.setdefault(n['parentId'],[]).append(n)
    for rows in children.values():rows.sort(key=lambda n:(n['code'].lower(),n['id']))
    def path(identity):
        chain=[];seen=set()
        while identity:
            assert identity not in seen;seen.add(identity);node=byid[identity];chain.append(node);identity=node['parentId']
        return chain[::-1]
    for node in nodes:depth[node['id']]=len(path(node['id']))
    wide=max((p for p in children if p),key=lambda p:len(children[p]));deep=max(nodes,key=lambda n:depth[n['id']])
    def branch(identity):
        out=[];stack=[byid[identity]]
        while stack:
            n=stack.pop();out.append(n);stack.extend(children.get(n['id'],[]))
        return sorted(out,key=lambda n:n['code'])
    large=max(children[None],key=lambda n:len(branch(n['id'])))
    return byid,children,path,branch,wide,deep,large,dict(nodes=len(nodes),maximumDepth=max(depth.values()),maximumSiblings=max(map(len,children.values())),roots=len(children[None]),graphSHA256=digest(sorted(nodes,key=lambda n:n['code'])))



def table_work(private, container):
    frozen = frozen_database(private, container)
    synthetic=(private/'tree300-synthetic.json').exists()
    if frozen:
        container,database=frozen
    elif synthetic:
        expected='chnhcm-bu-'+hashlib.sha256((str(ROOT)+':tree300-synthetic-d').encode()).hexdigest()[:12]+'-postgres-1'
        assert container==expected
    else:
        assert container in [None,'chnhcm-vanke127'];container='chnhcm-vanke127'
    if not frozen:
        labels=json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Config.Labels}}',container],text=True))
        assert labels.get('io.chnhcm.temporary')=='true'
        assert labels.get('io.chnhcm.profile')=='tree300-synthetic-d' if synthetic else labels.get('io.chnhcm.task')=='vanke127'
        database='chnhcm' if synthetic else 'vanke127'
    statement="SELECT coalesce(json_object_agg(relname,json_build_object('seqRows',seq_tup_read,'indexRows',idx_tup_fetch,'seqScans',seq_scan,'indexScans',idx_scan)),'{}'::json) FROM pg_stat_user_tables WHERE schemaname='organization' AND relname IN ('department','department_read_segment','department_hierarchy_relation','department_dictionary','department_shape','bu_read_segment')"
    return json.loads(subprocess.check_output(['docker','exec',container,'psql','-U','postgres','-d',database,'-X','-Atq','-c',statement],text=True))


def work_delta(before, after):
    return {table:{key:values[key]-before[table][key] for key in values} for table,values in after.items()}


def run(args):
    private=args.private.resolve();assert not private.is_relative_to(ROOT)
    assert not args.output.exists(),'preserve earlier results'
    metadata=MeasuredClient(args.base);status,login,_=metadata.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));assert status==200;metadata.csrf=login['csrf']
    status,context,_=metadata.call('GET','/api/v1/organization/context');assert status==200;day=context['currentDate']
    byid,children,path,branch,wide,deep,large,shape=fixtures(oracle(private,day))
    q=lambda params:urllib.parse.urlencode({k:v for k,v in params.items() if v is not None})
    permitted=None
    def authorized(rows):return rows if permitted is None else [n for n in rows if n['id'] in permitted]
    def check_nodes(expected, size=50, previous=None):
        def check(value):
            assert value['businessAt']==day
            wanted=authorized(expected)[:size]
            actual=value['items'];assert [n['id'] for n in actual]==[n['id'] for n in wanted]
            assert value['hasMore']==(len(authorized(expected))>size)
            assert value['hasPrevious']==(previous() if previous else False)
            assert bool(value.get('nextCursor'))==value['hasMore']
            assert bool(value.get('previousCursor'))==value['hasPrevious']
            for n,want in zip(actual,wanted):
                assert all(n[k]==want[k] for k in ['code','name','parentId']) and n['kind']=='node'
                assert n['hasChildren']==bool(authorized(children.get(n['id'],[])))
        return check
    def check_list(expected, page=1):
        def check(value):
            wanted=authorized(expected);position=max(1,(len(wanted)+19)//20) if page==-1 else page
            assert value['businessAt']==day and value['count']==len(wanted)
            assert [r['id'] for r in value['items']]==[r['id'] for r in wanted[(position-1)*20:position*20]]
        return check
    def check_path(value):
        expected=path(deep['id']);assert value['businessAt']==day
        assert [n['id'] for n in value['items']]==[n['id'] for n in expected]
        for n,want in zip(value['items'],expected):
            assert all(n[k]==want[k] for k in ['code','name','parentId'])
            operational=permitted is None or n['id'] in permitted
            assert n['kind']==('node' if operational else 'context')
            assert n.get('hasChildren')==(bool(authorized(children.get(n['id'],[]))) if operational else None)
    def check_path_list(value):
        wanted=authorized(branch(large['id']))
        assert value['businessAt']==day and value.get('count') is None
        assert [n['id'] for n in value['items']]==[n['id'] for n in wanted[:20]]
        assert value['hasMore']==(len(wanted)>20)
    common_name=Counter(n['name'] for n in byid.values()).most_common(1)[0][0]
    all_nodes=sorted(byid.values(),key=lambda n:n['code'])
    large_nodes=branch(large['id'])
    cases=[
      ('roots','GET','/api/v1/departments/tree/nodes',None,check_nodes(children[None]),300),
      ('wide-first','GET','/api/v1/departments/tree/nodes?'+q(dict(parentId=wide)),None,check_nodes(children[wide]),300),
      ('wide-last','GET','/api/v1/departments/tree/nodes?'+q(dict(parentId=wide,aroundId=children[wide][-1]['id'])),None,check_nodes(children[wide][-1:],previous=lambda:len(authorized(children[wide]))>1),300),
      ('deep-children','GET','/api/v1/departments/tree/nodes?'+q(dict(parentId=deep['id'])),None,check_nodes([]),300),
      ('search-exact','GET','/api/v1/departments/tree/search?'+q(dict(q=deep['code'])),None,check_nodes(sorted([n for n in byid.values() if deep['code'].lower() in n['code'].lower() or deep['code'].lower() in n['name'].lower()],key=lambda n:(n['code'].lower(),n['id'])),20),1000),
      ('search-empty','GET','/api/v1/departments/tree/search?q=__TREE300_NO_MATCH__',None,check_nodes([]),1000),
      ('location-deep','GET','/api/v1/departments/'+deep['id']+'/tree-location',None,check_path,1000),
      ('scope-large','GET','/api/v1/departments?'+q(dict(scopeDepartmentId=large['id'])),None,check_list(branch(large['id'])),None),
      ('scope-self','GET','/api/v1/departments?'+q(dict(scopeDepartmentId=large['id'],includeDescendants='false')),None,check_list([large]),None),
    ]
    cases.extend([
      ('search-common-name','GET','/api/v1/departments/tree/search?'+q(dict(q=common_name)),None,check_nodes(sorted([n for n in all_nodes if common_name.lower() in n['name'].lower() or common_name.lower() in n['code'].lower()],key=lambda n:(n['code'].lower(),n['id'])),20),1000),
      ('scope-path','GET','/api/v1/departments?'+q(dict(scopeDepartmentId=large['id'],filterField='departmentPath',filterOperator='contains',filterValue=large['name'])),None,check_path_list,300),
    ])
    for label, params, expected in [('enterprise',{},all_nodes),('large',dict(scopeDepartmentId=large['id']),large_nodes)]:
        # One independently authenticated draft per client. Resolve must keep
        # the same membership, revision and page facts established by start.
        cases.append(('start-'+label,'POST','/api/v1/departments/selection/start',dict(query=params,selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20),check_list(expected),None))
        cases.append(('resolve-'+label,'POST','/api/v1/departments/selection/resolve',dict(query=params,page=1,pageSize=20),check_list(expected),None))
    last_page=max(1,(len(large_nodes)+19)//20)
    cases.append(('scope-large-last','GET','/api/v1/departments?'+q(dict(scopeDepartmentId=large['id'],page=last_page)),None,check_list(large_nodes,-1),None))
    if args.cases:
        assert set(args.cases)<=set(c[0] for c in cases)
        cases=[c for c in cases if c[0] in args.cases]
    sparse_ids={n['id'] for n in all_nodes[::100]}|{deep['id'],large['id'],wide,children[wide][-1]['id']}
    result=dict(version=1,completed=False,formal=not args.diagnostic,shape=shape,day=day,formalPerGroup=0 if args.diagnostic else 200,warmupPerSession=2,groups=[],limitations=['This runner covers ordinary tree reads and fixed selection start/resolve; CSV, mixed changes, cold observations and browser latency are separately required.'])
    manifest=dict(version=1,formal=not args.diagnostic,shape=shape,roles=args.roles,sessions=args.sessions,warmupsPerSession=2,formalPerGroup=0 if args.diagnostic else 200,diagnosticBatches=1 if args.diagnostic else 0,cases=[dict(name=n,method=m,pathTemplate=p.split('?')[0].replace(deep['id'],':id'),targetMS=t) for n,m,p,b,c,t in cases])
    manifest['oracleImplementationSHA256']=hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    path_policy=dict(id=tree_scope_targets.POLICY_ID,contract=tree_scope_targets.CONTRACT,scope='large subtree only',
        phase='warm',stageTargetMS={'1':400,'20':2000},previousStageTargetMS={'1':300,'20':1500},
        longTermTargetMS=path_targets.LONG_TERM_TARGET_MS,
        sourceSHA256=hashlib.sha256(Path(tree_scope_targets.__file__).read_bytes()).hexdigest())
    manifest['pathTargetPolicy']=path_policy
    result['pathTargetPolicy']=path_policy
    scope_policy=dict(id=tree_scope_targets.POLICY_ID,contract=tree_scope_targets.CONTRACT,cases={name:{str(n):tree_scope_targets.limits(name,n,phase='warm') for n in [1,20]} for name in tree_scope_targets.CASES},sourceSHA256=path_policy['sourceSHA256'])
    manifest['largeSubtreeTargetPolicy']=scope_policy;result['largeSubtreeTargetPolicy']=scope_policy
    for case in manifest['cases']:
        if case['name']=='scope-path':
            case.update(targetMS=None,stageTargetMS=path_policy['stageTargetMS'],
                longTermTargetMS=path_targets.LONG_TERM_TARGET_MS)
    manifest['groups']=[dict(name=name,role=role,sessions=sessions,authorizedObjects=len(sparse_ids) if role=='sparse-reader' else len(all_nodes),referencePolicy='all minimal department references',warmupsPerSession=2,attempts=sessions if args.diagnostic else 200,targetMS=target_for(name,sessions,target),expectedGraphSHA256=shape['graphSHA256']) for role in args.roles for sessions in args.sessions for name,method,endpoint,body,check,target in cases]
    for group in manifest['groups']:
        if group['name'] in tree_scope_targets.CASES:group.update(tree_scope_targets.limits(group['name'],group['sessions'],phase='warm'))
    args.output.with_suffix('.matrix.json').write_text(json.dumps(manifest,indent=2)+'\n')
    save=lambda:args.output.write_text(json.dumps(result,indent=2)+'\n')
    save();last_login=[];drafts={}
    journal=args.output.with_suffix('.attempts.jsonl').open('x')
    atexit.register(journal.close)
    journal_lock=threading.Lock()
    def login_clients(role,count):
      clients=[]
      for _ in range(count):
        while len(last_login)>=18 and time.monotonic()-last_login[-18]<61:time.sleep(min(5,61-(time.monotonic()-last_login[-18])))
        client=MeasuredClient(args.base);last_login.append(time.monotonic());status,login,_=client.call('POST','/api/v1/session/login',dict(login='reader' if role=='sparse-reader' else role,password='admin1234567'));assert status==200;client.csrf=login['csrf'];clients.append(client)
      return clients,login
    for role in args.roles:
      permitted=sparse_ids if role=='sparse-reader' else None
      clients,login=login_clients(role,args.sessions[0])
      with sparse_reads(private,login,permitted,args.output.parent.name+'-'+args.output.stem,args.container):
        for sessions in args.sessions:
          if sessions!=args.sessions[0]:clients,_=login_clients(role,sessions)
          drafts.clear()
          for name,method,endpoint,body,check,target in cases:
            if name=='scope-large-last':endpoint='/api/v1/departments?'+q(dict(scopeDepartmentId=large['id'],page=max(1,(len(authorized(large_nodes))+19)//20)))
            target=target_for(name,sessions,target)
            if args.resolve_setup and name.startswith('resolve-'):
              for client in clients:
                for attempt in range(5):
                  status,value,elapsed=client.call('POST','/api/v1/departments/selection/start',dict(query=body['query'],selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20))
                  sample=dict(status=status,ms=elapsed,requestSent=True,responseBytes=client.response_bytes,capacityRequest=client.capacity_request)
                  if status!=200:sample['error']=value.get('code','http_error')
                  journal.write(json.dumps(dict(role=role,sessions=sessions,name=name,phase='setup',sample=sample))+'\n');journal.flush()
                  if status==200:
                    assert value['pageState']=='available' and value['businessAt']==day
                    check(value['page']);assert value['summary']['selectedCount']==value['page']['count']
                    ids=sorted(row['id'] for row in value['page']['items'])
                    assert sorted(value['memberIds'])==ids and sorted(value['selectedIds'])==ids
                    drafts[id(client),name.removeprefix('resolve-')]=value['summary'];break
                else:raise RuntimeError('resolve prerequisite unavailable; setup attempts retained')
            stats_before=table_work(private,args.container)
            samples=[];warmups=[];observation=ProcessObservation(args.server_pid);observation.start();group_started=time.perf_counter()
            def invoke_inner(client):
              started=time.perf_counter()
              try:
                  request=body
                  previous=None
                  if name.startswith('resolve-'):
                      previous=drafts.get((id(client),name.removeprefix('resolve-')))
                      if previous is None:return dict(ms=0,status=0,error='setup_draft_missing',requestSent=False,responseBytes=0,capacityRequest=None)
                      request=dict(body,draftId=previous['draftId'],expectedRevision=previous['revision'])
                  status,value,elapsed=client.call(method,endpoint,request)
                  if status!=200:return dict(ms=elapsed,status=status,error=value.get('code','http_error'))
                  try:
                      if name.startswith(('start-','resolve-')):
                          assert value['pageState']=='available' and value['businessAt']==day
                          check(value['page'])
                          summary=value['summary']
                          assert summary['selectedCount']==value['page']['count']
                          ids=[r['id'] for r in value['page']['items']]
                          assert sorted(value['memberIds'])==sorted(ids) and sorted(value['selectedIds'])==sorted(ids)
                          if previous is not None:assert summary==previous
                          else:drafts[id(client),name.removeprefix('start-')]=summary
                      else:check(value)
                  except Exception:return dict(ms=elapsed,status=status,error='semantic_mismatch')
                  return dict(ms=elapsed,status=status,responseBytes=client.response_bytes,capacityRequest=client.capacity_request)
              except Exception as error:return dict(ms=(time.perf_counter()-started)*1000,status=0,error=type(error).__name__)
            active=0;maximum_active=0;active_lock=threading.Lock();submitted=time.perf_counter()
            def invoke(client):
                nonlocal active,maximum_active
                waited=(time.perf_counter()-submitted)*1000
                with active_lock:active+=1;maximum_active=max(active,maximum_active)
                try:
                    value=invoke_inner(client);value['clientQueueMS']=waited;value.setdefault('requestSent',True)
                    value.setdefault('capacityRequest',client.capacity_request)
                    value.setdefault('responseBytes',client.response_bytes)
                    with journal_lock:
                        journal.write(json.dumps(dict(role=role,sessions=sessions,name=name,phase=phase,sample=value))+'\n');journal.flush()
                    return value
                finally:
                    with active_lock:active-=1
            with concurrent.futures.ThreadPoolExecutor(max_workers=sessions) as pool:
              phase='warmup'
              for _ in range(2):
                  submitted=time.perf_counter();warmups.extend(pool.map(invoke,clients[:sessions]))
              phase='formal'
              for _ in range(1 if args.diagnostic else 200//sessions):
                  submitted=time.perf_counter();samples.extend(pool.map(invoke,clients[:sessions]))
            successful=sorted(s['ms'] for s in samples if 'error' not in s);failures=sum('error' in s for s in samples)
            p95=successful[math.ceil(len(successful)*.95)-1] if successful else None
            group=dict(name=name,role=role,sessions=sessions,maximumClientInFlight=maximum_active,attempts=len(samples),executedRequests=sum(s['requestSent'] for s in samples),failures=failures,successP95MS=p95,targetMS=target,passed=failures==0 and p95 is not None and p95<=target,wallSeconds=time.perf_counter()-group_started,process=observation.finish(),warmups=warmups,samples=samples)
            if name in tree_scope_targets.CASES:
                group.update(tree_scope_targets.limits(name,sessions,phase='warm') if args.diagnostic else tree_scope_targets.assess(group,phase='warm'))
            time.sleep(1.1) # Let idle backend statistics flush; excluded from request samples.
            group['databaseWorkIncludingWarmups']=work_delta(stats_before,table_work(private,args.container))
            group['databaseWorkScope']='isolated-table cumulative counters; includes warmups and may lag; query.jsonl holds exact projected-fact counts'
            result['groups'].append(group);save();print(json.dumps({k:group[k] for k in ['name','role','sessions','attempts','failures','successP95MS','passed']}),flush=True)
    result['completed']=True;result['passed']=all(g['passed'] for g in result['groups']);save();journal.close()

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('base');p.add_argument('output',type=Path);p.add_argument('--server-pid',type=int,required=True);p.add_argument('--container');p.add_argument('--roles',nargs='+',choices=['admin','reader','sparse-reader'],default=['admin','reader','sparse-reader']);p.add_argument('--sessions',nargs='+',type=int,choices=[1,20],default=[1,20]);p.add_argument('--cases',nargs='+');p.add_argument('--resolve-setup',action='store_true',help='prepare an acknowledged draft for every resolve client before timing, journal every setup attempt');p.add_argument('--diagnostic',action='store_true',help='one batch after warmups; explicitly not a formal P95 run');run(p.parse_args())
