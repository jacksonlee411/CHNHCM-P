#!/usr/bin/env python3
"""Write real browser expectations only to the private case directory."""
import argparse
import json
from pathlib import Path
from verify import Oracle


def run(private, admission):
    o=Oracle(private);codes=[c for c in o.objects if c!=o.root]
    depth=lambda c:len(o.path(c,o.heads[c],'2026-11-07',current=True))
    deepest=max(codes,key=depth)
    historic=max(codes,key=lambda c:len(o.events[c]))
    invalid=json.loads(admission.read_text())['invalidRowNumbers']['create.name']
    parents={r[2] for s in o.heads.values() for r in o.frames[s]['records']}
    old_name=next(o.rows['create',n][0]['org_code'] for n in invalid
                  if o.rows['create',n][0]['org_code'] not in parents and
                  o.record_at(o.frames[o.heads[o.rows['create',n][0]['org_code']]],'2026-11-07')[1]==o.rows['create',n][0]['name'])
    cases=[]
    for label,code in [('root-bu',o.root),('deep-department',deepest),('history-rich-department',historic),('historical-name-leaf',old_name)]:
        first=o.frames[o.events[code][0][1]];head=o.frames[o.heads[code]]
        cases.append(dict(label=label,code=code,kind=o.objects[code]['kind'],id=o.objects[code]['id'],
                          originalSource=first['source'],originalName=first['records'][0][1],
                          originalRecord=first['records'][0][3],recordCount=len(head['records']),
                          records=[dict(date=r[0],name=r[1],id=r[3]) for r in head['records']],
                          count=1 if code==o.root else 87343,edit=code==old_name))
    path=private/'browser-plan.json';path.touch(mode=0o600,exist_ok=True);path.write_text(json.dumps(cases,ensure_ascii=False))
    print(json.dumps(dict(browserCases=len(cases),sourceBasedExpectations=True,privateOutput=True)))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('admission',type=Path);a=p.parse_args();run(a.private,a.admission)
