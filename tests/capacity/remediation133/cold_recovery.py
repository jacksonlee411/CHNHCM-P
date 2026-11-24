#!/usr/bin/env python3
"""Recover the complete final affected-capacity receipt set after DB restart."""
import argparse,hashlib,json,subprocess,sys,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import case,resource,sql

def run(a):
 assert not a.output.exists()
 case.BIN=a.binary.resolve();name=resource(a.private);receipts=[];counts={}
 for receipt_path in a.receipts:
  assert receipt_path.resolve().parent == a.private.resolve()
  rows=json.loads(receipt_path.read_text());counts[receipt_path.name]=len(rows);receipts+=rows
 assert receipts and len({r['receipt']['submissionId'] for r in receipts})==len(receipts)
 def state():
  return json.loads(sql(name,"SELECT json_build_object('sources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history)+(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history),'terminals',(SELECT count(*) FROM organization.bu_terminal)+(SELECT count(*) FROM organization.department_terminal),'heads',(SELECT json_object_agg(id,source) FROM (SELECT bu_id id,head_source_id source FROM organization.business_unit UNION ALL SELECT department_id,head_source_id FROM organization.department) h))"))
 before=state();result=dict(completed=False,expected=len(receipts),phaseReceipts=counts,recovered=0,exactSources=0,failures=[],binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),capacityAcceptance=False)
 server=case.Server(a.private,a.private);original_timelines={}
 result['wholeTimelineCorrectionsChecked']=0
 try:
  subprocess.run(['docker','restart',name],check=True,stdout=subprocess.DEVNULL)
  for _ in range(100):
   r=subprocess.run(['docker','exec',name,'pg_isready','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   if r.returncode==0:break
   time.sleep(.1)
  assert r.returncode==0;result['databaseRestarted']=True
  server.start();c=case.Client();c.base=server.base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf']
  for index,row in enumerate(receipts):
   try:
    receipt=row['receipt'];v,_=c.call('POST',row['route'])
    assert set(receipt)<=set(v) and set(v)-set(receipt)<={'expectedSourceVersion'}
    assert all(v[k]==expected for k,expected in receipt.items() if k!='currentChanged')
    assert v['currentChanged']==(before['heads'][row['path'].rsplit('/',1)[1]]!=receipt['sourceVersion'])
    if 'expectedSourceVersion' in set(v)-set(receipt):
     log,_=c.call('GET',row['path']+'/changes?throughSourceVersion='+receipt['sourceVersion']+'&pageSize=20')
     assert log['items'][0]['sourceVersion']==receipt['sourceVersion'] and log['items'][0]['previousSourceVersion']==v['expectedSourceVersion']
    result['recovered']+=1
    v,_=c.call('GET',row['path']+'?sourceVersion='+receipt['sourceVersion']);assert v['readOnly'] and v['resolvedSourceVersion']==receipt['sourceVersion'];result['exactSources']+=1
    expected=row.get('expectedCorrection')
    if expected:
     key=(row['path'],expected['originalSource'])
     if key not in original_timelines:
      original,_=c.call('GET',row['path']+'?sourceVersion='+expected['originalSource'])
      assert original['readOnly'] and original['resolvedSourceVersion']==expected['originalSource']
      original_timelines[key]=original['timeline']
     original=original_timelines[key];actual=v['timeline']
     assert len(original)==len(actual)==expected['records'];targets=0
     for before_record,after_record in zip(original,actual,strict=True):
      if before_record['recordId']!=expected['recordId']:
       assert after_record==before_record,'unmodified record changed'
       continue
      targets+=1;wanted=dict(before_record,name=expected['name'],revision=receipt['targetRevision'])
      # BU timeline rows omit the optional summary; department rows may carry
      # it. Preserve absence and reject losing a summary already projected.
      if 'operationSummary' in after_record:
       summary=after_record['operationSummary']
       assert summary['recordId']==expected['recordId'] and summary['revision']==receipt['targetRevision'] and summary['operationSourceVersion']==receipt['sourceVersion']
       source_log,_=c.call('GET',row['path']+'/changes?throughSourceVersion='+receipt['sourceVersion']+'&pageSize=20')
       source_log=source_log['items'][0]
       assert source_log['sourceVersion']==receipt['sourceVersion']
       assert summary['recordedAt']==receipt['recordedAt']==source_log['recordedAt']
       assert summary['operator']==source_log['operator']
       wanted['operationSummary']=summary
      else:assert 'operationSummary' not in before_record,'operation summary disappeared'
      assert after_record==wanted,'target correction changed unexpected facts'
     assert targets==1;result['wholeTimelineCorrectionsChecked']+=1
   except Exception as e:result['failures'].append(dict(index=index,error=type(e).__name__,code=getattr(e,'code',None)))
  result['completed']=True
 finally:
  server.close();after=state();result['newSources']=after['sources']-before['sources'];result['newTerminals']=after['terminals']-before['terminals'];result['passed']=result['completed'] and not result['failures'] and before==after and result['expected']==result['recovered']==result['exactSources'];a.output.write_text(json.dumps(result,indent=2)+'\n')
 if not result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--receipts',type=Path,nargs='+',required=True);run(p.parse_args())
