#!/usr/bin/env python3
"""Check actual Playwright inventories before running an admitted CI shard."""
import argparse
from collections import Counter
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

from ci_policy import ROOT, POLICY, browser_files, validate_plan
from ci_coverage import inventory_part
from ci_shards import FULL_BROWSER, balanced_groups, browser_seconds, selected_browser_count


PREFLIGHT_CASES = (
    ('operation-permissions.spec.ts', 'TL380/382 在途日志失效、历史切换和原上界刷新'),
    ('staffing307.spec.ts', 'Staffing307 admin 双对象桌面窄屏短屏布局和取消'),
    ('staffing309.spec.ts', '309 grade admin V01 AC017/020/021/022/023/028/030'),
    ('staffing309.spec.ts', '309 grade admin V03 AC017/020/021/022/023/028/030'),
    ('staffing312-hierarchy.spec.ts', '139 查询者聚焦结构日期后收到默认日期仍可输入并查询'),
    ('staffing314.spec.ts', '314 T04 双维查询、返回与固定选择 390px'),
)
PREFLIGHT_TITLES = tuple(title for _, title in PREFLIGHT_CASES)


def title_args(titles):
    # Playwright grep matches the full title including the project and suite.
    return ['--grep', '(?:' + '|'.join(re.escape(title) for title in titles) + ')$']


def inventory(report, expected_titles=()):
    if report.get('errors'):
        raise ValueError('Playwright discovery errors')
    found = {}
    titles = []

    def entry_path(value):
        path = Path(value)
        if path.is_absolute():
            path = path.relative_to(report['config']['rootDir'])
        return path.as_posix()

    def visit(suite, owner=None):
        # File filters select the entry suite. spec.file can point to a
        # shared helper imported by several distinct entry suites.
        if not owner and suite.get('file', '').endswith('.spec.ts'):
            owner = entry_path(suite['file'])
        for spec in suite.get('specs', []):
            for test in spec['tests']:
                key = (test['projectName'], spec['file'], spec['id'])
                if key in found:
                    raise ValueError('duplicate discovered test')
                found[key] = owner or entry_path(spec['file'])
                if expected_titles and test['projectName'] == 'chromium':
                    titles.append(spec['title'])
        for child in suite.get('suites', []):
            visit(child, owner)
    visit(report)
    if not found:
        raise ValueError('empty Playwright inventory')
    if expected_titles and sorted(titles) != sorted(expected_titles):
        raise ValueError('missing or ambiguous preflight browser case')
    return found


def verify_partition(whole, shards, dependencies):
    counts = Counter(key for shard in shards for key in shard)
    if any(not shard for shard in shards) or set(counts) != set(whole):
        raise ValueError('empty shard, missing or unknown test')
    if any(count != 1 and key[0] not in dependencies for key, count in counts.items()):
        raise ValueError('non-dependency test runs more than once')
    return {'unique_tests': len(whole), 'executions': sum(counts.values()),
            'shard_counts': list(map(len, shards)),
            'inventory_sha256': hashlib.sha256(json.dumps(sorted(whole)).encode()).hexdigest()}


def selected_files(scope, root=ROOT):
    return browser_files(json.loads((root / POLICY).read_text()), root) if scope == 'geography' else []


def arguments(files):
    return ['(?:^|/)' + re.escape(p) + '$' for p in files]


def shard_files(selected, total, dependencies):
    entries = sorted({owner for key, owner in selected.items() if key[0] not in dependencies})
    if not entries and total == 1:
        return [sorted(set(selected.values()))]
    return balanced_groups(entries, total, browser_seconds)


def discover(args, expected_titles=()):
    result = subprocess.check_output(['corepack', 'pnpm', 'exec', 'playwright', 'test',
                                     '--config', 'tests/e2e/playwright.config.ts',
                                     '--list', '--reporter=json', *args], cwd=ROOT)
    return inventory(json.loads(result), expected_titles)


def check_inventory(files, total, split=False, coverage_output=None):
    dependencies = json.loads((ROOT / POLICY).read_text())['dependency_projects']
    whole = discover([])
    selected = discover(arguments(files)) if files else whole
    expected = {key for key, owner in whole.items() if not files or owner in files or key[0] in dependencies}
    if set(selected) != expected or not set(files) <= set(selected.values()):
        raise ValueError('consumer files absent from configuration or selection differs from full inventory')
    early = title_args(PREFLIGHT_TITLES)
    shards = [discover(early, PREFLIGHT_TITLES)] if split else []
    remaining = ['--grep-invert', early[1]] if split else []
    shards.extend(discover(arguments(group) + remaining)
                  for group in shard_files(selected, total, dependencies))
    result = verify_partition(selected, shards, dependencies)
    if split and coverage_output:
        parts = [inventory_part([(*key, owner) for key, owner in shard.items()], shard.values())
                 for shard in shards]
        coverage_output.write_text(json.dumps(parts))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['check', 'run', 'preflight'])
    parser.add_argument('--scope', choices=['full', 'geography'], help='local inventory only; CI uses CI_PLAN')
    parser.add_argument('--shard', type=int)
    parser.add_argument('--coverage-output', type=Path, help='write the checked full partition for PR reuse')
    args = parser.parse_args()
    if os.environ.get('CI_PLAN'):
        plan = json.loads(os.environ['CI_PLAN'])
        validate_plan(plan, os.environ)
        scope = plan['scope']
        total, files = plan['browser_shards'], plan['browser_files']
        if not total:
            raise ValueError('browser suite is not in the current plan')
    elif args.scope:
        scope = args.scope
        files = selected_files(args.scope)
        total = selected_browser_count(files) if files else FULL_BROWSER
    else:
        parser.error('CI_PLAN or an explicit local --scope is required')
    early = title_args(PREFLIGHT_TITLES)
    split = scope == 'full'
    remaining = ['--grep-invert', early[1]] if split else []
    if args.command == 'check':
        print(json.dumps(check_inventory(files, total, split, args.coverage_output)))
    else:
        if args.command == 'preflight':
            if scope != 'full' or args.shard is not None:
                parser.error('preflight requires a full selection and no shard')
            selected_args = early
        else:
            if args.shard is None:
                parser.error('run requires a shard')
            if not 1 <= args.shard <= total:
                parser.error('invalid browser shard')
            dependencies = json.loads((ROOT / POLICY).read_text())['dependency_projects']
            groups = shard_files(discover(arguments(files)), total, dependencies)
            selected_args = arguments(groups[args.shard - 1]) + remaining
        selected = discover(selected_args, PREFLIGHT_TITLES if args.command == 'preflight' else ())
        print(json.dumps({'phase': args.command, 'shard': args.shard, 'total': total, 'tests': sorted(selected)}), flush=True)
        # Pass an argument vector directly: file patterns never enter a shell.
        profile = ['--profile', 'ci-browser'] if os.environ.get('CI_PLAN') else []
        os.execvp('python3', ['python3', 'scripts/local-bu.py', 'smoke', *profile, '--', *selected_args])


if __name__ == '__main__':
    main()
