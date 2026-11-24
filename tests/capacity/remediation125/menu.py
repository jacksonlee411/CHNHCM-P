#!/usr/bin/env python3
"""Real normal-menu first entries on the complete 125 candidate, with its
owned API, DB and data-file caches reset by the same formal cold helper."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

from measure import Lane,case,ROOT,save

class WebServer(case.Server):
    def start(self):
        self.p=subprocess.Popen([str(case.BIN),'-mode','serve-web','-input',str(self.dir)],cwd=ROOT,stdout=subprocess.PIPE,stderr=self.log,text=True)
        raw=self.p.stdout.readline()
        if not raw:raise RuntimeError('owned web API failed startup')
        self.info=json.loads(raw);self.base=self.info['url'];return self.base

def main(private,output,binary):
    case.Server=WebServer
    lane=Lane(private,output,binary)
    try:
        lane.result['menuEntries']=[]
        for role in ['admin','reader']:
            for kind,segment in [('bu','business-units'),('dep','departments')]:
                lane.cold(f'normal-menu-{role}-{kind}')
                for cache in ['cold','warm']:
                    request=dict(url=lane.server.base,role=role,segment=segment,count=len(lane.codes[kind]),screenshot=str(private/f'{output.stem}-{role}-{kind}-{cache}.png'))
                    p=subprocess.run(['node',str(ROOT/'tests/capacity/remediation125/menu.mjs')],input=json.dumps(request),text=True,capture_output=True)
                    if p.returncode:
                        # A failure may contain DOM values; keep the raw error
                        # private and publish only that this entry failed.
                        path=private/f'{output.stem}-{role}-{kind}-{cache}-failure.txt'
                        fd=os.open(path,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600)
                        with os.fdopen(fd,'w') as f:f.write(p.stderr)
                        lane.result['menuEntries'].append(dict(role=role,kind=kind,cache=cache,passed=False))
                    else:
                        lane.result['menuEntries'].append(dict(cache=cache,**json.loads(p.stdout)))
                    save(output,lane.result)
        lane.result['preparation'].append(dict(ok=all(v['passed'] for v in lane.result['menuEntries'])))
        lane.result['completed']=True
    finally:lane.close()
    if not lane.result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);a=p.parse_args()
    assert not a.output.exists()
    main(a.private,a.output,a.binary.resolve())
