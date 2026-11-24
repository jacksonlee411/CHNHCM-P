import copy
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from datetime import datetime, timezone
from unittest.mock import patch
import zipfile

import ci_coverage as coverage
import ci_policy as policy
from ci_impact import full, journey_consumers


class Runs:
    def __init__(self):
        self.info = {}
        self.jobs = {}
        self.records = {}
        self.final = {}

    def pages(self, path, key=None):
        if path.startswith('commits/'):
            return [self.pr]
        if key == 'workflow_runs':
            return [v for v in self.info.values() if 'status=success' not in path or v.get('conclusion') == 'success']
        rid = int(path.split('/')[2])
        if key == 'artifacts':
            artifacts = [dict(id=rid, name=f'ci-coverage-{self.records[rid]["attempt"]}', expired=False)]
            if rid in self.final:
                artifacts.append(dict(id=1000 + rid, name='ci-evidence-1', expired=False))
            return artifacts
        return self.jobs[rid]

    def raw(self, path):
        buf = io.BytesIO()
        aid = int(path.split('/')[2])
        with zipfile.ZipFile(buf, 'w') as out:
            if aid >= 1000:
                out.writestr('ci-evidence.json', json.dumps(self.final[aid - 1000]))
            else:
                out.writestr('ci-coverage.json', json.dumps(self.records[aid]))
        return buf.getvalue()

    def json(self, path):
        if path.startswith('pulls/'):
            return self.pr
        return self.info[int(path.split('/')[-1])]


class CoverageTests(unittest.TestCase):
    def setUp(self):
        self.plan = dict(remote_browser=True, schema=2, repo='owner/repo', event='pull_request', ref='refs/pull/4/merge',
                         pr=4, contract='policy', run_id='30', tree='current', head='head', base='base',
                         **full('complete diff'))
        self.inventories = {kind: [coverage.inventory_part([str(i)], [f'Test{i}' if kind == 'integration'
                                                                     else f'case{i}.spec.ts'])
                                   for i in range(count + 1)] for kind, count in coverage.SUITES.items()}
        self.units = coverage.inventory_units(self.inventories)
        self.api = Runs()
        self.add_source(10)
        self.delta = {}
        self.selection = patch.object(coverage, 'delta_from',
                                      side_effect=lambda prior, *_: self.delta.get(int(prior['run_id'])))
        self.selection.start()
        self.addCleanup(self.selection.stop)

    def add_source(self, rid):
        plan = dict(self.plan, run_id=str(rid), coverage_attempt=1, carried={})
        self.api.info[rid] = dict(id=rid, head_sha=plan['head'], event='pull_request', path=coverage.WORKFLOW,
                                  head_repository=dict(full_name=plan['repo']), pull_requests=[])
        self.api.records[rid] = dict(attempt=1, plan=plan, inventories=copy.deepcopy(self.inventories))
        self.api.jobs[rid] = [dict(name=name, run_attempt=1, id=rid * 100 + i, runner_id=999,
                                   status='completed', conclusion='success')
                              for i, name in enumerate(['quality', *self.units])]

    def job(self, rid, name):
        return next(j for j in self.api.jobs[rid] if j['name'] == name)

    def credits(self):
        return coverage.find_carried(self.api, self.plan, 'feature/fix', self.inventories)

    def test_document_followup_gates_and_main_reuse_require_real_source_jobs(self):
        with tempfile.TemporaryDirectory() as folder:
            for kind, parts in self.inventories.items():
                Path(folder, f'ci-{kind}.json').write_text(json.dumps(parts))
            event = dict(pull_request=dict(head=dict(repo={'full_name': 'owner/repo'}, ref='feature/fix')))
            result = policy.schedule(self.plan, event, dict(RUNNER_TEMP=folder, GITHUB_RUN_ATTEMPT='1'), self.api)
            candidate = result['plan']
            self.api.records[30] = json.loads(Path(folder, 'ci-coverage.json').read_text())
        self.assertEqual(set(candidate['carried']), set(self.units))
        self.assertFalse(any(result[key] for key in ('run_preflight', 'run_integration', 'run_browser')))
        self.api.pr = dict(state='open', draft=False, head={'sha': 'head'}, base={'sha': 'base', 'ref': 'main'})
        for kind in coverage.SUITES:
            results = {'quality': 'success', 'integration': 'success', 'preflight': 'skipped', kind+'-shard': 'skipped'}
            self.assertIn('previous successful', policy.gate(candidate, kind, results, self.api, {}))
        coverage.verify_carried(self.api, candidate, self.units)
        self.api.pr.update(number=4, merged_at=datetime.now(timezone.utc).isoformat(), merge_commit_sha='merged')
        self.api.pr['head']['repo'] = {'full_name': 'owner/repo'}
        self.api.info[30] = dict(self.api.info[10], id=30, created_at=self.api.pr['merged_at'], run_attempt=1,
                                 status='completed', conclusion='success')
        self.api.jobs[30] = [dict(self.job(10, 'quality'), name=n) for n in ('quality', 'integration', 'browser')]
        self.api.final[30] = dict(schema=2, attempt=1, plan=candidate)
        with patch.object(policy, 'git', return_value='base head'), patch.object(policy, 'selection_for', return_value=candidate):
            self.assertEqual(policy.find_reuse(dict(self.plan, checkout='merged'), self.api), '30')
        # A later failed partial rerun invalidates the prior success at the gate.
        self.api.jobs[10].append(dict(self.job(10, 'browser shard 2/16'), run_attempt=2, conclusion='failure'))
        with self.assertRaisesRegex(ValueError, 'prior|previous'):
            coverage.verify_carried(self.api, candidate, self.units)

    def test_failed_cancelled_unallocated_and_changed_shards_run_on_new_candidate(self):
        self.add_source(20)
        self.job(20, 'browser shard 1/16')['conclusion'] = 'failure'
        self.job(20, 'browser shard 2/16')['conclusion'] = 'cancelled'
        self.job(20, 'integration shard 2/2')['runner_id'] = None
        self.delta[20] = dict(browser_files=['case3.spec.ts'], browser_shards=1, integration_shards=0)
        self.api.records[20]['inventories']['browser'][4] = coverage.inventory_part(['old-id'], ['case4.spec.ts'])
        expected = {'browser shard 1/16', 'browser shard 2/16', 'browser shard 3/16',
                    'browser shard 4/16', 'integration shard 2/2'}
        carried = self.credits()
        self.assertEqual(set(self.units) - carried.keys(), expected)
        output = policy.plan_outputs(dict(self.plan, carried=carried), {'pull_request': {'head': {}}}, {})
        self.assertEqual([r['shard'] for r in output['browser_matrix']], [1, 2, 3, 4])
        self.assertEqual([r['shard'] for r in output['integration_matrix']], [2])
        self.assertFalse(output['run_preflight'])

    def test_inherited_slots_bind_to_the_actual_run_that_executed_them(self):
        self.add_source(20)
        self.api.jobs[20] = [self.job(20, 'quality'), self.job(20, 'browser shard 4/16')]
        self.delta[10] = dict(browser_files=[], browser_shards=0, integration_shards=4, integration_filter='^Test1$')
        got = self.credits()
        self.assertNotIn('integration shard 1/2', got)
        self.assertEqual(got['browser shard 4/16']['run'], 20)
        self.assertEqual(got['browser shard 1/16']['run'], 10)
        self.assertEqual(got['preflight']['run'], 10)

    def test_newer_failure_from_a_different_partition_reopens_its_layer(self):
        self.add_source(20)
        self.api.jobs[20] = [self.job(20, 'quality'),
                             dict(self.job(20, 'browser shard 1/16'), name='browser shard 1/2', conclusion='failure')]
        self.assertEqual(set(self.credits()), {f'integration shard {n}/2' for n in range(1, 3)})

    def test_other_pr_fork_contract_missing_jobs_and_diagnostic_have_no_credit(self):
        for field, value in [('pr', 2), ('contract', 'changed'), ('scope', 'selected'), ('ref', 'refs/heads/main')]:
            with self.subTest(field=field), patch.dict(self.api.records[10]['plan'], {field: value}):
                self.assertFalse(self.credits())
        for field, value in [('event','workflow_dispatch'), ('head_repository', {'full_name': 'fork/repo'}),
                             ('path','another.yml'), ('id',31), ('pull_requests',[{'number': 2}])]:
            with self.subTest(field=field), patch.dict(self.api.info[10], {field: value}):
                self.assertFalse(self.credits())
        with patch.dict(self.api.records[10], attempt=2):
            self.assertFalse(self.credits())
        with patch.object(coverage, 'checkpoint', return_value=None):
            self.assertFalse(self.credits())
        self.job(10, 'quality')['conclusion'] = 'failure'
        self.assertFalse(self.credits())

    def test_shared_runtime_changes_require_every_slot_and_preflight(self):
        self.delta[10] = full('shared source')
        self.assertFalse(self.credits())
        self.delta[10] = dict(integration_shards=0, browser_shards=1, browser_files=['case0.spec.ts'])
        self.assertEqual(set(self.units) - self.credits().keys(), {'preflight'})

    def test_light_quality_checks_both_merge_trees_after_a_docs_only_push(self):
        previous = 'a' * 40
        self.api.info[10]['head_sha'] = self.api.records[10]['plan']['head'] = previous
        with patch.object(coverage, 'selection_for', return_value={'scope': 'docs'}):
            record, source = coverage.documentation_checkpoint(self.api, self.plan, previous, Path('.'))
            self.assertEqual(record['inventories'], self.inventories)
            self.assertEqual(source['job'], self.job(10, 'quality')['id'])
            self.delta[10] = full('new migration on base')
            self.assertIsNone(coverage.documentation_checkpoint(self.api, self.plan, previous, Path('.')))
        with patch.object(coverage, 'selection_for', return_value={'scope': 'selected'}):
            self.assertIsNone(coverage.documentation_checkpoint(self.api, self.plan, previous, Path('.')))

    def test_multiple_docs_pushes_reference_the_original_runtime_quality(self):
        self.add_source(20)
        previous = 'a' * 40
        self.api.info[20]['head_sha'] = self.api.records[20]['plan']['head'] = previous
        source = {'run': 10, 'job': self.job(10, 'quality')['id']}
        self.api.records[20]['plan']['quality_source'] = source
        with patch.object(coverage, 'selection_for', return_value={'scope': 'docs'}):
            actual, ref = coverage.documentation_checkpoint(self.api, self.plan, previous, Path('.'))
            self.assertEqual((actual, ref), (self.api.records[10], source))

    def test_explicit_fresh_full_schedules_all_jobs_with_existing_green_evidence(self):
        event = dict(pull_request=dict(head=dict(repo={'full_name': 'owner/repo'}, ref='feature/fix')))
        with tempfile.TemporaryDirectory() as folder:
            for kind, parts in self.inventories.items():
                Path(folder, f'ci-{kind}.json').write_text(json.dumps(parts))
            outputs = policy.schedule(dict(self.plan, fresh_full=True), event,
                                      dict(RUNNER_TEMP=folder, GITHUB_RUN_ATTEMPT='1',
                                           CI_FULL_PR_HEAD=self.plan['head']), self.api)
        self.assertFalse(outputs['plan']['carried'])
        self.assertEqual((len(outputs['integration_matrix']), len(outputs['browser_matrix'])), (2, 16))
        self.assertTrue(outputs['run_preflight'])

    def test_browser_approval_does_not_hold_integration_or_preflight(self):
        event = dict(pull_request=dict(head=dict(repo={'full_name': 'owner/repo'}, ref='topic')))
        with tempfile.TemporaryDirectory() as folder:
            for kind, parts in self.inventories.items():
                Path(folder, f'ci-{kind}.json').write_text(json.dumps(parts))
            env = dict(RUNNER_TEMP=folder, GITHUB_RUN_ATTEMPT='1', CI_FULL_PR_HEAD='older-head')
            # Approval for the browser suite must preserve verified DB work.
            db_job = 'integration shard 1/2'
            with patch.object(coverage, 'find_carried', return_value={db_job: self.job(10, db_job)}):
                result = policy.schedule(self.plan, event, dict(env, CI_FULL_BROWSER_HEAD=self.plan['head']), self.api)
                self.assertTrue(result['run_browser'])
                self.assertEqual(len(result['integration_matrix']), 1)
            for carried in ({}, {'preflight': self.job(10, 'preflight')},
                            {'integration shard 1/2': self.job(10, 'integration shard 1/2')},
                            {'browser shard 1/16': self.job(10, 'browser shard 1/16')}):
                with self.subTest(carried=carried), patch.object(coverage, 'find_carried', return_value=carried):
                    result = policy.schedule(dict(self.plan, remote_browser=False), event, env, self.api)
                    self.assertTrue(result['run_integration'])
                    self.assertEqual(result['run_preflight'], 'preflight' not in carried)
                    self.assertFalse(result['run_browser'])
                    self.assertEqual(len(result['integration_matrix']), 2 - int('integration shard 1/2' in carried))

    def test_db_only_preflight_cannot_cover_requested_chromium(self):
        self.api.records[10]['plan']['remote_browser'] = False
        self.api.jobs[10] = [job for job in self.api.jobs[10] if 'browser shard' not in job['name']]
        self.assertEqual(set(self.credits()), {n for n in self.units if n.startswith('integration shard')})
        self.plan['remote_browser'] = False
        self.assertEqual(set(self.credits()), {n for n in self.units if 'integration' in self.units[n]})

    def test_forged_omission_is_rejected_by_the_gate(self):
        credits = self.credits()
        self.api.records[30] = dict(attempt=1, plan=dict(self.plan, carried={}, coverage_attempt=1),
                                    inventories=self.inventories)
        with self.assertRaisesRegex(ValueError, 'current quality'):
            coverage.verify_carried(self.api, dict(self.plan, carried=credits, coverage_attempt=1), self.units)

    def test_journey_uses_transitive_entry_files_and_detects_global_setup(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            folder = root / 'tests/e2e'
            folder.mkdir(parents=True)
            (folder/'grade-journey.ts').touch()
            (folder/'helper.ts').write_text("export * from './grade-journey'")
            (folder/'a.spec.ts').write_text("import('./helper')")
            (folder/'b.spec.ts').write_text("import './grade-journey'")
            (folder/'nested').mkdir()
            (folder/'nested/a.spec.ts').write_text("import '../grade-journey.js'")
            path = 'tests/e2e/grade-journey.ts'
            self.assertEqual(journey_consumers(path, root), {'a.spec.ts', 'b.spec.ts', 'nested/a.spec.ts'})
            (folder/'runtime-setup.ts').write_text("require('./helper')")
            self.assertFalse(journey_consumers(path, root))


class MergeTreeTests(unittest.TestCase):
    def test_reconstructed_merge_and_new_base_use_actual_content(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / 'origin'
            repo.mkdir()
            git = lambda *args: coverage.git(*args, root=repo)
            git('init', '-q', '-b', 'main')
            git('config', 'user.email', 'ci@example.invalid')
            git('config', 'user.name', 'Fixture')
            (repo / '.github').mkdir()
            (repo / coverage.POLICY).write_bytes((coverage.ROOT / coverage.POLICY).read_bytes())
            git('add', '.')
            git('commit', '-qm', 'base')
            git('branch', 'topic')
            (repo / 'docs').mkdir()
            (repo / 'docs/base.md').write_text('base changes before the topic merges')
            git('add', '.')
            git('commit', '-qm', 'main')
            base = git('rev-parse', 'HEAD')
            git('checkout', '-q', 'topic')
            (repo / 'go.mod').write_text('shared runtime')
            git('add', '.')
            git('commit', '-qm', 'feature')
            head = git('rev-parse', 'HEAD')
            prior = dict(base=base, head=head, tree=git('merge-tree', '--write-tree', base, head))
            git('checkout', '-q', 'main')
            (repo / 'docs/base.md').write_text('only documentation advanced')
            git('commit', '-qam', 'new main')
            git('merge', '-q', '--no-ff', '-m', 'current candidate', head)
            current = dict(tree=git('rev-parse', 'HEAD^{tree}'))
            clone = Path(tmp) / 'fresh'
            subprocess.run(['git', 'clone', '--quiet', '--no-local', str(repo), str(clone)], check=True)
            self.assertNotEqual(subprocess.run(['git', 'cat-file', '-e', prior['tree']], cwd=clone).returncode, 0)
            self.assertEqual(coverage.delta_from(prior, current, clone)['scope'], 'docs')
            (clone / 'go.mod').write_text('new shared change')
            coverage.git('add', 'go.mod', root=clone)
            self.assertEqual(coverage.delta_from(prior, dict(tree=coverage.git('write-tree', root=clone)), clone)['scope'], 'full')
