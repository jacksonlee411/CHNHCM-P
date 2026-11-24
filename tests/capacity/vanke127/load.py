#!/usr/bin/env python3
"""Resumable real-HTTP import; private rows, requests and responses stay outside Git."""
import argparse
import collections
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import signal
import sqlite3
import subprocess
import time
import urllib.error
import urllib.request
import uuid

from source import read_rows
from runtime import BIN, ROOT


class Client:
    def __init__(self, base):
        self.base, self.csrf = base, ''
        self.http = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def call(self, method, path, body=None, key=None):
        headers = {'Content-Type': 'application/json', 'Origin': self.base, 'X-CSRF-Token': self.csrf}
        if key:
            headers['Idempotency-Key'] = key
        req = urllib.request.Request(self.base + path, method=method, headers=headers, data=None if body is None else json.dumps(body, ensure_ascii=False).encode())
        start = time.perf_counter()
        try:
            with self.http.open(req, timeout=25) as response:
                status, result = response.status, json.load(response)
        except urllib.error.HTTPError as response:
            status, result = response.code, json.load(response)
        return status, result, (time.perf_counter() - start) * 1000


def authenticated_clients(base, workers):
    """Create independent worker sessions, respecting the normal login rate limit."""
    clients, login_times = [], []
    for _ in range(workers):
        if len(login_times) >= 20:
            delay = 60.2 - (time.perf_counter() - login_times[-20])
            if delay > 0:
                print(json.dumps(dict(event='login-rate-window', established=len(clients), remaining=workers-len(clients), waitSeconds=round(delay, 2))), flush=True)
                while delay > 0:
                    time.sleep(min(delay, 50))
                    delay = 60.2 - (time.perf_counter() - login_times[-20])
        client = Client(base)
        login_times.append(time.perf_counter())
        status, login, _ = client.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
        assert status == 200
        client.csrf = login['csrf']
        clients.append(client)
    return clients


def source_db(private, source):
    path = private / 'import.sqlite'
    new = not path.exists()
    db = sqlite3.connect(path)
    path.chmod(0o600)
    db.execute('PRAGMA journal_mode=WAL')
    db.executescript('''CREATE TABLE IF NOT EXISTS rows(phase TEXT,row_no INTEGER,code TEXT,payload TEXT,priority INTEGER,result TEXT,PRIMARY KEY(phase,row_no));
CREATE TABLE IF NOT EXISTS objects(code TEXT PRIMARY KEY,object_id TEXT,kind TEXT,head TEXT);
CREATE TABLE IF NOT EXISTS attempts(phase TEXT,row_no INTEGER,attempt INTEGER,method TEXT,path TEXT,body TEXT,key TEXT,status INTEGER,response TEXT,ms REAL,PRIMARY KEY(phase,row_no,attempt));''')
    if not new:
        return db
    data = {phase: read_rows(source / ('orgunit-import-' + phase + '-from-source.xlsx')) for phase in ['create', 'version', 'parent-change']}
    creates = {r['org_code']: r for r in data['create']}
    depths, visiting = {}, set()
    def depth(code):
        if code in depths:
            return depths[code]
        assert code not in visiting, 'cycle in initial topology'
        visiting.add(code)
        p = creates[code].get('parent_org_code')
        value = depth(p) + 1 if p else 0
        visiting.remove(code)
        depths[code] = value
        return value
    for code in creates:
        depth(code)
    for phase, rows in data.items():
        for i, row in enumerate(rows, 3):
            db.execute('INSERT INTO rows VALUES(?,?,?,?,?,NULL)', (phase, i, row['org_code'], json.dumps(row, ensure_ascii=False), depths[row['org_code']] if phase == 'create' else i))
    db.commit()
    return db


def acknowledged_record(previous, row, result):
    """Only reuse identities returned by the real API; never allocate record IDs."""
    assert result['expectedSourceVersion']==previous['head'], 'unexpected source receipt'
    if result['outcome']=='no_change':
        assert result['observedSourceVersion']==previous['head'] and result['selectedRecordId']==previous['recordId']
        return previous
    assert result['outcome']=='accepted'
    day=row.get('target_effective_date') or row['effective_date']
    assert result['effectiveDate']==day and day>=previous['effectiveDate'], 'chronological import invariant failed'
    if day==previous['effectiveDate']:
        assert result['action']=='CORRECT' and result['targetRecordId']==previous['recordId']
    else:
        assert result['action']=='ADD'
    return dict(head=result['sourceVersion'],recordId=result['targetRecordId'],revision=result['targetRevision'],effectiveDate=result['effectiveDate'])


def acknowledged_records(db, order):
    latest={};changed=[]
    for code,payload,result in db.execute("SELECT code,payload,result FROM rows WHERE phase='create'"):
        assert result,'all organizations must be created before historical changes'
        row,value=json.loads(payload),json.loads(result)['response']
        assert value['outcome']=='accepted' and value['effectiveDate']==row['effective_date']
        latest[code]=dict(head=value['sourceVersion'],recordId=value['targetRecordId'],revision=value['targetRevision'],effectiveDate=value['effectiveDate'])
    historical={(p,n):(c,json.loads(v),json.loads(r) if r else None) for p,n,c,v,r in db.execute("SELECT phase,row_no,code,payload,result FROM rows WHERE phase<>'create'")}
    pending_seen=False
    for phase,no in order:
        code,row,result=historical[phase,no]
        if result is None:pending_seen=True;continue
        assert not pending_seen,'acknowledged historical rows are not a chronological prefix'
        latest[code]=acknowledged_record(latest[code],row,result['response']);changed.append(code)
    for code,head in db.execute('SELECT code,head FROM objects'):
        assert latest[code]['head']==head,'local receipt directory mismatch'
    return latest,changed


def prepare_history(db, phase, code, row, previous):
    obj = db.execute('SELECT object_id,kind FROM objects WHERE code=?', (code,)).fetchone()
    assert obj, 'missing imported object'
    oid, kind = obj
    base = '/api/v1/' + kind + '/' + oid
    day = row.get('target_effective_date') or row['effective_date']
    assert day >= previous['effectiveDate'], 'historical order moved backwards'
    target = previous if day == previous['effectiveDate'] else None
    adaptations = []
    if phase == 'version':
        changes = {'name': row['core.name']} if row.get('core.name') else {}
        if row.get('ext.d_org_type'):
            adaptations.append('extension-retained-in-source-only')
    else:
        parent = db.execute('SELECT object_id,kind FROM objects WHERE code=?', (row['new_parent_org_code'],)).fetchone()
        assert parent, 'missing mapped parent'
        changes = {'parentDepartmentId': parent[0] if parent[1] == 'departments' else None}
    if row['operation'] == 'correct_parent' and target:
        path = base + '/records/' + target['recordId'] + '/correct'
        body = dict(expectedSourceVersion=previous['head'], expectedRecordRevision=target['revision'], changes=changes)
    else:
        if row['operation'] == 'correct_parent':
            original = db.execute("SELECT result FROM rows WHERE phase='version' AND code=? AND json_extract(payload,'$.effective_date')=?", (code, day)).fetchone()
            assert original and json.loads(original[0]).get('outcome') == 'no_change', 'missing correction target is not a known no-change version'
            adaptations.append('correct-target-no-change-version-becomes-dated-addition')
        assert previous['effectiveDate'] < day and not target, 'addition position conflict'
        path = base + '/records'
        body = dict(expectedSourceVersion=previous['head'], anchorRecordId=previous['recordId'], effectiveDate=day, operationType='CHANGE', operationReason='NAME_ADJUSTMENT' if phase == 'version' else 'ORGANIZATION_REASSIGNMENT', changes=changes)
    return path, body, adaptations


def run(private, source, evidence, phase, limit):
    requested_phase=phase
    db = source_db(private, source)
    log = (private / 'server.log').open('a')
    server = subprocess.Popen([str(BIN), '-mode', 'serve-import', '-private', str(private)], cwd=ROOT, stdout=subprocess.PIPE, stderr=log, text=True)
    info = json.loads(server.stdout.readline())
    client = Client(info['url'])
    started, processed, counts, durations = time.perf_counter(), 0, collections.Counter(), []
    source_counts=collections.Counter()
    preparation_checks=0
    try:
        status, login, _ = client.call('POST', '/api/v1/session/login', {'login': 'admin', 'password': 'admin1234567'})
        assert status == 200, 'login failed'
        client.csrf = login['csrf']
        if phase == 'history':
            order = json.loads((private/'history-order.json').read_text())
            pending = {(p,n):(p,n,c,v) for p,n,c,v in db.execute("SELECT phase,row_no,code,payload FROM rows WHERE phase<>'create' AND result IS NULL")}
            rows = [pending[(p,n)] for p,n in order if (p,n) in pending]
            latest,changed=acknowledged_records(db,order)
            # Compare resumed and new cursors with actual detail responses once.
            # Subsequent writes retain ordinary optimistic concurrency and all Go checks.
            probes=list(dict.fromkeys(changed[-20:]+[r[2] for r in rows[:20]]))
            for code in probes:
                oid,kind=db.execute('SELECT object_id,kind FROM objects WHERE code=?',(code,)).fetchone()
                status,detail,_=client.call('GET','/api/v1/'+kind+'/'+oid)
                assert status==200,'receipt preparation readback failed'
                last=detail['timeline'][-1];known=latest[code]
                assert detail['headSourceVersion']==known['head'] and all(last[k]==known[k] for k in ['recordId','revision','effectiveDate']),'receipt differs from actual latest record'
                preparation_checks+=1
            print(json.dumps(dict(event='acknowledged_record_readback',compared=preparation_checks,matched=True)),flush=True)
        else:
            rows = db.execute('SELECT phase,row_no,code,payload FROM rows WHERE phase=? AND result IS NULL ORDER BY priority,code,row_no', (phase,)).fetchall()
        for phase, row_no, code, payload in rows:
            if (private / 'stop-after-row').exists() or limit and processed >= limit:
                break
            row = json.loads(payload)
            key = str(uuid.uuid5(uuid.NAMESPACE_URL, 'chnhcm-vanke127/history-v1/' + phase + '/' + str(row_no)))
            adaptations = []
            if phase == 'create':
                kind = 'business-units' if row.get('is_business_unit') == 'true' else 'departments'
                path = '/api/v1/' + kind
                body = dict(code=code, name=row['name'], effectiveDate=row['effective_date'], operationType='CREATE', operationReason='NEW_ORGANIZATION')
                if kind == 'departments':
                    root = db.execute("SELECT object_id FROM objects WHERE kind='business-units'").fetchone()
                    parent = db.execute('SELECT object_id,kind FROM objects WHERE code=?', (row['parent_org_code'],)).fetchone()
                    if not root or not parent:
                        raise RuntimeError('missing parent from earlier source row; stop to preserve resumability')
                    body.update(managementBuId=root[0], parentDepartmentId=parent[0] if parent[1] == 'departments' else None)
            else:
                previous=latest[code]
                path, body, adaptations = prepare_history(db, phase, code, row, previous)
            attempt = db.execute('SELECT coalesce(max(attempt),0)+1 FROM attempts WHERE phase=? AND row_no=?', (phase, row_no)).fetchone()[0]
            db.execute('INSERT INTO attempts VALUES(?,?,?,?,?,?,?,NULL,NULL,NULL)', (phase, row_no, attempt, 'POST', path, json.dumps(body, ensure_ascii=False), key))
            db.commit()
            status, result, ms = client.call('POST', path, body, key)
            db.execute('UPDATE attempts SET status=?,response=?,ms=? WHERE phase=? AND row_no=? AND attempt=?', (status, json.dumps(result, ensure_ascii=False), ms, phase, row_no, attempt))
            db.commit()
            if status not in (200, 201):
                print(json.dumps(dict(phase=phase, row=row_no, status=status, code=result.get('code'), field=result.get('field'), ms=ms, processed=processed)), flush=True)
                raise RuntimeError('import first failure preserved; row remains pending')
            stored = dict(status=status, outcome=result['outcome'], response=result, adaptations=adaptations, requestKey=key)
            db.execute('UPDATE rows SET result=? WHERE phase=? AND row_no=?', (json.dumps(stored, ensure_ascii=False), phase, row_no))
            if phase == 'create':
                oid = result.get('buId') if kind == 'business-units' else result.get('departmentId')
                assert oid
                db.execute('INSERT INTO objects VALUES(?,?,?,?)', (code, oid, kind, result['sourceVersion']))
            else:
                latest[code]=acknowledged_record(previous,row,result)
                db.execute('UPDATE objects SET head=? WHERE code=?', (latest[code]['head'], code))
            db.commit()
            processed += 1
            counts[result['outcome']] += 1
            source_counts[phase+'.'+result['outcome']] += 1
            durations.append(ms)
            if processed % 100 == 0:
                progress = dict(phase=requested_phase, lastSourcePhase=phase, processed=processed, pending=len(rows)-processed, elapsedSeconds=round(time.perf_counter()-started,2), meanRequestMS=round(sum(durations)/len(durations),2), outcomes=dict(counts),sourceOutcomes=dict(source_counts))
                evidence.write_text(json.dumps(progress, indent=2)+'\n')
                print(json.dumps(progress), flush=True)
        out = dict(phase=requested_phase, lastSourcePhase=phase if processed else None, processed=processed, pending=len(rows)-processed, elapsedSeconds=round(time.perf_counter()-started,2), meanRequestMS=round(sum(durations)/len(durations),2) if durations else None, outcomes=dict(counts), sourceOutcomes=dict(source_counts), acknowledgedRecordReadbacks=preparation_checks, complete=processed==len(rows))
        evidence.write_text(json.dumps(out, indent=2)+'\n')
        print(json.dumps(out), flush=True)
    finally:
        server.send_signal(signal.SIGTERM)
        server.wait(timeout=10)
        log.close()
        db.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('source', type=Path)
    p.add_argument('evidence', type=Path)
    p.add_argument('--phase', choices=['create', 'history'], default='create')
    p.add_argument('--limit', type=int, default=0)
    a = p.parse_args()
    run(a.private, a.source, a.evidence, a.phase, a.limit)
