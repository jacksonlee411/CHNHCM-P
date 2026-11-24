#!/usr/bin/env python3
"""Partition the actual Go integration test roots; each runner stays sequential."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess

from ci_shards import FULL_INTEGRATION, SECONDS, balanced_groups

PACKAGES = ('./tests/integration', './internal/bootstrap')
PREFLIGHT_ROOTS = {
    ('chnhcm.local/chnhcm/internal/bootstrap', 'TestCountry200SourceFieldRemovalMigration'),
    ('chnhcm.local/chnhcm/internal/bootstrap', 'TestT07ExactCurrentMigrationSetAdmission'),
}


def discover(output):
    roots = set()
    for line in output.splitlines():
        event = json.loads(line)
        name = event.get('Output', '').strip()
        if event.get('Action') == 'output' and re.fullmatch(r'Test\w+', name):
            root = (event['Package'], name)
            if root in roots:
                raise ValueError(f'duplicate test root: {root}')
            roots.add(root)
    if not roots:
        raise ValueError('no compiled integration tests discovered')
    return sorted(roots)


def partition(roots, shard, total):
    if not 1 <= shard <= total:
        raise ValueError('require 1 <= shard <= total <= test count')
    groups = balanced_groups(roots, total, lambda root: SECONDS['integration'].get(root[1], 3))
    return groups[shard - 1]


def select_roots(roots, pattern):
    selected = [root for root in roots if re.search(pattern, root[1])]
    if not selected:
        raise ValueError('integration filter selected no test roots')
    return selected


def remaining_roots(eligible):
    if not PREFLIGHT_ROOTS.issubset(eligible):
        raise ValueError('missing preflight integration root')
    return [root for root in eligible if root not in PREFLIGHT_ROOTS]


def run_recorded(command, package, names, path):
    # Persist only terminal root outcomes. Go's regular verbose output stays on the console.
    outcomes = {}
    with path.open('a') as output, subprocess.Popen([*command, '-json', package], stdout=subprocess.PIPE,
                                                    text=True) as process:
        for line in process.stdout:
            event = json.loads(line)
            if 'Output' in event:
                print(event['Output'], end='', flush=True)
            test, action = event.get('Test'), event['Action']
            if test in names and action in ('pass', 'fail', 'skip'):
                status = {'pass': 'passed', 'fail': 'failed', 'skip': 'skipped'}[action]
                outcomes[test] = status
                print(json.dumps(dict(package=package, name=test, status=status)), file=output, flush=True)
        return process.wait() != 0 or outcomes != dict.fromkeys(names, 'passed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--shard', type=int)
    parser.add_argument('--check', action='store_true', help='verify every planned partition without executing tests')
    parser.add_argument('--coverage-output', type=Path, help='write the checked full partition for PR reuse')
    parser.add_argument('--total', type=int, default=FULL_INTEGRATION)
    parser.add_argument('--list', action='store_true', help='only print the selected roots')
    parser.add_argument('--results', type=Path, help='record actual local preflight root outcomes')
    parser.add_argument('--after-preflight', action='store_true', help='local replay of the full post-preflight partitions')
    parser.add_argument('--preflight', action='store_true', help='run the two old-database upgrade checks')
    parser.add_argument('--filter', default='^Test', help='explicit admitted test-root pattern')
    args = parser.parse_args()
    if args.results and not args.preflight:
        parser.error('--results requires --preflight')
    split_preflight = args.preflight or args.after_preflight or bool(args.coverage_output)
    if os.environ.get('CI_PLAN'):
        from ci_policy import validate_plan
        plan = json.loads(os.environ['CI_PLAN'])
        validate_plan(plan, os.environ)
        if not plan['integration_shards']:
            raise ValueError('integration suite is not in the current plan')
        split_preflight = plan['scope'] == 'full'
        args.total, args.filter = plan['integration_shards'], plan['integration_filter']
    if args.preflight and (not split_preflight or args.shard is not None or args.check):
        parser.error('preflight requires a full selection and no shard')
    if not args.preflight and not args.check and args.shard is None:
        parser.error('--shard is required for execution')
    found = subprocess.run(['go', 'test', '-tags=integration', '-json', '-list', '^Test', *PACKAGES],
                           check=True, text=True, stdout=subprocess.PIPE)
    roots = discover(found.stdout)
    eligible = select_roots(roots, args.filter)
    remainder = remaining_roots(eligible) if split_preflight else eligible
    if args.check:
        parts = [sorted(PREFLIGHT_ROOTS)] if split_preflight else []
        parts.extend(partition(remainder, i, args.total) for i in range(1, args.total + 1))
        if sorted(root for part in parts for root in part) != eligible:
            raise ValueError('integration selection differs from the planned partitions')
        if split_preflight and args.coverage_output:
            from ci_coverage import inventory_part
            args.coverage_output.write_text(json.dumps(
                [inventory_part(part, [name for _, name in part]) for part in parts]))
        print(json.dumps({'discovered': len(roots), 'eligible': len(eligible),
                          'shard_counts': list(map(len, parts)), 'filter': args.filter}))
        return 0
    selected = sorted(PREFLIGHT_ROOTS) if args.preflight else partition(remainder, args.shard, args.total)
    print(json.dumps({'shard': args.shard, 'total': args.total,
                      'discovered': len(roots), 'eligible': len(eligible), 'filter': args.filter,
                      'selected': selected}), flush=True)
    if args.list:
        return 0
    failed = False
    for package in sorted({package for package, _ in selected}):
        names = [name for owner, name in selected if owner == package]
        pattern = '^(' + '|'.join(re.escape(name) for name in names) + ')$'
        command = ['go', 'test', '-tags=integration', '-timeout=20m', '-count=1', '-v', '-run', pattern]
        if args.results:
            failed |= run_recorded(command, package, names, args.results)
        else:
            failed |= subprocess.run([*command, package]).returncode != 0
    return int(failed)


if __name__ == '__main__':
    raise SystemExit(main())
