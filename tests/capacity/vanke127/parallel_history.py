#!/usr/bin/env python3
"""Independent leaf commands, with ancestor barriers and a durable batch journal."""
import argparse
import collections
import concurrent.futures
import fcntl
import json
from pathlib import Path
import queue
import signal
import subprocess
import time
import uuid

from load import Client, acknowledged_record, acknowledged_records, prepare_history, source_db
from runtime import BIN, ROOT


def dispatch(db, jobs, clients, parallel=True):
    results = {}
    pending = []
    for index, job in enumerate(jobs):
        old = db.execute('SELECT status,response,ms FROM attempts WHERE phase=? AND row_no=? AND status IN (200,201) ORDER BY attempt DESC LIMIT 1', (job['phase'], job['row'])).fetchone()
        if old:
            results[index] = (old[0], json.loads(old[1]), old[2])
            continue
        attempt = db.execute('SELECT coalesce(max(attempt),0)+1 FROM attempts WHERE phase=? AND row_no=?', (job['phase'], job['row'])).fetchone()[0]
        db.execute('INSERT INTO attempts VALUES(?,?,?,?,?,?,?,NULL,NULL,NULL)', (job['phase'], job['row'], attempt, 'POST', job['path'], json.dumps(job['body'], ensure_ascii=False), job['key']))
        pending.append((index, attempt))
    db.commit()  # Every exact request is durable before any request is sent.
    errors = []
    available = queue.Queue()
    for client in clients:
        available.put(client)
    def call(job):
        client = available.get()
        try:
            return client.call('POST', job['path'], job['body'], job['key'])
        finally:
            available.put(client)
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(clients) if parallel else 1) as pool:
        futures = {}
        for index, attempt in pending:
            job = jobs[index]
            future = pool.submit(call, job)
            futures[future] = (index, attempt)
        for future in concurrent.futures.as_completed(futures):
            index, attempt = futures[future]
            job = jobs[index]
            try:
                status, result, ms = future.result()
            except Exception as error:
                errors.append(dict(phase=job['phase'], row=job['row'], errorType=type(error).__name__))
                continue
            db.execute('UPDATE attempts SET status=?,response=?,ms=? WHERE phase=? AND row_no=? AND attempt=?', (status, json.dumps(result, ensure_ascii=False), ms, job['phase'], job['row'], attempt))
            db.commit()
            if status not in (200, 201):
                errors.append(dict(phase=job['phase'], row=job['row'], status=status, code=result.get('code'), ms=ms))
            else:
                results[index] = (status, result, ms)
    if errors:
        print(json.dumps(dict(event='batch-failure-preserved', failures=errors, successfulReceipts=len(results))), flush=True)
        raise RuntimeError('batch journal retained; resume exact requests before any new command')
    assert len(results) == len(jobs)
    # Publish the complete chronological prefix atomically. If interrupted,
    # replay only missing receipts with the original key/body, never new IDs.
    for index, job in enumerate(jobs):
        status, response, _ = results[index]
        raw = json.loads(db.execute('SELECT payload FROM rows WHERE phase=? AND row_no=?', (job['phase'], job['row'])).fetchone()[0])
        after = acknowledged_record(job['previous'], raw, response)
        stored = dict(status=status, outcome=response['outcome'], response=response, adaptations=job['adaptations'], requestKey=job['key'])
        db.execute('UPDATE rows SET result=? WHERE phase=? AND row_no=? AND result IS NULL', (json.dumps(stored, ensure_ascii=False), job['phase'], job['row']))
        db.execute('UPDATE objects SET head=? WHERE code=?', (after['head'], job['code']))
    db.execute('DELETE FROM history_batch WHERE id=1')
    db.commit()
    return [results[i] for i in range(len(jobs))]


def run(private, source, evidence, workers, limit, window=0):
    assert 1 <= workers <= 64
    window = window or workers
    assert workers <= window <= 256
    lock = (private / 'history-run.lock').open('a')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    db = source_db(private, source)
    db.execute('CREATE TABLE IF NOT EXISTS history_batch(id INTEGER PRIMARY KEY CHECK(id=1),jobs TEXT NOT NULL)')
    db.commit()
    log = (private / 'server.log').open('a')
    server = subprocess.Popen([str(BIN), '-mode', 'serve-import', '-private', str(private)], cwd=ROOT, stdout=subprocess.PIPE, stderr=log, text=True)
    start = time.perf_counter()
    output = dict(workers=workers, window=window, complete=False, processed=0, scope='distinct globally independent leaves between nonleaf barriers; per-object and ancestor history order retained')
    try:
        info = json.loads(server.stdout.readline())
        clients = []
        login_times = []
        for _ in range(workers):
            if len(login_times) >= 20:
                delay = 60.2 - (time.perf_counter() - login_times[-20])
                if delay > 0:
                    print(json.dumps(dict(event='login-rate-window', established=len(clients), remaining=workers-len(clients), waitSeconds=round(delay, 2))), flush=True)
                    while delay > 0:
                        time.sleep(min(delay, 50)); delay = 60.2 - (time.perf_counter() - login_times[-20])
            client = Client(info['url'])
            login_times.append(time.perf_counter())
            status, login, _ = client.call('POST', '/api/v1/session/login', {'login': 'admin', 'password': 'admin1234567'})
            assert status == 200
            client.csrf = login['csrf']; clients.append(client)
        active = db.execute('SELECT jobs FROM history_batch WHERE id=1').fetchone()
        output['resumedJournalRows'] = 0
        if active:
            jobs = json.loads(active[0])
            dispatch(db, jobs, clients, parallel=False)
            output['resumedJournalRows'] = len(jobs)
        order = json.loads((private / 'history-order.json').read_text())
        latest, changed = acknowledged_records(db, order)
        pending = {(p, n): (p, n, c, json.loads(v)) for p, n, c, v in db.execute("SELECT phase,row_no,code,payload FROM rows WHERE phase<>'create' AND result IS NULL")}
        rows = [pending[p, n] for p, n in order if (p, n) in pending]
        parents = {v[0] for v in db.execute("SELECT json_extract(payload,'$.parent_org_code') FROM rows WHERE phase='create' UNION SELECT json_extract(payload,'$.new_parent_org_code') FROM rows WHERE phase='parent-change'")}
        probes = list(dict.fromkeys(changed[-20:] + [r[2] for r in rows[:20]]))
        for code in probes:
            oid, kind = db.execute('SELECT object_id,kind FROM objects WHERE code=?', (code,)).fetchone()
            status, detail, _ = clients[0].call('GET', '/api/v1/' + kind + '/' + oid)
            assert status == 200
            last, known = detail['timeline'][-1], latest[code]
            assert detail['headSourceVersion'] == known['head'] and all(last[k] == known[k] for k in ['recordId', 'revision', 'effectiveDate'])
        output.update(acknowledgedRecordReadbacks=len(probes), initialPending=len(rows), eligibleLeafRows=sum(c not in parents for _, _, c, _ in rows), apiConnectionLimit=info['apiConnectionLimit'])
        print(json.dumps(dict(event='ready', **output)), flush=True)
        processing = time.perf_counter(); cursor = 0; next_progress = 100
        counts, batch_counts = collections.Counter(), collections.Counter()
        classes = {}
        durations = []
        while cursor < len(rows) and (not limit or cursor < limit) and not (private / 'stop-after-row').exists():
            batch_started = time.perf_counter()
            selected = [rows[cursor]]
            phase, _, code, row = selected[0]
            used = {code}
            if code not in parents:
                for candidate in rows[cursor+1:cursor+window]:
                    p, _, c, r = candidate
                    if c in parents or c in used or limit and cursor + len(selected) >= limit:
                        break
                    selected.append(candidate); used.add(c)
            jobs = []
            for phase, number, code, row in selected:
                path, body, adaptations = prepare_history(db, phase, code, row, latest[code])
                key = str(uuid.uuid5(uuid.NAMESPACE_URL, 'chnhcm-vanke127/history-v1/' + phase + '/' + str(number)))
                jobs.append(dict(phase=phase, row=number, code=code, path=path, body=body, key=key, adaptations=adaptations, previous=latest[code]))
            db.execute('INSERT INTO history_batch VALUES(1,?)', (json.dumps(jobs, ensure_ascii=False),)); db.commit()
            results = dispatch(db, jobs, clients)
            for (_, _, code, row), (status, result, ms) in zip(selected, results):
                latest[code] = acknowledged_record(latest[code], row, result)
                counts[result['outcome']] += 1; durations.append(ms)
            batch_counts[len(selected)] += 1; cursor += len(selected)
            label = 'serial-nonleaf' if selected[0][2] in parents else ('serial-leaf' if len(selected) == 1 else 'parallel-leaf')
            metric = classes.setdefault(label, dict(rows=0, seconds=0.0, batches=0))
            metric['rows'] += len(selected); metric['seconds'] += time.perf_counter()-batch_started; metric['batches'] += 1
            metric['rowsPerSecond'] = metric['rows']/metric['seconds']
            elapsed = time.perf_counter() - processing
            output.update(processed=cursor, pending=len(rows)-cursor, processingSeconds=round(elapsed, 3), rowsPerSecond=round(cursor/elapsed, 3), meanRequestMS=round(sum(durations)/len(durations), 3), outcomes=dict(counts), batches=dict(batch_counts), classes=classes, totalSeconds=round(time.perf_counter()-start, 3))
            if cursor >= next_progress:
                evidence.write_text(json.dumps(output, indent=2) + '\n'); print(json.dumps(output), flush=True)
                next_progress = cursor + 100
        output['complete'] = cursor == len(rows)
        evidence.write_text(json.dumps(output, indent=2) + '\n'); print(json.dumps(output), flush=True)
    finally:
        server.send_signal(signal.SIGTERM); server.wait(timeout=10)
        log.close(); db.close(); lock.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path); p.add_argument('source', type=Path); p.add_argument('evidence', type=Path)
    p.add_argument('--workers', type=int, default=2); p.add_argument('--limit', type=int, default=0)
    p.add_argument('--window', type=int, default=0)
    a = p.parse_args(); run(a.private, a.source, a.evidence, a.workers, a.limit, a.window)
