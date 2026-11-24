#!/usr/bin/env python3
"""131 full-size read matrix; immutable independent input, owned cold resets.

No reduced data, retries or request-timeout changes. Every timed request remains
in its denominator, including failures. Diagnostics run in a separate process.
"""
import argparse
import copy
from collections import Counter
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import threading
import time
from types import SimpleNamespace
from urllib.parse import urlencode
from zoneinfo import ZoneInfo

from lists import ROOT, case, expected, module, resource
from growth import sql

sys.path.append(str(ROOT/'tests/capacity'))
import path_targets

product_protocol = module('query133_protocol', ROOT/'tests/capacity/remediation133/protocol.py')
capacity = module('capacity125', ROOT/'tests/capacity/remediation125/measure.py')
ACCEPTANCE_POLICY = 'q131-path-stage-1000ms-20261115'


def read_target_ms(operation, permission_negative=False, *, sessions):
    if permission_negative:
        return None
    # This driver uses the complete22104/87343-department fixtures. The user
    # revised selection start/resolve targets for this ten-thousand-plus scale.
    if operation in ['start-explicit', 'start-all', 'resolve']:
        return {1: 1000, 20: 5000}[sessions]
    if operation == 'path':
        return path_targets.target_ms(sessions=sessions)
    # Ordinary lists and first unsearched create-parent queries retain their
    # separately approved targets; other reads stay unchanged.
    if operation == 'list':
        return capacity.operation_target_ms('dep', 'list', sessions=sessions)
    # The first unsearched create-parent page keeps its existing 1/20 targets.
    if operation == 'parent-create-first':
        return {1: 300, 20: 1000}[sessions]
    return 300


def read_stats(samples, operation, permission_negative=False, *, sessions):
    result = capacity.stats(samples, read_target_ms(operation, permission_negative, sessions=sessions))
    if operation == 'path' and not permission_negative:
        long_term = capacity.stats(samples, path_targets.target_ms(sessions=sessions, long_term=True))
        result.update(targetPolicy=path_targets.POLICY_ID,
                      longTermTargetMS=long_term['targetMS'], longTermPassed=long_term['passed'])
    return result


def reevaluate_group(group, operation):
    assert group['completed'] and group['requests'] == group['expectedRequests']
    admitted = copy.deepcopy(group)
    admitted.update(read_stats(admitted['samples'], operation,
                              group['permissionNegative'], sessions=group['sessions']))
    admitted.update(originalTargetMS=group['targetMS'], originalPassed=group['passed'])
    return admitted


class Matrix:
    def __init__(self, args):
        if args.baseline_model4: raise ValueError("current driver admits only the current model; use the historical frozen commit for historical model evidence")
        self.args = args
        self.private = args.private.resolve()
        assert self.private.stat().st_mode & 0o777 == 0o700 and ROOT not in self.private.parents
        assert not args.output.exists(), 'first results must remain immutable'
        self.container = resource(self.private)
        case.BIN = args.binary.resolve()
        self.day = datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
        self.rows, self.shape = expected(args.shape, self.private, self.day)
        self.ordered = sorted(self.rows.values(), key=lambda r: (r['code'], r['id']))
        self.server = case.Server(self.private, self.private)
        self.clients = {}
        self.result = dict(completed=False, shape=args.shape, phase=args.phase,
                           objects=len(self.rows), inputShape=self.shape, businessAt=self.day,
                           apiConnections=12, binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(),
                           driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                           acceptancePolicy=ACCEPTANCE_POLICY,
                           pathTargetPolicy=dict(id=path_targets.POLICY_ID, contract=path_targets.CONTRACT,
                               stageTargetMS={'1': path_targets.target_ms(sessions=1), '20': path_targets.target_ms(sessions=20)},
                               longTermTargetMS=path_targets.LONG_TERM_TARGET_MS,
                               sourceSHA256=hashlib.sha256(Path(path_targets.__file__).read_bytes()).hexdigest()),
                           groups={}, preparation=[], coldResets=[], failures=[],
                           timing='complete HTTP and JSON decode; assertions after timing; all failures retained',
                           observationsEnabled=args.phase == 'observe')
        self.result['roles'] = args.role or ['admin', 'reader']
        names = ['snapshot.json', 'organization.enterprise.jsonl.gz'] + (
            ['owners.json', 'bu-prototype.json', 'dep-prototype.json', 'bu-typed.jsonl.gz', 'dep-typed.jsonl.gz']
            if args.shape == 'synthetic' else ['import.sqlite', 'history-order.json'])
        self.result['oracleInputSHA256'] = {}
        for name in names:
            with (self.private/name).open('rb') as stream:
                self.result['oracleInputSHA256'][name] = hashlib.file_digest(stream, 'sha256').hexdigest()
        self.result['capacityAcceptance'] = args.phase != 'observe'
        self.reused = {}
        if args.completed_evidence:
            assert args.phase == 'warm' and args.reuse_group
            previous = json.loads(args.completed_evidence.read_text())
            for key in ['shape', 'phase', 'objects', 'businessAt', 'apiConnections', 'binarySHA256', 'oracleInputSHA256']:
                assert previous[key] == self.result[key], 'reused group has different fixed input'
            assert not any(previous['businessSourceGrowth'].values())
            for tag in args.reuse_group:
                group = previous['groups'][tag]
                assert group['completed'] and group['requests'] == group['expectedRequests'] == 200
                assert not any(str(s.get('code', '')).startswith('AUTH_') for s in group['samples'])
                _, operation, _ = tag.split('/')
                self.reused[tag] = reevaluate_group(group, operation)
            self.result['reusedEvidence'] = dict(file=args.completed_evidence.name,
                sha256=hashlib.sha256(args.completed_evidence.read_bytes()).hexdigest(),
                originalRunCompleted=previous['completed'],
                groups={tag: {key: group[key] for key in ['requests', 'failures', 'httpErrors', 'semanticErrors', 'p95MS', 'targetMS', 'passed', 'originalTargetMS', 'originalPassed', 'targetPolicy', 'longTermTargetMS', 'longTermPassed'] if key in group} for tag, group in self.reused.items()},
                scope='explicitly admitted complete original samples evaluated under the current approved target; original target/outcome and immutable report hash retained; not new requests or a single new all-green run')
        else:
            assert not args.reuse_group
        self.output = args.output
        self.result['before'] = case.cli('size', self.private)
        if self.reused:
            for key in ['BU', 'department', 'buSources', 'departmentSources', 'departmentSegments', 'buSegments', 'securityChangeEvents']:
                assert self.result['before']['counts'][key] == previous['after']['counts'][key], 'fixture changed since reused groups'
        self.rss = []
        self.stop = threading.Event()
        self.monitor = threading.Thread(target=self.memory)
        self.prepare_parents()
        self.monitor.start()

    def memory(self):
        while not self.stop.wait(.1):
            if self.server.p:
                try:
                    raw = Path('/proc', str(self.server.p.pid), 'status').read_text()
                    self.rss.append(int(next(line.split()[1] for line in raw.splitlines() if line.startswith('VmRSS:'))))
                except (FileNotFoundError, StopIteration):
                    pass

    def save(self):
        capacity.save(self.output, self.result)

    def prepare_parents(self):
        if self.args.shape == 'vanke':
            oracle = module('vanke_capacity_oracle', ROOT/'tests/capacity/vanke127/verify.py').Oracle(self.private)
            timelines = {c: oracle.frames[oracle.heads[c]]['records'] for c in self.rows}
            referenced = {r[2] for rows in timelines.values() for r in rows}
            target = max((c for c in self.rows if c not in referenced), key=lambda c: (timelines[c][-1][0], c))
            self.parent_start = timelines[target][-1][0]
            self.parent_first = {c: rows[0][0] for c, rows in timelines.items()}
            self.parent_bu = oracle.objects[oracle.root]['id']
            self.parent_codes = set(self.rows)
            self.search = max(Counter(c[:2] for c in self.rows), key=lambda p: sum(c.startswith(p) for c in self.rows))
        else:
            oracle = case.Oracle(self.private)
            fan = [c for c in self.rows if c.startswith('R6_FAN_')]
            assert len(fan) >= 2001, 'full fanout fixture required for a nonempty second page'
            referenced = {r['Data']['Parent'] for c in fan for r in oracle.frame(c)['State']['Timeline']['Records']}
            target = next(c for c in sorted(fan) if self.rows[c]['id'] not in referenced)
            self.parent_bu = self.rows[target]['managementBu']['id']
            self.parent_codes = {c for c, r in self.rows.items() if r['managementBu']['id'] == self.parent_bu}
            self.parent_first = {}
            for c in self.parent_codes:
                timeline = oracle.frame(c)['State']['Timeline']['Records']
                assert all(r['State'] == 'active' and r['Data']['BU'] == self.parent_bu for r in timeline)
                self.parent_first[c] = timeline[0]['Date']
            self.parent_start = oracle.frame(target)['State']['Timeline']['Records'][-1]['Date']
            self.search = 'R6_FAN_'
        self.target = target
        self.parent_expected = {}
        for mode in ['create', 'maintain']:
            start = self.day if mode == 'create' else self.parent_start
            for searched in [False, True]:
                self.parent_expected[mode, searched] = [dict(id=r['id'], code=r['code'], name=r['name'], eligibility='eligible')
                    for r in self.ordered if r['code'] in self.parent_codes and self.parent_first[r['code']] <= start
                    and (mode == 'create' or r['code'] != target)
                    and (not searched or self.search.lower() in (r['code']+' '+r['name']).lower())]
                assert len(self.parent_expected[mode, searched]) > 20

    def authenticate(self, role):
        clients = []
        for _ in range(20):
            c = case.Client(); c.base = self.server.base
            try:
                value, ms = c.call('POST', '/api/v1/session/login', dict(login=role, password='admin1234567'))
            except Exception as error:
                self.result['preparation'].append(dict(kind='authentication', role=role, ok=False,
                    status=getattr(error, 'status', None), code=getattr(error, 'code', None), ms=getattr(error, 'elapsed_ms', None)))
                raise
            c.csrf = value['csrf']; clients.append(c)
            c.last_successful_activity = time.monotonic()
            self.result['preparation'].append(dict(kind='authentication', role=role, ok=True, ms=ms))
        self.clients[role] = clients
        return clients

    def login(self, role):
        clients = self.authenticate(role)
        detail, ms = clients[0].call('GET', '/api/v1/departments/'+self.rows[self.target]['id'])
        record = detail['timeline'][-1]
        assert record['effectiveDate'] == self.parent_start
        self.context = dict(mode='maintain', action='CORRECT', departmentId=self.rows[self.target]['id'],
                            expectedSourceVersion=detail['headSourceVersion'], recordId=record['recordId'], expectedRecordRevision=record['revision'])
        self.result['preparation'].append(dict(kind='parent-context', role=role, ok=True, ms=ms))
        sample, value = capacity.measured(clients[0], 'GET', '/api/v1/departments', check=lambda v: self.page(v, self.ordered))
        self.result['preparation'].append(dict(kind='independent-list-oracle', role=role, **sample))
        if value is None:
            return
        assert sample['ok'], 'successful HTTP violated the independent oracle'
        rejected = []
        for fault in ['count', 'identity', 'name', 'path', 'omitted-row']:
            wrong = copy.deepcopy(value)
            if fault == 'count': wrong['count'] -= 1
            elif fault == 'identity': wrong['items'][0]['id'] = '00000000-0000-0000-0000-000000000000'
            elif fault == 'name': wrong['items'][0]['name'] += '-invalid'
            elif fault == 'path': wrong['items'][0]['path']['departmentPath'] += '-invalid'
            else: wrong['items'].pop()
            try: self.page(wrong, self.ordered)
            except AssertionError: rejected.append(fault)
            else: raise AssertionError('oracle accepted corrupted HTTP 200')
        self.result['preparation'][-1]['rejected'] = rejected

    def page(self, value, rows, page=1):
        assert value['count'] == len(rows) and value['page'] == page and value['pageSize'] == 20
        assert value['businessAt'] == self.day
        assert 'selectionVersion' not in value
        wanted = rows[(page-1)*20:page*20]
        assert len(value['items']) == len(wanted)
        for actual, original in zip(value['items'], wanted, strict=True):
            for field in ['id', 'code', 'name', 'shortName', 'managementBu', 'status', 'effectiveDate']:
                assert actual.get(field) == original.get(field), field
            assert actual['path']['departmentPath'] == original['path']
            assert actual['path']['pathMode'] == original['pathMode']
            assert actual['path']['pathBusinessAt'] == original['pathBusinessAt']

    def job(self, role, group, client):
        return lambda: product_protocol.operation(self, role, group, client)

    def group(self, role, name, sessions):
        tag = f'{role}/{name}/{sessions}'
        if tag in self.reused:
            print(json.dumps(dict(group=tag, reused=True, evidence=self.args.completed_evidence.name)), flush=True)
            return
        rss_start = len(self.rss)
        # Long single-session groups must not leave the other nineteen idle
        # beyond the product's 30-minute expiry before their measured wave.
        # Renew only a group whose clients would be idle for 25 minutes. This
        # also respects the unchanged 20-login/minute product limit.
        if any(time.monotonic()-c.last_successful_activity > 25*60 for c in self.clients[role][:sessions]):
            self.authenticate(role)
        clients = self.clients[role][:sessions]
        values = []
        permission_negative = role == 'reader' and name.startswith('parent-create')
        count = 1 if self.args.phase == 'observe' else 200 if self.args.phase == 'warm' else 100
        if self.args.phase == 'warm':
            sample = self.job(role, name, clients[0])()
            if sample['ok']: clients[0].last_successful_activity = time.monotonic()
            self.result['preparation'].append(dict(kind='warm-preparation', group=tag, **sample))
        waves = count//sessions
        for wave in range(waves):
            # Budget failures roll back session activity. Refresh before expiry
            # using only the last successful call, including in a long failed
            # single-session group; never retry a measured request.
            if any(time.monotonic()-c.last_successful_activity > 25*60 for c in clients):
                clients = self.authenticate(role)[:sessions]
            if self.args.phase in ['cold-first', 'cold-wave']:
                capacity.Lane.cold(self, tag+'/'+str(wave))
                # Only login and parent/selection preparation precede the reset;
                # no target business request prewarms the resumed process.
            actual_day = datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
            if actual_day != self.day:
                self.result['calendarBoundary'] = dict(group=tag, wave=wave, previousDay=self.day, currentDay=actual_day,
                    reason='fixed-date fixture ended; preserve partial samples and require a separately recorded new run, never rebind an existing selection')
                raise RuntimeError('business date changed during fixed-input capacity lane')
            start = time.monotonic_ns()
            wall_start = datetime.now(ZoneInfo('UTC')).isoformat()
            barrier = threading.Barrier(sessions, timeout=30) if sessions > 1 else None
            def run(client):
                call = client.call
                interval = {}
                def observed_call(*args, **kwargs):
                    interval['actualCallStartedNS'] = time.monotonic_ns()
                    try:
                        return call(*args, **kwargs)
                    finally:
                        interval['actualCallFinishedNS'] = time.monotonic_ns()
                client.call = observed_call
                try:
                    if barrier: barrier.wait()
                    sample = self.job(role, name, client)()
                    sample.setdefault('actualCallStartedNS',interval['actualCallStartedNS']); sample.setdefault('actualCallFinishedNS',interval['actualCallFinishedNS'])
                    return sample
                finally:
                    client.call = call
            samples = capacity.parallel([lambda c=c: run(c) for c in clients]) if sessions > 1 else [run(clients[0])]
            end = time.monotonic_ns()
            for c, sample in zip(clients, samples, strict=True):
                if sample['ok']: c.last_successful_activity = time.monotonic()
            wall_end = datetime.now(ZoneInfo('UTC')).isoformat()
            for i, sample in enumerate(samples):
                sample.update(wave=wave, client=i, waveStartedNS=start, waveFinishedNS=end,
                              waveStartedUTC=wall_start, waveFinishedUTC=wall_end)
            values.extend(samples)
            group = read_stats(values, name, permission_negative, sessions=sessions)
            events = sorted((t, delta) for sample in values for t, delta in
                            [(sample['actualCallStartedNS'], 1), (sample['actualCallFinishedNS'], -1)])
            active = peak = 0
            for _, delta in events:
                active += delta
                peak = max(peak, active)
            group.update(completed=wave == waves-1, expectedRequests=count, sessions=sessions,
                         observedAPIPeakRSSKiB=max(self.rss[rss_start:], default=None),
                         maximumObservedConcurrentCalls=peak,
                         concurrencyScope='HTTP call through complete response decode, excluding semantic assertions; monotonic timestamps',
                         coordinatedStart=sessions > 1,
                         coldScope='independent first request' if self.args.phase == 'cold-first' else 'shared cold-start first wave' if self.args.phase == 'cold-wave' else 'warm' if self.args.phase == 'warm' else 'diagnostic single sample',
                         permissionNegative=permission_negative)
            self.result['groups'][tag] = group
            self.save()
        print(json.dumps(dict(group=tag, phase=self.args.phase, requests=len(values), failures=group['failures'], p95MS=group['p95MS'])), flush=True)

    def run(self):
        for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
            os.environ.pop(key, None)
        if self.args.profile:
            assert self.args.phase == 'observe', 'profiling is excluded from formal latency groups'
            os.environ['CHNHCM_CAPACITY_PROFILE'] = self.args.profile
        self.result['profile'] = self.args.profile
        self.log_start = None
        try:
            if self.args.phase == 'observe':
                os.environ['CHNHCM_QUERY131_STATS'] = str(self.private/(self.output.stem+'-work.jsonl'))
                diagnostics = self.private/(self.output.stem+'-stages'); diagnostics.mkdir(mode=0o700)
                os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(diagnostics)
                # Task-owned database only. Raw SQL/parameters stay private and the
                # logging cost is excluded from every formal latency group.
                self.log_start = datetime.now(ZoneInfo('UTC')).isoformat()
                sql(self.container, "ALTER SYSTEM SET log_line_prefix='%m [%p] %q%u@%d '; ALTER SYSTEM SET log_statement='all'")
                sql(self.container, 'SELECT pg_reload_conf()')
            self.server.start(); assert self.server.info['apiConnectionLimit'] == 12
            assert self.server.info['model'] == 'm02-compact-5'
            self.result['startup'] = self.server.info
            planned = set()
            for role in self.result['roles']:
                self.login(role)
                groups = ['list', 'path']
                groups += ['start-explicit', 'start-all', 'resolve']
                groups += [f'parent-{m}-{v}' for m in ['create', 'maintain'] for v in ['first', 'search', 'second']]
                for group in groups:
                    if self.args.group and group not in self.args.group: continue
                    sessions = [1, 20] if self.args.phase == 'warm' else [20] if self.args.phase == 'cold-wave' else [1]
                    for n in sessions:
                        # A bounded diagnostic subclass may expand one entry to
                        # explicitly declared concurrency waves; ordinary lanes
                        # retain their existing one-group expectation.
                        planned.update(f'{role}/{group}/{wave_sessions}'
                            for wave_sessions in getattr(self.args, 'waves', [n]))
                        self.group(role, group, n)
            assert set(self.result['groups']) | set(self.reused) == planned
            self.result['completed'] = bool(self.result['groups']) and all(g.get('completed') for g in self.result['groups'].values())
            self.result['actualTimedRequests'] = sum(g['requests'] for g in self.result['groups'].values())
            self.result['groupCount'] = len(self.result['groups'])
            self.result['reusedTimedRequests'] = sum(g['requests'] for g in self.reused.values())
            self.result['coveredGroupCount'] = self.result['groupCount'] + len(self.reused)
        finally:
            self.server.close(); self.stop.set(); self.monitor.join()
            self.result['peakRSSKiB'] = max(self.rss, default=None)
            self.result['after'] = case.cli('size', self.private)
            before, after = self.result['before']['counts'], self.result['after']['counts']
            self.result['businessSourceGrowth'] = {k: after[k]-before[k] for k in ['buSources', 'departmentSources']}
            if self.args.phase == 'observe' and self.log_start:
                sql(self.container, "ALTER SYSTEM RESET log_statement; ALTER SYSTEM RESET log_line_prefix")
                sql(self.container, 'SELECT pg_reload_conf()')
                log = subprocess.run(['docker', 'logs', '--timestamps', '--since', self.log_start, self.container], capture_output=True, text=True, check=True)
                raw_log = self.private/(self.output.stem+'-postgres.log')
                raw_log.write_text(log.stdout+log.stderr); raw_log.chmod(0o600)
                events = []
                for line in (log.stdout+log.stderr).splitlines():
                    if 'chnhcm_api@storage126' not in line or not re.search(r'LOG:\s+(execute [^:]*:|statement:)', line): continue
                    stamp, _, rest = line.partition(' ')
                    events.append(dict(at=datetime.fromisoformat(stamp.replace('Z', '+00:00')), text=rest))
                for group in self.result['groups'].values():
                    for sample in group.get('samples', []):
                        start = datetime.fromisoformat(sample['waveStartedUTC']); end = datetime.fromisoformat(sample['waveFinishedUTC'])
                        stable = abs((end-start).total_seconds()*1000-(sample['waveFinishedNS']-sample['waveStartedNS'])/1e6) <= 100
                        sample['sqlWallClockBracketStable'] = stable
                        sample['sqlCommandCalls'] = sum(start <= e['at'] <= end for e in events) if stable else None
                self.result['sqlCommandsWholeDiagnosticIncludingPreparation'] = len(events)
                self.result['sqlCountScope'] = 'PostgreSQL logged top-level API command calls, including BEGIN/COMMIT and multi-statement batches; sequential diagnostic request wall-clock brackets; raw log private'
                work = self.private/(self.output.stem+'-work.jsonl')
                if work.exists():
                    self.result['work'] = [json.loads(line) for line in work.read_text().splitlines()]
                    captured = {}
                    for observation in self.result['work']:
                        if observation.get('Request'):
                            captured.setdefault(str(observation['Request']), []).append(observation)
                    for group in self.result['groups'].values():
                        for sample in group.get('samples', []):
                            sample['captureObservations'] = [o for request in sample.get('requestIDs',[sample.get('request')]) for o in captured.get(str(request),[])]
                stages = self.private/(self.output.stem+'-stages')/'requests.jsonl'
                if stages.exists():
                    requests = {str(v['Request']): v for v in (json.loads(line) for line in stages.read_text().splitlines())}
                    for group in self.result['groups'].values():
                        for sample in group.get('samples', []):
                            request = requests.get(str(sample.get('request')), {})
                            events = request.get('Stages', [])
                            sample['qualifiedTraversals'] = sum(e['Name'] == 'references.qualify' for e in events)
                            sample['stages'] = [e for e in events if e['Name'] != 'references.qualify']
                            sample['memory'] = {k: request[k] for k in ['ProcessAllocatedBytes', 'ProcessHeapChangeBytes', 'ProcessMallocs', 'ConcurrentMemoryWindow'] if k in request}
                            sample['memoryScope'] = 'process-wide runtime delta across one diagnostic HTTP request; includes runtime and observation overhead; concurrent windows are not attributable to one request'
            self.result['passed'] = self.result['completed'] and all(g['passed'] for g in list(self.result['groups'].values())+list(self.reused.values())) and all(x['ok'] for x in self.result['preparation']) and not any(self.result['businessSourceGrowth'].values())
            self.save()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('shape', choices=['synthetic', 'vanke']); p.add_argument('private', type=Path); p.add_argument('output', type=Path)
    p.add_argument('--binary', required=True, type=Path)
    p.add_argument('--phase', required=True, choices=['warm', 'cold-first', 'cold-wave', 'observe'])
    p.add_argument('--group', action='append')
    p.add_argument('--role', action='append', choices=['admin', 'reader'], help='explicit role partition, default both; sample minima per group stay unchanged')
    p.add_argument('--profile', choices=['cpu', 'allocs', 'mutex', 'block'])
    p.add_argument('--baseline-model4', action='store_true', help='isolated frozen A only; model 4 has no atomic start/page protocol')
    p.add_argument('--completed-evidence', type=Path, help='explicit admission of previously completed warm groups; original failures remain in that immutable evidence')
    p.add_argument('--reuse-group', action='append', default=[])
    args = p.parse_args(); matrix = Matrix(args); matrix.run()
    if not matrix.result['passed']: raise SystemExit(1)
