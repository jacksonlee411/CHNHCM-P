#!/usr/bin/env python3
"""Actual owned cursor plans, collected outside all formal capacity samples."""
import argparse,json,re,subprocess,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import sql,resource

def run(a):
 name=resource(a.private);assert name=='chnhcm-storage130-b' and not a.output.exists()
 settings={'session_preload_libraries':"'auto_explain'",'auto_explain.log_min_duration':'0','auto_explain.log_analyze':'on','auto_explain.log_buffers':'on','auto_explain.log_timing':'off','auto_explain.log_format':"'json'",'auto_explain.log_nested_statements':'on'}
 old=json.loads(sql(name,"SELECT coalesce(to_json(rolconfig),'[]') FROM pg_roles WHERE rolname='chnhcm_api'"));assert not any(v.split('=')[0] in settings for v in old)
 evidence=a.output.with_name(a.output.stem+'-request.json')
 try:
  for k,v in settings.items():sql(name,f'ALTER ROLE chnhcm_api SET {k}={v}')
  subprocess.run([sys.executable,str(ROOT/'tests/capacity/remediation133/diagnose.py'),str(a.private),str(evidence),'--binary',str(a.binary),'--operation',a.operation,'--sessions','1'],check=True)
 finally:
  for k in settings:sql(name,f'ALTER ROLE chnhcm_api RESET {k}')
 raw=(a.private/(evidence.stem+'-postgres.log')).read_text()
 raw='\n'.join(re.sub(r'^\d{4}-\d{2}-\d{2}T\S+\s','',line) for line in raw.splitlines())
 keys=['Node Type','Parent Relationship','Subplan Name','Relation Name','Plan Rows','Plan Width','Actual Rows','Actual Loops','Rows Removed by Filter','Rows Removed by Join Filter','Shared Hit Blocks','Shared Read Blocks','Temp Read Blocks','Temp Written Blocks','Sort Space Used','Sort Space Type']
 def clean(n):return {**{k:n[k] for k in keys if k in n},'Plans':[clean(p) for p in n.get('Plans',[])]}
 plans=[]
 for match in re.finditer(r'\{\s*"Query Text"',raw):
  try:value,_=json.JSONDecoder().raw_decode(raw[match.start():])
  except json.JSONDecodeError:continue
  query=value['Query Text']
  if query.startswith('DECLARE '):plans.append(dict(cursor=query.split()[1],plan=clean(value['Plan'])))
 result=dict(operation=a.operation,formalCapacity=False,diagnostic=evidence.name,plans=plans)
 a.output.write_text(json.dumps(result,indent=2)+'\n');assert plans,'actual cursor execution plan missing; preserve request and raw private logs'
 print(json.dumps(dict(plans=len(plans),operation=a.operation)))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--operation',choices=['parent-create-first','parent-maintain-second','path'],required=True);run(p.parse_args())
