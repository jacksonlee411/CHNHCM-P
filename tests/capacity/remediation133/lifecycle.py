#!/usr/bin/env python3
"""Real 87343-member/20-session product drafts; no prototype admission claims."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/query131'))
from lists import case,resource
from growth import sql

def pin_owned_port(container):
 # Docker's anonymous host-port assignment changes on container restart. Pin
 # this explicitly disposable fixture before starting the API, preserving its
 # existing volume and all data. Never print its environment (DB credentials).
 spec=json.loads(subprocess.check_output(['docker','inspect',container]))[0]
 task={'chnhcm-storage130-b':'storage130','chnhcm-storage126-product':'storage126-product'}[container]
 labels=spec['Config']['Labels'];assert labels['io.chnhcm.temporary']=='true' and labels['io.chnhcm.parent-task']=='storage130' and labels['io.chnhcm.task']==task
 mounts=spec['Mounts'];assert len(mounts)==1 and mounts[0]['Type']=='volume' and mounts[0]['Name']==container+'-data'
 binding=spec['HostConfig']['PortBindings']['5432/tcp'][0]
 assert binding['HostIp']=='127.0.0.1'
 if binding['HostPort']:return False
 port=spec['NetworkSettings']['Ports']['5432/tcp'][0];assert port['HostIp']=='127.0.0.1'
 subprocess.run(['docker','stop','--time','30',container],check=True,capture_output=True)
 subprocess.run(['docker','rm',container],check=True,capture_output=True)
 command=['docker','run','-d','--name',container,'-p','127.0.0.1:'+port['HostPort']+':5432','-v',mounts[0]['Name']+':'+mounts[0]['Destination']]
 for k,v in labels.items():command+=['--label',k+'='+v]
 for value in spec['Config']['Env']:command+=['-e',value]
 command+=[spec['Config']['Image']]+spec['Config']['Cmd']
 subprocess.run(command,check=True,capture_output=True)
 deadline=time.monotonic()+60
 while subprocess.run(['docker','exec',container,'pg_isready','-U','postgres','-d','storage126'],capture_output=True).returncode:
  assert time.monotonic()<deadline;time.sleep(.5)
 return True

def run(args):
 private=args.private.resolve();assert private.stat().st_mode&0o777==0o700 and ROOT not in private.parents
 assert not args.output.exists()
 container=resource(private);case.BIN=args.binary.resolve();server=case.Server(private,private)
 result=dict(completed=False,product=True,prototype=False,binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),members=87343,sessions=20,steps=[],errors=[],formalCapacity=False)
 def save():args.output.write_text(json.dumps(result,indent=2)+'\n')
 def measure(name,fn):
  began=time.monotonic();entry=dict(name=name,ok=False);result['steps'].append(entry)
  try:value=fn();entry['ok']=True;return value
  except Exception as error:entry.update(error=type(error).__name__,status=getattr(error,'status',None),code=getattr(error,'code',None));raise
  finally:entry['ms']=(time.monotonic()-began)*1000;save()
 def state():
  return json.loads(sql(container,"""SELECT jsonb_build_object(
   'drafts',(SELECT count(*) FROM organization.department_selection_draft),
   'members',(SELECT coalesce(sum(member_count),0) FROM organization.department_selection_draft),
   'selected',(SELECT coalesce(sum(selected_count),0) FROM organization.department_selection_draft),
   'blocks',(SELECT count(*) FROM organization.department_selection_block),
   'maximumBlock',(SELECT coalesce(max(cardinality(members)),0) FROM organization.department_selection_block),
   'logicalBytes',(SELECT coalesce(sum(pg_column_size(d)),0) FROM organization.department_selection_draft d)+(SELECT coalesce(sum(pg_column_size(b)),0) FROM organization.department_selection_block b),
   'physicalBytes',pg_total_relation_size('organization.department_selection_draft')+pg_total_relation_size('organization.department_selection_block'),
   'ttlExact',(SELECT coalesce(bool_and(expires_at-created_at=interval '30 minutes'),true) FROM organization.department_selection_draft),
   'walLSN',pg_current_wal_lsn()::text)"""))
 def parallel(fn,clients):
  def one(pair):
   index,client=pair;began=time.monotonic()
   try:fn(client);return dict(client=index,ok=True,ms=(time.monotonic()-began)*1000)
   except Exception as error:return dict(client=index,ok=False,ms=(time.monotonic()-began)*1000,error=type(error).__name__,status=getattr(error,'status',None),code=getattr(error,'code',None))
  with ThreadPoolExecutor(max_workers=20) as pool:samples=list(pool.map(one,enumerate(clients)))
  result['steps'][-1]['samples']=samples;save()
  assert all(s['ok'] for s in samples),'parallel failures retained in all-client samples'
  return samples
 def start(client):
  v,_=client.call('POST','/api/v1/departments/selection/start',dict(query={},selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20))
  assert v['summary']['selectedCount']==87343
  client.draft=v['summary'];client.ids=[v['page']['items'][0]['id'],v['page']['items'][-1]['id']]
  return v
 def body(client,**edit):return dict(query={},draftId=client.draft['draftId'],expectedRevision=client.draft['revision'],page=1,pageSize=20,**edit)
 def resolve(client,**edit):
  v,_=client.call('POST','/api/v1/departments/selection/resolve',body(client,**edit));client.draft=v['summary'];return v
 clients=[]
 try:
  for key in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE']:os.environ.pop(key,None)
  result['fixturePortPinned']=pin_owned_port(container)
  result['before']=case.cli('size',private);assert result['before']['counts']['department']==87343
  assert sum(result['before']['counts'][k] for k in ['buSources','departmentSources'])==132290
  base=server.start();assert server.info['apiConnectionLimit']==12
  for _ in range(20):
   c=case.Client();c.base=base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];clients.append(c)
  result['empty']=state()
  measure('20 concurrent full captures',lambda:parallel(start,clients));result['created']=state();assert result['created']['members']==20*87343 and result['created']['drafts']==20 and result['created']['ttlExact']
  created=result['created'];assert created['maximumBlock']<=1024 and created['logicalBytes']<=20*3*1024**2
  result['creationWALBytes']=int(sql(container,f"SELECT pg_wal_lsn_diff('{created['walLSN']}'::pg_lsn,'{result['empty']['walLSN']}'::pg_lsn)::bigint"))
  assert created['physicalBytes']<=96*1024**2 and result['creationWALBytes']<=160*1024**2
  def exclude(c):
   v=resolve(c,removeIds=c.ids);assert v['summary']['selectedCount']==87341 and not set(c.ids)&set(v['selectedIds'])
  measure('20 concurrent exclusions',lambda:parallel(exclude,clients))
  def restore(c):assert resolve(c,restoreIds=c.ids)['summary']['selectedCount']==87343
  measure('20 concurrent restores',lambda:parallel(restore,clients))
  # A normal database stop/start preserves logged intent. API remains the same
  # process; new health traffic is separately recorded, not a hidden retry.
  before_ids=[c.draft['draftId'] for c in clients]
  measure('normal database stop',lambda:subprocess.run(['docker','stop','--time','30',container],check=True,capture_output=True))
  measure('normal database start',lambda:subprocess.run(['docker','start',container],check=True,capture_output=True))
  deadline=time.monotonic()+60
  while subprocess.run(['docker','exec',container,'pg_isready','-U','postgres','-d','storage126'],capture_output=True).returncode:
   assert time.monotonic()<deadline;time.sleep(.5)
  result['afterDBRestart']=state();assert result['afterDBRestart']['members']==20*87343
  measure('resolve all original drafts after DB restart',lambda:parallel(lambda c:resolve(c),clients))
  assert [c.draft['draftId'] for c in clients]==before_ids
  def cancel(c):c.call('POST','/api/v1/departments/selection/cancel',dict(draftId=c.draft['draftId'],expectedRevision=c.draft['revision']),want=204)
  measure('cancel five owner drafts',lambda:parallel(cancel,clients[:5]))
  measure('logout five sessions',lambda:parallel(lambda c:c.call('DELETE','/api/v1/session',want=204),clients[5:10]))
  assert state()['drafts']==10
  # Make five actual product drafts eligible for expiry using one fixed time;
  # production TTL stays 30 minutes. The real periodic sweeper must remove them.
  expiry_ids=','.join("'"+c.draft['draftId']+"'::uuid" for c in clients[10:15])
  sql(container,f"UPDATE organization.department_selection_draft SET created_at=statement_timestamp()-interval '31 minutes',expires_at=statement_timestamp()-interval '1 minute' WHERE draft_id IN ({expiry_ids})")
  for c in clients[10:15]:
   _,_=c.call('POST','/api/v1/departments/selection/resolve',body(c),want=409)
  began=time.monotonic()
  while state()['drafts']!=5:
   assert time.monotonic()-began<65;time.sleep(1)
  result['periodicExpiryReclaimMS']=(time.monotonic()-began)*1000
  result['expiryMethod']='five real draft timestamps advanced to expiry; unchanged 30-minute TTL and real periodic sweep'
  measure('normal API restart',lambda:server.restart(clients));assert state()['drafts']==0
  for c in clients[15:]:c.call('POST','/api/v1/departments/selection/resolve',body(c),want=409)
  # Reuse the fifteen non-revoked sessions; renew five after the unchanged
  # login window has elapsed, then measure sustained replacement on twenty.
  live=clients[:5]+clients[10:]
  for _ in range(5):
   c=case.Client();c.base=server.base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];live.append(c)
  result['replacementRounds']=[]
  for index in range(10):
   measure(f'sustained full replacement round {index+1}',lambda:parallel(start,live))
   current=state();assert current['drafts']==20 and current['members']==20*87343 and current['physicalBytes']<=512*1024**2
   result['replacementRounds'].append(current);save()
  measure('final cancellation',lambda:parallel(cancel,live));assert state()['drafts']==0 and state()['blocks']==0
  result['completed']=True
 finally:
  server.close();result['after']=case.cli('size',private)
  result['businessSourceGrowth']={k:result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources','departmentSources']}
  result['passed']=result['completed'] and not any(result['businessSourceGrowth'].values());save()
 return result
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',required=True,type=Path)
 run(p.parse_args())
