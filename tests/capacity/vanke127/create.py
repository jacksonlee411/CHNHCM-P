#!/usr/bin/env python3
"""Bounded concurrent creation; each depth waits for all of its parents."""
import argparse
import collections
import concurrent.futures
import json
from pathlib import Path
import signal
import subprocess
import time
import uuid

from load import authenticated_clients, source_db
from runtime import ROOT, BIN


def run(private, source, out, workers, limit):
    assert 1 <= workers <= 128
    db = source_db(private, source)
    log = (private / 'server.log').open('a')
    server = subprocess.Popen([str(BIN), '-mode', 'serve-import', '-private', str(private)], cwd=ROOT, stdout=subprocess.PIPE, stderr=log, text=True)
    info = json.loads(server.stdout.readline())
    start, count, durations = time.perf_counter(), 0, []
    total_start = start
    try:
        clients = authenticated_clients(info['url'], workers)
        start = time.perf_counter()
        pending = db.execute("SELECT row_no,code,payload,priority FROM rows WHERE phase='create' AND result IS NULL ORDER BY priority,code").fetchall()
        initial = len(pending)
        objects = {row[0]: row[1:3] for row in db.execute('SELECT code,object_id,kind FROM objects')}
        levels = collections.defaultdict(list)
        for row in pending:
            levels[row[3]].append(row)
        with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
            for depth, rows in levels.items():
                for offset in range(0, len(rows), workers):
                    if (private / 'stop-after-row').exists() or limit and count >= limit:
                        break
                    jobs = []
                    for index, (row_no, code, payload, _) in enumerate(rows[offset:offset+workers]):
                        row = json.loads(payload)
                        kind = 'business-units' if row.get('is_business_unit') == 'true' else 'departments'
                        path = '/api/v1/' + kind
                        body = dict(code=code, name=row['name'], effectiveDate=row['effective_date'], operationType='CREATE', operationReason='NEW_ORGANIZATION')
                        if kind == 'departments':
                            root = next(v[0] for v in objects.values() if v[1] == 'business-units')
                            parent = objects[row['parent_org_code']]
                            body.update(managementBuId=root, parentDepartmentId=parent[0] if parent[1] == 'departments' else None)
                        key = str(uuid.uuid5(uuid.NAMESPACE_URL, 'chnhcm-vanke127/history-v1/create/' + str(row_no)))
                        attempt = db.execute("SELECT coalesce(max(attempt),0)+1 FROM attempts WHERE phase='create' AND row_no=?", (row_no,)).fetchone()[0]
                        db.execute("INSERT INTO attempts VALUES('create',?,?,?,?,?,?,NULL,NULL,NULL)", (row_no, attempt, 'POST', path, json.dumps(body, ensure_ascii=False), key))
                        jobs.append((index, path, body, row_no, code, kind, key, attempt))
                    # Persist every exact request before sending any command.
                    # One durable transaction per batch avoids two fsyncs per row.
                    db.commit()
                    jobs = [(pool.submit(clients[index].call, 'POST', path, body, key), row_no, code, kind, key, attempt)
                            for index, path, body, row_no, code, kind, key, attempt in jobs]
                    failures = []
                    for future, row_no, code, kind, key, attempt in jobs:
                        status, result, ms = future.result()
                        db.execute("UPDATE attempts SET status=?,response=?,ms=? WHERE phase='create' AND row_no=? AND attempt=?", (status, json.dumps(result, ensure_ascii=False), ms, row_no, attempt))
                        if status != 201:
                            failures.append(dict(row=row_no, status=status, code=result.get('code'), field=result.get('field')))
                        else:
                            stored = dict(status=status, outcome=result['outcome'], response=result, adaptations=[], requestKey=key)
                            db.execute("UPDATE rows SET result=? WHERE phase='create' AND row_no=?", (json.dumps(stored, ensure_ascii=False), row_no))
                            oid = result.get('buId') if kind == 'business-units' else result.get('departmentId')
                            assert oid and result['outcome'] == 'accepted'
                            db.execute('INSERT INTO objects VALUES(?,?,?,?)', (code, oid, kind, result['sourceVersion']))
                            objects[code] = (oid, kind)
                            count += 1
                            durations.append(ms)
                    # Receipts and the object directory advance atomically. If
                    # interrupted before this commit, deterministic keys replay
                    # the same requests against the acknowledged parent layer.
                    db.commit()
                    if failures:
                        print(json.dumps({'firstFailures': failures}), flush=True)
                        raise RuntimeError('failed rows retained for diagnosis')
                    if count % 100 < workers:
                        progress = dict(processed=count, pending=initial-count, depth=depth, workers=workers, apiConnectionLimit=info['apiConnectionLimit'], elapsedSeconds=round(time.perf_counter()-start,2), meanRequestMS=round(sum(durations)/len(durations),2), rowsPerSecond=round(count/(time.perf_counter()-start),2))
                        out.write_text(json.dumps(progress, indent=2)+'\n')
                        print(json.dumps(progress), flush=True)
                if (private / 'stop-after-row').exists() or limit and count >= limit:
                    break
        result = dict(processed=count, pending=initial-count, workers=workers, sessions=len(clients), apiConnectionLimit=info['apiConnectionLimit'], elapsedSeconds=round(time.perf_counter()-start,2), totalSeconds=round(time.perf_counter()-total_start,2), meanRequestMS=round(sum(durations)/len(durations),2) if durations else None, rowsPerSecond=round(count/(time.perf_counter()-start),2), completed=count==initial)
        out.write_text(json.dumps(result, indent=2)+'\n')
        print(json.dumps(result), flush=True)
    finally:
        server.send_signal(signal.SIGTERM)
        server.wait(timeout=10)
        log.close()
        db.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('source', type=Path)
    p.add_argument('out', type=Path)
    p.add_argument('--workers', type=int, default=4)
    p.add_argument('--limit', type=int, default=0)
    a = p.parse_args()
    run(a.private, a.source, a.out, a.workers, a.limit)
