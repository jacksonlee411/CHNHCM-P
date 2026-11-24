#!/usr/bin/env python3
"""Merge source field deltas by business date before issuing snapshot commands."""
import argparse
import bisect
import collections
import json
from pathlib import Path
import sqlite3


def run(private, report):
    db=sqlite3.connect(private/'import.sqlite')
    states={};records={};names={};parents={};groups=collections.defaultdict(list)
    for phase,no,payload in db.execute('SELECT phase,row_no,payload FROM rows ORDER BY phase,row_no'):
        r=json.loads(payload);code=r['org_code'];day=r.get('target_effective_date') or r['effective_date']
        if phase=='create':
            states[code]=[r['name'],r.get('parent_org_code','')]
            records[code]={day:states[code].copy()};names[code]={day:r['name']};parents[code]={day:r.get('parent_org_code','')}
        else:
            groups[day].append((phase,no,r))
            if phase=='version' and r.get('core.name'):names[code][day]=r['core.name']
            if phase=='parent-change':parents[code][day]=r['new_parent_org_code']
    order=[];nochange=collections.Counter();deferred_count=0
    for day,rows in sorted(groups.items()):
        for phase,no,r in sorted(rows,key=lambda x:(x[0]!='version',x[1])):
            if phase!='version':continue
            code=r['org_code'];old=states[code].copy()
            if r.get('core.name'):states[code][0]=r['core.name']
            if old!=states[code]:records[code][day]=states[code].copy()
            else:nochange[phase]+=1
            order.append([phase,no])
        pending=[x for x in rows if x[0]=='parent-change']
        while pending:
            again=[]
            for phase,no,r in pending:
                code,parent=r['org_code'],r['new_parent_org_code'];seen={code};at=parent;cycle=False
                while at:
                    if at in seen:cycle=True;break
                    seen.add(at);at=states[at][1]
                if cycle:again.append((phase,no,r));deferred_count+=1;continue
                if states[code][1]!=parent:
                    states[code][1]=parent;records[code][day]=states[code].copy()
                else:nochange[phase]+=1
                order.append([phase,no])
            if len(again)==len(pending):raise RuntimeError('no admissible same-day parent schedule')
            pending=again
    comparisons,mismatches=0,0
    for code,rs in records.items():
        nd,pd,rd=sorted(names[code]),sorted(parents[code]),sorted(rs)
        for day in sorted(set(nd+pd+rd)):
            actual=rs[rd[bisect.bisect_right(rd,day)-1]]
            expected=[names[code][nd[bisect.bisect_right(nd,day)-1]],parents[code][pd[bisect.bisect_right(pd,day)-1]]]
            comparisons+=1;mismatches+=actual!=expected
    assert len(order)==44958 and mismatches==0
    path=private/'history-order.json';path.touch(mode=0o600,exist_ok=True);path.write_text(json.dumps(order))
    result=dict(scheduled=len(order),distinctDates=len(groups),sameDateDeferrals=deferred_count,modeledNoChanges=dict(nochange),finalFactsCompared=comparisons,finalFactMismatches=mismatches,scope='independent chronological field-delta mapping; actual Go/API still required')
    report.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('report',type=Path);a=p.parse_args();run(a.private,a.report)
