#!/usr/bin/env python3
"""Run the complete real-DB suites in bounded, serial batches.
The batch timeout is a runner limit; API/SQL/guard budgets are unchanged.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import time

ROOT=Path(__file__).resolve().parents[3]

def run(destination):
    destination.mkdir(parents=True,exist_ok=False)
    result=dict(completed=False,packages=[],runs=[],productTimeoutsChanged=False)
    report=destination/'summary.json'
    def save():report.write_text(json.dumps(result,indent=2)+'\n')
    for package in ['./tests/integration','./internal/bootstrap']:
        listed=subprocess.check_output(['go','test','-tags=integration',package,'-list','^Test'],cwd=ROOT,text=True)
        names=[n for n in listed.splitlines() if n.startswith('Test')]
        result['packages'].append(dict(package=package,tests=names));save()
        for start in range(0,len(names),15):
            batch=names[start:start+15];label=package.rsplit('/',1)[1]+'-'+str(start//15+1).zfill(2)
            command=['go','test','-tags=integration','-timeout=20m',package,'-run','^('+'|'.join(batch)+')$','-count=1','-json']
            before=time.monotonic()
            with (destination/(label+'.jsonl')).open('w') as log:code=subprocess.run(command,cwd=ROOT,stdout=log,stderr=subprocess.STDOUT).returncode
            outcomes=[]
            for line in (destination/(label+'.jsonl')).read_text().splitlines():
                try:event=json.loads(line)
                except ValueError:continue
                if event.get('Test') and event.get('Action') in ['pass','fail','skip']:outcomes.append({k:event[k] for k in ['Test','Action','Elapsed'] if k in event})
            row=dict(label=label,package=package,requestedRoots=batch,exitCode=code,elapsedSeconds=time.monotonic()-before,outcomes=outcomes,logSHA256=hashlib.sha256((destination/(label+'.jsonl')).read_bytes()).hexdigest())
            result['runs'].append(row);save()
            print(json.dumps(dict(label=label,exitCode=code,passed=sum(v['Action']=='pass' for v in outcomes),failed=sum(v['Action']=='fail' for v in outcomes),skipped=sum(v['Action']=='skip' for v in outcomes))),flush=True)
    result['completed']=True;result['passed']=all(r['exitCode']==0 for r in result['runs']);save()
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('destination',type=Path);run(p.parse_args().destination)
