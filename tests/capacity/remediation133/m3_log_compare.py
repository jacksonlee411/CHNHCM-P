#!/usr/bin/env python3
"""Read-only same-database log projection comparison; no capacity acceptance."""
import argparse,hashlib,json,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from measure import case,stats,measured

def run(a):
 assert not a.output.exists()
 oracle=case.Oracle(a.private);paths={c:oracle.path(c) for c in ['R6_PROTO_BU','R6_LONG_HISTORY']}
 result=dict(completed=False,capacityAcceptance=False,groups={},responsesEqual=True,errors=[],binaries={})
 expected={}
 for label,binary in [('baseline',a.baseline),('candidate',a.binary)]:
  case.BIN=binary.resolve();result['binaries'][label]=hashlib.sha256(case.BIN.read_bytes()).hexdigest()
  server=case.Server(a.private,a.private)
  try:
   server.start();c=case.Client();c.base=server.base;login,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=login['csrf']
   for code,path in paths.items():
    for page in [1,2,999999]:
     tag=f'{code}-page{page}';samples=[]
     for _ in range(20):
      sample,value=measured(c,'GET',path+f'/changes?page={page}&pageSize=20')
      if sample['ok']:
       digest=hashlib.sha256(json.dumps(value,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
       if tag not in expected:expected[tag]=digest
       if digest!=expected[tag]:sample.update(ok=False,semanticError=True);result['responsesEqual']=False
       sample['responseBytes']=len(json.dumps(value,ensure_ascii=False).encode());sample['total']=value['total'];sample['returnedOperations']=len(value['items'])
      samples.append(sample)
     result['groups'][label+'/'+tag]=stats(samples,None)
     a.output.write_text(json.dumps(result,indent=2)+'\n')
  finally:server.close()
 result['completed']=True;result['passed']=result['responsesEqual'] and all(g['failures']==0 for g in result['groups'].values())
 a.output.write_text(json.dumps(result,indent=2)+'\n')
 if not result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--baseline',type=Path,required=True);run(p.parse_args())
