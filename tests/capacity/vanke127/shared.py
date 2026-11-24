#!/usr/bin/env python3
"""Verify the external V127 workbooks and prepare an external import ledger."""
import argparse
import hashlib
import json
from pathlib import Path

from source import read_rows
from load import source_db

ROOT = Path(__file__).resolve().parents[3]



def verify(source):
    source = source.resolve()
    if source.is_relative_to(ROOT):
        raise ValueError('source directory must be outside checkout')
    manifest = json.loads((source / 'manifest.json').read_text())
    counts = {}
    for item in manifest['files']:
        path = source / item['file']
        assert path.parent == source and path.suffix == '.xlsx'
        raw = path.read_bytes()
        assert len(raw) == item['bytes'], 'source size mismatch'
        assert hashlib.sha256(raw).hexdigest() == item['sha256'], 'source digest mismatch'
        rows = read_rows(path)
        assert len(rows) == item['rows'], 'source row count mismatch'
        assert sorted({k for row in rows for k in row}) == item['columns'], 'source column mismatch'
        assert all(not row.get('manager_pernr') and not row.get('core.manager_pernr') for row in rows)
        counts[item['phase']] = len(rows)
    assert counts == {'create': 87344, 'version': 11741, 'parent-change': 33217}
    return dict(dataset=manifest['id'], rows=sum(counts.values()), phases=counts,
                allFileDigestsMatched=True, allColumnsMatched=True, personnelFieldsEmpty=True)


def prepare(private, source):
    private = private.resolve()
    assert private.is_absolute() and not private.is_relative_to(ROOT), 'private directory must be outside checkout'
    assert private.is_dir() and private.stat().st_mode & 0o777 == 0o700, 'private directory must have mode 0700'
    assert not (private / 'import.sqlite').exists() and not (private / 'source-rows.jsonl').exists(), 'existing import ledger retained; resume it with V127 tools'
    result = verify(source)
    db = source_db(private, source.resolve())
    try:
        # Keep original phase/Excel row and all fields. Product result identities
        # are created by this host's Go/API, never copied from a running database.
        path = private / 'source-rows.jsonl'
        with path.open('x', encoding='utf-8') as output:
            path.chmod(0o600)
            for phase, number, payload in db.execute('SELECT phase,row_no,payload FROM rows ORDER BY phase,row_no'):
                output.write(json.dumps(dict(Phase=phase, Row=number, Payload=json.loads(payload), Result=None), ensure_ascii=False) + '\n')
        assert db.execute('SELECT count(*) FROM rows WHERE result IS NOT NULL').fetchone()[0] == 0
        result['preparedRows'] = db.execute('SELECT count(*) FROM rows').fetchone()[0]
        result['productWrites'] = 0
        return result
    finally:
        db.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['verify', 'prepare'])
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--private', type=Path)
    args = parser.parse_args()
    if args.mode == 'prepare' and args.private is None:
        parser.error('prepare requires --private')
    print(json.dumps(prepare(args.private, args.source) if args.mode == 'prepare' else verify(args.source), ensure_ascii=False))
