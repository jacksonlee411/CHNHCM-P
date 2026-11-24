#!/usr/bin/env python3
"""133 M0: six real HTTP entries, five independent-session waves, 270 samples.

Reuses 131's immutable full-input oracle and private observation collector.
No retry, reduced fixture, changed request budget or formal capacity claim.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import json
import re
from pathlib import Path
import sys
import subprocess
import threading
import time
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'tests/capacity/query131'))
from capacity import Matrix, capacity, sql

GROUPS = ['list', 'start-explicit', 'start-all', 'resolve', 'parent-create-first', 'path']

class Diagnosis(Matrix):
    def run(self):
        self.resources = []
        stop = threading.Event()
        def observe():
            while not stop.wait(.5):
                try:
                    waits = json.loads(sql(self.container, "SELECT coalesce(json_agg(v),'[]') FROM (SELECT coalesce(wait_event_type,'running') kind,count(*) n FROM pg_stat_activity WHERE usename='chnhcm_api' AND state='active' GROUP BY 1)v"))
                    cpu = subprocess.check_output(['docker','exec',self.container,'cat','/sys/fs/cgroup/cpu.stat'],text=True)
                    item = dict(atNS=time.monotonic_ns(), dbCPU=dict(zip(cpu.split()[::2],map(int,cpu.split()[1::2]))), waits=waits)
                    if self.server.p:
                        stat = Path('/proc',str(self.server.p.pid),'stat').read_text().split()
                        item['apiCPUTicks'] = int(stat[13])+int(stat[14])
                    self.resources.append(item)
                except Exception as e:
                    self.resources.append(dict(errorType=type(e).__name__))
        def temp():
            return json.loads(sql(self.container,"SELECT json_build_object('tempBytes',temp_bytes,'tempFiles',temp_files) FROM pg_stat_database WHERE datname=current_database()"))
        self.result['dbTempBefore'] = temp()
        monitor = threading.Thread(target=observe);monitor.start()
        try:
            sql(self.container,"ALTER SYSTEM SET log_min_duration_statement='0'")
            sql(self.container,'SELECT pg_reload_conf()')
            super().run()
        finally:
            stop.set();monitor.join()
            sql(self.container,'ALTER SYSTEM RESET log_min_duration_statement')
            sql(self.container,'SELECT pg_reload_conf()')
            self.result['dbTempAfter'] = temp()
            self.result['resources'] = self.resources
            self.result['resourceScope'] = '500ms sampling; database cgroup cumulative CPU and active wait classes; API process CPU ticks; includes preparation and observer overhead, no per-request CPU attribution'
            path = self.private/(self.output.stem+'-stages')/'requests.jsonl'
            if path.exists():
                observed = {str(v['Request']):v for v in map(json.loads,path.read_text().splitlines())}
                for group in self.result['groups'].values():
                    for sample in group['samples']:
                        obs = observed.get(str(sample.get('request')), {})
                        sample['workCounters'] = obs.get('Work')
                        sample.setdefault('memory',{}).update({k:obs[k] for k in ['ProcessGCCycles','ProcessGCPauseNS'] if k in obs})
            raw = self.private/(self.output.stem+'-postgres.log')
            durations = []
            if raw.exists():
                for line in raw.read_text().splitlines():
                    match = re.search(r'duration: ([\d.]+) ms',line)
                    if match and 'chnhcm_api@storage126' in line:
                        durations.append((datetime.fromisoformat(line.split(' ',1)[0].replace('Z','+00:00')),float(match[1])))
            for group in self.result['groups'].values():
                sample = group['samples'][0]
                begin,end = map(datetime.fromisoformat,[sample['waveStartedUTC'],sample['waveFinishedUTC']])
                values = [ms for at,ms in durations if begin<=at<=end]
                group['sqlDurationMS'] = sum(values)
                group['sqlDurationEvents'] = len(values)
                group['sqlDurationScope'] = 'shared wave sum of PostgreSQL parse/bind/execute durations; includes waits and client backpressure, not pure CPU and not additive with capture stages'
            self.save()

    def group(self, role, name, ignored_sessions):
        for n in self.args.waves:
            tag = f'{role}/{name}/{n}'
            barrier = threading.Barrier(n, timeout=30)
            start = time.monotonic_ns()
            wall = datetime.now(timezone.utc).isoformat()
            def call(client):
                barrier.wait()
                began = time.monotonic_ns()
                sample = self.job(role, name, client)()
                sample.update(actualCallStartedNS=began, actualCallFinishedNS=time.monotonic_ns())
                return sample
            with ThreadPoolExecutor(max_workers=n) as pool:
                samples = list(pool.map(call, self.clients[role][:n]))
            end = time.monotonic_ns()
            finished = datetime.now(timezone.utc).isoformat()
            for sample in samples:
                sample.update(wave=0, waveStartedUTC=wall, waveFinishedUTC=finished,
                              waveStartedNS=start, waveFinishedNS=end)
            self.result['groups'][tag] = dict(completed=True, expectedRequests=n, requests=n,
                sessions=n, samples=samples, failures=sum(not s['ok'] for s in samples),
                permissionNegative=False, diagnosticOnly=True, coordinatedStart=True,
                passed=all(s['ok'] for s in samples))
            self.save()
            print(json.dumps(dict(group=tag, requests=n, failures=sum(not s['ok'] for s in samples))), flush=True)

if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--operation', choices=GROUPS+['parent-maintain-second'], help='explicit supplementary diagnosis only')
    p.add_argument('--sessions', type=int, choices=[1,4,8,12,20])
    args = p.parse_args()
    assert bool(args.operation) == bool(args.sessions), 'supplement requires both operation and sessions'
    args = SimpleNamespace(**vars(args), shape='vanke', phase='observe', role=['admin'],
        completed_evidence=None, reuse_group=None, baseline_model4=False, profile=None,
        group=[args.operation] if args.operation else GROUPS, waves=[args.sessions] if args.sessions else [1,4,8,12,20])
    run = Diagnosis(args)
    run.run()
    assert run.result['completed'] and run.result['actualTimedRequests'] == (args.sessions if args.operation else 270)
