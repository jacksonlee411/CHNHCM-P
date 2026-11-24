#!/usr/bin/env python3
"""CI selection and gates. GitHub access is read-only; uncertainty never grants reuse."""
import argparse
from datetime import datetime, timezone, timedelta
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import zipfile

import ci_coverage as coverage
from ci_shards import FULL_BROWSER, TIMINGS, selected_browser_count, selected_integration_count
from ci_coverage import (ROOT, WORKFLOW, POLICY, SHA, changed_paths, git, latest_jobs,
                         run, selection_for, unpack_evidence)
from ci_impact import SELECTION_KEYS, full, select, validate_mapping

CONTRACT_PATHS = (WORKFLOW, POLICY, TIMINGS, 'scripts/ci_shards.py', '.github/actions/browser-runtime/action.yml',
                  'scripts/ci_policy.py', 'scripts/ci_coverage.py', 'scripts/ci_impact.py', 'scripts/ci_browser.py',
                  'scripts/ci_preview.py', 'scripts/integration-shard.py')


def digest_contract(root=ROOT):
    return hashlib.sha256(b''.join(p.encode() + b'\0' + (root / p).read_bytes() + b'\0' for p in CONTRACT_PATHS)).hexdigest()


def browser_files(policy, root=ROOT):
    files = list(policy['geography']['browser_files'])
    for source, tests in policy['geography']['extensions'].items():
        # Either source or a test activates the entire consumer group.
        if (root / source).exists() or any((root / 'tests/e2e' / p).exists() for p in tests):
            files.extend(tests)
    if len(set(files)) != len(files) or any(not (root / 'tests/e2e' / p).is_file() for p in files):
        raise ValueError('missing or duplicate geography consumer test')
    return files


class GitHub:
    def __init__(self, repo):
        if not re.fullmatch(r'[\w.-]+/[\w.-]+', repo):
            raise ValueError('invalid repository identity')
        self.repo = repo

    def raw(self, path):
        return run('gh', 'api', f'repos/{self.repo}/{path}')

    def json(self, path):
        return json.loads(self.raw(path))

    def pages(self, path, key=None):
        rows = []
        for page in range(1, 101):
            value = self.json(f'{path}{"&" if "?" in path else "?"}per_page=100&page={page}')
            batch = value[key] if key else value
            if not isinstance(batch, list):
                raise ValueError('invalid GitHub page')
            rows.extend(batch)
            if len(batch) < 100:
                if key and value.get('total_count', len(rows)) != len(rows):
                    raise ValueError('incomplete GitHub pagination')
                return rows
        raise ValueError('GitHub page budget exceeded')


def geography_selection(policy, root):
    pattern = policy['geography']['integration_filter']
    files = browser_files(policy, root)
    return dict(integration_filter=pattern, integration_shards=selected_integration_count(pattern),
                browser_files=files, browser_shards=selected_browser_count(files))


def classify(paths, policy, regular=True, root=ROOT):
    decision = select(paths, policy, regular, root)
    return decision['scope'], decision['reason']


def remote_browser_requested(plan, env):
    return (env.get('CI_REPOSITORY_VISIBILITY') == 'public'
            or plan['event'] == 'workflow_dispatch'
            or env.get('CI_FULL_BROWSER_HEAD') == plan['head']
            or (plan['event'] == 'pull_request' and env.get('CI_FULL_PR_HEAD') == plan['head']))


def job_names(plan, kind):
    if kind == 'browser' and not plan['remote_browser']:
        return []
    prefix = 'diagnostic ' if plan['event'] == 'workflow_dispatch' else ''
    count = plan[kind + '_shards']
    return [f'{prefix}{kind} shard {i}/{count}' for i in range(1, count + 1)]


def scheduled_jobs(plan, kind):
    return [name for name in job_names(plan, kind) if name not in plan.get('carried', {})]


def preflight_jobs(plan):
    if plan['scope'] == 'full' and 'preflight' not in plan.get('carried', {}):
        return ['diagnostic preflight' if plan['event'] == 'workflow_dispatch' else 'preflight']
    return []


def shard_routes(plan, event, suite, setting):
    limit = {'browser': FULL_BROWSER, 'integration': 8}[suite]
    setting = setting or '0'
    if setting not in {str(i) for i in range(limit + 1)}:
        raise ValueError(f'CI_{suite.upper()}_SELF_HOSTED_SHARDS must be a shard count from 0 to {limit}')
    repo = event.get('repository', {})
    trusted = repo.get('private') is True and repo.get('full_name') == plan['repo']
    if plan['event'] == 'pull_request':
        head_repo = event['pull_request']['head'].get('repo') or {}
        trusted = trusted and head_repo.get('full_name') == plan['repo']
    else:
        trusted = trusted and plan['event'] in {'push', 'workflow_dispatch'} and plan['ref'] == 'refs/heads/main'
    shards = plan[suite + '_shards']
    count = min(int(setting), shards) if trusted else 0
    ephemeral = ['self-hosted', 'linux', 'x64', 'chnhcm-ci-ephemeral']
    return [dict(shard=i, runner=ephemeral if i <= count else ['ubuntu-24.04'])
            for i in range(1, max(1, shards) + 1)]


def plan_outputs(plan, event, env):
    browser = shard_routes(plan, event, 'browser', env.get('CI_BROWSER_SELF_HOSTED_SHARDS'))
    integration = shard_routes(plan, event, 'integration', env.get('CI_INTEGRATION_SELF_HOSTED_SHARDS'))
    preflight_runner = (browser if plan['remote_browser'] else integration)[0]['runner']
    carried = plan.get('carried', {})
    browser = [row for row in browser if f'browser shard {row["shard"]}/{plan["browser_shards"]}' not in carried]
    integration = [row for row in integration if f'integration shard {row["shard"]}/{plan["integration_shards"]}' not in carried]
    if not plan['remote_browser'] or not plan['browser_shards']:
        browser = []
    counts = {suite: sum('self-hosted' in row['runner'] for row in routes)
              for suite, routes in [('browser', browser), ('integration', integration)]}
    placeholder = [dict(shard=1, runner=['ubuntu-24.04'])]
    return {'plan': dict(plan, self_hosted_shards=counts), 'scope': plan['scope'],
            'runtime_quality': plan['scope'] in {'full', 'selected'} and 'quality_source' not in plan,
            'integration_matrix': integration or placeholder, 'browser_matrix': browser or placeholder,
            'run_integration': bool(scheduled_jobs(plan, 'integration')),
            'run_browser': bool(scheduled_jobs(plan, 'browser')),
            'run_preflight': bool(preflight_jobs(plan)), 'has_carried': bool(carried or plan.get('quality_source')),
            'preflight_runner': preflight_runner}


def schedule(plan, event, env, api, root=ROOT):
    if plan['event'] != 'pull_request' or plan['scope'] != 'full':
        return plan_outputs(plan, event, env)
    folder = Path(env['RUNNER_TEMP'])
    inventories = {kind: json.loads((folder / f'ci-{kind}.json').read_text()) for kind in coverage.SUITES}
    coverage.inventory_units(inventories)
    head = event['pull_request']['head']
    carried = {}
    if (head.get('repo') or {}).get('full_name') == plan['repo'] and not plan.get('fresh_full'):
        try:
            carried = coverage.find_carried(api, plan, head['ref'], inventories, root)
        except coverage.READ_ERRORS as error:
            print('Previous PR coverage unavailable.', type(error).__name__)
    attempt = int(env['GITHUB_RUN_ATTEMPT'])
    outputs = plan_outputs(dict(plan, carried=carried, coverage_attempt=attempt), event, env)
    record = dict(plan=outputs['plan'], attempt=attempt, inventories=inventories)
    (folder / 'ci-coverage.json').write_text(json.dumps(record, separators=(',', ':')))
    return outputs


def require_jobs(jobs, expected):
    latest = latest_jobs(jobs)
    for name in expected:
        value = latest.get(name, {})
        if value.get('status') != 'completed' or value.get('conclusion') != 'success':
            raise ValueError(f'missing or unsuccessful job: {name}')


def check_current_pr(plan, pr):
    if (pr.get('state') != 'open' or pr.get('draft') or pr['head']['sha'] != plan['head']
            or pr['base']['sha'] != plan['base'] or pr['base']['ref'] != 'main'):
        raise ValueError('PR is draft, closed or no longer the tested head/base')


def reuse_matches(plan, evidence, run_info, pr, parents, now):
    created = datetime.fromisoformat(run_info['created_at'].replace('Z', '+00:00'))
    prior = evidence.get('plan', {})
    return (evidence.get('schema') == 2 and prior.get('scope') in {'full', 'selected', 'docs'}
            and prior.get('schema') == 2
            and (not plan['remote_browser'] or prior.get('remote_browser') is True)
            and prior.get('event') == 'pull_request' and prior.get('repo') == plan['repo']
            and prior.get('pr') == pr['number'] and prior.get('head') == pr['head']['sha']
            and parents == [prior.get('base'), prior.get('head')]
            and prior.get('tree') == plan['tree'] and prior.get('contract') == plan['contract']
            and prior.get('run_id') == str(run_info['id'])
            and evidence.get('attempt') == run_info['run_attempt']
            and prior.get('ref') == f'refs/pull/{pr["number"]}/merge'
            and run_info.get('event') == 'pull_request' and run_info.get('path') == WORKFLOW
            and run_info.get('head_sha') == prior.get('head')
            and run_info.get('status') == 'completed' and run_info.get('conclusion') == 'success'
            # GitHub may clear this list after merge. The commit->merged PR,
            # run head SHA and retained artifact PR identity remain mandatory.
            and (not run_info.get('pull_requests') or any(p['number'] == pr['number'] for p in run_info['pull_requests']))
            and timedelta(0) <= now - created <= timedelta(days=7))


def find_reuse(plan, api, root=ROOT):
    parents = git('show', '-s', '--format=%P', plan['checkout'], root=root).split()
    prs = api.pages(f'commits/{plan["checkout"]}/pulls')
    for pr in prs:
        if not (pr.get('merged_at') and pr.get('merge_commit_sha') == plan['checkout']
                and pr['base']['ref'] == 'main' and pr['head']['repo']['full_name'] == plan['repo']):
            continue
        runs = api.pages(f'actions/workflows/ci.yml/runs?event=pull_request&head_sha={pr["head"]["sha"]}&status=success', 'workflow_runs')
        for info in runs:
            artifacts = api.pages(f'actions/runs/{info["id"]}/artifacts', 'artifacts')
            matches = [a for a in artifacts if a['name'] == f'ci-evidence-{info["run_attempt"]}' and not a['expired']]
            if len(matches) != 1:
                continue
            evidence = unpack_evidence(api.raw(f'actions/artifacts/{matches[0]["id"]}/zip'))
            if not reuse_matches(plan, evidence, info, pr, parents, datetime.now(timezone.utc)):
                continue
            prior = evidence['plan']
            expected = selection_for(prior['base'], plan['checkout'], root)
            if any(prior.get(key) != expected[key] for key in SELECTION_KEYS):
                continue
            jobs = api.pages(f'actions/runs/{info["id"]}/jobs?filter=all', 'jobs')
            jobs = [j for j in jobs if j['run_attempt'] >= prior.get('coverage_attempt', 1)]
            require_jobs(jobs, ['quality', 'integration', 'browser'] + preflight_jobs(prior)
                         + scheduled_jobs(prior, 'integration') + scheduled_jobs(prior, 'browser'))
            coverage.verify_carried(api, prior, prior.get('carried', {}), root, verify_quality=True)
            return str(info['id'])
    return ''


def build_plan(event, env, api, root=ROOT):
    policy = json.loads((root / POLICY).read_text())
    validate_mapping(policy, root)
    checkout = git('rev-parse', 'HEAD', root=root)
    if checkout != env['GITHUB_SHA'] or not SHA.fullmatch(checkout):
        raise ValueError('checkout does not match the triggering SHA')
    head = event['pull_request']['head']['sha'] if env['GITHUB_EVENT_NAME'] == 'pull_request' else checkout
    plan = dict(schema=2, repo=env['GITHUB_REPOSITORY'], event=env['GITHUB_EVENT_NAME'],
                run_id=env['GITHUB_RUN_ID'], ref=env['GITHUB_REF'], checkout=checkout,
                tree=git('rev-parse', 'HEAD^{tree}', root=root), contract=digest_contract(root),
                head=head, base='', pr=0, reuse_run='', **full('default complete validation'))
    plan['remote_browser'] = remote_browser_requested(plan, env)
    if plan['event'] == 'pull_request':
        pr = event['pull_request']
        plan.update(pr=pr['number'], base=pr['base']['sha'])
        if plan['ref'] != f'refs/pull/{plan["pr"]}/merge' or pr['base']['ref'] != 'main':
            raise ValueError('not the main PR merge reference')
        parents = git('show', '-s', '--format=%P', checkout, root=root).split()
        if parents != [plan['base'], plan['head']]:
            raise ValueError('merge candidate does not bind the event head/base')
        if pr['draft']:
            plan.update(scope='draft', reason='draft precheck only; merge gates must fail',
                        integration_filter='', integration_shards=0, browser_files=[], browser_shards=0)
        else:
            plan.update(selection_for(plan['base'], checkout, root))
            if plan['scope'] == 'full' and env.get('CI_FULL_PR_HEAD') == plan['head']:
                plan['fresh_full'] = True
            elif (plan['scope'] == 'full' and event.get('before') and
                    (pr['head'].get('repo') or {}).get('full_name') == plan['repo']):
                try:
                    match = coverage.documentation_checkpoint(api, plan, event['before'], root)
                    if match:
                        record, plan['quality_source'] = match
                        for kind, parts in record['inventories'].items():
                            (Path(env['RUNNER_TEMP']) / f'ci-{kind}.json').write_text(json.dumps(parts))
                except coverage.READ_ERRORS as error:
                    plan.pop('quality_source', None)
                    print('Running current quality and inventory discovery.', type(error).__name__)
    elif plan['event'] == 'push' and plan['ref'] == 'refs/heads/main':
        try:
            plan['reuse_run'] = find_reuse(plan, api, root)
        except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError, zipfile.BadZipFile):
            plan['reason'] = 'reuse evidence unavailable or invalid; complete validation'
        if plan['reuse_run']:
            plan.update(scope='reuse', reason='equivalent PR impact selection and successful jobs verified',
                        integration_filter='', integration_shards=0, browser_files=[], browser_shards=0)
        elif SHA.fullmatch(event.get('before', '')) and event['before'] != '0' * 40:
            plan.update(base=event['before'], **selection_for(event['before'], checkout, root))
    elif plan['event'] == 'workflow_dispatch':
        suite = event.get('inputs', {}).get('suite')
        if suite not in ('geography-legal-entity', 'browser-full', 'full'):
            raise ValueError('请选择要运行的远程诊断范围')
        if suite == 'browser-full':
            plan.update(scope='selected', reason='explicit full browser diagnostic', integration_filter='', integration_shards=0)
        elif suite == 'geography-legal-entity':
            plan.update(scope='selected', reason='explicit diagnostic scope, not a PR gate',
                        **geography_selection(policy, root))
    else:
        raise ValueError('unsupported formal CI event')
    return plan


def validate_plan(plan, env, root=ROOT):
    if (plan.get('schema') != 2 or plan.get('scope') not in {'docs', 'draft', 'full', 'selected', 'reuse'}
            or plan.get('repo') != env['GITHUB_REPOSITORY'] or plan.get('run_id') != env['GITHUB_RUN_ID']
            or plan.get('event') != env['GITHUB_EVENT_NAME'] or plan.get('checkout') != env['GITHUB_SHA']
            or plan.get('checkout') != git('rev-parse', 'HEAD', root=root)
            or plan.get('tree') != git('rev-parse', 'HEAD^{tree}', root=root)
            or plan.get('contract') != digest_contract(root)):
        raise ValueError('missing or mismatched CI plan identity')
    if plan.get('remote_browser') is not remote_browser_requested(plan, env):
        raise ValueError('browser execution mode differs from the candidate request')
    if plan.get('ref') != env['GITHUB_REF']:
        raise ValueError('invalid selection reference')
    if 'carried' in plan:
        eligible = {'preflight', *job_names(plan, 'integration'), *job_names(plan, 'browser')}
        if (plan['event'] != 'pull_request' or plan['scope'] != 'full'
                or type(plan.get('coverage_attempt')) is not int or plan['coverage_attempt'] < 1
                or not isinstance(plan['carried'], dict) or not plan['carried'].keys() <= eligible):
            raise ValueError('invalid carried PR jobs')
    refs = list(plan.get('carried', {}).values())
    if 'quality_source' in plan:
        refs.append(plan['quality_source'])
    for ref in refs:
        if (plan['scope'] != 'full' or plan['event'] != 'pull_request' or plan.get('fresh_full')
                or not isinstance(ref, dict) or set(ref) != {'run', 'job'}
                or type(ref['run']) is not int or not 0 < ref['run'] < int(plan['run_id'])
                or type(ref['job']) is not int or ref['job'] <= 0):
            raise ValueError('invalid original PR job reference')
    push_diff = plan['event'] == 'push' and bool(plan['base'])
    if push_diff:
        before = json.loads(Path(env['GITHUB_EVENT_PATH']).read_text()).get('before')
        if plan['ref'] != 'refs/heads/main' or plan['base'] != before:
            raise ValueError('main selection does not match the push event')
    if push_diff or (plan['event'] == 'pull_request' and plan['scope'] != 'draft'):
        expected = selection_for(plan['base'], plan['checkout'], root)
        if any(plan.get(key) != expected[key] for key in SELECTION_KEYS):
            raise ValueError('selection differs from the complete candidate diff')
    elif plan['scope'] in {'reuse', 'draft'}:
        if any(plan.get(k) for k in ['integration_filter', 'integration_shards', 'browser_files', 'browser_shards']):
            raise ValueError('omitted suites contain execution fields')
    elif plan['scope'] == 'full':
        if any(plan.get(k) != full('')[k] for k in SELECTION_KEYS if k != 'reason'):
            raise ValueError('invalid full plan')
    elif plan['event'] == 'workflow_dispatch':
        policy = json.loads((root / POLICY).read_text())
        suite = json.loads(Path(env['GITHUB_EVENT_PATH']).read_text()).get('inputs', {}).get('suite')
        if suite == 'browser-full':
            expected = ('', 0, [], FULL_BROWSER)
        elif suite == 'geography-legal-entity':
            selected = geography_selection(policy, root)
            expected = tuple(selected[key] for key in SELECTION_KEYS[2:])
        else:
            raise ValueError('unknown diagnostic suite')
        if plan.get('scope') != 'selected' or tuple(plan.get(k) for k in SELECTION_KEYS[2:]) != expected:
            raise ValueError('invalid diagnostic selection')
    else:
        raise ValueError('unverified selection')
    if plan['scope'] in {'docs', 'draft'} and plan['event'] != 'pull_request' and not push_diff:
        raise ValueError('invalid PR-only scope')
    if plan['scope'] == 'reuse' and (plan['event'] != 'push' or plan['ref'] != 'refs/heads/main' or not plan.get('reuse_run')):
        raise ValueError('invalid reuse scope')


def gate(plan, kind, results, api, env, root=ROOT):
    if plan['scope'] == 'draft' or results.get('quality') != 'success':
        raise ValueError('draft or unsuccessful quality cannot pass a merge gate')
    if plan['event'] == 'pull_request':
        check_current_pr(plan, api.json(f'pulls/{plan["pr"]}'))
    if kind == 'browser' and results.get('integration') != 'success':
        raise ValueError('integration gate did not succeed')
    early = preflight_jobs(plan)
    if results.get('preflight') != ('success' if early else 'skipped'):
        raise ValueError('preflight result does not match the planned scope')
    shard_result = results.get(kind + '-shard')
    if kind + '_shards' not in plan:
        raise ValueError('missing suite selection')
    expected = scheduled_jobs(plan, kind)
    coverage.verify_carried(api, plan, ['preflight', *job_names(plan, kind)], root, verify_quality=kind == 'browser')
    reused = len(job_names(plan, kind)) - len(expected)
    if not expected:
        if shard_result != 'skipped' or (plan['scope'] == 'reuse' and not plan['reuse_run']):
            raise ValueError('unexpected result for explicitly omitted suite')
        if kind == 'browser' and not plan['remote_browser'] and plan['browser_shards']:
            return '浏览器按影响在本地验证。本次未要求远程执行，未运行远程浏览器。'
        return f'{kind}: {reused} previous successful shards verified' if reused else 'not executed: ' + plan['reason']
    if shard_result != 'success':
        raise ValueError('planned shards did not all succeed')
    jobs = api.pages(f'actions/runs/{env["GITHUB_RUN_ID"]}/jobs?filter=all', 'jobs')
    jobs = [j for j in jobs if j['run_attempt'] >= plan.get('coverage_attempt', 1)]
    actual = {j['name'] for j in jobs if j['name'].startswith(('diagnostic ' if plan['event'] == 'workflow_dispatch' else '') + kind + ' shard ')}
    if actual != set(expected):
        raise ValueError('unexpected or missing shard identity')
    require_jobs(jobs, early + expected)
    return f'{kind}: all {len(expected)} planned shards successful; {reused} previous shards verified'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['plan', 'schedule', 'gate', 'quality'])
    parser.add_argument('--kind', choices=['integration', 'browser'])
    args = parser.parse_args()
    env = os.environ
    api = GitHub(env['GITHUB_REPOSITORY'])
    if args.command in ('plan', 'schedule'):
        event = json.loads(Path(env['GITHUB_EVENT_PATH']).read_text())
        if args.command == 'plan':
            outputs = plan_outputs(build_plan(event, env, api), event, env)
        else:
            plan = json.loads(env['CI_PLAN'])
            validate_plan(plan, env)
            outputs = schedule(plan, event, env, api)
        with open(env['GITHUB_OUTPUT'], 'a') as f:
            for k, value in outputs.items():
                f.write(k + '=' + (value if isinstance(value, str) else json.dumps(value, separators=(',', ':'))) + '\n')
        message = json.dumps(outputs['plan'], ensure_ascii=False)
    else:
        plan = json.loads(env['CI_PLAN'])
        validate_plan(plan, env)
        if args.command == 'quality':
            if plan['scope'] == 'draft':
                raise ValueError('draft precheck completed; candidate is not mergeable')
            if plan['event'] == 'pull_request':
                check_current_pr(plan, api.json(f'pulls/{plan["pr"]}'))
            message = 'quality: current candidate validated'
        else:
            needs = json.loads(env['CI_NEEDS'])
            message = gate(plan, args.kind, {k: v['result'] for k, v in needs.items()}, api, env)
            if args.kind == 'browser' and plan['event'] == 'pull_request' and plan['scope'] != 'draft':
                evidence = dict(schema=2, plan=plan, attempt=int(env['GITHUB_RUN_ATTEMPT']))
                Path(env['RUNNER_TEMP'], 'ci-evidence.json').write_text(json.dumps(evidence))
    print(message)
    with open(env['GITHUB_STEP_SUMMARY'], 'a') as f:
        if args.command == 'schedule' and 'carried' in outputs['plan']:
            current = outputs['plan']
            early = '执行' if outputs['run_preflight'] else '复用'
            browser_count = len(scheduled_jobs(current, 'browser')) if outputs['run_browser'] else 0
            f.write(f'本轮派发 DB/API {len(scheduled_jobs(current, "integration"))}/{current["integration_shards"]} 片，'
                    f'浏览器 {browser_count}/{current["browser_shards"]} 片，前置回归{early}。\n\n')
            for rid in sorted({ref['run'] for ref in current['carried'].values()}):
                names = sorted(n for n, ref in current['carried'].items() if ref['run'] == rid)
                url = f'{env["GITHUB_SERVER_URL"]}/{current["repo"]}/actions/runs/{rid}'
                f.write(f'- [来源运行 {rid}]({url}) · {", ".join(names)}\n')
        if args.command == 'schedule' and not outputs['plan']['remote_browser']:
            f.write('浏览器默认本地验证，本次未要求远程执行。\n')
        f.write('\n```json\n' + message + '\n```\n')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
        # Never print subprocess output, authentication headers or signed artifact URLs.
        print('CI policy rejected the operation:', str(error) if type(error) is ValueError else type(error).__name__)
        raise SystemExit(1)
