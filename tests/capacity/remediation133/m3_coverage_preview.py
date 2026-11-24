#!/usr/bin/env python3
"""Coverage-changing BU preview on both full shapes, including all failures.

A future deactivation would strand existing open-ended departments. The
independent expected result is a 409 reference conflict, never a save. A 503 is
retained as a failed request. Each 1/20-session warm/cold group keeps 200/100.
"""
import argparse,hashlib,json,sys,time
from pathlib import Path
from types import SimpleNamespace
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from measure import Lane,parallel,stats
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,sql,save

def run(a):
 assert not a.output.exists();case.BIN=a.binary.resolve();container=resource(a.private)
 oid=sql(container,'SELECT bu_id FROM organization.business_unit ORDER BY code LIMIT 1');path='/api/v1/business-units/'+oid
 # Prove a dependent extends beyond the proposed stop using the retained input
 # oracle, independently of the candidate response.
 if a.private.name=='synthetic':
  oracle=case.Oracle(a.private);dependent=oracle.frame('R6_D_00000')['State']['Timeline'];assert dependent['Records'][-1]['End']=='' and dependent['Records'][-1]['State']=='active' and dependent['Records'][-1]['Data']['BU']==oid
 witness=int(sql(container,"SELECT count(*) FROM organization.department_read_segment WHERE q_bu='"+oid+"' AND q_state='active' AND upper_inf(span)"))
 assert witness>0
 server=case.Server(a.private,a.private);clients=[]
 terminal_count=lambda:int(sql(container,'SELECT (SELECT count(*) FROM organization.bu_terminal)+(SELECT count(*) FROM organization.department_terminal)'))
 before_terminals=terminal_count()
 result=dict(completed=False,expectedTimedRequests=600,actualTimedRequests=0,groups={},coldResets=[],expectedStatus=409,expectedCode='M02_DEPARTMENT_REFERENCE_CONFLICT',openEndedReferenceWitnesses=witness,formalPreviewGroups=True,full125CapacityAcceptance=False,binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),before=case.cli('size',a.private))
 try:
  server.start();assert server.info['apiConnectionLimit']==12
  for _ in range(20):
   c=case.Client();c.base=server.base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];clients.append(c)
  detail,_=clients[0].call('GET',path);last=detail['timeline'][-1];assert last['state']=='active' and last['effectiveDate']<'2057-01-01'
  body=dict(action='DEACTIVATE',expectedSourceVersion=detail['headSourceVersion'],anchorRecordId=last['recordId'],effectiveDate='2057-01-01',operationType='DEACTIVATE',operationReason='ORG_RESTRUCTURING')
  lane=SimpleNamespace(server=server,result=result,output=a.output,clients={'admin':clients})
  def request(c):
   started=time.perf_counter()
   try:
    value,elapsed=c.call('POST',path+'/preview',body,want=409)
    return dict(ok=value.get('code')==result['expectedCode'],ms=elapsed,status=409,code=value.get('code'),semanticError=value.get('code')!=result['expectedCode'])
   except Exception as e:return dict(ok=False,ms=getattr(e,'elapsed_ms',(time.perf_counter()-started)*1000),status=getattr(e,'status',None),code=getattr(e,'code',None),errorType=type(e).__name__,httpError=True)
  for sessions in [1,20]:
   for phase in ['warm','cold']:
    samples=[]
    for block in range(1 if phase=='warm' else 5):
     if phase=='cold':Lane.cold(lane,f'coverage-change-{sessions}-{block}')
     for _ in range((200 if phase=='warm' else 20)//sessions):
      samples+=parallel([lambda c=c:request(c) for c in clients[:sessions]])
      result['actualTimedRequests']+=sessions
      if result['actualTimedRequests']%20==0:print(json.dumps(dict(sessions=sessions,phase=phase,attempted=result['actualTimedRequests'])),flush=True)
     result['groups'][f'{sessions}-{phase}']=stats(samples,300);save(a.output,result)
  after,_=clients[0].call('GET',path);assert after['headSourceVersion']==detail['headSourceVersion'] and after['timeline']==detail['timeline'];result['completed']=result['actualTimedRequests']==600
 finally:
  server.close();result['after']=case.cli('size',a.private);result['newSources']={k:result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources','departmentSources']};result['newTerminals']=terminal_count()-before_terminals;result['passed']=result['newTerminals']==0 and result['completed'] and all(g['passed'] for g in result['groups'].values()) and not any(result['newSources'].values());save(a.output,result)
 if not result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);run(p.parse_args())
