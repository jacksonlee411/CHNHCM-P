#!/usr/bin/env python3
"""Cooperative same-host slot plus observed overlap detection, never kill peers.
No claim of peer acknowledgement. Record workload names, not command arguments.
"""
import argparse
from datetime import datetime, timezone
import fcntl
import json
import os
from pathlib import Path
import subprocess
import threading
import time

ROOT=Path(__file__).resolve().parents[3]


def competing():
    found=[]
    for entry in Path('/proc').glob('[0-9]*'):
        try:
            cwd=Path(os.readlink(entry/'cwd'))
            argv=(entry/'cmdline').read_bytes().split(b'\0')
            if not argv or not argv[0]:continue
            names=[Path(value.decode(errors='replace')).name for value in argv if value]
            arguments=[value.decode(errors='replace') for value in argv if value]
            executable=names[0]
            label=None
            if executable=='go' and len(names)>1 and names[1] in ['test','build','generate','vet']:label='go-'+names[1]
            if executable.endswith('.test'):label='go-test-binary'
            if any(v in names for v in ['create.py','parallel_history.py','synthetic.py']):label='fixture-import'
            if any(v in names for v in ['history.py','verify.py','archive_results.py','history_order.py']):label='fixture-verification'
            if executable=='make' and any(v in names for v in ['check','docs','build','integration','smoke','boundaries']):label='validation'
            if executable.startswith('python') and ('unittest' in arguments or any(v.startswith('check-') and v.endswith('.py') for v in names)):label='validation'
            if '-mode' in arguments and any(v in executable for v in ['vanke127','tree300-synthetic','storage130','storage126']):
                position=arguments.index('-mode')
                if position+1<len(arguments) and arguments[position+1] in ['dump','migrate','initialize','read-index','compact','maintain','prepare','import','frozen-tree','export','verify','verify-history']:label='fixture-maintenance'
            if executable in ['node','pnpm','corepack'] and any(v in names for v in ['build','typecheck','tsc','generate']):label='javascript-build'
            if 'local-bu.py' in names and 'smoke' in names:label='browser-suite'
            if cwd!=ROOT and 'CHNHCM-wt-dev-' in str(cwd) and any(v in executable for v in ['node','chromium','python','vite']):
                # A retained idle server is not a capacity job. Only classify
                # explicit tests/builds/imports; process presence is not CPU use.
                if any('playwright' in v or '/capacity/' in v for v in arguments):label='peer-test'
            if label:found.append(dict(pid=int(entry.name),worktree=cwd.name,workload=label))
        except (OSError,ValueError):pass
    return found


def run(args):
    assert not args.output.exists()
    assert args.working_directory.resolve().is_relative_to(ROOT)
    if args.measured_test: assert Path(args.command[0]).name.endswith('.test')
    common=Path(subprocess.check_output(['git','rev-parse','--git-common-dir'],cwd=ROOT,text=True).strip())
    directory=common/'chnhcm-coordination';directory.mkdir(exist_ok=True)
    notice=directory/'tree300-capacity.json'
    lock=(directory/'capacity.lock').open('a')
    fcntl.flock(lock,fcntl.LOCK_EX)
    owner=subprocess.check_output(['git','branch','--show-current'],cwd=ROOT,text=True).strip()
    report=dict(owner=owner,task=args.task,peerAcknowledgement=False,completed=False,overlaps=[],waitObservations=[],quietSeconds=args.quiet_seconds)
    def save():args.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    def announce(active):notice.write_text(json.dumps(dict(owner=owner,task=args.task,exclusiveActive=active,state='measuring' if active else 'waiting',updatedAt=datetime.now(timezone.utc).isoformat(),policy='不停止他人任务；发现重叠保留原结果并标记该轮非独占；不声明已获其他任务确认'),ensure_ascii=False,indent=2)+'\n')
    quiet_since=time.monotonic();announce(False)
    while time.monotonic()-quiet_since<args.quiet_seconds:
        jobs=competing()
        if jobs:
            report['waitObservations'].append(dict(at=datetime.now(timezone.utc).isoformat(),jobs=jobs));save();quiet_since=time.monotonic()
        time.sleep(2)
    report['startedAt']=datetime.now(timezone.utc).isoformat();report['loadAtStart']=os.getloadavg();save();announce(True)
    stop=threading.Event(); measured_pid=None
    def monitor():
        while not stop.wait(.5):
            jobs=competing()
            if args.measured_test:
                jobs=[job for job in jobs if not (job['pid']==measured_pid and job['workload']=='go-test-binary')]
            if jobs:report['overlaps'].append(dict(at=datetime.now(timezone.utc).isoformat(),jobs=jobs));save()
    watcher=threading.Thread(target=monitor,daemon=True);watcher.start()
    try:
        process=subprocess.Popen(args.command,cwd=args.working_directory)
        measured_pid=process.pid
        if args.measured_test:
            report['measuredTestPID']=measured_pid
            report['measuredTestScope']='Only the directly launched test process is the measured workload; other build/test/import processes remain overlap failures.'
        report['exitCode']=process.wait()
    finally:
        stop.set();watcher.join();report['completed']=True;report['endedAt']=datetime.now(timezone.utc).isoformat();report['loadAtEnd']=os.getloadavg();report['observedExclusive']=not report['overlaps'];save();announce(False);lock.close()
    if report['exitCode'] or not report['observedExclusive']:raise SystemExit(1)


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--task',default='CHNHCM-300');parser.add_argument('--measured-test',action='store_true');parser.add_argument('--working-directory',type=Path,default=ROOT);parser.add_argument('output',type=Path);parser.add_argument('--quiet-seconds',type=int,default=60);parser.add_argument('command',nargs=argparse.REMAINDER)
    args=parser.parse_args();args.command=args.command[1:] if args.command[:1]==['--'] else args.command;assert args.command;run(args)
