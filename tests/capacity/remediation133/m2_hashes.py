#!/usr/bin/env python3
"""Install identical-format hash algorithms in an owned M2 comparison copy.

Not an upgrade route: normal current-model environments initialize migrations.
The frozen original snapshots and fixed retained runtimes are never changed.
"""
import argparse
import hashlib
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'tests/capacity/storage130'))
from growth import resource, sql


def run(args):
    assert not args.output.exists()
    assert args.private.parent.name == 'remediation133-m2-20261112'
    name = resource(args.private)
    assert sql(name, 'SELECT contract FROM organization.storage_contract') == 'm02-compact-5'
    source = (ROOT/'db/migrations/00002_organization.sql').read_text()
    statements = []
    for function in ['history_parts_hash', 'history_dictionary_hash']:
        match = re.search(r'CREATE FUNCTION organization\.'+function+r'\(.*?\$\$;', source, re.S)
        assert match is not None
        statements.append(match.group().replace('CREATE FUNCTION', 'CREATE OR REPLACE FUNCTION', 1))
    sql(name, 'BEGIN;\n'+'\n'.join(statements)+'\nCOMMIT;')
    checks = {}
    for kind in ['bu', 'department']:
        for table, expression in [('dictionary', 'organization.history_dictionary_hash(words,binaries,ranges)'),
                                  ('shape', 'organization.history_parts_hash(parts)')]:
            checks[kind+'_'+table] = json.loads(sql(name, f'SELECT json_build_object(\'rows\',count(*),\'mismatches\',count(*) FILTER(WHERE digest IS DISTINCT FROM {expression})) FROM organization.{kind}_{table}'))
    result = dict(passed=all(v['mismatches'] == 0 for v in checks.values()), model='m02-compact-5',
                  sameDigestFormat=True, checks=checks,
                  migrationSHA256=hashlib.sha256(source.encode()).hexdigest())
    args.output.write_text(json.dumps(result, indent=2)+'\n')
    assert result['passed'], 'retained digest differs; keep owned case and first failure'
    print(json.dumps(result))


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    run(p.parse_args())
