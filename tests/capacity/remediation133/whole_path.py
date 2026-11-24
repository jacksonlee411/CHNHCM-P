#!/usr/bin/env python3
"""A complete full-scale path selection, not a first-page substitute."""
import argparse,hashlib,json,os,sys,time
from datetime import datetime
from zoneinfo import ZoneInfo
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/query131'))
from lists import case,resource,expected
from growth import sql

def run(a):
 private=a.private.resolve();assert private.stat().st_mode&0o777==0o700 and ROOT not in private.parents and not a.output.exists()
 container=resource(private);case.BIN=a.binary.resolve();server=case.Server(private,private)
 day=datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
 rows,_=expected('vanke',private,day)
 # 306 removes the implicit enterprise prefix. A separator matches only
 # non-root paths; derive the entire matching set from immutable inputs.
 term='/';members={r['id'] for r in rows.values() if term in r['path']}
 assert len(rows)==87343 and len(members)>80000
 result=dict(completed=False,formalCapacity=False,expectedMembers=len(members),totalDepartments=len(rows),excludedRootPaths=len(rows)-len(members),businessAt=day,queryPolicy='department-path-separator-after-306',binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),steps=[])
 before=sql(container,'SELECT sum(cardinality(source_refs)) FROM organization.department_history')
 try:
  for key in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE']:os.environ.pop(key,None)
  c=case.Client();c.base=server.start();v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf']
  query=dict(filterField='departmentPath',filterOperator='contains',filterValue=term)
  began=time.monotonic();sample=dict(operation='full-path-capture',ok=False);result['steps'].append(sample)
  try:
   v,ms=c.call('POST','/api/v1/departments/selection/start',dict(query=query,selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20))
   assert v['summary']['selectedCount']==len(members) and 'count' not in v['page'] and v['page']['hasMore']
   assert all(r['id'] in members for r in v['page']['items'])
   sample.update(ok=True,status=200,selectedCount=v['summary']['selectedCount'])
  except Exception as e:
   sample.update(status=getattr(e,'status',None),code=getattr(e,'code',None),error=type(e).__name__);raise
  finally:sample['ms']=(time.monotonic()-began)*1000
  d=v['summary'];v,ms=c.call('POST','/api/v1/departments/selection/resolve',dict(query=query,draftId=d['draftId'],expectedRevision=d['revision'],removeIds=[v['page']['items'][0]['id']],page=1,pageSize=20));assert v['summary']['selectedCount']==len(members)-1
  result['steps'].append(dict(operation='full-path-page-exclusion',ok=True,status=200,ms=ms,selectedCount=len(members)-1));result['completed']=True
 finally:
  server.close();result['businessSourcesUnchanged']=sql(container,'SELECT sum(cardinality(source_refs)) FROM organization.department_history')==before
  result['passed']=result['completed'] and result['businessSourcesUnchanged'];a.output.write_text(json.dumps(result,indent=2)+'\n')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);run(p.parse_args())
