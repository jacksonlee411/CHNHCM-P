#!/usr/bin/env python3
"""Resume the acknowledged original Vanke replay after local test jobs finish.
The existing importer owns all requests and recovery; no fixture shortcuts.
"""
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import time

from exclusive import competing, ROOT


def run(args):
    private=args.private.resolve();assert not private.is_relative_to(ROOT)
    target=args.output;assert not target.exists();target.mkdir()
    report=dict(completed=False,phase='waiting-for-functional-jobs',steps=[],peerAcknowledgement=False)
    save=lambda:(target/'summary.json').write_text(json.dumps(report,indent=2)+'\n')
    save();quiet=time.monotonic()
    while time.monotonic()-quiet<60:
        if competing():quiet=time.monotonic()
        time.sleep(2)
    report['startedAt']=datetime.now(timezone.utc).isoformat();save()
    env=dict(os.environ,CHNHCM_VANKE_BINARY=str(ROOT/'.local/vanke127-tree300'))
    source=args.source.resolve()
    if source.is_relative_to(ROOT):
        raise ValueError('source directory must be outside checkout')
    def step(name,command):
        report['phase']=name;save();print(json.dumps(dict(phase=name,at=datetime.now(timezone.utc).isoformat())),flush=True)
        start=time.monotonic()
        with (target/(name+'.log')).open('w') as log:
            result=subprocess.run(command,cwd=ROOT,env=env,stdout=log,stderr=subprocess.STDOUT)
        report['steps'].append(dict(name=name,exitCode=result.returncode,seconds=time.monotonic()-start));save()
        if result.returncode:raise RuntimeError(name+' failed; exact requests and original log retained')
    (private/'stop-after-row').unlink(missing_ok=True)
    step('create',['python3','tests/capacity/vanke127/create.py',str(private),str(source),str(target/'create.json'),'--workers','4'])
    assert json.loads((target/'create.json').read_text())['completed']
    step('order',['python3','tests/capacity/vanke127/history_order.py',str(private),str(target/'order.json')])
    step('history',['python3','tests/capacity/vanke127/parallel_history.py',str(private),str(source),str(target/'history.json'),'--workers','4','--window','4'])
    assert json.loads((target/'history.json').read_text())['complete']
    step('archive',['python3','tests/capacity/vanke127/archive_results.py',str(private),str(target/'archive.json')])
    step('dump',[env['CHNHCM_VANKE_BINARY'],'-mode','dump','-private',str(private)])
    step('verify',['python3','tests/capacity/vanke127/verify.py',str(private),str(target/'verify.json')])
    report['completed']=True;report['phase']='verified-input-ready';save()


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('private',type=Path);parser.add_argument('output',type=Path);parser.add_argument('--source',required=True,type=Path);run(parser.parse_args())
