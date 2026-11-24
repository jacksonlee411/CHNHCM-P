import copy
from datetime import datetime, timezone, timedelta
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import zipfile

import ci_policy as ci


class API:
    def __init__(self, pr=None, jobs=()):
        self.pr, self.jobs = pr, list(jobs)

    def json(self, path):
        return self.pr

    def pages(self, path, key=None):
        return self.jobs


def jobs(names, attempt=1, conclusion='success'):
    return [dict(id=i + attempt * 100, name=n, run_attempt=attempt,
                 status='completed', conclusion=conclusion) for i, n in enumerate(names)]


class CandidateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.git('init', '-q', '-b', 'main')
        self.git('config', 'user.email', 'fixture@example.invalid')
        self.git('config', 'user.name', 'Fixture')
        for name in ci.CONTRACT_PATHS:
            p = self.root / name
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes((ci.ROOT / name).read_bytes())
        policy = json.loads((self.root / ci.POLICY).read_text())
        for spec in policy['domains'].values():
            for name in spec['browser_files']:
                self.write('tests/e2e/' + name, '')
        self.write('docs/guide.md', 'initial')
        self.write('internal/core.go', 'core')
        self.base = self.commit()

    def git(self, *args):
        return ci.git(*args, root=self.root)

    def write(self, name, content):
        p = self.root / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)

    def commit(self):
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture')
        return self.git('rev-parse', 'HEAD')

    def candidate(self, changes, draft=False):
        self.git('checkout', '-qb', 'feature')
        changes()
        head = self.commit()
        self.git('checkout', '-q', 'main')
        self.git('merge', '--no-ff', '-qm', 'merge', head)
        env = dict(GITHUB_SHA=self.git('rev-parse', 'HEAD'), GITHUB_EVENT_NAME='pull_request',
                   GITHUB_REPOSITORY='owner/repo', GITHUB_RUN_ID='123', GITHUB_REF='refs/pull/1/merge')
        pr = dict(number=1, state='open', draft=draft, head={'sha': head}, base={'sha': self.base, 'ref': 'main'})
        return ci.build_plan({'pull_request': pr}, env, API(pr), self.root), env, pr

    def test_complete_diff_includes_earlier_code_change(self):
        def changes():
            self.write('internal/core.go', 'code changed')
            self.commit()
            self.write('docs/guide.md', 'last commit is only documentation')
        plan, env, _ = self.candidate(changes)
        self.assertEqual(plan['scope'], 'full')
        ci.validate_plan(plan, env, self.root)

    def test_full_candidate_restores_inventory_only_for_verified_docs_followup(self):
        before = []
        def change():
            self.write('internal/core.go', 'product change')
            before.append(self.commit())
            self.write('docs/guide.md', 'explain the product change')
        original, env, pr = self.candidate(change)
        pr['head'].update(repo={'full_name': 'owner/repo'}, ref='topic')
        event = dict(before=before[0], pull_request=pr)
        source = {'run': 2, 'job': 3}
        inventories = {kind: [kind] for kind in ('browser', 'integration')}
        with tempfile.TemporaryDirectory() as directory, patch.object(ci.coverage, 'documentation_checkpoint',
                return_value=({'inventories': inventories}, source)) as lookup:
            env['RUNNER_TEMP'] = directory
            plan = ci.build_plan(event, env, API(pr), self.root)
            self.assertEqual((original['scope'], plan['scope'], plan['quality_source']), ('full', 'full', source))
            self.assertFalse(ci.plan_outputs(plan, event, env)['runtime_quality'])
            for kind, parts in inventories.items():
                self.assertEqual(json.loads(Path(directory, f'ci-{kind}.json').read_text()), parts)
            fresh = ci.build_plan(event, dict(env, CI_FULL_PR_HEAD=pr['head']['sha']), API(pr), self.root)
            self.assertTrue(fresh['fresh_full'])
            self.assertTrue(ci.plan_outputs(fresh, event, env)['runtime_quality'])
            lookup.assert_called_once()

    def test_docs_plan_and_identity_cannot_be_reused_after_checkout_change(self):
        plan, env, _ = self.candidate(lambda: self.write('docs/guide.md', 'changed'))
        self.assertEqual(plan['scope'], 'docs')
        ci.validate_plan(plan, env, self.root)
        self.write('internal/core.go', 'later')
        self.commit()
        with self.assertRaises(ValueError):
            ci.validate_plan(plan, env, self.root)

    def test_two_commit_depth_runner_can_recompute_merge_diff(self):
        plan, env, _ = self.candidate(lambda: self.write('docs/guide.md', 'changed'))
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / 'checkout'
            subprocess.run(['git', 'clone', '--quiet', '--depth=2', self.root.as_uri(), str(clone)],
                           check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            ci.validate_plan(plan, env, clone)

    def test_frontend_candidate_cannot_remove_selected_consumer_or_add_backend_work(self):
        path = 'apps/web/src/features/organization/Countries.tsx'
        plan, env, _ = self.candidate(lambda: self.write(path, 'export const Country = 1'))
        self.assertEqual((plan['scope'], plan['integration_shards']), ('selected', 0))
        ci.validate_plan(plan, env, self.root)
        for field, value in [('browser_files', []), ('browser_shards', 0), ('integration_shards', 8),
                             ('scope', 'docs'), ('integration_filter', '^Test')]:
            altered = dict(plan, **{field: value})
            with self.subTest(field=field), self.assertRaises(ValueError):
                ci.validate_plan(altered, env, self.root)

    def test_rename_from_code_to_document_is_full(self):
        def changes():
            (self.root / 'internal/core.go').rename(self.root / 'docs/core.md')
        plan, _, _ = self.candidate(changes)
        self.assertEqual(plan['scope'], 'full')

    def test_document_symlink_is_full(self):
        plan, _, _ = self.candidate(lambda: (self.root / 'docs/link.md').symlink_to('../internal/core.go'))
        self.assertEqual(plan['scope'], 'full')

    def test_draft_scope_and_stale_base_are_rejected(self):
        plan, env, pr = self.candidate(lambda: self.write('docs/guide.md', 'draft'), draft=True)
        self.assertEqual(plan['scope'], 'draft')
        with self.assertRaises(ValueError):
            ci.gate(plan, 'integration', {'quality': 'success'}, API(pr), env)
        for field in ['head', 'base']:
            stale = copy.deepcopy(pr)
            stale['draft'] = False
            stale[field]['sha'] = '0' * 40
            with self.assertRaises(ValueError):
                ci.check_current_pr(plan, stale)

    def test_direct_main_and_unavailable_evidence_select_full(self):
        env = dict(GITHUB_SHA=self.base, GITHUB_EVENT_NAME='push', GITHUB_REPOSITORY='owner/repo',
                   GITHUB_RUN_ID='12', GITHUB_REF='refs/heads/main')
        for error in [None, ValueError('missing'), subprocess.CalledProcessError(1, 'gh')]:
            with self.subTest(error=error), patch.object(ci, 'find_reuse', return_value='', side_effect=error):
                self.assertEqual(ci.build_plan({}, env, API(), self.root)['scope'], 'full')

    def test_main_without_pr_evidence_uses_the_push_diff(self):
        self.write('docs/guide.md', 'changed')
        head = self.commit()
        event = {'before': self.base}
        payload = self.root / 'event.json'
        payload.write_text(json.dumps(event))
        env = dict(GITHUB_SHA=head, GITHUB_EVENT_NAME='push', GITHUB_REPOSITORY='owner/repo',
                   GITHUB_RUN_ID='12', GITHUB_REF='refs/heads/main', GITHUB_EVENT_PATH=str(payload))
        plan = ci.build_plan(event, env, API(), self.root)
        self.assertEqual(plan['scope'], 'docs')
        ci.validate_plan(plan, env, self.root)
        self.assertFalse(ci.schedule(plan, event, env, API())['run_browser'])
        with self.assertRaisesRegex(ValueError, 'push event'):
            ci.validate_plan(dict(plan, base=head), env, self.root)

    def test_full_browser_requires_this_commit_and_manual_full_requires_an_explicit_suite(self):
        env = dict(GITHUB_SHA=self.base, GITHUB_EVENT_NAME='push', GITHUB_REPOSITORY='owner/repo',
                   GITHUB_RUN_ID='12', GITHUB_REF='refs/heads/main')
        plan = ci.build_plan({}, env, API(), self.root)
        pending = ci.schedule(plan, {}, dict(env, CI_FULL_PR_HEAD='old'), API())
        self.assertFalse(pending['run_browser'])
        self.assertTrue(pending['run_integration'])
        results = {'quality': 'success', 'integration': 'success', 'preflight': 'success', 'browser-shard': 'skipped'}
        self.assertIn('未运行远程浏览器', ci.gate(pending['plan'], 'browser', results, API(), env))
        with self.assertRaises(ValueError):
            ci.gate(pending['plan'], 'browser', dict(results, **{'browser-shard': 'success'}), API(), env)
        requested = ci.build_plan({}, dict(env, CI_FULL_BROWSER_HEAD=self.base), API(), self.root)
        self.assertTrue(ci.schedule(requested, {}, env, API())['run_browser'])
        for result in ['skipped', 'failure', 'cancelled']:
            with self.subTest(result=result), self.assertRaises(ValueError):
                ci.gate(requested, 'browser', dict(results, **{'browser-shard': result}), API(), env)
        env['GITHUB_EVENT_NAME'] = 'workflow_dispatch'
        for suite in (None, 'select-suite', 'unknown'):
            with self.subTest(suite=suite), self.assertRaisesRegex(ValueError, '诊断范围'):
                ci.build_plan({'inputs': {'suite': suite}}, env, API(), self.root)
        event = {'inputs': {'suite': 'full'}}
        plan = ci.build_plan(event, env, API(), self.root)
        self.assertTrue(ci.schedule(plan, event, env, API())['run_preflight'])
        event = {'inputs': {'suite': 'geography-legal-entity'}}
        limited = ci.build_plan(event, env, API(), self.root)
        outputs = ci.schedule(limited, event, env, API())
        self.assertTrue(outputs['run_browser'])
        self.assertEqual(len(outputs['browser_matrix']), 4)
        event = {'inputs': {'suite': 'browser-full'}}
        payload = self.root / 'event.json'
        payload.write_text(json.dumps(event))
        env['GITHUB_EVENT_PATH'] = str(payload)
        plan = ci.build_plan(event, env, API(), self.root)
        ci.validate_plan(plan, env, self.root)
        outputs = ci.schedule(plan, event, env, API())
        self.assertTrue(outputs['run_browser'])
        self.assertFalse(outputs['run_integration'])

    def test_public_push_and_pr_schedule_browser_without_an_override(self):
        for event in ('push', 'pull_request'):
            plan = dict(event=event, head='candidate')
            self.assertTrue(ci.remote_browser_requested(plan, {'CI_REPOSITORY_VISIBILITY': 'public'}))
            self.assertFalse(ci.remote_browser_requested(plan, {'CI_REPOSITORY_VISIBILITY': 'private'}))

    def test_browser_only_full_selection_still_requires_approval(self):
        plan, env, pr = self.candidate(lambda: self.write('packages/ui/src/example.tsx', 'export default 1'))
        self.assertEqual((plan['scope'], plan['browser_shards'], plan['integration_shards']), ('selected', 11, 0))
        outputs = ci.schedule(plan, {'pull_request': pr}, env, API())
        self.assertFalse(outputs['run_browser'])
        self.assertFalse(outputs['run_integration'])
        env['CI_FULL_BROWSER_HEAD'] = plan['head']
        event = {'pull_request': pr}
        approved = ci.schedule(ci.build_plan(event, env, API(), self.root), event, env, API())
        self.assertTrue(approved['run_browser'])

    def test_selected_browser_defaults_to_local_with_full_integration(self):
        def change():
            self.write('internal/bootstrap/example_test.go', 'package bootstrap')
            self.write('tests/e2e/first-hire.spec.ts', 'changed scenario')
        plan, env, pr = self.candidate(change)
        outputs = ci.schedule(plan, {'pull_request': pr}, env, API())
        self.assertEqual(len(outputs['integration_matrix']), 8)
        self.assertTrue(outputs['run_integration'])
        self.assertEqual(plan['browser_files'], ['first-hire.spec.ts'])
        self.assertFalse(outputs['run_browser'])
        event = {'pull_request': pr}
        env['CI_FULL_BROWSER_HEAD'] = plan['head']
        requested = ci.build_plan(event, env, API(), self.root)
        self.assertTrue(ci.schedule(requested, event, env, API())['run_browser'])

    def test_forged_selection_fields_are_rejected(self):
        plan, env, _ = self.candidate(lambda: self.write('internal/core.go', 'changed'))
        for field, value in [('integration_shards', 0), ('browser_shards', 1), ('ref', 'refs/heads/main'),
                             ('integration_filter', '^TestOne'), ('browser_files', ['one.spec.ts']), ('contract', 'old'), ('remote_browser', True),
                             ('quality_source', {}), ('quality_source', {'run': 124, 'job': 5})]:
            altered = dict(plan, **{field: value})
            with self.subTest(field=field), self.assertRaises(ValueError):
                ci.validate_plan(altered, env, self.root)


class SelectionAndGateTests(unittest.TestCase):
    def test_unknown_and_executable_documents_are_full(self):
        policy = json.loads((ci.ROOT / ci.POLICY).read_text())
        for path in ['docs/verification/data/a.md', 'docs/a.py', 'docs/new-kind/a.md',
                     '.github/ci-scopes.json', 'scripts/ci_policy.py', 'go.mod',
                     'internal/core.go', 'unknown.txt']:
            with self.subTest(path=path):
                self.assertEqual(ci.classify([path], policy)[0], 'full')
        self.assertEqual(ci.classify([], policy)[0], 'full')

    def test_real_geography_path_and_new_consumer(self):
        policy = json.loads((ci.ROOT / ci.POLICY).read_text())
        path = policy['domains']['country']['frontend'][0]
        self.assertEqual(ci.classify([path, 'docs/README.md'], policy)[0], 'selected')
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            file = root / path
            file.parent.mkdir(parents=True)
            file.write_text('')
            other = root / 'apps/web/src/other.ts'
            other.write_text("export * from './features/organization/Countries'")
            self.assertEqual(ci.classify([path], policy, root=root)[0], 'selected')
            file.unlink()
            self.assertEqual(ci.classify([path], policy, root=root)[0], 'selected')

    def test_consumer_group_cannot_silently_disappear(self):
        policy = {'geography': {'browser_files': ['core.spec.ts'], 'extensions': {'apps/Feature.tsx': ['new.spec.ts']}}}
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            (root / 'tests/e2e').mkdir(parents=True)
            (root / 'tests/e2e/core.spec.ts').touch()
            self.assertEqual(ci.browser_files(policy, root), ['core.spec.ts'])
            (root / 'apps').mkdir()
            (root / 'apps/Feature.tsx').touch()
            with self.assertRaises(ValueError):
                ci.browser_files(policy, root)

    def test_pagination_requires_every_page(self):
        api = ci.GitHub('owner/repo')
        with patch.object(api, 'json', side_effect=[{'jobs': list(range(100)), 'total_count': 101},
                                                  {'jobs': [100], 'total_count': 101}]) as calls:
            self.assertEqual(len(api.pages('jobs?filter=all', 'jobs')), 101)
            self.assertIn('&per_page=100&page=2', calls.call_args.args[0])
        with patch.object(api, 'json', return_value={'jobs': [1], 'total_count': 2}), self.assertRaises(ValueError):
            api.pages('jobs', 'jobs')

    def test_all_attempts_preserve_success_but_latest_failure_overrides(self):
        ci.require_jobs(jobs(['a', 'b']) + jobs(['b'], 2), ['a', 'b'])
        for conclusion in ['failure', 'skipped', 'cancelled', None]:
            with self.subTest(conclusion=conclusion), self.assertRaises(ValueError):
                ci.require_jobs(jobs(['a', 'b']) + jobs(['b'], 2, conclusion), ['a', 'b'])

    def test_full_gate_requires_exact_jobs_and_results(self):
        plan = dict(scope='full', event='push', integration_shards=8)
        env = {'GITHUB_RUN_ID': '1'}
        names = ['preflight'] + ci.job_names(plan, 'integration')
        results = {'quality': 'success', 'preflight': 'success', 'integration-shard': 'success'}
        ci.gate(plan, 'integration', results, API(jobs=jobs(names)), env)
        for bad in [jobs(names[1:]), jobs(names[:-1]), jobs(names + ['integration shard 9/8']), jobs(names, conclusion='skipped')]:
            with self.subTest(bad=bad[-1]), self.assertRaises(ValueError):
                ci.gate(plan, 'integration', results, API(jobs=bad), env)
        for result in ['skipped', 'cancelled', 'failure', None]:
            with self.subTest(preflight=result), self.assertRaises(ValueError):
                ci.gate(plan, 'integration', dict(results, preflight=result), API(jobs=jobs(names)), env)
            with self.subTest(result=result), self.assertRaises(ValueError):
                ci.gate(plan, 'integration', dict(results, **{'integration-shard': result}), API(jobs=jobs(names)), env)

    def test_docs_na_requires_quality_and_expected_skip(self):
        plan = dict(remote_browser=False, scope='docs', event='pull_request', pr=1, head='head', base='base', reason='admitted docs', integration_shards=0, browser_shards=0)
        pr = dict(state='open', draft=False, head={'sha': 'head'}, base={'sha': 'base', 'ref': 'main'})
        valid = {'quality': 'success', 'preflight': 'skipped', 'integration': 'success', 'browser-shard': 'skipped'}
        self.assertIn('not executed', ci.gate(plan, 'browser', valid, API(pr), {}))
        for key, value in [('quality', 'failure'), ('preflight', 'success'), ('integration', 'cancelled'), ('browser-shard', 'success')]:
            with self.subTest(key=key), self.assertRaises(ValueError):
                ci.gate(plan, 'browser', dict(valid, **{key: value}), API(pr), {})

    def test_selected_suite_omission_must_be_explicit(self):
        plan = dict(scope='selected', event='push', integration_shards=0, browser_shards=1, reason='UI only')
        result = ci.gate(plan, 'integration', {'quality': 'success', 'preflight': 'skipped', 'integration-shard': 'skipped'}, API(), {})
        self.assertIn('not executed', result)
        for missing in [dict(plan, integration_shards=4), {k: v for k, v in plan.items() if k != 'integration_shards'}]:
            with self.assertRaises(ValueError):
                ci.gate(missing, 'integration', {'quality': 'success', 'preflight': 'skipped', 'integration-shard': 'skipped'}, API(), {})

    def test_manual_jobs_cannot_satisfy_formal_gate(self):
        plan = dict(scope='full', event='push', integration_shards=8)
        names = ['diagnostic preflight'] + ci.job_names(dict(plan, event='workflow_dispatch'), 'integration')
        with self.assertRaises(ValueError):
            ci.gate(plan, 'integration', {'quality': 'success', 'preflight': 'success', 'integration-shard': 'success'}, API(jobs=jobs(names)), {'GITHUB_RUN_ID': '1'})

    def test_runner_rollout_keeps_the_same_shards_and_limits_private_runner_access(self):
        plan = dict(repo='owner/repo', event='pull_request', ref='refs/pull/1/merge', browser_shards=4, integration_shards=8)
        repo = dict(full_name='owner/repo', private=True)
        event = dict(repository=repo, pull_request=dict(head=dict(repo=repo)))
        hosted = ci.shard_routes(plan, event, 'browser', '')
        self.assertTrue(all(r['runner'] == ['ubuntu-24.04'] for r in hosted))
        for setting, expected in [('1', 1), ('11', 4)]:
            routes = ci.shard_routes(plan, event, 'browser', setting)
            self.assertEqual([r['shard'] for r in routes], [1, 2, 3, 4])
            self.assertEqual(sum('chnhcm-ci-ephemeral' in r['runner'] for r in routes), expected)
        db = ci.shard_routes(plan, event, 'integration', '3')
        self.assertEqual([r['shard'] for r in db if 'self-hosted' in r['runner']], [1, 2, 3])
        with self.assertRaises(ValueError):
            ci.shard_routes(plan, event, 'integration', '9')
        for source in [dict(event, repository=dict(repo, private=False)),
                       dict(event, pull_request=dict(head=dict(repo=None))),
                       dict(event, pull_request=dict(head=dict(repo=dict(full_name='fork/repo'))))]:
            self.assertEqual(ci.shard_routes(plan, source, 'browser', '11'), hosted)
            self.assertTrue(all(r['runner'] == ['ubuntu-24.04'] for r in ci.shard_routes(plan, source, 'integration', '8')))
        self.assertEqual(ci.shard_routes(dict(plan, event='workflow_dispatch', ref='refs/heads/topic'), event, 'browser', '11'), hosted)
        for trigger in ['push', 'workflow_dispatch']:
            main = dict(plan, event=trigger, ref='refs/heads/main')
            self.assertIn('self-hosted', ci.shard_routes(main, event, 'browser', '1')[0]['runner'])
        unknown = dict(plan, event='workflow_run', ref='refs/heads/main')
        self.assertEqual(ci.shard_routes(unknown, event, 'browser', '11'), hosted)
        for setting in ['12', '-1', 'ubuntu-24.04', '1.5']:
            with self.assertRaises(ValueError):
                ci.shard_routes(plan, event, 'browser', setting)

    def test_plan_records_effective_routes_before_any_shard_starts(self):
        repo = dict(private=True, full_name='owner/repo')
        event = dict(repository=repo, pull_request=dict(head=dict(repo=repo)))
        plan = dict(remote_browser=True, repo='owner/repo', event='pull_request', scope='full', browser_shards=11, integration_shards=8)
        env = dict(CI_BROWSER_SELF_HOSTED_SHARDS='11', CI_INTEGRATION_SELF_HOSTED_SHARDS='3')
        for candidate, source, expected in [
                (plan, event, (11, 3)),
                (dict(plan, scope='selected', browser_shards=2, integration_shards=0), event, (2, 0)),
                (plan, dict(event, repository=dict(repo, private=False)), (0, 0))]:
            outputs = ci.plan_outputs(candidate, source, env)
            self.assertEqual(outputs['plan']['self_hosted_shards'], dict(zip(['browser', 'integration'], expected)))
            self.assertEqual(outputs['preflight_runner'], outputs['browser_matrix'][0]['runner'])


class ReuseTests(unittest.TestCase):
    def setUp(self):
        self.now = datetime.now(timezone.utc)
        self.plan = dict(remote_browser=True, repo='owner/repo', tree='tree', contract='contract')
        prior = dict(remote_browser=True, reason='test full', schema=2, scope='full', event='pull_request', repo='owner/repo', pr=1, head='head', base='base',
                     tree='tree', contract='contract', run_id='123', ref='refs/pull/1/merge', integration_shards=8,
                     browser_shards=11, integration_filter='^Test', browser_files=[])
        self.evidence = dict(schema=2, plan=prior, attempt=1)
        self.info = dict(id=123, created_at=self.now.isoformat(), run_attempt=1, event='pull_request', path=ci.WORKFLOW,
                         head_sha='head', status='completed', conclusion='success', pull_requests=[{'number': 1}])
        self.pr = dict(number=1, head={'sha': 'head'})

    def matches(self, evidence=None, info=None, parents=None, now=None):
        return ci.reuse_matches(self.plan, evidence or self.evidence, info or self.info, self.pr,
                                parents or ['base', 'head'], now or self.now)

    def test_local_browser_evidence_cannot_satisfy_an_explicit_main_request(self):
        self.evidence['plan']['remote_browser'] = False
        self.assertFalse(self.matches())
        self.plan['remote_browser'] = False
        self.assertTrue(self.matches())

    def test_full_matching_candidate(self):
        self.assertTrue(self.matches())
        self.assertTrue(self.matches(info=dict(self.info, pull_requests=[])))

    def test_selected_and_docs_proofs_require_recomputed_selection_and_complete_jobs(self):
        plan = dict(self.plan, checkout='merged')
        pr = dict(self.pr, merged_at=self.now.isoformat(), merge_commit_sha='merged', base={'ref': 'main'},
                  head={'sha': 'head', 'repo': {'full_name': 'owner/repo'}})
        for scope, integration, browser in [('docs', 0, 0), ('selected', 0, 1), ('selected', 4, 2), ('selected', 8, 0)]:
            evidence = copy.deepcopy(self.evidence)
            prior = evidence['plan']
            prior.update(scope=scope, integration_shards=integration, browser_shards=browser,
                         integration_filter='^Test' if integration else '',
                         browser_files=['country200.spec.ts'] if browser else [])
            names = ['quality', 'integration', 'browser'] + ci.job_names(prior, 'integration') + ci.job_names(prior, 'browser')
            archive = io.BytesIO()
            with zipfile.ZipFile(archive, 'w') as z:
                z.writestr('ci-evidence.json', json.dumps(evidence))
            api = API()
            api.raw = lambda path: archive.getvalue()
            for change in ['none', 'missing_job', 'broader_diff', 'failed_latest_attempt']:
                expected = dict(prior)
                actual_jobs = jobs(names)
                if change == 'missing_job':
                    actual_jobs.pop()
                elif change == 'broader_diff':
                    expected['browser_files'] = ['country200.spec.ts', 'department-journeys.spec.ts']
                elif change == 'failed_latest_attempt':
                    actual_jobs += jobs([names[-1]], attempt=2, conclusion='failure')
                pages = [[pr], [self.info], [{'name': 'ci-evidence-1', 'id': 9, 'expired': False}], actual_jobs]
                with self.subTest(scope=scope, change=change), patch.object(ci, 'git', return_value='base head'), \
                        patch.object(api, 'pages', side_effect=pages), patch.object(ci, 'selection_for', return_value=expected):
                    if change in {'missing_job', 'failed_latest_attempt'}:
                        with self.assertRaises(ValueError):
                            ci.find_reuse(plan, api)
                    else:
                        self.assertEqual(ci.find_reuse(plan, api), '123' if change == 'none' else '')

    def test_partial_or_different_proof_cannot_reuse(self):
        for field, value in [('scope', 'reuse'), ('event', 'workflow_dispatch'), ('base', 'other'),
                             ('head', 'other'), ('tree', 'other'), ('contract', 'other'), ('repo', 'other/repo'),
                             ('run_id', '999'), ('pr', 2), ('ref', 'refs/heads/main')]:
            evidence = copy.deepcopy(self.evidence)
            evidence['plan'][field] = value
            with self.subTest(field=field):
                self.assertFalse(self.matches(evidence=evidence))
        self.assertFalse(self.matches(parents=['head']))
        self.assertFalse(self.matches(now=self.now + timedelta(days=8)))
        self.assertFalse(self.matches(now=self.now - timedelta(seconds=1)))

    def test_failed_cancelled_or_other_attempt_cannot_reuse(self):
        for field, value in [('run_attempt', 2), ('conclusion', 'failure'), ('status', 'in_progress'),
                             ('event', 'workflow_dispatch'), ('path', 'different.yml'), ('head_sha', 'other'),
                             ('pull_requests', [{'number': 2}])]:
            with self.subTest(field=field):
                self.assertFalse(self.matches(info=dict(self.info, **{field: value})))

    def test_archive_does_not_extract_paths_or_accept_extra_files(self):
        for filenames in [['ci-evidence.json'], ['../ci-evidence.json'], ['ci-evidence.json', 'extra.json']]:
            data = io.BytesIO()
            with zipfile.ZipFile(data, 'w') as z:
                for name in filenames:
                    z.writestr(name, json.dumps(self.evidence))
            if filenames == ['ci-evidence.json']:
                self.assertEqual(ci.unpack_evidence(data.getvalue()), self.evidence)
            else:
                with self.assertRaises(ValueError):
                    ci.unpack_evidence(data.getvalue())

    def test_reuse_queries_source_run_and_all_actual_shards(self):
        plan = dict(self.plan, checkout='merged')
        pr = dict(self.pr, merged_at=self.now.isoformat(), merge_commit_sha='merged', base={'ref': 'main'},
                  head={'sha': 'head', 'repo': {'full_name': 'owner/repo'}})
        source_jobs = jobs(['quality', 'integration', 'browser'] + ci.job_names(self.evidence['plan'], 'integration')
                           + ci.job_names(self.evidence['plan'], 'browser') + ['preflight'])
        data = io.BytesIO()
        with zipfile.ZipFile(data, 'w') as z:
            z.writestr('ci-evidence.json', json.dumps(self.evidence))
        api = API()
        api.raw = lambda path: data.getvalue()
        for missing in [False, True]:
            values = [[pr], [self.info], [{'name': 'ci-evidence-1', 'id': 9, 'expired': False}],
                      source_jobs[:-1] if missing else source_jobs]
            with patch.object(ci, 'git', return_value='base head'), patch.object(api, 'pages', side_effect=values), patch.object(ci, 'selection_for', return_value=self.evidence['plan']):
                if missing:
                    with self.assertRaises(ValueError):
                        ci.find_reuse(plan, api)
                else:
                    self.assertEqual(ci.find_reuse(plan, api), '123')
        with patch.object(ci, 'git', return_value='base head'), patch.object(api, 'pages', side_effect=[[pr], [self.info], []]):
            self.assertEqual(ci.find_reuse(plan, api), '')
