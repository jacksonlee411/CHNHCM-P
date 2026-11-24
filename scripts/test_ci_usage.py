import json
import subprocess
import unittest
from unittest.mock import Mock

import ci_usage as usage


class DeliveryUsageTests(unittest.TestCase):
    def test_placement_uses_the_final_schedule_after_carried_jobs_are_removed(self):
        api = Mock(repo='owner/repo')
        plan = dict(schema=2, repo=api.repo, run_id='1', event='pull_request', pr=19, head='head',
                    scope='full', browser_shards=11, integration_shards=8)
        initial, actual = dict(browser=11, integration=8), dict(browser=1, integration=0)
        api.raw.return_value = '\n'.join('timestamp ' + json.dumps(dict(plan, self_hosted_shards=counts))
                                        for counts in (initial, actual)).encode()
        self.assertEqual(usage.read_placement_plan(api, self.job, 1, self.pr, 19)['self_hosted_shards'], actual)

    def setUp(self):
        self.job = dict(id=100, name='browser shard 1/11', run_attempt=1, status='completed', conclusion='success',
                        runner_id=9, runner_group_name='GitHub Actions', labels=['ubuntu-24.04'],
                        started_at='2026-11-16T07:00:10Z', completed_at='2026-11-16T07:01:11Z')
        self.delivery = dict(pr=19, run_ids=[1, 2], final_run=1, head='head', main_run=2, merge_sha='merge')
        run = dict(status='completed', path=usage.WORKFLOW, created_at='2026-11-16T07:00:00Z',
                   updated_at='2026-11-16T07:03:00Z', conclusion='success')
        self.pr = dict(run, event='pull_request', head_sha='head', run_attempt=2)
        self.main = dict(run, event='push', head_branch='main', head_sha='merge', run_attempt=1)

    def test_all_attempts_and_wall_clock_charge_failed_and_cancelled_jobs(self):
        api = Mock()
        api.json.side_effect = [self.pr, self.main]
        first = dict(self.job, conclusion='failure')
        retry = dict(self.job, id=101, labels=['self-hosted', 'linux'], runner_group_name='Default', run_attempt=2)
        cancelled = dict(self.job, id=102, conclusion='cancelled', completed_at=self.job['started_at'])
        skipped = dict(self.job, id=103, runner_id=None, conclusion='skipped', started_at=None, completed_at=None)
        api.pages.side_effect = [[first, retry], [cancelled, skipped]]
        result = usage.measure(api, self.delivery)
        self.assertEqual(result['minutes'], dict(github_hosted=2, self_hosted=2, unallocated=0))
        self.assertEqual(result['job_counts'], dict(github_hosted=2, self_hosted=1, unallocated=1))
        self.assertEqual(result['delivery_elapsed_seconds'], 71)
        self.assertEqual(result['runs'][1]['wait_seconds'], 10)
        placement, = result['runs'][0]['runner_placements']
        self.assertEqual((placement['job'], placement['attempt'], placement['runner_id'], placement['group']),
                         (101, 2, 9, 'self_hosted'))

    def test_only_an_expanded_allocated_full_pr_matrix_counts(self):
        slots = [dict(self.job, name=f'{suite} shard {i}/{size}', id=id_offset+i)
                 for suite, size, id_offset in [('integration', 8, 10), ('browser', 11, 30)]
                 for i in range(1, size+1)]
        retry = dict(slots[0], run_attempt=2, id=60)
        api = Mock()
        self.delivery.update(run_ids=[3, 1, 2])
        api.json.side_effect = [dict(self.pr, run_attempt=1, head_sha='old', conclusion='failure',
                                     created_at='2026-11-16T06:00:00Z'), self.pr, self.main]
        early = [dict(job, started_at='2026-11-16T06:10:00Z') for job in slots]
        api.pages.side_effect = [early, slots + [retry], slots]
        result = usage.measure(api, self.delivery)
        self.assertEqual((result['full_pr_matrix_starts'], result['before_final_pr_run']), (2, 1))
        self.assertFalse(result['runs'][1]['pr_matrix_attempts'][1]['full_start'])
        skipped = [dict(job, runner_id=None) for job in slots]
        self.assertFalse(usage.matrix_starts(skipped)[0]['full_start'])
        self.assertTrue(usage.matrix_starts(skipped[:-1] + slots[-1:])[0]['full_start'])

    def test_new_full_matrix_and_failed_only_retry_are_counted_separately(self):
        slots = [dict(self.job, name=f'{suite} shard {i}/{size}') for suite, size in usage.SUITES.items()
                 for i in range(1, size + 1)]
        retry = dict(slots[-1], run_attempt=2)
        self.assertEqual([row['full_start'] for row in usage.matrix_starts(slots + [retry])], [True, False])

    def test_incomplete_membership_attempts_or_candidate_are_rejected(self):
        for delivery in [dict(self.delivery, run_ids=[1]), dict(self.delivery, run_ids=[1, 2, 2])]:
            with self.assertRaises(ValueError):
                usage.measure(Mock(), delivery)
        for jobs in [[self.job], [self.job, dict(self.job, run_attempt=2)]]:
            api = Mock()
            api.json.return_value = self.pr
            api.pages.return_value = jobs
            with self.assertRaisesRegex(ValueError, 'duplicated jobs or missing attempts'):
                usage.measure(api, self.delivery)
        api = Mock()
        api.json.side_effect = [dict(self.pr, run_attempt=1), dict(self.main, head_sha='other')]
        api.pages.return_value = [self.job]
        with self.assertRaisesRegex(ValueError, 'main run'):
            usage.measure(api, self.delivery)

    def test_unclassified_allocations_remain_visible_even_at_zero_minutes(self):
        unknown = dict(self.job, runner_group_name=None, completed_at=self.job['started_at'])
        self.assertEqual(usage.job_usage(unknown), ('unclassified', 0))
        for job in [dict(self.job, status='in_progress'), dict(self.job, runner_id=0),
                    dict(self.job, completed_at='2026-11-16T06:00:00Z')]:
            with self.assertRaises(ValueError):
                usage.job_usage(job)

    def test_partial_rerun_inherits_plan_and_a_new_quality_attempt_replaces_it(self):
        api = Mock(repo='owner/repo')
        api.json.side_effect = [dict(self.pr, run_attempt=3), self.main]
        quality = dict(self.job, id=101, name='quality')
        shard = dict(self.job, id=102, run_attempt=2, conclusion='failure')
        new_quality = dict(quality, id=103, run_attempt=3)
        api.pages.side_effect = [[shard, new_quality, quality], [dict(quality, id=104)]]
        plan = dict(schema=2, repo=api.repo, run_id='1', event='pull_request', pr=19, head='head', scope='full',
                    browser_shards=11, integration_shards=8, self_hosted_shards=dict(browser=1, integration=0))
        def log(data):
            return ('2026-11-16T07:00:11Z ' + json.dumps(data)).encode()
        api.raw.side_effect = [log(plan), log(dict(plan, self_hosted_shards=dict(browser=11, integration=3))),
                               subprocess.CalledProcessError(1, 'gh')]
        result = usage.measure(api, self.delivery, rollout=True)
        first, partial, changed = result['runs'][0]['attempt_placement']
        self.assertEqual([(a['attempt'], a['quality_job'], a['plan_attempt']) for a in [first, partial, changed]],
                         [(1, 101, 1), (2, 101, 1), (3, 103, 3)])
        self.assertEqual(first['plan'], partial['plan'])
        self.assertEqual(first['plan']['self_hosted_shards'], dict(browser=1, integration=0))
        self.assertEqual(changed['plan']['self_hosted_shards'], dict(browser=11, integration=3))
        self.assertEqual(partial['minutes'], dict(github_hosted=2))
        self.assertIsNone(result['runs'][1]['attempt_placement'][0]['plan'])
        self.assertEqual(result['minutes'], dict(github_hosted=8))

    def test_historical_or_mismatched_plans_cannot_supply_runner_counts(self):
        api = Mock(repo='owner/repo')
        plan = dict(schema=2, repo=api.repo, run_id='1', event='pull_request', pr=19, head='head', scope='full')
        def read(**changes):
            api.raw.return_value = ('time ' + json.dumps(dict(plan, **changes))).encode()
            return usage.read_placement_plan(api, self.job, 1, self.pr, 19)
        self.assertEqual(read(), dict(scope='full', self_hosted_shards=None))
        self.assertIsNone(read(pr=20))
        self.assertIsNone(read(head='different'))
        with self.assertRaisesRegex(ValueError, 'invalid runner counts'):
            read(browser_shards=1, integration_shards=0, self_hosted_shards=dict(browser=11, integration=3))
