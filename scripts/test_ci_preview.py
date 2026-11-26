import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from ci_policy import POLICY
from ci_preview import API_CLIENT, preview, response_texts
from ci_ready import ReadyRun, candidate, check_generated, check_ready, execute_ready


API = """paths:
  /api/v1/jobs:
    get:
      operationId: listJob
      responses:
        '200':
          description: Successful response
          content: {$ref: '#/components/schemas/JobList'}
    post:
      operationId: createJob
      responses:
        '200':
          description: Successful write
"""
CLIENT = """export interface operations {
    listJob: {
        parameters: {};
        responses: {
            /** @description Successful response */
            200: {
            };
        };
    };
}
"""
INDEX = """-- +goose Up
CREATE INDEX job_read_active_date ON staffing.job_read (active, start_date);
-- +goose Down
DROP INDEX staffing.job_read_active_date;
"""


class PreviewTests(unittest.TestCase):
    def setUp(self):
        folder = tempfile.TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        self.root = Path(folder.name)
        self.git('init', '-q')
        for name, value in [('user.name', 'test'), ('user.email', 'test@example.invalid')]:
            self.git('config', name, value)
        self.write(POLICY, json.dumps(dict(domains={
            d: dict(frontend=[], browser_files=[d+'.spec.ts'], backend_prefixes=[], backend_consumers=[d])
            for d in ('staffing', 'country')}, shared_browser_files=[], business_tools={},
            integration_prefixes={'staffing': ['Staffing'], 'country': ['Country']})))
        self.write('.gitignore', '.local/\n')
        self.write('api/openapi.yaml', API)
        self.write(API_CLIENT, CLIENT)
        self.sql = 'db/migrations/00002_index.sql'
        self.write('db/migrations/00001_base.sql', '-- baseline')
        self.write('tests/e2e/staffing.spec.ts', '')
        self.write('tests/e2e/country.spec.ts', '')
        self.git('add', '.')
        self.git('commit', '-qm', 'baseline')
        self.base = self.git('rev-parse', 'HEAD')

    def git(self, *args):
        return subprocess.check_output(['git', *args], text=True, cwd=self.root).strip()

    def write(self, path, value):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(value)

    def report(self, head=None):
        return preview(self.base, head, self.root)

    def test_shadow_reads_only_admitted_response_and_preserves_other_full_triggers(self):
        changed = API.replace('Successful response', 'Current jobs')
        self.write('api/openapi.yaml', changed)
        self.assertEqual(self.report()['shadow']['scope'], 'full')
        self.write(API_CLIENT, CLIENT.replace('Successful response', 'Current jobs'))
        p = self.report()
        self.assertEqual(p['current']['scope'], 'full')
        self.assertEqual((p['shadow']['integration_shards'], p['shadow']['browser_files']), (4, ['staffing.spec.ts']))
        for other in [changed.replace('JobList', 'AccessPolicy'), changed.replace('createJob', 'replaceJob'),
                      changed.replace('Successful write', 'Updated'), API.replace('Successful response', '*alias'),
                      changed + 'security: []\n']:
            self.assertNotEqual(response_texts(other)[0], response_texts(API)[0])
        self.write(API_CLIENT, CLIENT.replace('Successful response', 'Other change'))
        self.assertEqual(self.report()['shadow']['scope'], 'full')

    def test_migration_requires_matching_down_and_only_limits_browser_layer(self):
        self.write(self.sql, INDEX)
        plan = self.report()['shadow']
        self.assertEqual((plan['scope'], plan['integration_filter'], plan['integration_shards'],
                          plan['browser_files']), ('selected', '^Test', 4, ['staffing.spec.ts']))
        for bad in [INDEX.replace('staffing.job_read (', 'access.grants ('),
                    INDEX.replace('CREATE INDEX', 'CREATE UNIQUE INDEX'), INDEX + 'DELETE FROM staffing.job;',
                    INDEX.replace('active, start_date', 'lower(name)'), INDEX.replace('DROP INDEX staffing.', 'DROP INDEX access.')]:
            self.write(self.sql, bad)
            self.assertEqual(self.report()['shadow']['scope'], 'full')
        self.write('db/migrations/00001_base.sql', INDEX)
        (self.root / self.sql).unlink()
        self.assertEqual(self.report()['shadow']['scope'], 'full')

    def test_working_copy_and_committed_replay_use_their_own_inventory(self):
        self.write(self.sql, INDEX)
        self.git('add', self.sql)
        self.git('commit', '-qm', 'index')
        self.write('tests/e2e/unmapped.spec.ts', '')
        self.assertEqual(self.report()['shadow']['scope'], 'full')
        replay = self.report('HEAD')
        self.assertFalse(replay['working_tree'])
        self.assertEqual(replay['changed_files'], 1)
        self.assertEqual(replay['shadow']['browser_files'], ['staffing.spec.ts'])
        self.write('notes.md', '')
        self.git('add', 'notes.md')
        (self.root / 'link').symlink_to('notes.md')
        self.assertEqual(self.report()['current']['reason'], 'empty diff or non-regular entry')

    def test_copy_detects_new_and_different_generated_files_without_mutating_candidate(self):
        self.write('Makefile', 'generate:\n\tcp input output\n\tcp input new-output\n')
        self.write('input', 'expected')
        self.write('output', 'developer edit')
        before = self.git('status', '--porcelain')
        with self.assertRaisesRegex(SystemExit, '(?s)new-output.*output'):
            check_generated(self.root)
        self.assertEqual((self.root / 'output').read_text(), 'developer edit')
        self.assertEqual(before, self.git('status', '--porcelain'))

    def test_ready_only_runs_regressions_present_in_a_selected_browser_plan(self):
        record = ReadyRun(self.root)
        with patch('ci_ready.check_generated') as generate, patch('ci_ready.check_inventory', return_value={}), \
                patch('ci_ready.discover', return_value={('chromium', 'file', 'id'): 'file'}) as inventory, \
                patch('ci_ready.run_check') as run:
            (record.folder / 'browser.jsonl').write_text('{"id":"id","project":"chromium","status":"passed"}\n')
            for files in [['country.spec.ts'], ['staffing314.spec.ts']]:
                check_ready(dict(scope='selected', browser_shards=1, browser_files=files), record, self.root)
            self.assertEqual(generate.call_count, 2)
            inventory.assert_called_once()
            run.assert_called_once()
            args = run.call_args.args[0]
            self.assertIn('scripts/local-bu.py', args)
            self.assertIn('899px', args[-1])
            self.assertNotIn('139', args[-1])

    def test_ready_identifies_working_inputs_without_staging_and_matches_commit_tree(self):
        clean = candidate(self.base, self.root)
        self.write('api/openapi.yaml', 'staged')
        self.git('add', 'api/openapi.yaml')
        self.write('api/openapi.yaml', 'working')
        self.write('new file', 'content')
        (self.root / API_CLIENT).unlink()
        before = self.git('diff', '--cached')
        dirty = candidate(self.base, self.root)
        self.assertTrue(dirty['dirty'])
        self.assertNotEqual(clean['tree'], dirty['tree'])
        self.assertEqual(self.git('diff', '--cached'), before)
        self.git('add', '-A')
        self.git('commit', '-qm', 'candidate')
        self.assertEqual(dirty['tree'], self.git('rev-parse', 'HEAD^{tree}'))

    def test_no_candidate_and_changed_during_execution(self):
        folder = execute_ready(self.base, self.root)
        result = json.loads((folder / 'result.json').read_text())
        self.assertEqual((result['status'], result['scope'], result['steps']), ('no_candidate', 'none', []))
        self.write('docs/guides/notes.md', 'first')
        with patch('ci_ready.check_ready', side_effect=lambda *_: self.write('docs/guides/notes.md', 'second')):
            with self.assertRaisesRegex(ValueError, '候选已变化'):
                execute_ready(self.base, self.root)
        reports = [json.loads(p.read_text()) for p in folder.parent.glob('*/result.json')]
        self.assertEqual(sorted(r['status'] for r in reports), ['failed', 'no_candidate'])

    def test_generated_detects_deleted_outputs_and_accepts_identical_rewrites(self):
        self.write('original', 'keep')
        self.write('Makefile', 'generate:\n\tcp original output\n')
        self.write('output', 'keep')
        check_generated(self.root)
        self.write('Makefile', 'generate:\n\trm output\n')
        with self.assertRaisesRegex(SystemExit, 'output'):
            check_generated(self.root)
        self.assertEqual((self.root / 'output').read_text(), 'keep')

    def test_symlink_generation_input_and_hidden_index_entry_are_rejected(self):
        self.write('Makefile', 'generate:\n\t@true\n')
        (self.root / 'link').symlink_to('Makefile')
        with self.assertRaisesRegex(ValueError, 'generation input'):
            check_generated(self.root)
        self.git('update-index', '--assume-unchanged', 'api/openapi.yaml')
        self.write('api/openapi.yaml', 'hidden')
        with self.assertRaisesRegex(ValueError, '索引隐藏'):
            candidate(self.base, self.root)

    def test_interruption_and_parallel_directories_preserve_attempts(self):
        from concurrent.futures import ThreadPoolExecutor
        with ThreadPoolExecutor(max_workers=2) as pool:
            attempts = list(pool.map(ReadyRun, [self.root, self.root]))
        self.assertNotEqual(attempts[0].folder, attempts[1].folder)
        self.write('docs/guides/notes.md', 'candidate')
        with patch('ci_ready.run_check', side_effect=KeyboardInterrupt), self.assertRaises(KeyboardInterrupt):
            execute_ready(self.base, self.root)
        reports = [json.loads(p.read_text()) for p in attempts[0].folder.parent.glob('*/result.json')]
        interrupted, = [r for r in reports if r['status'] == 'interrupted']
        self.assertEqual(interrupted['steps'][0]['status'], 'interrupted')
        self.assertEqual(interrupted['first_failure'], 'docs')

    def test_failed_attempt_is_retained_when_identical_candidate_passes(self):
        self.write('Makefile', '.PHONY: docs\ndocs:\n\t@test -f .local/ready\n')
        self.git('add', 'Makefile')
        self.git('commit', '-qm', 'test document checker')
        base = self.git('rev-parse', 'HEAD')
        self.write('docs/guides/example.md', 'candidate')
        with self.assertRaises(subprocess.CalledProcessError):
            execute_ready(base, self.root)
        first_path, = (self.root / '.local/ci-ready').glob('*/result.json')
        first_bytes = first_path.read_bytes()
        self.write('.local/ready', '')
        second = execute_ready(base, self.root)
        first = json.loads(first_bytes)
        self.assertEqual(first_path.read_bytes(), first_bytes)
        self.assertEqual(first['first_failure'], 'docs')
        retry = json.loads((second / 'result.json').read_text())
        self.assertEqual(first['candidate'], retry['candidate'])
        self.assertEqual((first['status'], retry['status']), ('failed', 'passed'))

    def test_generation_compares_content_even_if_length_and_timestamp_match(self):
        self.write('output', 'old')
        self.write('Makefile', 'generate:\n\tcp -p output stamp\n\tprintf new >output\n\ttouch -r stamp output\n\trm stamp\n')
        with self.assertRaisesRegex(SystemExit, 'output'):
            check_generated(self.root)
        self.assertEqual((self.root / 'output').read_text(), 'old')
