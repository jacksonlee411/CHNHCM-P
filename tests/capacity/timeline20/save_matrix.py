#!/usr/bin/env python3
"""20-record ordinary corrections against a complete, owned background.

Preparation, formal timing and restart recovery are separate invocations.
Twenty clients own twenty independent objects. Never retry a failed write;
resolve its original identity outside timing. Cold means API/PG restart with
host cache retained, explicitly distinct from the older cache-eviction lane.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import math
import os
from pathlib import Path
import subprocess
import sys
import threading
import time
import uuid

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/storage130'))
from growth import case, resource, save, sql


def login(base):
    c = case.Client(); c.base = base
    value, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
    c.csrf = value['csrf']
    return c


def run(a):
    private = a.private.resolve(); assert not private.is_relative_to(ROOT)
    assert private.stat().st_mode & 0o777 == 0o700 and not a.output.exists()
    container = resource(private)
    marker_path = private / 'tree300-frozen.json'
    marker = json.loads(marker_path.read_text())
    assert marker['completeInputVerified'] and marker['model'] == 'm02-compact-5'
    assert marker['containerID'] == subprocess.check_output(['docker', 'inspect', '--format', '{{.Id}}', container], text=True).strip()
    case.BIN = a.binary.resolve()
    for key in ('CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE', 'CHNHCM_CAPACITY_SPLIT_CONSTRAINTS'):
        os.environ.pop(key, None)
    state_path = private / 'timeline20-concurrent-state.json'
    journal = private / 'timeline20-concurrent-receipts.jsonl'
    result = dict(phase=a.phase, completed=False, groups=[], binarySHA256=hashlib.sha256(case.BIN.read_bytes()).hexdigest(), driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                  apiConnections=12, recordCount=20, targetMS=500, coldDefinition='API and PostgreSQL restart; host filesystem cache retained',
                  backgroundBefore=json.loads(sql(container, "SELECT json_build_object('departments',(SELECT count(*) FROM organization.department),'businessUnits',(SELECT count(*) FROM organization.business_unit),'departmentSources',(SELECT sum(cardinality(source_refs)) FROM organization.department_history),'buSources',(SELECT sum(cardinality(source_refs)) FROM organization.bu_history))")))
    save(a.output, result)
    result['fixtureProofSHA256'] = hashlib.sha256(marker_path.read_bytes()).hexdigest()
    prefix = json.loads(state_path.read_text())['objects'] if a.resume_preparation else []
    assert not a.resume_preparation or a.phase == 'prepare'
    assert result['backgroundBefore']['departments'] == (87343 + sum(o['kind']=='department' for o in prefix) if a.phase == 'prepare' else 87363)
    assert result['backgroundBefore']['businessUnits'] == (1 + sum(o['kind']=='bu' for o in prefix) if a.phase == 'prepare' else 21)
    assert result['backgroundBefore']['departmentSources'] + result['backgroundBefore']['buSources'] >= marker['decodedSources']
    server = case.Server(private, private); server.start()
    assert server.info['apiConnectionLimit'] == 12
    clients = [login(server.base) for _ in range(20)]
    lock = threading.Lock()
    def persist(value):
        with lock:
            with journal.open('a') as f:
                journal.chmod(0o600); f.write(json.dumps(value, ensure_ascii=False)+'\n'); f.flush(); os.fsync(f.fileno())
    def command(c, route, body, want=200):
        key = str(uuid.uuid4())
        persist(dict(type='attempt', phase=a.phase, key=key, route=route, body=body))
        try:
            v, ms = c.call('POST', route, body, key, want=want)
        except Exception as e:
            persist(dict(type='failure', phase=a.phase, key=key, status=getattr(e, 'status', None), code=getattr(e, 'code', None)))
            raise
        persist(dict(type='receipt', phase=a.phase, key=key, route=route, receipt=v))
        assert v['outcome'] == 'accepted'
        return v, ms
    def restart():
        server.stop(); subprocess.run(['docker', 'restart', container], check=True, stdout=subprocess.DEVNULL)
        for _ in range(120):
            if subprocess.run(['docker', 'exec', container, 'pg_isready', '-U', 'postgres'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0: break
            time.sleep(.25)
        else: raise RuntimeError('owned database not ready')
        server.start()
        for c in clients: c.base = server.base
    try:
        if a.phase == 'prepare':
            if a.resume_preparation:
                state=json.loads(state_path.read_text()); assert state['generatedThroughRealHTTP'] and journal.exists() and len(state['objects'])<40
                for obj in state['objects']:
                    detail,_=clients[0].call('GET',obj['route'])
                    assert detail['headSourceVersion']==obj['source'] and len(detail['timeline'])==20
                    for actual,want in zip(detail['timeline'],obj['records'],strict=True): assert all(actual[k]==want[k] for k in want)
            else:
                assert not state_path.exists() and not journal.exists()
                state = dict(objects=[], generatedThroughRealHTTP=True)
            for kind, route, field in [('bu', 'business-units', 'buId'), ('department', 'departments', 'departmentId')]:
                for i in range(20):
                    ordinal=i+(20 if kind=='department' else 0)
                    if ordinal<len(state['objects']):
                        assert state['objects'][ordinal]['kind']==kind
                        continue
                    body = dict(code=f'CAP12520-{kind}-{i:02d}', name=f'20条容量{kind}-{i}-0', effectiveDate='2000-01-01', operationType='CREATE', operationReason='NEW_ORGANIZATION')
                    if kind == 'department': body.update(managementBuId=state['objects'][i]['id'], parentDepartmentId=None)
                    v, _ = command(clients[i], '/api/v1/'+route, body, want=201)
                    obj = dict(kind=kind, id=v[field], route='/api/v1/'+route+'/'+v[field], source=v['sourceVersion'], records=[])
                    for position in range(20):
                        if position:
                            body = dict(expectedSourceVersion=obj['source'], anchorRecordId=obj['records'][-1]['recordId'], effectiveDate=f'{2000+position}-01-01', operationType='CHANGE', operationReason='NAME_ADJUSTMENT', changes=dict(name=f'20条容量{kind}-{i}-{position}'))
                            v, _ = command(clients[i], obj['route']+'/records', body, want=201)
                            obj['source'] = v['sourceVersion']
                        obj['records'].append(dict(recordId=v.get('targetRecordId') or v['selectedRecordId'], revision=v['targetRevision'], name=f'20条容量{kind}-{i}-{position}', effectiveDate=f'{2000+position}-01-01'))
                    detail, _ = clients[i].call('GET', obj['route'])
                    assert len(detail['timeline']) == 20
                    for actual, expected in zip(detail['timeline'], obj['records'], strict=True):
                        assert all(actual[k] == expected[k] for k in expected if k!='revision')
                        # ADD changes the preceding end boundary and therefore
                        # its revision. Capture current concurrency tokens only
                        # after independently checking the complete values/IDs.
                        expected['revision']=actual['revision']
                    obj['originalSource'] = obj['source']; obj['originalRecords'] = json.loads(json.dumps(obj['records']))
                    state['objects'].append(obj); save(state_path, state)
            result.update(preparedObjects=40, generatedSources=800)
        else:
            state = json.loads(state_path.read_text()); assert state['generatedThroughRealHTTP'] and len(state['objects']) == 40
            if a.phase == 'recover':
                restart(); entries = [json.loads(line) for line in journal.read_text().splitlines()]
                receipts = [row for row in entries if row['type'] == 'receipt']
                closed = [dict(row, receipt=row['resolution']) for row in entries if row['type'] == 'resolution' and row['resolution']['outcome'] == 'not_applied']
                attempts = [row['key'] for row in entries if row['type'] == 'attempt']
                assert len(receipts) >= 800 and len(set(attempts)) == len(attempts)
                assert set(attempts) == {row['key'] for row in receipts + closed}
                for row in receipts + closed:
                    route = '/api/v1/'+('organization' if '/business-units' in row['route'] else 'departments')+'/submissions/'+row['key']+'/resolve'
                    actual, _ = clients[0].call('POST', route)
                    assert all(actual.get(k) == v for k, v in row['receipt'].items() if k != 'currentChanged')
                for obj in state['objects']:
                    for source, expected in [('', obj['records']), (obj['originalSource'], obj['originalRecords'])]:
                        detail, _ = clients[0].call('GET', obj['route']+('?'+'sourceVersion='+source if source else ''))
                        assert len(detail['timeline']) == 20
                        for actual, want in zip(detail['timeline'], expected, strict=True): assert all(actual[k] == want[k] for k in want)
                result.update(recoveredReceipts=len(receipts), recoveredNotApplied=len(closed), allAttemptIdentitiesResolved=True, currentAndOriginalObjects=40)
            else:
                for kind in ('bu', 'department'):
                    objects = [o for o in state['objects'] if o['kind'] == kind]
                    for position in (0, 10, 19):
                        for sessions in ([1, 20] if a.phase == 'warm' else [20]):
                            group = dict(kind=kind, position=position, sessions=sessions, expected=200, samples=[], warmups=[], resets=0)
                            result['groups'].append(group); save(a.output, result)
                            def write(i, gate, ordinal):
                                obj=objects[i]; record=obj['records'][position]; name=f'{a.phase}-{kind}-{position}-{ordinal}'
                                body=dict(expectedSourceVersion=obj['source'],expectedRecordRevision=record['revision'],changes=dict(name=name))
                                key=str(uuid.uuid4()); route=obj['route']+'/records/'+record['recordId']+'/correct'
                                persist(dict(type='attempt',phase=a.phase,key=key,route=route,body=body))
                                gate.wait(); started=time.perf_counter(); accepted=None; status=None
                                try:
                                    accepted,ms=clients[i].call('POST',route,body,key); status=200
                                    assert accepted['outcome']=='accepted' and accepted['sourceVersion']!=obj['source'] and accepted['targetRecordId']==record['recordId'] and accepted['targetRevision']!=record['revision']
                                    sample=dict(ok=True,status=200,ms=ms)
                                except Exception as e:
                                    sample=dict(ok=False,status=getattr(e,'status',status),code=getattr(e,'code',None),error=type(e).__name__,ms=getattr(e,'elapsed_ms',(time.perf_counter()-started)*1000))
                                    recovery='/api/v1/'+('organization' if kind=='bu' else 'departments')+'/submissions/'+key+'/resolve'
                                    resolved,_=clients[i].call('POST',recovery)
                                    assert resolved['outcome'] in ('accepted','not_applied')
                                    persist(dict(type='resolution',phase=a.phase,key=key,route=route,resolution=resolved))
                                    accepted=resolved if resolved['outcome']=='accepted' else None;sample['resolvedOutcome']=resolved['outcome']
                                if accepted:
                                    persist(dict(type='receipt',phase=a.phase,key=key,route=route,receipt=accepted))
                                    obj['source']=accepted['sourceVersion'];record.update(revision=accepted['targetRevision'],name=name)
                                return sample
                            if a.phase=='warm':
                                for wave in range(2):
                                    gate=threading.Barrier(sessions)
                                    with ThreadPoolExecutor(max_workers=sessions) as pool:
                                        futures=[pool.submit(write,i,gate,-1-wave*sessions-i) for i in range(sessions)]
                                        group['warmups'].extend(f.result() for f in futures)
                                    save(state_path,state);save(a.output,result)
                                assert all(s['ok'] for s in group['warmups']), 'warmup failure retained; formal group not admitted'
                            for wave in range(200//sessions):
                                if a.phase=='cold': restart(); group['resets']+=1
                                gate=threading.Barrier(sessions)
                                with ThreadPoolExecutor(max_workers=sessions) as pool:
                                    futures=[pool.submit(write,i,gate,wave*sessions+i) for i in range(sessions)]
                                    group['samples'].extend(f.result() for f in futures)
                                save(state_path,state);save(a.output,result)
                            values=sorted(s['ms'] for s in group['samples'])
                            group.update(p95MS=values[math.ceil(len(values)*.95)-1],p99MS=values[math.ceil(len(values)*.99)-1],maximumMS=max(values),errors=sum(not s['ok'] for s in group['samples']))
                            group['passed']=len(values)==200 and group['errors']==0 and group['p95MS']<=500
                            save(a.output,result)
        result['completed']=True
    finally:
        server.close();result['passed']=result['completed'] and all(g.get('passed',False) for g in result['groups']);save(a.output,result)


if __name__ == '__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--phase',choices=['prepare','warm','cold','recover'],required=True);p.add_argument('--resume-preparation',action='store_true',help='continue only after all acknowledged complete objects have been reconstructed and verified');run(p.parse_args())
