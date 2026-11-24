"""Read-only resolution of original accepted receipts after the owned restart.
The separate history fingerprint pass covers every source; HTTP sampling does
not claim to resolve every imported submission.
"""
import json
import sqlite3

from matrix import MeasuredClient


def resolve_originals(private, base, journal=None):
    expected = []
    synthetic = (private / 'tree300-synthetic.json').exists()
    if synthetic:
        assert journal and journal.resolve().is_relative_to(private)
        for line in journal.read_text().splitlines():
            row = json.loads(line)
            if row.get('status') == 200 and row['response'].get('outcome') == 'accepted':
                expected.append(('departments', row['response']))
        assert len(expected) == 4, 'both changes and their restorations must be acknowledged'
    else:
        db = sqlite3.connect(f'file:{private / "import.sqlite"}?mode=ro', uri=True)
        for phase in ['create', 'version', 'parent-change']:
            rows = [json.loads(row[0]) for row in db.execute('SELECT result FROM rows WHERE phase=? ORDER BY row_no', (phase,))]
            accepted = [row['response'] for row in rows if row['outcome'] == 'accepted']
            assert len(accepted) >= 4
            for row in accepted[:2] + accepted[-2:]:
                expected.append(('departments' if row.get('departmentId') else 'organization', row))
        if db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='tree300_resubmission'").fetchone():
            for (raw,) in db.execute('SELECT closed_result FROM tree300_resubmission ORDER BY phase,row_no'):
                closed = json.loads(raw)
                assert closed['outcome'] == 'not_applied'
                expected.append(('departments', closed))
        closed_file = private / 'tree300-final-closed-receipts.jsonl'
        if closed_file.exists():
            known = {row['submissionId'] for _, row in expected}
            for line in closed_file.read_text().splitlines():
                closed = json.loads(line)
                assert closed['outcome'] == 'not_applied'
                if closed['submissionId'] not in known:
                    expected.append(('departments', closed))
                    known.add(closed['submissionId'])
        db.close()
    client = MeasuredClient(base)
    status, session, _ = client.call('POST', '/api/v1/session/login', dict(login='admin', password='admin1234567'))
    assert status == 200
    client.csrf = session['csrf']
    samples = []
    for kind, original in expected:
        sample = dict(passed=False)
        try:
            status, value, ms = client.call('POST', '/api/v1/' + kind + '/submissions/' + original['submissionId'] + '/resolve')
            sample.update(status=status, ms=ms, capacityRequest=client.capacity_request)
            assert status == 200
            assert all(value.get(key) == wanted for key, wanted in original.items() if key != 'currentChanged')
            sample['passed'] = True
        except Exception as error:
            sample['error'] = type(error).__name__
        samples.append(sample)
    current = []
    if synthetic:
        nodes = {node['id']: node for node in json.loads((private / 'tree300-synthetic.json').read_text())['nodes']}
        latest = {value['departmentId']: value for kind, value in expected}
        for oid, want in latest.items():
            status, detail, ms = client.call('GET', '/api/v1/departments/' + oid)
            sample = dict(status=status, ms=ms, passed=False)
            if status == 200:
                record = detail['selectedRecord']
                parent = record.get('parentDepartment')
                sample['passed'] = (detail['headSourceVersion'] == want['sourceVersion'] and record['recordId'] == want['targetRecordId'] and record['revision'] == want['targetRevision'] and record['name'] == nodes[oid]['name'] and (parent['id'] if parent else None) == nodes[oid]['parentId'])
            current.append(sample)
    return dict(planned=len(expected), attempts=len(samples), passed=all(row['passed'] for row in samples + current), samples=samples, currentRestorations=current,
                scope='four mixed-write receipts' if synthetic else 'twelve original workbook receipts across three phases plus confirmed closed receipts',
                writesPerformed=0)
