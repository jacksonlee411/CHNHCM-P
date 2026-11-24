"""Explicitly owned model-5 fixture imported from the complete 125 input.

This adapter changes fixture preparation only. It never opens a retained
development database or derives expected membership from product queries.
"""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]


def owned_database(private, container=None):
    marker = private / 'tree300-frozen.json'
    if not marker.exists():
        return None
    assert private.resolve().is_relative_to(Path.home() / '.local/state/chnhcm-tasks')
    assert not private.resolve().is_relative_to(ROOT)
    assert private.stat().st_mode & 0o777 == 0o700
    assert marker.stat().st_mode & 0o777 == 0o600
    owner = json.loads(marker.read_text())
    assert owner['task'] in ('tree300-closeout', 'tree300-frozen-synthetic') and owner['completeInputVerified']
    assert owner['model'] == 'm02-compact-5'
    synthetic = owner['task'] == 'tree300-frozen-synthetic'
    expected_container = 'chnhcm-storage126-product' if synthetic else 'chnhcm-storage130-b'
    assert owner['container'] == expected_container
    if synthetic:
        assert owner['decodedSources'] == 87344 and len(owner['sourceFactsSHA256']) == 64
    assert container in (None, owner['container'])
    actual = json.loads(subprocess.check_output([
        'docker', 'inspect', '--format',
        '{"id":{{json .Id}},"labels":{{json .Config.Labels}}}', owner['container']
    ], text=True))
    assert actual['id'] == owner['containerID']
    assert all(actual['labels'].get(key) == value for key, value in {
        'io.chnhcm.task': 'storage126-product' if synthetic else 'storage130',
        'io.chnhcm.parent-task': 'storage130',
        'io.chnhcm.temporary': 'true',
    }.items())
    return owner['container'], 'storage126'


def workbook_oracle(private):
    sys.path.insert(0, str(ROOT / 'tests/capacity/vanke127'))
    from verify import Oracle
    return Oracle(private)


def nodes(private, day):
    owner = json.loads((private / 'tree300-frozen.json').read_text())
    expected_digest = owner.get('preparedOracles', {}).get(day)
    if owner.get('oracleDate') == day:
        assert expected_digest in (None, owner['oracleSHA256'])
        expected_digest = owner['oracleSHA256']
    if expected_digest:
        # The full workbook oracle was evaluated before the measured phase.
        # Keep its immutable graph without retaining two complete histories
        # in the orchestration and request-generator processes during timing.
        path = private / ('tree300-expected-' + day + '.json')
        raw = path.read_bytes()
        assert path.stat().st_mode & 0o777 == 0o600
        assert hashlib.sha256(raw).hexdigest() == expected_digest
        value = json.loads(raw)
        assert value['businessAt'] == day and len(value['nodes']) == 87343
        return value['nodes']
    assert owner['task'] != 'tree300-frozen-synthetic', 'prepare the synthetic graph for the new business day before timing'
    source = workbook_oracle(private)
    result = []
    for code, obj in source.objects.items():
        if obj['kind'] != 'departments':
            continue
        record = source.record_at(source.frames[source.heads[code]], day)
        parent = source.objects[record[2]]
        result.append(dict(id=obj['id'], code=code, name=record[1],
                           parentId=parent['id'] if parent['kind'] == 'departments' else None))
    assert len(result) == 87343
    return result
