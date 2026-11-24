#!/usr/bin/env python3
"""Publish only aggregate, equal-state A/B growth evidence from private results."""
import argparse
from collections import defaultdict
import json
import math
from pathlib import Path


def statistics(values):
    values = sorted(values)
    return dict(samples=len(values),p95MS=values[math.ceil(.95*len(values))-1] if values else None,
                maxMS=max(values) if values else None,totalMS=sum(values))


def lsn(value):
    high,low=value.split('/')
    return int(high,16)*2**32+int(low,16)


def public_size(value):
    return {k:value[k] for k in ['capturedAt','model','databaseBytes','walBytes','walLSN','counts','relations','settings','orphanDictionaries'] if k in value}


def department_bytes(value):
    return sum(row['total_bytes'] for row in value['relations'] if row['relation'].startswith('organization.department'))


def run(private,output,cycles):
    assert not output.exists()
    evidence=dict(completed=False,cycles=[],scope='serial1000-command growth; same maintenance states; no FULL between cycles; capacity target measured separately')
    for cycle in cycles:
        values={role:json.loads((private/'logs'/f'{role}-growth-cycle{cycle}.json').read_text()) for role in ['a','b']}
        row=dict(cycle=cycle,roles={},checks={})
        for role,value in values.items():
            stages=defaultdict(list)
            for raw in (private/role/f'growth-{cycle}-diagnostics/requests.jsonl').read_text().splitlines():
                request=json.loads(raw)
                if request['Method']!='POST':
                    continue
                for stage in request['Stages']:
                    name=stage['Name']
                    if not name.startswith('backend.'):
                        stages[name].append(stage['DurationMS'])
            result={k:value[k] for k in ['cycle','completed','passed','expectedCommands','accepted','actions','sourceDelta','writeSeconds','everyAcceptedReceiptAndExactSourceAfterRestart','failures','binarySHA256','lockResidence'] if k in value}
            result['writes']=statistics([v['ms'] for v in value['writes']])
            result['stages']={k:statistics(v) for k,v in stages.items()}
            samples_path=private/role/f'growth-{cycle}-lock-samples.jsonl'
            if samples_path.exists() and result.get('lockResidence',{}).get('clock','').startswith('local monotonic_ns'):
                previous=None;gaps=[]
                for raw in samples_path.read_text().splitlines():
                    sample=json.loads(raw)
                    start,end=sample['localQueryStartNS'],sample['localQueryEndNS']
                    assert end>=start and (previous is None or start>=previous)
                    if previous is not None:gaps.append((start-previous)/1e6)
                    previous=start
                result['lockResidence']['monotonicQueryBracketsVerified']=True
                result['lockResidence']['actualSamplingIntervals']=statistics(gaps)
                result['lockResidence']['transactionKey']='backend PID and virtualtransaction, so distinct transactions never merge on a reused connection'
            for state in ['before','afterWrites','afterOrdinaryVacuum','afterOfflineCleanup']:
                result[state]=public_size(value[state])
                result[state]['departmentGroupTotalBytes']=department_bytes(value[state])
            result['walAdvanceWriteBytes']=lsn(value['afterWrites']['walLSN'])-lsn(value['before']['walLSN'])
            result['walAdvanceIncludingMaintenanceBytes']=lsn(value['afterOfflineCleanup']['walLSN'])-lsn(value['before']['walLSN'])
            result['maintenanceSeconds']=value['maintenance'].get('seconds',value['maintenance'].get('maintenance',{}).get('seconds'))
            row['roles'][role]=result
        a,b=row['roles']['a'],row['roles']['b']
        row['checks']=dict(all1000AppliedBoth=a['passed'] and b['passed'] and a['accepted']==b['accepted']==1000,
                           sameSourceDeltaAndActionTypes=a['sourceDelta']==b['sourceDelta']==1000 and a['actions']==b['actions'],
                           ordinaryDepartmentNotLarger=b['afterOrdinaryVacuum']['departmentGroupTotalBytes']<=a['afterOrdinaryVacuum']['departmentGroupTotalBytes'],
                           cleanupDepartmentSmaller=b['afterOfflineCleanup']['departmentGroupTotalBytes']<a['afterOfflineCleanup']['departmentGroupTotalBytes'],
                           cleanupWholeDatabaseSmaller=b['afterOfflineCleanup']['databaseBytes']<a['afterOfflineCleanup']['databaseBytes'],
                           bothUnreferencedDictionaryCountsZero=all(n==0 for x in [a,b] for n in x['afterOfflineCleanup']['orphanDictionaries'].values()))
        if any('lockResidence' in x for x in [a,b]):
            row['checks']['monotonicLockSamplingValid']=all(x.get('lockResidence',{}).get('monotonicQueryBracketsVerified') and not x['lockResidence']['errors'] and all(v['lowerMS']>=0 and (v['upperMS'] is None or v['upperMS']>=v['lowerMS']) for v in x['lockResidence']['intervals']) for x in [a,b])
        row['passed']=all(row['checks'].values())
        evidence['cycles'].append(row)
    evidence['completed']=True
    evidence['passed']=all(row['passed'] for row in evidence['cycles'])
    output.write_text(json.dumps(evidence,indent=2)+'\n')
    print(json.dumps(dict(passed=evidence['passed'],cycles=[dict(cycle=r['cycle'],checks=r['checks']) for r in evidence['cycles']])),flush=True)
    if not evidence['passed']:
        raise SystemExit(1)


if __name__=='__main__':
    p=argparse.ArgumentParser()
    p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--cycles',type=int,nargs=2,default=[1,2])
    a=p.parse_args()
    run(a.private,a.output,a.cycles)
