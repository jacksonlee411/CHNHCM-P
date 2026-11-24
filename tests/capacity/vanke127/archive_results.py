#!/usr/bin/env python3
"""Refresh original-cell/result mappings and verify the database readback."""
import argparse
import collections
import json
from pathlib import Path
import sqlite3
from runtime import cli


def result_reference(value):
    if value is None:return None
    # The immutable full result already belongs to the product terminal/history.
    # Source provenance retains its exact lookup identities and mapping decisions.
    response=value['response']
    out={k:value[k] for k in ['status','outcome','requestKey','adaptations']}
    for key in ['sourceVersion','observedSourceVersion']:
        if key in response:out[key]=response[key]
    return out


def run(private,out):
    db=sqlite3.connect(private/'import.sqlite');path=private/'source-rows.jsonl';counts=collections.Counter()
    path.touch(mode=0o600,exist_ok=True)
    with path.open('w') as f:
        for phase,no,payload,result in db.execute('SELECT phase,row_no,payload,result FROM rows ORDER BY phase,row_no'):
            value=json.loads(result) if result else None
            f.write(json.dumps(dict(Phase=phase,Row=no,Payload=json.loads(payload),Result=result_reference(value)),ensure_ascii=False)+'\n')
            counts[phase+'.'+(value['outcome'] if value else 'pending')]+=1
    result=dict(archive=cli('archive',private),sourceAudit=cli('source-audit',private),localOutcomes=dict(counts),resultStorage='exact source/submission references; complete original results remain in product history/terminals')
    out.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result));db.close()


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('out',type=Path);a=p.parse_args();run(a.private,a.out)
