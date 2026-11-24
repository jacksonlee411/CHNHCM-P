#!/usr/bin/env python3
"""Fixed-build tree matrix on a fully verified private fixture.
Run inside exclusive.py after preparation; this file never builds or imports.
"""
import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import time

from matrix import ROOT, MeasuredClient, ProcessObservation, fixtures, oracle


@contextmanager
def server(private,binary,kind,label,diagnostic=False):
    diagnostics=private/label;diagnostics.mkdir(mode=0o700)
    env=dict(os.environ)
    # Detailed observers call ReadMemStats and trace SQL. They belong in a
    # separate diagnostic run, never in formal request or browser timings.
    for name in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE','CHNHCM_CAPACITY_SPLIT_CONSTRAINTS']:
        env.pop(name,None)
    if diagnostic:
        env.update(CHNHCM_QUERY131_STATS=str(diagnostics/'query.jsonl'),CHNHCM_CAPACITY_DIAGNOSTICS=str(diagnostics))
    fd=os.open(diagnostics/'server.log',os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,'w') as log:
        command=([str(binary)] if kind=='synthetic' else
                 [str(binary),'-mode','serve-web','-input',str(private)] if kind=='frozen' else
                 [str(binary),'-mode','serve','-private',str(private)])
        process=subprocess.Popen(command,cwd=ROOT,env=env,stdout=subprocess.PIPE,stderr=log,text=True)
        try:
            line=process.stdout.readline()
            if not line:raise RuntimeError('fixture server failed; private diagnostics retained')
            info=json.loads(line)
            yield process,info,diagnostics
        finally:
            if process.poll() is None:process.send_signal(signal.SIGTERM);process.wait(timeout=15)


def run(args):
    assert not (args.matrix_only and args.workloads_only)
    assert not (args.workloads_only and (args.baseline or args.preflight))
    private=args.private.resolve();assert not private.is_relative_to(ROOT)
    args.output.mkdir(exist_ok=False)
    binary=args.binary.resolve()
    diagnostic=args.preflight or args.baseline
    report=dict(version=1,completed=False,formal=not diagnostic,detailedObservers=diagnostic,kind=args.kind,startedAt=datetime.now(timezone.utc).isoformat(),binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),steps=[],matrixSubset=dict(roles=args.roles,sessions=args.sessions,cases=args.cases),matrixOnly=args.matrix_only,workloadsOnly=args.workloads_only,resolveSetup=args.resolve_setup,skipMixed=args.skip_mixed)
    save=lambda:(args.output/'summary.json').write_text(json.dumps(report,indent=2)+'\n')
    save()
    with server(private,binary,args.kind,args.output.name,diagnostic=diagnostic) as (process,info,diagnostics):
        report['apiConnectionLimit']=info['apiConnectionLimit']
        client=MeasuredClient(info['url']);status,login,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));assert status==200;client.csrf=login['csrf']
        status,context,_=client.call('GET','/api/v1/organization/context');assert status==200
        byid,children,path,branch,wide,deep,large,shape=fixtures(oracle(private,context['currentDate']))
        report['shape']=shape;report['day']=context['currentDate'];save()
        browser_config=diagnostics/'browser.json'
        with os.fdopen(os.open(browser_config,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'w') as out:
            json.dump(dict(base=info['url'],graphSHA256=shape['graphSHA256'],targets=[dict(deep,case='deepest'),dict(children[wide][-1],case='wide-last')]),out)
        def step(name,command):
            observation=ProcessObservation(process.pid);observation.start();started=time.monotonic()
            with (args.output/(name+'.log')).open('w') as log:code=subprocess.run(command,cwd=ROOT,stdout=log,stderr=subprocess.STDOUT).returncode
            report['steps'].append(dict(name=name,exitCode=code,seconds=time.monotonic()-started,server=observation.finish()));save()
            if code:raise RuntimeError(name+' stopped; preserve original attempts')
        command=['python3','tests/capacity/tree300/matrix.py',str(private),info['url'],str(args.output/'matrix.json'),'--server-pid',str(process.pid)]
        if args.kind in ('synthetic','frozen'):command+=['--container',info['container']]
        if args.baseline:command+=['--diagnostic','--roles','admin','--cases','roots','wide-first','location-deep']
        if args.preflight:command+=['--diagnostic','--roles','admin','--sessions','1']
        if args.roles:command+=['--roles',*args.roles]
        if args.sessions:command+=['--sessions',*[str(n) for n in args.sessions]]
        if args.cases:command+=['--cases',*args.cases]
        if args.resolve_setup:command+=['--resolve-setup']
        if not args.workloads_only:step('matrix',command)
        if not args.baseline and not args.preflight and not args.matrix_only:
            time.sleep(61) # Separate setup login rate windows; not a timed sample.
            if args.kind in ('vanke','frozen'):step('csv',['python3','tests/capacity/tree300/export.py',str(private),info['url'],str(args.output/'csv.json'),'--server-pid',str(process.pid)])
            step('browser',['node','tests/capacity/tree300/browser.mjs',str(browser_config),str(args.output/'browser.json')])
            if args.kind=='synthetic' and not args.skip_mixed:
                time.sleep(61)
                step('mixed',['python3','tests/capacity/tree300/mixed.py',str(private),info['url'],str(args.output/'mixed.json')])
        report['completed']=True;report['endedAt']=datetime.now(timezone.utc).isoformat();save()


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('kind',choices=['vanke','synthetic','frozen']);parser.add_argument('private',type=Path);parser.add_argument('binary',type=Path);parser.add_argument('output',type=Path);parser.add_argument('--roles',nargs='+',choices=['admin','reader','sparse-reader']);parser.add_argument('--sessions',nargs='+',type=int,choices=[1,20]);parser.add_argument('--cases',nargs='+');parser.add_argument('--resolve-setup',action='store_true');parser.add_argument('--matrix-only',action='store_true');parser.add_argument('--workloads-only',action='store_true');parser.add_argument('--skip-mixed',action='store_true');mode=parser.add_mutually_exclusive_group();mode.add_argument('--baseline',action='store_true');mode.add_argument('--preflight',action='store_true');run(parser.parse_args())
