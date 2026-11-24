#!/usr/bin/env python3
"""Full-fixture CSV staging cost/failure prototype, not a product exporter."""
import argparse,csv,json,os,sys,tempfile,time
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/query131'))
from lists import expected

def run(private,output):
 assert not output.exists() and private.stat().st_mode&0o777==0o700
 rows,_=expected('vanke',private,datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d'))
 ordered=sorted(rows.values(),key=lambda r:(r['code'],r['id']))
 assert len(ordered)==87343
 budget=64*1024*1024
 result=dict(rows=len(ordered),budgetBytes=budget,scope='storage and zero-send prototype only; no authorization or production CSV parity claim',completed=False)
 def fields(r):
  bu=r.get('managementBu')or{}
  return [r.get('code'),r.get('name'),r.get('shortName'),bu.get('code'),bu.get('name'),r.get('path'),r.get('status'),r.get('effectiveDate')]
 for late_failure in [False,True]:
  received=0;path=None;start=time.perf_counter()
  try:
   with tempfile.NamedTemporaryFile(mode='w',encoding='utf-8-sig',newline='',dir=private,delete=False)as f:
    path=Path(f.name);assert path.stat().st_mode&0o777==0o600
    writer=csv.writer(f);writer.writerow(['code','name','short','BU code','BU name','path','status','date'])
    for i,row in enumerate(ordered):
     if late_failure and i==len(ordered)-1:raise PermissionError('late refusal fixture')
     writer.writerow(fields(row))
    f.flush();size=path.stat().st_size;assert size<=budget
   with path.open(encoding='utf-8-sig',newline='')as f:
    reader=csv.reader(f);next(reader)
    for actual,row in zip(reader,ordered,strict=True):
     assert actual==['' if x is None else str(x)for x in fields(row)]
   received=size
   result.update(csvBytes=size,stageAndVerifyMS=1000*(time.perf_counter()-start))
  except PermissionError:
   result['lateFailureBytesSent']=received;assert late_failure and received==0
  finally:
   if path:path.unlink()
 result['completed']=True;output.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);a=p.parse_args();run(a.private,a.output)
