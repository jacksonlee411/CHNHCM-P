#!/usr/bin/env python3
"""M1 shared-pool/guard/draft pressure against real independent writes.

Run only after the frozen 132290-source formal matrix. Five new task-owned
objects preserve the original histories. Every attempted operation is retained;
submission lookup handles failures without retrying the business write.
"""
import argparse,hashlib,json,os,sys,time,threading,uuid
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/query131'))
from lists import case,resource
from growth import sql

def run(a):
 private=a.private.resolve();assert private.stat().st_mode&0o777==0o700 and ROOT not in private.parents and not a.output.exists()
 container=resource(private);case.BIN=a.binary.resolve();server=case.Server(private,private)
 result=dict(completed=False,formalCapacity=False,scope='20 simultaneous sessions: 15 M1 reads and 5 independent corrections, four waves; original frozen business rows untouched',binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),preparation=[],samples=[],recovery=[],failures=[])
 receipts=[]
 def save():a.output.write_text(json.dumps(result,indent=2)+'\n')
 def measured(kind,fn):
  began=time.monotonic()
  try:value=fn();return dict(kind=kind,ok=True,ms=(time.monotonic()-began)*1000),value
  except Exception as e:return dict(kind=kind,ok=False,ms=(time.monotonic()-began)*1000,error=type(e).__name__,status=getattr(e,'status',None),code=getattr(e,'code',None)),None
 try:
  for key in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE']:os.environ.pop(key,None)
  result['before']=case.cli('size',private)
  assert result['before']['counts']['department']==87343 and sum(result['before']['counts'][k] for k in ['buSources','departmentSources'])==132290
  base=server.start();assert server.info['apiConnectionLimit']==12
  clients=[]
  for i in range(20):
   c=case.Client();c.base=base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf'];clients.append(c)
  bu=clients[0].call('GET','/api/v1/business-units')[0]['items'][0]['id'];writes=[]
  for i in range(5):
   c=clients[15+i];code='M1_MIX_'+uuid.uuid4().hex[:10]
   body=dict(code=code,name='M1混合读写对象'+str(i),effectiveDate='2026-01-01',managementBuId=bu,operationType='CREATE',operationReason='NEW_ORGANIZATION')
   v,_=c.call('POST','/api/v1/departments',body,str(uuid.uuid4()),want=201);writes.append(v['departmentId']);receipts.append(v)
  total=87348
  def start(c):
   v,_=c.call('POST','/api/v1/departments/selection/start',dict(query={},selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20));assert v['summary']['selectedCount']==total;return v
  for c in clients[3:6]:c.draft=start(c)['summary']
  for wave in range(4):
   prepared=[]
   for i,oid in enumerate(writes):
    detail,_=clients[15+i].call('GET','/api/v1/departments/'+oid);record=detail['selectedRecord']
    prepared.append((oid,record['recordId'],dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=record['revision'],changes=dict(name=f'M1混合修改{wave}-{i}')),str(uuid.uuid4())))
   barrier=threading.Barrier(20,timeout=30)
   def job(index):
    c=clients[index];kind=['capture','resolve','path','parent','list'][index//3] if index<15 else 'write'
    def operation():
     if kind=='capture':return start(c)
     if kind=='resolve':
      v,_=c.call('POST','/api/v1/departments/selection/resolve',dict(query={},draftId=c.draft['draftId'],expectedRevision=c.draft['revision'],page=2,pageSize=20));assert v['summary']['selectedCount']==total;return v
     if kind=='path':
      v,_=c.call('GET','/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue=%E7%BB%84%E7%BB%87');assert 'count' not in v and len(v['items'])==20 and v['hasMore'];return v
     if kind=='parent':
      v,_=c.call('POST','/api/v1/departments/references/parents',dict(context=dict(mode='create',managementBuId=bu,effectiveDate='2026-01-01'),pageSize=20));assert len(v['items'])==20 and v['hasMore'];return v
     if kind=='list':
      v,_=c.call('GET','/api/v1/departments');assert v['count']==total;return v
     oid,rid,body,key=prepared[index-15]
     v,_=c.call('POST',f'/api/v1/departments/{oid}/records/{rid}/correct',body,key);assert v['outcome']=='accepted' and v['submissionId']==key;return v
    barrier.wait();sample,value=measured(kind,operation);sample.update(wave=wave,session=index)
    return sample,value
   with ThreadPoolExecutor(max_workers=20) as pool:outcomes=list(pool.map(job,range(20)))
   for index,(sample,value) in enumerate(outcomes):
    result['samples'].append(sample)
    if index>=15:
     oid,rid,body,key=prepared[index-15];c=clients[index]
     if value:receipts.append(value)
     def recover():
      v,_=c.call('POST','/api/v1/departments/submissions/'+key+'/resolve')
      if value:assert all(v.get(k)==val for k,val in value.items() if k!='currentChanged')
      else:assert v['outcome'] in ['accepted','not_applied','rejected']
      return v
     recovered,v=measured('original-result',recover);result['recovery'].append(recovered)
     if v and v['outcome']=='accepted':
      detail,_=c.call('GET','/api/v1/departments/'+oid);assert detail['selectedRecord']['name']==body['changes']['name']
   save()
  result['completed']=True
  server.restart(clients)
  for v in receipts:
   recovered,_=clients[0].call('POST','/api/v1/departments/submissions/'+v['submissionId']+'/resolve')
   assert all(recovered.get(k)==val for k,val in v.items() if k!='currentChanged')
  result['restartReceiptsVerified']=len(receipts)
  assert sql(container,"SELECT count(*) FROM pg_stat_activity WHERE application_name='chnhcm' AND state='idle in transaction'")=='0'
 finally:
  server.close();result['after']=case.cli('size',private)
  result['sourceGrowth']={k:result['after']['counts'][k]-result['before']['counts'][k] for k in ['buSources','departmentSources']}
  result['passed']=result['completed'] and all(s['ok'] for s in result['samples']+result['recovery']) and result.get('restartReceiptsVerified')==25 and result['sourceGrowth']==dict(buSources=0,departmentSources=25)
  groups={}
  for kind in ['capture','resolve','path','parent','list','write']:
   rows=[s for s in result['samples'] if s['kind']==kind];times=sorted(s['ms'] for s in rows)
   groups[kind]=dict(requests=len(rows),failures=sum(not s['ok'] for s in rows),p95MS=times[max(0,__import__('math').ceil(len(times)*.95)-1)] if times else None)
  result['groups']=groups;save()
 return result
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);a=p.parse_args();r=run(a)
 if not r['passed']:raise SystemExit(1)
