#!/usr/bin/env python3
"""Verify original synthetic facts and retain a private per-source fingerprint.
Comparisons include all decoded state, immutable sources, results and paths.
The public report contains counts and digests only, never the private facts.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import subprocess

from matrix import ROOT, digest


def owned_database(private):
    from frozen import owned_database as frozen_database
    frozen = frozen_database(private)
    if frozen:
        return frozen
    synthetic = (private / 'tree300-synthetic.json').exists()
    container = ('chnhcm-bu-' + hashlib.sha256((str(ROOT) + ':tree300-synthetic-d').encode()).hexdigest()[:12] + '-postgres-1') if synthetic else 'chnhcm-vanke127'
    labels = json.loads(subprocess.check_output(['docker', 'inspect', '--format', '{{json .Config.Labels}}', container], text=True))
    assert labels.get('io.chnhcm.temporary') == 'true'
    assert labels.get('io.chnhcm.profile') == 'tree300-synthetic-d' if synthetic else labels.get('io.chnhcm.task') == 'vanke127'
    return container, 'chnhcm' if synthetic else 'vanke127'


def physical_history(private):
    container, database = owned_database(private)
    tables = ['bu_history', 'department_history', 'bu_dictionary', 'department_dictionary', 'bu_shape', 'department_shape']
    statement = ' UNION ALL '.join("SELECT json_build_object('table','" + table + "','rows',count(*),'rowContentMD5',md5(coalesce(string_agg(fingerprint,'' ORDER BY fingerprint),''))) FROM (SELECT md5(to_jsonb(h)::text) fingerprint FROM organization." + table + ' h) items' for table in tables)
    raw = subprocess.check_output(['docker', 'exec', container, 'psql', '-U', 'postgres', '-d', database, '-X', '-Atq', '-v', 'ON_ERROR_STOP=1', '-c', statement], text=True)
    return sorted((json.loads(line) for line in raw.splitlines()), key=lambda row: row['table'])


def run(args):
    private = args.private.resolve()
    assert not private.is_relative_to(ROOT)
    assert not args.output.exists() and not args.snapshot.exists()
    assert args.snapshot.resolve().is_relative_to(private)
    assert args.dump.resolve().is_relative_to(private)
    fd = os.open(args.snapshot, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    os.close(fd)
    snapshot = sqlite3.connect(args.snapshot)
    snapshot.execute('CREATE TABLE sources(source TEXT PRIMARY KEY, digest TEXT NOT NULL)')
    baseline = sqlite3.connect(f'file:{args.baseline.resolve()}?mode=ro', uri=True) if args.baseline else None
    if baseline:
        assert args.baseline.resolve().is_relative_to(private)
    expected, identities, originals, original_revisions = {}, {}, {}, {}
    synthetic = (private / 'tree300-synthetic.json').exists()
    if synthetic:
        ledger = sqlite3.connect(f'file:{private / "tree300-synthetic.sqlite"}?mode=ro', uri=True)
        for code, key, raw_body, raw_result in ledger.execute('SELECT code,key,body,result FROM requests'):
            assert raw_result, 'unacknowledged creation remains'
            body, result = json.loads(raw_body), json.loads(raw_result)
            assert result['outcome'] == 'accepted'
            oid = result.get('departmentId') or result['buId']
            expected[result['sourceVersion']] = dict(code=code, key=key, body=body, result=result, id=oid)
            identities[oid] = result['sourceVersion']
            originals[oid] = body
            original_revisions[oid] = result
        ledger.close()
        assert len(expected) == 87344
    counts, original_seen, additional, preserved = {}, set(), 0, 0
    heads = {}
    raw_digest = hashlib.sha256()
    with args.dump.open() as stream:
        for line in stream:
            raw_digest.update(line.encode())
            item = json.loads(line)
            fact, kind = item['Fact'], item['Kind']
            source = fact['Operation']['Source']['ID']
            owner = (kind, fact['Object'])
            number = fact['Operation']['Source']['Number']
            if owner not in heads or number > heads[owner][0]:
                heads[owner] = (number, source)
            fingerprint = digest(item)
            snapshot.execute('INSERT INTO sources VALUES(?,?)', (source, fingerprint))
            counts[kind] = counts.get(kind, 0) + 1
            if baseline:
                old = baseline.execute('SELECT digest FROM sources WHERE source=?', (source,)).fetchone()
                if old:
                    assert old[0] == fingerprint, 'original decoded source was changed'
                    preserved += 1
                else:
                    additional += 1
            if synthetic and source in expected:
                want = expected[source]
                original_seen.add(source)
                body, result = want['body'], want['result']
                operation, timeline = fact['Operation'], fact['State']['Timeline']
                assert fact['Object'] == want['id'] and timeline['Code'] == want['code']
                assert timeline['Source'] == source and len(timeline['Records']) == 1
                record = timeline['Records'][0]
                data = record['Data']
                assert (record['Date'], record['End'], record['ID'], record['Revision']) == (body['effectiveDate'], '', result['targetRecordId'], result['targetRevision'])
                assert record['State'] == 'active' and record['Type'] == 'CREATE'
                assert data['Name'] == body['name']
                assert operation['Source']['Submission'] == want['key'] and operation['Source']['Number'] == 1
                assert operation['Source']['Previous'] == '' and operation['Source']['Action'] == 'CREATE'
                assert operation['Terminal']['ID'] == want['key'] and operation['Terminal']['Outcome'] == 'accepted'
                for key in ['submissionId', 'outcome', 'buId', 'departmentId', 'sourceVersion', 'targetRecordId', 'targetRevision', 'selectedRecordId', 'effectiveDate', 'action']:
                    assert operation['Terminal']['Result'].get(key) == result.get(key)
                if kind == 'department':
                    assert data['BU'] == body['managementBuId'] and data['Parent'] == (body['parentDepartmentId'] or '')
                    segments = fact['Evidence']['Segments']
                    assert len(segments) == 1
                    segment = segments[0]
                    assert (segment['Start'], segment['End'], segment['Mode'], segment['RecordID'], segment['Revision']) == (body['effectiveDate'], '', 'active', record['ID'], record['Revision'])
                    chain, cursor = [], want['id']
                    while cursor:
                        chain.append(cursor)
                        cursor = originals[cursor]['parentDepartmentId']
                    nodes = segment['Nodes']
                    assert len(nodes) == len(chain) + 1 and nodes[0]['Kind'] == 'enterprise'
                    for node, oid in zip(nodes[1:], reversed(chain)):
                        original = original_revisions[oid]
                        assert (node['Kind'], node['DepartmentID'], node['Source'], node['RecordID'], node['Revision'], node['Name']) == ('department', oid, identities[oid], original['targetRecordId'], original['targetRevision'], originals[oid]['name'])
    snapshot.commit()
    if synthetic:
        assert original_seen == set(expected)
    total = sum(counts.values())
    if baseline:
        assert preserved == baseline.execute('SELECT count(*) FROM sources').fetchone()[0]
        assert additional == args.expected_additional
        baseline.close()
    elif synthetic:
        assert total == len(expected)
    container, database = owned_database(private)
    statement = "SELECT jsonb_build_array('BU',bu_id::text,head_source_id::text) FROM organization.business_unit UNION ALL SELECT jsonb_build_array('department',department_id::text,head_source_id::text) FROM organization.department"
    raw_heads = subprocess.check_output(['docker', 'exec', container, 'psql', '-U', 'postgres', '-d', database, '-X', '-Atq', '-v', 'ON_ERROR_STOP=1', '-c', statement], text=True)
    observed_heads = {}
    for line in raw_heads.splitlines():
        kind, oid, source = json.loads(line)
        assert (kind, oid) not in observed_heads
        observed_heads[kind, oid] = source
    assert observed_heads == {key: value[1] for key, value in heads.items()}, 'current owner head differs from latest preserved source'
    aggregate = hashlib.sha256()
    for source, fingerprint in snapshot.execute('SELECT source,digest FROM sources ORDER BY source'):
        aggregate.update((source + ':' + fingerprint + '\n').encode())
    snapshot.close()
    physical = physical_history(private)
    if args.storage_baseline:
        previous_physical = sorted(json.loads(args.storage_baseline.read_text())['physicalHistory'], key=lambda row: row['table'])
        assert physical == previous_physical, 'read-only interval changed stored history/dictionary/shape content'
    result = dict(completed=True, passed=True, counts=counts, totalSources=total,
                  originalSyntheticSourcesVerified=len(original_seen), preservedSources=preserved,
                  additionalSources=additional, currentHeadsVerified=len(observed_heads), sourceFingerprintSHA256=aggregate.hexdigest(),
                  dumpSHA256=raw_digest.hexdigest(), comparisonIncludes='all decoded state, operation, terminal result and frozen path fields')
    result['physicalHistory'] = physical
    result['readOnlyStorageUnchanged'] = True if args.storage_baseline else None
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result), flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path)
    parser.add_argument('dump', type=Path)
    parser.add_argument('snapshot', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--storage-baseline', type=Path, help='earlier public checksum report for a read-only interval')
    parser.add_argument('--expected-additional', type=int, default=0)
    run(parser.parse_args())
