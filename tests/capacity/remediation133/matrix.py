#!/usr/bin/env python3
"""M1 original R153 business denominator, with every necessary subrequest timed.

No reuse, retry, changed timeouts or pool size. Diagnostic mode is separately
labelled and never admitted as the formal 6400-operation result.
"""
import argparse
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import sys
import threading
import time
from types import SimpleNamespace
from urllib.parse import urlencode
from zoneinfo import ZoneInfo

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'tests/capacity/query131'))
from capacity import Matrix, capacity, case, read_stats
import protocol

BOUNDARY = ROOT/'docs/verification/data/131-t04-capacity-and-handoff/selected-stage-capacity-boundary-r153.json'

class M1Matrix(Matrix):
    def __init__(self, args):
        super().__init__(SimpleNamespace(**vars(args), shape='vanke', phase='warm', role=None,
            completed_evidence=None, reuse_group=[], baseline_model4=False, profile=None, group=None))
        self.diagnostic = args.diagnostic
        self.entries = json.loads(BOUNDARY.read_text())['groupResults']
        assert len(self.entries) == 32 and sum(v['requests'] for v in self.entries) == 6400
        assert self.result['before']['counts']['department'] == 87343
        assert sum(self.result['before']['counts'][k] for k in ['buSources','departmentSources']) == 132290
        self.result.update(driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
            originalBoundarySHA256=hashlib.sha256(BOUNDARY.read_bytes()).hexdigest(),
            originalGroups=[v['tag'] for v in self.entries], diagnosticOnly=self.diagnostic,
            capacityAcceptance=not self.diagnostic, observationsEnabled=False,
            timing='monotonic business operation through all required HTTP responses/JSON, queueing and intermediate validation; final oracle assertion follows completion',
            protocolMapping={'list':'one ordinary exact-count GET','path':'one cursor first-page GET',
                'start-explicit':'one start with two identities','start-all':'one start with all fixed identities',
                'resolve':'start full identities then resolve page 2 inside each timed operation',
                'parent-first-search':'one qualified cursor page','parent-second':'first cursor page then second page inside each timed operation'},
            expectedFormalHTTPRequests=8000, expectedFormalBusinessOperations=6400)
        self.result['driverFilesSHA256'] = {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest() for path in [Path(__file__), ROOT/'tests/capacity/remediation133/protocol.py', ROOT/'tests/capacity/query131/capacity.py', ROOT/'tests/capacity/path_targets.py', ROOT/'tests/capacity/storage126/measure.py']}

    def operation(self, role, name, client):
        return protocol.operation(self, role, name, client)

    def execute_group(self, entry):
        tag=entry['tag'];role,name,n=tag.split('/');n=int(n)
        if any(time.monotonic()-c.last_successful_activity>25*60 for c in self.clients[role][:n]):self.authenticate(role)
        clients=self.clients[role][:n]
        values=[];count=n if self.diagnostic else 200
        if not self.diagnostic:
            warm=self.operation(role,name,clients[0]);self.result['preparation'].append(dict(kind='warm-preparation',group=tag,**warm))
            if warm['ok']:clients[0].last_successful_activity=time.monotonic()
        for wave in range(count//n):
            if datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')!=self.day:raise RuntimeError('business day changed; preserve first results')
            if any(time.monotonic()-c.last_successful_activity>25*60 for c in clients):clients=self.authenticate(role)[:n]
            barrier=threading.Barrier(n,timeout=30) if n>1 else None
            def execute(client):
                if barrier:barrier.wait()
                return self.operation(role,name,client)
            samples=capacity.parallel([lambda c=c:execute(c) for c in clients]) if n>1 else [execute(clients[0])]
            for i,sample in enumerate(samples):
                sample.update(wave=wave,client=i)
                if sample['ok']:clients[i].last_successful_activity=time.monotonic()
            values.extend(samples)
            group=read_stats(values,name,sessions=n)
            group.update(completed=len(values)==count,expectedRequests=count,sessions=n,httpRequests=sum(v['httpRequests'] for v in values),coordinatedStart=n>1,permissionNegative=False)
            self.result['groups'][tag]=group;self.save()
        print(json.dumps(dict(group=tag,operations=len(values),httpRequests=group['httpRequests'],failures=group['failures'],p95MS=group['p95MS'])),flush=True)

    def run(self):
        for key in ['CHNHCM_QUERY131_STATS','CHNHCM_CAPACITY_DIAGNOSTICS','CHNHCM_CAPACITY_PROFILE']:os.environ.pop(key,None)
        try:
            self.server.start();assert self.server.info['apiConnectionLimit']==12
            assert self.server.info['model']=='m02-compact-5';self.result['startup']=self.server.info
            for entry in self.entries:
                role=entry['tag'].split('/')[0]
                if role not in self.clients:self.login(role)
                self.execute_group(entry)
            self.result['completed']=all(g['completed'] for g in self.result['groups'].values()) and len(self.result['groups'])==32
        finally:
            self.server.close();self.stop.set();self.monitor.join()
            self.result['after']=case.cli('size',self.private)
            self.result['businessSourceGrowth']={k:self.result['after']['counts'][k]-self.result['before']['counts'][k] for k in ['buSources','departmentSources']}
            groups=list(self.result['groups'].values())
            self.result.update(actualTimedBusinessOperations=sum(g['requests'] for g in groups),actualTimedHTTPRequests=sum(g['httpRequests'] for g in groups),peakRSSKiB=max(self.rss,default=None))
            self.result['availabilityPassed']=self.result['completed'] and all(g['failures']==0 for g in groups) and not any(self.result['businessSourceGrowth'].values())
            self.result['latencyPassed']=self.result['completed'] and all(g['passed'] for g in groups)
            self.result['passed']=self.result['availabilityPassed'] and self.result['latencyPassed']
            self.save()

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',required=True,type=Path);p.add_argument('--diagnostic',action='store_true')
    run=M1Matrix(p.parse_args());run.run()
    if not run.result['availabilityPassed']:raise SystemExit(1)
