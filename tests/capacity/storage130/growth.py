#!/usr/bin/env python3
"""Two explicitly selected 1000-command growth cycles on owned A/B cases.

All inputs, receipts, labels and failures remain in the external private case.
This serial lane measures growth; it makes no concurrent-throughput claim.
"""
import argparse
import copy
from collections import Counter
import importlib.util
import hashlib
import json
import os
from pathlib import Path
import subprocess
import threading
import time
import uuid

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('case126', ROOT/'tests/capacity/storage126/measure.py')
case = importlib.util.module_from_spec(spec)
spec.loader.exec_module(case)


def save(path, value):
    with path.with_suffix('.writing').open('w') as f:
        json.dump(value, f, ensure_ascii=False, indent=2)
        f.write('\n')
    path.with_suffix('.writing').replace(path)
    path.chmod(0o600)


def resource(private):
    cfg = json.loads((private/'candidate-private.json').read_text())
    name, task = {'A': ('chnhcm-storage126-product', 'storage126-product'),
                  'B': ('chnhcm-storage130-b', 'storage130')}[cfg['Role']]
    assert cfg['Container'] == name and cfg['Volume'] == name+'-data'
    labels = json.loads(subprocess.check_output(['docker', 'inspect', '--format', '{{json .Config.Labels}}', name]))
    assert labels.get('io.chnhcm.task') == task and labels.get('io.chnhcm.temporary') == 'true'
    assert labels.get('io.chnhcm.parent-task') == 'storage130'
    return name


def sql(name, query):
    reply = subprocess.run(['docker','exec','-i',name,'psql','-X','-U','postgres','-d','storage126','-Atq','-v','ON_ERROR_STOP=1'],
                           input=query, text=True, capture_output=True)
    if reply.returncode:
        raise RuntimeError('owned growth SQL failed; private values omitted')
    return reply.stdout.strip()


def capture(private):
    value = case.cli('size', private)
    name = resource(private)
    value['orphanDictionaries'] = json.loads(sql(name, "SELECT jsonb_object_agg(kind,n) FROM ("+
        " UNION ALL ".join(f"SELECT '{k}' kind,count(*) n FROM organization.{k}_dictionary d WHERE NOT EXISTS(SELECT FROM organization.{k}_history h WHERE h.dictionary_id=d.id) AND NOT EXISTS(SELECT FROM organization.{k}_terminal t WHERE t.dictionary_id=d.id) AND NOT EXISTS(SELECT FROM organization.{k}_read_segment r WHERE r.dictionary_id=d.id)" for k in ['bu','department'])+") v"))
    return value


class LockSampler:
    """Observe actual advisory-lock residence with explicit sampling bounds.

    This owns one administrative observer connection, outside the 12 API pool.
    It is closed before any offline maintenance, and never changes business data.
    """
    def __init__(self, name, path):
        self.process = subprocess.Popen(['docker','exec','-i',name,'psql','-X','-U','postgres','-d','storage126','-Atq','-v','ON_ERROR_STOP=1'],
                                        stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
        self.path = path
        self.stop_event = threading.Event()
        self.intervals = []
        self.errors = []
        self.samples = 0
        self.thread = threading.Thread(target=self.observe)
        self.thread.start()

    def observe(self):
        previous = None
        active = {}
        query = "SELECT jsonb_build_object('at',extract(epoch FROM clock_timestamp())*1000,'holders',coalesce((SELECT jsonb_agg(DISTINCT l.pid::text||':'||l.virtualtransaction) FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.datname=current_database() AND a.usename='chnhcm_api' AND l.locktype='advisory' AND l.granted),'[]'::jsonb));\n"
        try:
            with self.path.open('x') as log:
                self.path.chmod(0o600)
                while not self.stop_event.is_set():
                    query_start = time.monotonic_ns()
                    self.process.stdin.write(query);self.process.stdin.flush()
                    raw = self.process.stdout.readline()
                    query_end = time.monotonic_ns()
                    if not raw:
                        raise RuntimeError('owned lock observer exited')
                    sample = json.loads(raw)
                    sample.update(localQueryStartNS=query_start,localQueryEndNS=query_end)
                    log.write(json.dumps(sample)+'\n')
                    holders = set(sample['holders'])
                    self.samples += 1
                    for pid in set(active)-holders:
                        first_end, last_start, before = active.pop(pid)
                        self.intervals.append(dict(lowerMS=max(0,last_start-first_end)/1e6,
                                                   upperMS=(query_end-before)/1e6 if before is not None else None,
                                                   leftCensored=before is None))
                    for pid in holders:
                        if pid not in active:
                            active[pid] = [query_end,query_start,previous]
                        else:
                            active[pid][1] = query_start
                    previous = query_start
                    self.stop_event.wait(.01)
                for first_end,last_start,before in active.values():
                    self.intervals.append(dict(lowerMS=max(0,last_start-first_end)/1e6,upperMS=None,rightCensored=True))
        except Exception as error:
            self.errors.append(type(error).__name__)
        finally:
            try:
                self.process.stdin.write('\\q\n');self.process.stdin.flush();self.process.stdin.close()
                self.process.wait(timeout=5)
            except (BrokenPipeError,OSError,subprocess.TimeoutExpired):
                self.errors.append('observer-shutdown')

    def close(self):
        self.stop_event.set();self.thread.join(timeout=10)
        assert not self.thread.is_alive() and self.process.poll() == 0, 'lock observer must close before maintenance'
        return dict(samples=self.samples,intervals=self.intervals,errors=self.errors,
                    requestedIntervalMS=10,clock='local monotonic_ns around each complete SQL observation; PG wall clock is diagnostic only',
                    coverage='sampled advisory-lock residence with query-time uncertainty; shorter unsampled locks are not zero; left/right censored intervals have no upper bound',
                    administrativeObserverConnections=1,apiConnections=12)


def run(private, binary, maintenance_binary, plan_path, cycle, output):
    assert private.is_absolute() and ROOT not in private.parents and private.stat().st_mode & 0o777 == 0o700
    assert not output.exists(), 'retain the first complete denominator'
    case.BIN = binary
    name = resource(private)
    plan = json.loads(plan_path.read_text())
    assert len(plan['objects']) == 500 and len(set(plan['objects'])) == 500
    diagnostics = private/f'growth-{cycle}-diagnostics'
    diagnostics.mkdir(mode=0o700)
    previous = os.environ.get('CHNHCM_CAPACITY_DIAGNOSTICS')
    os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(diagnostics)
    server = case.Server(private, private)
    client = case.Client()
    result = dict(completed=False, cycle=cycle, serial=True, expectedCommands=1000,
                  binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                  failures=[], writes=[], skippedDependentCommands=[], stage='starting', before=capture(private))
    save(output, result)
    cache = {}
    receipts = []
    actions = Counter()
    lock_sampler = None

    def detail(oid):
        if oid not in cache:
            cache[oid] = client.call('GET', '/api/v1/departments/'+oid)[0]
        return cache[oid]

    def write(oid, category, suffix, body, want=200, check=None):
        key = str(uuid.uuid4())
        started = time.perf_counter()
        sample = dict(index=len(result['writes']), category=category, ok=False)
        try:
            expected = copy.deepcopy(detail(oid)['timeline'])
            response, elapsed = client.call('POST', '/api/v1/departments/'+oid+suffix, body, key, want=want)
            sample['request'] = client.request_id
            assert response['outcome'] == 'accepted' and response['submissionId'] == key
            assert response['departmentId'] == oid and response['sourceVersion'] != body['expectedSourceVersion']
            sample['applied'] = True
            actions[response['action']] += 1
            receipts.append(dict(object=oid, key=key, receipt=response, category=category))
            if check:
                check(response)
            if category.startswith('correction-'):
                rid = suffix.split('/')[2]
                next(r for r in expected if r['recordId'] == rid)['name'] = body['changes']['name']
            elif category == 'date-add':
                row = copy.deepcopy(expected[-1])
                row.update(recordId=response['targetRecordId'],effectiveDate=body['effectiveDate'],name=body['changes']['name'])
                expected.append(row)
            elif category == 'date-move':
                rid = suffix.split('/')[2]
                next(r for r in expected if r['recordId'] == rid)['effectiveDate'] = body['newEffectiveDate']
            elif category == 'pair-add':
                # New IDs come from the accepted result's exact source below;
                # every business value/date/state is predicted from the input.
                for day, state in [(body['deactivationDate'],'inactive'),(body['reactivationDate'],'active')]:
                    row = copy.deepcopy(expected[-1]);row.update(recordId=None,effectiveDate=day,state=state)
                    expected.append(row)
            elif category == 'pair-restore':
                expected = expected[:-2]
            else:
                raise AssertionError('unclassified growth command')
            cache.pop(oid,None)
            after = detail(oid)
            assert after['headSourceVersion'] == response['sourceVersion']
            actual = after['timeline']
            assert len(actual) == len(expected)
            expected.sort(key=lambda r:r['effectiveDate'])
            for index, (row, want_row) in enumerate(zip(actual,expected,strict=True)):
                for field in ['recordId','name','effectiveDate','state','managementBuId','parentDepartmentId','shortName']:
                    if field == 'recordId' and want_row.get(field) is None:
                        uuid.UUID(row[field])
                    else:
                        assert row.get(field) == want_row.get(field), field
                assert row.get('endExclusive') == (expected[index+1]['effectiveDate'] if index+1 < len(expected) else None)
            sample.update(ok=True, ms=elapsed, action=response['action'])
            return response
        except Exception as error:
            sample.update(ms=getattr(error, 'elapsed_ms', (time.perf_counter()-started)*1000),
                          errorType=type(error).__name__, status=getattr(error,'status',None),code=getattr(error,'code',None))
            if not sample.get('applied'):
                try:
                    recovered, _ = client.call('POST','/api/v1/departments/submissions/'+key+'/resolve')
                    sample['resolvedOutcome'] = recovered.get('outcome')
                    if recovered.get('outcome') == 'accepted':
                        sample['applied'] = True
                        actions[recovered['action']] += 1
                        receipts.append(dict(object=oid,key=key,receipt=recovered,category=category))
                except Exception as resolution_error:
                    sample['resolveErrorType'] = type(resolution_error).__name__
            result['failures'].append(dict(sample))
            cache.pop(oid, None)
            return None
        finally:
            result['writes'].append(sample)
            if len(result['writes']) % 50 == 0:
                result['stage'] = f"commands-{len(result['writes'])}"
                save(output, result)
                print(json.dumps(dict(cycle=cycle, commands=len(result['writes']), failed=len(result['failures']))), flush=True)

    try:
        client.base = server.start()
        login, _ = client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'))
        client.csrf = login['csrf']
        result['startup'] = server.info
        lock_sampler = LockSampler(name, private/f'growth-{cycle}-lock-samples.jsonl')
        started = time.perf_counter()
        # 300 name changes followed by 300 real restorations. Every restoration
        # creates a new immutable source; there is no SQL undo or history purge.
        originals = {}
        for index, oid in enumerate(plan['objects'][:300]):
            before = detail(oid)
            row = before['timeline'][0]
            originals[oid] = (row['recordId'], row['name'])
            write(oid, 'correction-new', '/records/'+row['recordId']+'/correct',
                  dict(expectedSourceVersion=before['headSourceVersion'], expectedRecordRevision=row['revision'],
                       changes=dict(name=f'130增长{cycle:02d}资料{index:04d}')))
        for oid in plan['objects'][:300]:
            before = detail(oid)
            rid, original = originals[oid]
            row = next(r for r in before['timeline'] if r['recordId'] == rid)
            write(oid, 'correction-restore', '/records/'+rid+'/correct',
                  dict(expectedSourceVersion=before['headSourceVersion'], expectedRecordRevision=row['revision'],changes=dict(name=original)))
        # 100 future additions and 100 real date moves of those new records.
        added = {}
        for index, oid in enumerate(plan['objects'][300:400]):
            before = detail(oid)
            response = write(oid, 'date-add', '/records',
                  dict(expectedSourceVersion=before['headSourceVersion'], anchorRecordId=before['timeline'][-1]['recordId'],
                       effectiveDate=f'{2040+cycle}-01-01',operationType='CHANGE',operationReason='NAME_ADJUSTMENT',
                       changes=dict(name=f'130增长{cycle:02d}未来{index:04d}')),want=201)
            if response:
                added[oid] = response['targetRecordId']
        for oid in plan['objects'][300:400]:
            if oid not in added:
                result['skippedDependentCommands'].append(dict(object=oid, category='date-move',
                    predecessor='date-add', reason='addition not confirmed; never substitute an existing record'))
                continue
            before = detail(oid)
            rid = added[oid]
            row = next(r for r in before['timeline'] if r['recordId'] == rid)
            write(oid, 'date-move', '/records/'+rid+'/redate',
                  dict(expectedSourceVersion=before['headSourceVersion'],expectedRecordRevision=row['revision'],newEffectiveDate=f'{2040+cycle}-02-01'))
        # 100 fixed lifecycle pairs, then 100 legitimate pair cancellations.
        pairs_added = set()
        for oid in plan['objects'][400:]:
            before = detail(oid)
            response = write(oid, 'pair-add', '/lifecycle-pairs',
                  dict(expectedSourceVersion=before['headSourceVersion'],anchorRecordId=before['timeline'][-1]['recordId'],
                       deactivationDate=f'{2050+cycle}-01-01',reactivationDate=f'{2050+cycle}-02-01',
                       deactivationReason='ORG_RESTRUCTURING',reactivationReason='BUSINESS_RESUMPTION'))
            if response:
                pairs_added.add(oid)
        for oid in plan['objects'][400:]:
            if oid not in pairs_added:
                result['skippedDependentCommands'].append(dict(object=oid, category='pair-restore',
                    predecessor='pair-add', reason='pair addition not confirmed; never substitute existing records'))
                continue
            before = detail(oid)
            a, z = before['timeline'][-2:]
            write(oid, 'pair-restore', '/lifecycle-pairs/delete',
                  dict(expectedSourceVersion=before['headSourceVersion'],deactivationRecordId=a['recordId'],deactivationRevision=a['revision'],
                       reactivationRecordId=z['recordId'],reactivationRevision=z['revision']))
        result['writeSeconds'] = time.perf_counter()-started
        result['lockResidence'] = lock_sampler.close()
        lock_sampler = None
        result['actions'] = dict(actions)
        result['afterWrites'] = capture(private)
        sql(name, 'VACUUM (ANALYZE)')
        result['afterOrdinaryVacuum'] = capture(private)
        # Restart clears all process-local history caches before checking the
        # complete original receipt returned by every accepted growth command.
        recovery_diagnostics = private/f'growth-{cycle}-recovery-diagnostics'
        recovery_diagnostics.mkdir(mode=0o700)
        os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = str(recovery_diagnostics)
        server.restart([client])
        restored = 0
        for value in receipts:
            reply, _ = client.call('POST','/api/v1/departments/submissions/'+value['key']+'/resolve')
            assert all(reply.get(k) == v for k,v in value['receipt'].items() if k != 'currentChanged')
            exact, _ = client.call('GET','/api/v1/departments/'+value['object']+'?sourceVersion='+value['receipt']['sourceVersion'])
            assert exact['readOnly'] and exact['resolvedSourceVersion'] == value['receipt']['sourceVersion']
            restored += 1
        result['everyAcceptedReceiptAndExactSourceAfterRestart'] = restored
        server.stop()
        assert sql(name,"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND backend_type='client backend' AND pid<>pg_backend_pid()") == '0'
        maintenance = subprocess.check_output([str(maintenance_binary),'-mode','maintain','-input',str(private)],cwd=ROOT)
        result['maintenance'] = json.loads(maintenance)
        result['afterOfflineCleanup'] = capture(private)
        result['sourceDelta'] = result['afterWrites']['counts']['departmentSources']-result['before']['counts']['departmentSources']
        result['completed'] = True
        result['passed'] = len(result['writes']) == 1000 and not result['failures'] and result['sourceDelta'] == 1000 and restored == 1000 and all(n == 0 for n in result['afterOfflineCleanup']['orphanDictionaries'].values())
    except Exception as error:
        result['failureType'] = type(error).__name__
        raise
    finally:
        if lock_sampler is not None:
            result['lockResidence'] = lock_sampler.close()
        server.close()
        if previous is None:
            os.environ.pop('CHNHCM_CAPACITY_DIAGNOSTICS',None)
        else:
            os.environ['CHNHCM_CAPACITY_DIAGNOSTICS'] = previous
        result['accepted'] = len(receipts)
        result['attemptedCommands'] = len(result['writes'])
        result['unattemptedCommands'] = result['expectedCommands'] - result['attemptedCommands']
        save(output, result)
        save(private/f'growth-{cycle}-receipts.json', receipts)
    if not result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private',type=Path);p.add_argument('plan',type=Path);p.add_argument('output',type=Path)
    p.add_argument('--binary',type=Path,required=True);p.add_argument('--maintenance-binary',type=Path)
    p.add_argument('--cycle',type=int,choices=[1,2,3,4,5,6],required=True)
    a = p.parse_args()
    run(a.private,a.binary.resolve(),(a.maintenance_binary or a.binary).resolve(),a.plan,a.cycle,a.output)
