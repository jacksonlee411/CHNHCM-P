#!/usr/bin/env python3
"""Real broad-dependency writes on an explicitly disposable complete fixture.

Journal every request before sending. Failed/ambiguous writes are resolved and
stop the run; a new identity never conceals a failure. Successful corrections
are restored with another ordinary command, preserving both history sources.
"""
import argparse
import json
import os
from pathlib import Path
import sqlite3
import sys
import uuid

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/capacity/tree300'))
from measure import server
from matrix import MeasuredClient
sys.path.insert(0, str(ROOT / 'tests/capacity/storage130'))
from growth import resource, sql


def run(args):
    assert args.pairs > 0, 'at least one change/restore pair is required'
    private = args.private.resolve()
    assert not private.is_relative_to(ROOT) and private.stat().st_mode & 0o777 == 0o700
    assert not args.output.exists()
    container = resource(private)
    ledger = sqlite3.connect('file:' + str(private / 'import.sqlite') + '?mode=ro', uri=True)
    target = ledger.execute('SELECT object_id FROM objects WHERE code=?', (args.code,)).fetchone()[0]
    ledger.close()
    # Count all dated parent identities independently from the API. This is
    # shape evidence, not a replacement for Go qualification or lock discovery.
    rows = sql(container, "SELECT json_build_array(department_id::text,parents::text[]) FROM organization.department").splitlines()
    children = {}
    for raw in rows:
        identity, parents = json.loads(raw)
        for parent in parents:
            children.setdefault(parent, []).append(identity)
    seen, queue = {target}, [target]
    for identity in queue:
        for child in children.get(identity, []):
            if child not in seen:
                seen.add(child)
                queue.append(child)
    del rows, children
    report = dict(completed=False, passed=False, diagnostic=args.diagnostic,
                  totalDepartments=len(sql(container, 'SELECT department_id FROM organization.department').splitlines()),
                  datedDescendantsIncludingTarget=len(seen), attempts=[], readbacks=[])
    resume = None
    if args.retry_confirmed:
        prior = json.loads(args.retry_confirmed.read_text())
        assert len(prior['attempts']) == 1 and not prior['passed']
        assert prior['attempts'][0]['recovery']['outcome'] == 'not_applied'
        entries = [json.loads(line) for line in (private / (args.retry_confirmed.stem + '-requests.jsonl')).read_text().splitlines()]
        resume = next(entry for entry in entries if 'body' in entry)
        report['retryAfterConfirmedNotApplied'] = args.retry_confirmed.name
    def save():
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        args.output.chmod(0o600)
    journal_path = private / (args.output.stem + '-requests.jsonl')
    with os.fdopen(os.open(journal_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as journal:
        def record(value):
            journal.write(json.dumps(value, ensure_ascii=False) + '\n')
            journal.flush()
            os.fsync(journal.fileno())
        with server(private, args.binary.resolve(), 'frozen', args.output.stem + '-server', args.diagnostic) as (_, info, _):
            report['apiConnectionLimit'] = info['apiConnectionLimit']
            report['runtimeBudgets'] = dict(departmentPreparationMS=info.get('departmentPreparationBudgetMS', 2000),
                                           cumulativeLockAcquisitionMS=info.get('departmentLockAcquisitionBudgetMS', 2000))
            c = MeasuredClient(info['url'])
            status, login, _ = c.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
            assert status == 200
            c.csrf = login['csrf']
            route = '/api/v1/departments/' + target
            status, original, _ = c.call('GET', route)
            assert status == 200
            original_parent = original['selectedRecord'].get('parentDepartment')
            assert original_parent, 'representative target must have a current parent'
            parent = original_parent['id']
            receipts = []
            for index in range(args.pairs * 2):
                status, detail, _ = c.call('GET', route)
                assert status == 200
                selected = detail['selectedRecord']
                endpoint = route + '/records/' + selected['recordId'] + '/correct'
                destination = None if index % 2 == 0 else parent
                body = dict(expectedSourceVersion=detail['headSourceVersion'], expectedRecordRevision=selected['revision'],
                            changes=dict(parentDepartmentId=destination))
                key = str(uuid.uuid4())
                if index == 0 and resume:
                    assert endpoint == resume['path'] and body == resume['body'], 'original request no longer matches; do not replace it'
                    # Resolving not_applied closes the old identity. A fresh
                    # submission is permitted only after that definitive proof;
                    # retain its link instead of hiding the original failure.
                    record(dict(replacesClosedSubmission=resume['key'], newSubmission=key, provenOutcome='not_applied'))
                record(dict(index=index, path=endpoint, body=body, key=key))
                status, reply, ms = c.call('POST', endpoint, body, key)
                record(dict(index=index, status=status, response=reply, key=key))
                passed = status == 200 and reply.get('outcome') == 'accepted'
                metric = dict(index=index, phase='change' if index % 2 == 0 else 'restore', status=status, ms=ms,
                              passed=passed, code=reply.get('code'), capacityRequest=c.capacity_request)
                report['attempts'].append(metric)
                save()
                if not passed:
                    rs, resolved, r_ms = c.call('POST', '/api/v1/departments/submissions/' + key + '/resolve')
                    record(dict(key=key, resolveStatus=rs, response=resolved))
                    metric['recovery'] = dict(status=rs, outcome=resolved.get('outcome'), ms=r_ms,
                                              identityMatches=resolved.get('submissionId') == key)
                    break
                assert reply['submissionId'] == key
                receipts.append(reply)
                status, current, _ = c.call('GET', route)
                actual = current['selectedRecord'].get('parentDepartment')
                assert status == 200 and (actual['id'] if actual else None) == destination
                assert current['headSourceVersion'] == reply['sourceVersion']
                report['readbacks'].append(dict(index=index, currentParentMatches=True, sourceMatches=True))
                # Original source remains readable while current membership changes.
                status, historical, _ = c.call('GET', route + '?sourceVersion=' + original['headSourceVersion'])
                assert status == 200 and historical['readOnly']
                assert historical['selectedRecord']['parentDepartment']['id'] == parent
            report['completed'] = True
            report['passed'] = len(report['attempts']) == args.pairs * 2 and all(x['passed'] for x in report['attempts'])
            save()
            record(dict(acceptedReceipts=receipts))
    print(json.dumps(dict(completed=report['completed'], passed=report['passed'], diagnostic=report['diagnostic'],
                          attempts=len(report['attempts']), failures=sum(not item['passed'] for item in report['attempts']),
                          readbacks=len(report['readbacks']))), flush=True)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('binary', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--code', required=True)
    p.add_argument('--pairs', type=int, default=1)
    p.add_argument('--diagnostic', action='store_true')
    p.add_argument('--retry-confirmed', type=Path, help='resubmit the unchanged body with a new identity only after the original was closed as not_applied')
    run(p.parse_args())
