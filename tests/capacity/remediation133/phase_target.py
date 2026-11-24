#!/usr/bin/env python3
"""User's 2026-11-12 temporary target; original 300ms judgment stays intact.

A focused 200-operation run cannot replace the final 32-group/6400 matrix.
Uses its exact operation timer and mandatory first/second HTTP mapping.
"""
import argparse,hashlib,json,os
from pathlib import Path
from matrix import M1Matrix,case


def run(args):
    matrix=M1Matrix(args)
    matrix.result.update(capacityAcceptance=False,stageTargetMS=2000,
        stageTargetScope='admin/parent-maintain-second/20 only; other groups stage-accepted by user, original targets retained',
        expectedFormalHTTPRequests=None,expectedFormalBusinessOperations=None,
        expectedFocusedOperations=200,expectedFocusedHTTPRequests=400,
        focusedDriverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
    group='admin/parent-maintain-second/20'
    try:
        for key in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE']:os.environ.pop(key,None)
        matrix.server.start();assert matrix.server.info['apiConnectionLimit']==12
        matrix.result['startup']=matrix.server.info
        matrix.login('admin')
        matrix.execute_group(next(v for v in matrix.entries if v['tag']==group))
        matrix.result['completed']=True
    finally:
        matrix.server.close();matrix.stop.set();matrix.monitor.join()
        matrix.result['after']=case.cli('size',matrix.private)
        matrix.result['businessSourceGrowth']={k:matrix.result['after']['counts'][k]-matrix.result['before']['counts'][k] for k in ['buSources','departmentSources']}
        g=matrix.result['groups'].get(group,{})
        matrix.result['passed']=matrix.result['completed'] and g.get('requests')==200 and g.get('httpRequests')==400 and g.get('failures')==0 and g.get('p95MS',float('inf'))<=2000 and not any(matrix.result['businessSourceGrowth'].values())
        matrix.result['peakRSSKiB']=max(matrix.rss,default=None)
        matrix.save()
    return matrix.result


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('private',type=Path);parser.add_argument('output',type=Path);parser.add_argument('--binary',type=Path,required=True)
    args=parser.parse_args();args.diagnostic=False
    result=run(args)
    print(json.dumps(dict(passed=result['passed'],p95MS=result['groups']['admin/parent-maintain-second/20']['p95MS'],targetMS=2000)))
    if not result['passed']:raise SystemExit(1)
