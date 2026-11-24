#!/usr/bin/env python3
"""R129 complete BU previews: full shapes, 1/20 sessions, warm/cold separated.

Two explicit ordinary fixture writes add and remove one future CHANGE record.
All timed requests are read-only; fixture receipts stay in the private case.
"""
import argparse,copy,hashlib,json,sys,uuid
from pathlib import Path
from types import SimpleNamespace
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from measure import Lane,parallel,measured,stats
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,sql,save

def run(a):
 assert not a.output.exists()
 case.BIN=a.binary.resolve(); name=resource(a.private)
 oid=sql(name,"SELECT bu_id FROM organization.business_unit ORDER BY code LIMIT 1")
 path='/api/v1/business-units/'+oid
 server=case.Server(a.private,a.private)
 result=dict(completed=False,binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),groups={},coldResets=[],before=case.cli('size',a.private),apiConnections=12,actions=['CORRECT','NO_CHANGE','ADD','REDATE','DELETE'],fixtureWrites=0,diagnosticOnly=a.diagnostic)
 receipts=[];clients=[]
 try:
  server.start();assert server.info['apiConnectionLimit']==12
  for _ in range(20):
   c=case.Client();c.base=server.base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];clients.append(c)
  detail,_=clients[0].call('GET',path);original=copy.deepcopy(detail['timeline']);last=original[-1];assert last['effectiveDate']<'2055-01-01'
  body=dict(expectedSourceVersion=detail['headSourceVersion'],anchorRecordId=last['recordId'],effectiveDate='2055-01-01',operationType='CHANGE',operationReason='NAME_ADJUSTMENT',changes=dict(name='129预览未来资料'))
  key=str(uuid.uuid4());v,_=clients[0].call('POST',path+'/records',body,key,want=201);assert v['outcome']=='accepted';receipts.append(v);result['fixtureWrites']+=1
  detail,_=clients[0].call('GET',path);first,last=detail['timeline'][0],detail['timeline'][-1]
  head=detail['headSourceVersion'];bodies={}
  for action in result['actions']:
   b=dict(action=action,expectedSourceVersion=head)
   if action=='ADD':b.update(anchorRecordId=last['recordId'],effectiveDate='2056-01-01',operationType='CHANGE',operationReason='NAME_ADJUSTMENT',changes=dict(name='129只读新增辅助'))
   else:
    record=first if action in ['CORRECT','NO_CHANGE'] else last
    b.update(recordId=record['recordId'],expectedRecordRevision=record['revision'])
    if action in ['CORRECT','NO_CHANGE']:b.update(action='CORRECT',changes=dict(name=first['name'] if action=='NO_CHANGE' else '129只读更正辅助'))
    if action=='REDATE':b['newEffectiveDate']='2055-01-02'
   bodies[action]=b
  def check(action,value):
   assert value['outcome']==('no_change' if action=='NO_CHANGE' else 'accepted') and value['expectedSourceVersion']==head
   assert 'name' in value['readFields']
   assert not set(['timeline','defaults','beforeRecord','targetPreviousState','successorSnapshotsPreserved','writeFields','referenceCoverage']).intersection(value)
   if action in ['REDATE','DELETE']:assert value.get('afterRecord') and value['currentStatus']==detail['status']
  lane=SimpleNamespace(server=server,result=result,output=a.output,clients={'admin':clients})
  for action,body in bodies.items():
   check(action,clients[0].call('POST',path+'/preview',body)[0])
   for sessions in ([1] if a.diagnostic else [1,20]):
    for phase in (['warm'] if a.diagnostic else ['warm','cold']):
     samples=[]
     for block in range(1 if phase=='warm' else 5):
      if phase=='cold':Lane.cold(lane,f'{action}-{sessions}-{block}')
      for _ in range(((20 if a.diagnostic else 200) if phase=='warm' else 20)//sessions):
       samples+=parallel([lambda c=c:measured(c,'POST',path+'/preview',body,check=lambda v:check(action,v))[0] for c in clients[:sessions]])
     result['groups'][f'{action}-{sessions}-{phase}']=stats(samples,300);save(a.output,result)
  after,_=clients[0].call('GET',path);assert after['headSourceVersion']==head and after['timeline']==detail['timeline']
  key=str(uuid.uuid4());v,_=clients[0].call('POST',path+'/records/'+last['recordId']+'/delete',dict(expectedSourceVersion=head,expectedRecordRevision=last['revision']),key)
  assert v['outcome']=='accepted';receipts.append(v);result['fixtureWrites']+=1
  restored,_=clients[0].call('GET',path);assert len(restored['timeline'])==len(original)
  for got,want in zip(restored['timeline'],original):
   for field in ['recordId','effectiveDate','name','nameEn','state','operationType','operationReason']:assert got.get(field)==want.get(field)
  result['completed']=True
 finally:
  server.close();save(a.private/(a.output.stem+'-receipts.json'),receipts)
  result['after']=case.cli('size',a.private)
  result['businessSourceGrowth']={k:result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources','departmentSources']}
  result['passed']=result['completed'] and all(g['passed'] for g in result['groups'].values()) and result['businessSourceGrowth']==dict(buSources=2,departmentSources=0)
  save(a.output,result)
 print(json.dumps({k:result[k] for k in ['completed','passed','businessSourceGrowth']}))
 if not result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--diagnostic',action='store_true');run(p.parse_args())
