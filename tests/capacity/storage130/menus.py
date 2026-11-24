#!/usr/bin/env python3
"""Run the existing normal-menu browser journey on the complete owned Vanke B.
Raw screenshots and failures remain in the private case directory.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from measure import Lane, case, save
from menu import WebServer


class MenuLane:
    cold = Lane.cold


def run(private, output, binary):
    assert private.is_absolute() and ROOT not in private.parents and private.stat().st_mode & 0o777 == 0o700
    assert not output.exists()
    case.BIN = binary
    lane = MenuLane()
    lane.private, lane.output = private, output
    lane.server, lane.clients = WebServer(private, private), {}
    lane.result = dict(completed=False,binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                       coldResets=[],menuEntries=[],scope='complete Vanke B; real normal menu; one sample per entry, no p95 claim')
    try:
        lane.server.start()
        lane.result['startup'] = lane.server.info
        for role in ['admin','reader']:
            for segment,count in [('business-units',1),('departments',87343)]:
                lane.cold(f'vanke-normal-menu-{role}-{segment}')
                for cache in ['cold','warm']:
                    request=dict(url=lane.server.base,role=role,segment=segment,count=count,
                                 screenshot=str(private/f'{output.stem}-{role}-{segment}-{cache}.png'))
                    result=subprocess.run(['node',str(ROOT/'tests/capacity/remediation125/menu.mjs')],
                                          input=json.dumps(request),text=True,capture_output=True)
                    if result.returncode:
                        path=private/f'{output.stem}-{role}-{segment}-{cache}-failure.txt'
                        with os.fdopen(os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'w') as f:
                            f.write(result.stderr)
                        lane.result['menuEntries'].append(dict(role=role,segment=segment,cache=cache,passed=False))
                    else:
                        lane.result['menuEntries'].append(dict(cache=cache,**json.loads(result.stdout)))
                    save(output,lane.result)
        lane.result['completed'] = True
        lane.result['passed'] = all(v['passed'] for v in lane.result['menuEntries'])
    finally:
        lane.server.close()
        save(output,lane.result)
    if not lane.result.get('passed'):
        raise SystemExit(1)


if __name__ == '__main__':
    p=argparse.ArgumentParser()
    p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True)
    a=p.parse_args()
    run(a.private,a.output,a.binary.resolve())
