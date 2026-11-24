#!/usr/bin/env python3
"""Read every selected preview-fixture receipt after an owned DB restart."""
import argparse,hashlib,json,subprocess,sys,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,sql

def run(a):
 assert not a.output.exists() and a.private.parent.name in ['remediation133-m3-20261112','remediation133-coverage-20261113']
 case.BIN=a.binary.resolve();name=resource(a.private);receipts=[]
 for p in (a.receipts or sorted(a.private.glob('*preview*-receipts.json'))):
  assert p.resolve().parent==a.private.resolve() and p.name.endswith('-receipts.json')
  receipts+=json.loads(p.read_text())
 assert receipts and len({r['submissionId'] for r in receipts})==len(receipts)
 def state():
  return json.loads(sql(name,"SELECT json_build_object('sources',(SELECT sum(cardinality(source_refs)) FROM organization.bu_history)+(SELECT sum(cardinality(source_refs)) FROM organization.department_history),'terminals',(SELECT count(*) FROM organization.bu_terminal)+(SELECT count(*) FROM organization.department_terminal),'heads',(SELECT json_object_agg(id,source) FROM (SELECT bu_id id,head_source_id source FROM organization.business_unit UNION ALL SELECT department_id,head_source_id FROM organization.department) h))"))
 before=state();result=dict(completed=False,expected=len(receipts),recovered=0,exactSources=0,failures=[],binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),capacityAcceptance=False)
 server=case.Server(a.private,a.private)
 try:
  subprocess.run(['docker','restart',name],check=True,stdout=subprocess.DEVNULL)
  for _ in range(100):
   r=subprocess.run(['docker','exec',name,'pg_isready','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   if r.returncode==0:break
   time.sleep(.1)
  assert r.returncode==0;result['databaseRestarted']=True
  server.start();c=case.Client();c.base=server.base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf']
  for index,receipt in enumerate(receipts):
   try:
    kind='departments' if 'departmentId' in receipt else 'business-units';oid=receipt.get('departmentId',receipt.get('buId'));path='/api/v1/'+kind+'/'+oid
    resolve='/api/v1/departments/submissions/' if kind=='departments' else '/api/v1/organization/submissions/'
    v,_=c.call('POST',resolve+receipt['submissionId']+'/resolve')
    assert set(receipt)<=set(v) and set(v)-set(receipt)<={'expectedSourceVersion'}
    assert all(v[k]==expected for k,expected in receipt.items() if k!='currentChanged')
    assert v['currentChanged']==(before['heads'][oid]!=receipt['sourceVersion'])
    if 'expectedSourceVersion' in set(v)-set(receipt):
     log,_=c.call('GET',path+'/changes?throughSourceVersion='+receipt['sourceVersion']+'&pageSize=20')
     assert log['items'][0]['sourceVersion']==receipt['sourceVersion'] and log['items'][0]['previousSourceVersion']==v['expectedSourceVersion']
    result['recovered']+=1
    v,_=c.call('GET',path+'?sourceVersion='+receipt['sourceVersion'])
    assert v['readOnly'] and v['resolvedSourceVersion']==receipt['sourceVersion'];result['exactSources']+=1
   except Exception as e:result['failures'].append(dict(index=index,error=type(e).__name__,code=getattr(e,'code',None)))
  result['completed']=True
 finally:
  server.close();after=state();result['newSources']=after['sources']-before['sources'];result['newTerminals']=after['terminals']-before['terminals'];result['passed']=result['completed'] and not result['failures'] and before==after and result['expected']==result['recovered']==result['exactSources'];a.output.write_text(json.dumps(result,indent=2)+'\n')
 if not result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--receipts',type=Path,nargs='+');run(p.parse_args())
