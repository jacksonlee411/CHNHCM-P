#!/usr/bin/env python3
"""Original 125 write partition, with separate request-correlated cold diagnosis.

Formal mode keeps all six original groups, five cold resets per group and all
20 clients. Diagnosis runs one wave for each of the four affected combinations;
it never supplies a capacity verdict. Credentials stay in the owned private case.
"""
import argparse
import json
import os
from pathlib import Path
import sys
import subprocess
import threading
import time

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/remediation125'))
import measure


class WaitSampler:
    """Diagnostic-only SQL waits, without query text, inputs or credentials."""
    def __init__(self, container):
        self.process = subprocess.Popen(['docker', 'exec', '-i', container, 'psql', '-X', '-U', 'postgres', '-d', 'storage126', '-Atq', '-v', 'ON_ERROR_STOP=1'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
        self.finished = threading.Event()
        self.samples, self.errors = [], []
        self.thread = threading.Thread(target=self.observe)
        self.thread.start()

    def observe(self):
        query = "SELECT coalesce(json_agg(json_build_object('pid',pid,'type',wait_event_type,'event',wait_event,'shapeInsert',query LIKE 'INSERT INTO organization.%_shape%','blockers',pg_blocking_pids(pid))),'[]'::json) FROM pg_stat_activity WHERE datname=current_database() AND usename='chnhcm_api' AND state='active' AND wait_event_type IS NOT NULL;\n"
        try:
            while not self.finished.is_set():
                # Correlation only: sample counts are never wait durations.
                started = time.time_ns()
                self.process.stdin.write(query)
                self.process.stdin.flush()
                raw = self.process.stdout.readline()
                if not raw:
                    raise RuntimeError('observer exited')
                finished = time.time_ns()
                for sample in json.loads(raw):
                    sample.update(observedStartUnixNS=started, observedEndUnixNS=finished)
                    self.samples.append(sample)
                self.finished.wait(.01)
        except Exception as error:
            self.errors.append(type(error).__name__)
        finally:
            try:
                self.process.stdin.write('\\q\n')
                self.process.stdin.flush()
                self.process.stdin.close()
                self.process.wait(timeout=5)
            except (BrokenPipeError, OSError, subprocess.TimeoutExpired):
                self.errors.append('shutdown')

    def close(self):
        self.finished.set()
        self.thread.join(timeout=10)
        assert not self.thread.is_alive() and self.process.poll() == 0
        assert not self.errors, self.errors
        return self.samples


def run(a):
    assert not a.output.exists()
    if a.split_constraints and not a.diagnostic:
        raise ValueError('constraint separation is diagnostic only')
    os.environ.pop('CHNHCM_CAPACITY_SPLIT_CONSTRAINTS', None)
    for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
        os.environ.pop(key, None)
    runs = []
    waits = []
    original_server = measure.case.Server
    original_measured = measure.measured

    if a.diagnostic:
        class ObservedServer(original_server):
            def start(self):
                path = a.private / (a.output.stem + '-stages-' + str(len(runs)))
                path.mkdir(mode=0o700)
                runs.append(path)
                os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(path)
                if a.split_constraints:
                    os.environ['CHNHCM_CAPACITY_SPLIT_CONSTRAINTS'] = '1'
                try:
                    base = super().start()
                    self.wait_sampler = WaitSampler(self.info['container'])
                    return base
                finally:
                    os.environ.pop('CHNHCM_CAPACITY_DIAGNOSTICS', None)
                    os.environ.pop('CHNHCM_CAPACITY_SPLIT_CONSTRAINTS', None)

            def stop(self):
                sampler = getattr(self, 'wait_sampler', None)
                if sampler:
                    waits.append(dict(diagnosticRun=len(runs)-1, samples=sampler.close()))
                    self.wait_sampler = None
                return super().stop()

        def observed(*args, **kwargs):
            sample, value = original_measured(*args, **kwargs)
            sample['diagnosticRun'] = len(runs) - 1
            return sample, value

        measure.case.Server = ObservedServer
        measure.measured = observed
    lane = None
    try:
        lane = measure.Lane(a.private, a.output, a.binary)
        lane.result['diagnosticOnly'] = a.diagnostic
        lane.result['splitDeferredConstraints'] = a.split_constraints
        preparation = subprocess.run(['docker', 'exec', '-i', lane.server.info['container'],
            'psql', '-X', '-U', 'postgres', '-d', 'storage126', '-Atq', '-v', 'ON_ERROR_STOP=1'],
            input="SELECT json_build_object('migration', (SELECT max(version_id) FROM goose_db_version WHERE is_applied), 'tables', (SELECT json_agg(json_build_object('table',relname,'rows',n_live_tup,'lastAnalyze',last_analyze) ORDER BY relname) FROM pg_stat_user_tables WHERE schemaname='organization'));",
            text=True, capture_output=True, check=True)
        lane.result['databasePreparation'] = json.loads(preparation.stdout)
        if not a.diagnostic:
            if a.phase == 'mixed':
                lane.matrix(a.phase)
            else:
                lane.matrix(a.phase, actions=['save', 'pair'], modes=['BU', 'Department', 'mixed'])
            lane.result['completed'] = (lane.result['allScheduledGroupsAttempted'] and
                lane.result['actualTimedRequests'] == lane.result['expectedTimedRequests'] and
                not lane.result['unattemptedRequests'])
        else:
            clients = lane.login()
            for mode, action in [('Department', 'save'), ('Department', 'pair'), ('mixed', 'save'), ('mixed', 'pair')]:
                jobs, cleanups, kinds = [], [], []
                for w, client in enumerate(clients):
                    kind = 'dep' if mode == 'Department' or w % 2 else 'bu'
                    job, cleanup = lane.write_job(client, kind, action, 0, w)
                    jobs.append(job)
                    cleanups.append(cleanup)
                    kinds.append(kind)
                lane.cold(mode + '-' + action)
                samples = []
                for (sample, _), kind in zip(measure.parallel(jobs), kinds):
                    sample.update(kind=kind, action=action)
                    samples.append(sample)
                lane.result['groups'][mode + '/' + action] = measure.stats(samples, None)
                for cleanup in cleanups:
                    cleanup()
            lane.result['completed'] = True
    except Exception as error:
        if lane is not None:
            lane.result['driverFailure'] = dict(type=type(error).__name__,
                status=getattr(error, 'status', None), code=getattr(error, 'code', None))
        raise
    finally:
        if lane is not None:
            lane.close()
            if a.diagnostic:
                stages = []
                for n, path in enumerate(runs):
                    log = path / 'requests.jsonl'
                    if log.exists():
                        for line in log.read_text().splitlines():
                            row = json.loads(line)
                            row['diagnosticRun'] = n
                            stages.append(row)
                lane.result['observations'] = stages
                lane.result['sqlWaitObservations'] = waits
                lane.result['sqlWaitSampling'] = '10ms requested interval, one administrative observer; sampled events are not exact wait durations'
                lane.result['capacityAcceptance'] = False
                measure.save(a.output, lane.result)
        measure.case.Server = original_server
        measure.measured = original_measured
    if not lane.result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--binary', required=True, type=Path)
    p.add_argument('--phase', choices=['warm', 'cold', 'mixed'], default='cold')
    p.add_argument('--diagnostic', action='store_true')
    p.add_argument('--split-constraints', action='store_true')
    args = p.parse_args()
    args.private = args.private.resolve()
    args.binary = args.binary.resolve()
    run(args)
