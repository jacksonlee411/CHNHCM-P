#!/usr/bin/env python3
"""Re-submit one explicitly identified, confirmed-not-applied import command.
Never replace an ambiguous identity. Preserve the closed receipt, original job,
new identity and exact body before sending through the ordinary product API.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import sqlite3
import subprocess
import sys
import uuid

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/vanke127'))
from load import Client
from parallel_history import dispatch
from runtime import owned_resource


def run(args):
    private = args.private.resolve()
    assert not private.is_relative_to(ROOT) and not args.output.exists()
    container, database = owned_resource(private)
    db = sqlite3.connect(private / 'import.sqlite')
    raw = db.execute('SELECT jobs FROM history_batch WHERE id=1').fetchone()[0]
    jobs = json.loads(raw)
    assert len(jobs) == 1 and jobs[0]['phase'] == args.phase and jobs[0]['row'] == args.row
    job = jobs[0]
    def count():
        return int(subprocess.check_output(['docker', 'exec', container, 'psql', '-U', 'postgres', '-d', database, '-X', '-Atq', '-c', 'SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history'], text=True))
    report = dict(completed=False, passed=False, phase=args.phase, row=args.row,
                  originalJournalSHA256=hashlib.sha256(raw.encode()).hexdigest(), sourcesBefore=count())
    log_path = private / (args.output.stem + '-server.log')
    with os.fdopen(os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as log:
        process = subprocess.Popen([str(args.binary.resolve()), '-mode', 'serve-import', '-private', str(private)], cwd=ROOT,
                                   env=dict(os.environ, VANKE127_STAGES='1'), stdout=subprocess.PIPE, stderr=log, text=True)
        try:
            info = json.loads(process.stdout.readline())
            client = Client(info['url'])
            status, login, _ = client.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
            assert status == 200
            client.csrf = login['csrf']
            status, closed, ms = client.call('POST', '/api/v1/departments/submissions/' + job['key'] + '/resolve')
            assert status == 200 and closed.get('outcome') == 'not_applied' and closed.get('submissionId') == job['key']
            report['closedReceiptConfirmed'] = dict(status=status, outcome='not_applied', ms=ms)
            route = '/'.join(job['path'].split('/')[:5])
            status, current, ms = client.call('GET', route)
            assert status == 200 and current['headSourceVersion'] == job['body']['expectedSourceVersion']
            report['originalHeadStillCurrent'] = True
            replacement = str(uuid.uuid5(uuid.NAMESPACE_URL, job['key'] + '/tree300-confirmed-not-applied'))
            db.execute('CREATE TABLE IF NOT EXISTS tree300_resubmission(phase TEXT,row_no INTEGER,closed_key TEXT PRIMARY KEY,new_key TEXT UNIQUE,original_job TEXT,closed_result TEXT)')
            db.execute('INSERT INTO tree300_resubmission VALUES(?,?,?,?,?,?)', (job['phase'], job['row'], job['key'], replacement, json.dumps(job, ensure_ascii=False), json.dumps(closed, ensure_ascii=False)))
            job['key'] = replacement
            db.execute('UPDATE history_batch SET jobs=? WHERE id=1', (json.dumps(jobs, ensure_ascii=False),))
            db.commit()  # Closed proof and replacement are durable before send.
            values = dispatch(db, jobs, [client], parallel=False)
            report['results'] = [dict(status=status, outcome=value.get('outcome'), ms=ms, identityMatches=value.get('submissionId') == replacement) for status, value, ms in values]
            report['sourcesAfter'] = count()
            assert report['sourcesAfter'] - report['sourcesBefore'] == sum(value['outcome'] == 'accepted' for _, value, _ in values)
            assert db.execute('SELECT count(*) FROM history_batch').fetchone()[0] == 0
            report['passed'] = True
        except Exception as error:
            report['error'] = type(error).__name__
            report['sourcesAfter'] = count()
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                process.wait(timeout=15)
            db.close()
            report['completed'] = True
            args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report), flush=True)
    if not report['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path)
    parser.add_argument('binary', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--phase', choices=['version', 'parent-change'], required=True)
    parser.add_argument('--row', type=int, required=True)
    run(parser.parse_args())
