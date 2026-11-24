#!/usr/bin/env python3
"""First unsearched parent opening from normal menus on complete owned inputs.

Run before growth, separately from formal timing. Expected pages come from the
retained source oracle. Browser errors and business labels stay outside the repo.
"""
import argparse
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
from zoneinfo import ZoneInfo

from lists import ROOT, expected, module
from menus import MenuLane, WebServer, case
from measure import save


def require_original_counts(shape, counts):
    bus, departments, sources = (1, 87343, 132290) if shape == 'vanke' else (2023, 22104, 1104206)
    assert counts['BU'] == bus and counts['department'] == departments, 'complete original shape required'
    assert counts['buSources'] + counts['departmentSources'] == sources, 'run before growth or mutable125'


def synthetic_fixture(rows, oracle, day):
    fan = sorted(code for code in rows if code.startswith('R6_FAN_'))
    assert len(fan) == 2001, 'complete fanout source required'
    bu = rows[fan[0]]['managementBu']
    assert all(rows[code]['managementBu']['id'] == bu['id'] for code in fan)
    candidates = []
    for code, row in rows.items():
        if row['managementBu']['id'] != bu['id']:
            continue
        timeline = oracle.frame(code)['State']['Timeline']['Records']
        assert timeline and all(r['State'] == 'active' and r['Data']['BU'] == bu['id'] for r in timeline), 'candidate has a future or historical eligibility change'
        if timeline[0]['Date'] <= day:
            candidates.append(dict(id=row['id'], code=code, name=row['name'], eligibility='eligible'))
    candidates.sort(key=lambda row: (row['code'], row['id']))
    assert len(candidates) > 20, 'independent second page required'
    return dict(buCode=bu['code'], count=len(candidates), pages=[candidates[:20], candidates[20:40]])


def fixture(shape, private, day):
    rows, _ = expected(shape, private, day)
    assert len(rows) == (87343 if shape == 'vanke' else 22104)
    if shape == 'synthetic':
        return synthetic_fixture(rows, case.Oracle(private), day)
    oracle = module('parent_menu_vanke_oracle', ROOT/'tests/capacity/vanke127/verify.py').Oracle(private)
    values = [dict(id=row['id'], code=row['code'], name=row['name'], eligibility='eligible') for row in rows.values()]
    values.sort(key=lambda row: (row['code'], row['id']))
    return dict(buCode=oracle.root, count=len(values), pages=[values[:20], values[20:40]])


def run(shape, private, output, binary):
    private, binary, output = private.resolve(), binary.resolve(), output.resolve()
    assert ROOT not in private.parents and private.stat().st_mode & 0o777 == 0o700
    assert ROOT not in output.parents and not output.exists(), 'retain first private results'
    for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
        os.environ.pop(key, None)
    case.BIN = binary
    day = datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')
    lane = MenuLane()
    lane.private, lane.output = private, output
    lane.server, lane.clients = WebServer(private, private), {}
    browser_file = Path(__file__).with_suffix('.mjs')
    lane.result = dict(completed=False, passed=False, cleanupSucceeded=False, shape=shape,
        businessAt=day, binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),
        driverSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        browserSHA256=hashlib.sha256(browser_file.read_bytes()).hexdigest(),
        coldResets=[], entries=[], capacityAcceptance=False,
        scope='admin first unsearched parent opening via normal menus; two complete independent pages and unsaved exit; reader disabled-create boundary belongs to dual-role normal-menu lane',
        cacheScope='owned runtime reset before cold menu journey; login/list/BU lookup precede parent opening, so this is not an independently cold parent API sample; warm is the second scheduled journey; no P95 claim')
    save(output, lane.result)
    closed = False
    try:
        lane.result['before'] = case.cli('size', private)
        require_original_counts(shape, lane.result['before']['counts'])
        names = ['snapshot.json', 'organization.enterprise.jsonl.gz'] + (
            ['owners.json', 'bu-prototype.json', 'dep-prototype.json', 'bu-typed.jsonl.gz', 'dep-typed.jsonl.gz']
            if shape == 'synthetic' else ['import.sqlite', 'history-order.json'])
        def input_hashes():
            values = {}
            for name in names:
                with (private/name).open('rb') as stream:
                    values[name] = hashlib.file_digest(stream, 'sha256').hexdigest()
            return values
        lane.result['oracleInputSHA256'] = input_hashes()
        expected_input = fixture(shape, private, day)
        lane.result['expectedCandidateCount'] = expected_input['count']
        lane.server.start()
        assert lane.server.info['model'] == 'm02-compact-5' and lane.server.info['apiConnectionLimit'] == 12
        lane.cold(shape+'-first-parent-normal-menu')
        lane.result['startup'] = lane.server.info
        for cache in ['cold', 'warm']:
            assert datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d') == day, 'do not rebind across business dates'
            request = dict(url=lane.server.base, day=day, **expected_input)
            process = subprocess.run(['node', str(browser_file)], input=json.dumps(request), text=True, capture_output=True, cwd=ROOT)
            private_error = private/f'{output.stem}-{cache}-failure.txt'
            with os.fdopen(os.open(private_error, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as stream:
                stream.write(process.stderr)
            value = json.loads(process.stdout) if process.stdout.strip() else dict(passed=False, resultMissing=True)
            lane.result['entries'].append(dict(cache=cache, exitCode=process.returncode, result=value))
            save(output, lane.result)
        assert input_hashes() == lane.result['oracleInputSHA256'], 'original oracle changed during menu verification'
        lane.result['completed'] = True
    except Exception as error:
        lane.result['failure'] = dict(errorType=type(error).__name__, scope='original failure retained; no retry')
        raise
    finally:
        try:
            lane.server.close()
            closed = True
        finally:
            lane.result['cleanupSucceeded'] = closed
            try:
                lane.result['after'] = case.cli('size', private)
                if 'before' in lane.result:
                    lane.result['businessSourceGrowth'] = {key: lane.result['after']['counts'][key] - lane.result['before']['counts'][key] for key in ['buSources', 'departmentSources']}
                    assert not any(lane.result['businessSourceGrowth'].values()), 'menu journey wrote business sources'
                lane.result['passed'] = closed and lane.result['completed'] and len(lane.result['entries']) == 2 and all(entry['exitCode'] == 0 and entry['result'].get('passed') is True for entry in lane.result['entries'])
            finally:
                save(output, lane.result)
    if not lane.result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('shape', choices=['synthetic', 'vanke'])
    parser.add_argument('private', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    run(args.shape, args.private, args.output, args.binary)
