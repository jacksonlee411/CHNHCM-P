#!/usr/bin/env python3
"""Record exact current sources and offline binaries used by the 125 lane."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT=Path(__file__).resolve().parents[3]

def sources():
    paths=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','--','cmd','internal','db','api','apps/web','packages','go.mod','go.sum','pnpm-lock.yaml','tests/capacity/remediation125','tests/capacity/storage126','tests/capacity/vanke127'],cwd=ROOT,text=True).splitlines()
    return {name:hashlib.sha256((ROOT/name).read_bytes()).hexdigest() for name in sorted(set(paths)) if (ROOT/name).is_file()}

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('destination',type=Path);args=p.parse_args()
    destination=args.destination.resolve();destination.mkdir(parents=True,exist_ok=False)
    before=sources();binaries={}
    for name,target in [('storage126','./tests/capacity/storage126'),('vanke127','./tests/capacity/vanke127')]:
        path=destination/name;subprocess.run(['go','build','-o',str(path),target],cwd=ROOT,check=True)
        binaries[name]=hashlib.sha256(path.read_bytes()).hexdigest()
    after=sources();assert before==after,'sources changed during build'
    result=dict(baseCommit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),sourceSHA256=after,binarySHA256=binaries,goVersion=subprocess.check_output(['go','version'],text=True).strip(),scope='current working-tree source snapshot; no local commit or publication implied')
    (destination/'manifest.json').write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(dict(destination=str(destination),binarySHA256=binaries)))
