import copy
import sys
import unittest

from capacity import ROOT, capacity, module, path_targets, read_stats, read_target_ms, reevaluate_group
import tree_scope_targets

sys.path.insert(0, str(ROOT/'tests/capacity/tree300'))
tree = module('tree300_target_policy', ROOT/'tests/capacity/tree300/matrix.py')
sys.path.pop(0)


class ReadTargetTests(unittest.TestCase):
    def test_user_target_is_limited_to_the_named_operation(self):
        self.assertEqual(read_target_ms('parent-create-first', sessions=1), 300)
        self.assertEqual(read_target_ms('parent-create-first', sessions=20), 1000)
        self.assertEqual(read_target_ms('list', sessions=1), 300)
        self.assertEqual(read_target_ms('list', sessions=20), 1000)
        for operation in ['start-explicit', 'start-all', 'resolve']:
            with self.subTest(operation=operation):
                self.assertEqual(read_target_ms(operation, sessions=1), 1000)
                self.assertEqual(read_target_ms(operation, sessions=20), 5000)
        self.assertEqual(read_target_ms('path', sessions=1), 300)
        self.assertEqual(read_target_ms('path', sessions=20), 1000)
        for operation in ['parent-create-search', 'parent-create-second',
                          'parent-maintain-first', 'parent-maintain-search', 'parent-maintain-second']:
            with self.subTest(operation=operation):
                for sessions in [1, 20]:
                    self.assertEqual(read_target_ms(operation, sessions=sessions), 300)
        self.assertIsNone(read_target_ms('parent-create-first', permission_negative=True, sessions=20))

    def test_path_stage_target_keeps_long_term_and_error_requirements(self):
        for sessions, target in [(1, 300), (20, 1000)]:
            for elapsed, expected in [(target-1, True), (target, True), (target+.001, False)]:
                with self.subTest(sessions=sessions, elapsed=elapsed):
                    samples = [dict(ok=True, ms=elapsed) for _ in range(200)]
                    result = read_stats(samples, 'path', sessions=sessions)
                    self.assertEqual(result['passed'], expected)
                    self.assertEqual(result['longTermTargetMS'], 300)
                    self.assertEqual(result['longTermPassed'], elapsed <= 300)
                    self.assertEqual(result['targetPolicy'], path_targets.POLICY_ID)
        for failure in [dict(ok=False, ms=10, httpError=True),
                        dict(ok=False, ms=10, semanticError=True),
                        dict(ok=False, ms=None, httpError=True)]:
            samples = [dict(ok=True, ms=250) for _ in range(199)] + [failure]
            result = read_stats(samples, 'path', sessions=20)
            self.assertFalse(result['passed'])
            self.assertFalse(result['longTermPassed'])
            self.assertEqual((result['requests'], result['failures']), (200, 1))
        self.assertIsNone(read_stats([dict(ok=True, ms=600)], 'path', True, sessions=20)['targetMS'])
        with self.assertRaises(KeyError):
            path_targets.target_ms(sessions=2)

    def test_large_subtree_stage_does_not_relax_other_operations(self):
        for sessions in [1, 20]:
            self.assertEqual(tree.target_for('scope-path', sessions, 300), {1: 400, 20: 2000}[sessions])
            self.assertEqual(read_target_ms('path', sessions=sessions), {1: 300, 20: 1000}[sessions])
            self.assertEqual(tree.target_for('scope-self', sessions, None), {1: 300, 20: 1000}[sessions])
            self.assertEqual(tree.target_for('roots', sessions, 300), 300)
            self.assertEqual(tree.target_for('search-exact', sessions, 1000), 1000)
            self.assertEqual(tree.target_for('scope-large', sessions, None), {1: 300, 20: 1500}[sessions])
            for operation in ['start-large', 'resolve-large']:
                self.assertEqual(tree.target_for(operation, sessions, None), {1: 1000, 20: 5000}[sessions])

    def test_large_subtree_reassessment_preserves_errors_and_previous_targets(self):
        group = dict(name='scope-path', sessions=20, attempts=200, executedRequests=200,
                     samples=[dict(status=200, ms=1400) for _ in range(200)])
        before = copy.deepcopy(group)
        result = tree_scope_targets.assess(group, phase='warm')
        self.assertTrue(result['passed'])
        self.assertFalse(result['originalPassed'])
        self.assertFalse(result['longTermPassed'])
        self.assertEqual(group, before)
        for sample in [dict(status=503, ms=20), dict(status=200, ms=20, error='semantic_mismatch')]:
            bad = copy.deepcopy(group); bad['samples'][-1] = sample
            self.assertFalse(tree_scope_targets.assess(bad, phase='warm')['passed'])
        short = copy.deepcopy(group); short['samples'].pop(); short['attempts'] = short['executedRequests'] = 199
        self.assertFalse(tree_scope_targets.assess(short, phase='warm')['passed'])
        for elapsed, passed in [(2000, True), (2000.001, False)]:
            sample = copy.deepcopy(group)
            for row in sample['samples']: row['ms'] = elapsed
            self.assertEqual(tree_scope_targets.assess(sample, phase='warm')['passed'], passed)
        group['sessions'] = 1
        for row in group['samples']: row['ms'] = 350
        result = tree_scope_targets.assess(group, phase='warm')
        self.assertTrue(result['passed'])
        self.assertFalse(result['previousStagePassed'])
        self.assertFalse(result['longTermPassed'])
        with self.assertRaises(KeyError): tree_scope_targets.limits('scope-self', 20, phase='warm')
        with self.assertRaises(KeyError): tree_scope_targets.limits('scope-path', 2, phase='warm')

    def test_warm_target_does_not_leak_into_cold_mixed_or_first_page(self):
        for phase in ('cold', 'mixed'):
            for name in tree_scope_targets.CASES:
                for sessions, target in ((1, 300), (20, 1500)):
                    self.assertEqual(tree_scope_targets.limits(name, sessions, phase=phase)['stageTargetMS'], target)
        for name in ('scope-path', 'scope-large-last'):
            for sessions, target in ((1, 400), (20, 2000)):
                for ms, passed in ((target, True), (target + .001, False), (float('nan'), False), (float('inf'), False), (-1, False)):
                    group = dict(name=name, sessions=sessions, attempts=200, executedRequests=200,
                                 samples=[dict(status=200, ms=ms) for _ in range(200)])
                    self.assertEqual(tree_scope_targets.assess(group, phase='warm')['passed'], passed)
        with self.assertRaises(TypeError): tree_scope_targets.limits('scope-path', 20)
        with self.assertRaises(KeyError): tree_scope_targets.limits('scope-path', 20, phase='unknown')

    def test_path_reassessment_preserves_original_failure(self):
        original = capacity.stats([dict(ok=True, ms=600) for _ in range(200)], 300)
        original.update(completed=True, expectedRequests=200, permissionNegative=False, sessions=20)
        preserved = copy.deepcopy(original)
        result = reevaluate_group(original, 'path')
        self.assertTrue(result['passed'])
        self.assertFalse(result['longTermPassed'])
        self.assertFalse(result['originalPassed'])
        self.assertEqual((result['targetMS'], result['longTermTargetMS'], result['originalTargetMS']), (1000, 300, 300))
        result['samples'][0]['ok'] = False
        self.assertEqual(original, preserved)
        self.assertFalse(reevaluate_group(original | {'sessions': 1}, 'path')['passed'])

    def test_session_boundaries_keep_all_failures_in_the_verdict(self):
        for operation in ['parent-create-first', 'start-explicit', 'start-all', 'resolve']:
            targets = [(1, 300), (20, 1000)] if operation == 'parent-create-first' else [(1, 1000), (20, 5000)]
            for sessions, target in targets:
                for elapsed, expected in [(target-1, True), (target, True), (target+.001, False)]:
                    with self.subTest(operation=operation, sessions=sessions, elapsed=elapsed):
                        values = [dict(ok=True, ms=elapsed) for _ in range(200)]
                        self.assertEqual(capacity.stats(values, read_target_ms(operation, sessions=sessions))['passed'], expected)
        for failure in [dict(ok=False, ms=10, httpError=True),
                        dict(ok=False, ms=10, semanticError=True),
                        dict(ok=False, ms=None, httpError=True)]:
            values = [dict(ok=True, ms=200) for _ in range(199)] + [failure]
            for target in [1000, 5000]:
                result = capacity.stats(values, target)
                self.assertFalse(result['passed'])
                self.assertEqual(result['requests'], 200)
                self.assertEqual(result['failures'], 1)

    def test_reassessment_preserves_original_samples_and_verdict(self):
        original = capacity.stats([dict(ok=True, ms=1150.131) for _ in range(200)], 3000)
        original.update(completed=True, expectedRequests=200, permissionNegative=False, sessions=1)
        before = copy.deepcopy(original)
        reviewed = reevaluate_group(original, 'parent-create-first')
        self.assertFalse(reviewed['passed'])
        self.assertTrue(reviewed['originalPassed'])
        self.assertEqual(reviewed['originalTargetMS'], 3000)
        self.assertEqual(reviewed['targetMS'], 300)
        self.assertEqual(reviewed['samples'], original['samples'])
        reviewed['samples'][0]['ok'] = False
        self.assertEqual(original, before)
        concurrent = copy.deepcopy(original)
        concurrent['sessions'] = 20
        self.assertEqual(reevaluate_group(concurrent, 'parent-create-first')['targetMS'], 1000)
        concurrent['targetMS'] = 300
        reviewed_list = reevaluate_group(concurrent, 'list')
        self.assertEqual(reviewed_list['targetMS'], 1000)
        self.assertEqual(reviewed_list['originalTargetMS'], 300)
        self.assertEqual(reviewed_list['samples'], concurrent['samples'])
        old_selection = capacity.stats([dict(ok=True, ms=1150.131) for _ in range(200)], 300)
        old_selection.update(completed=True, expectedRequests=200, permissionNegative=False, sessions=20)
        preserved = copy.deepcopy(old_selection)
        for operation in ['start-explicit', 'start-all', 'resolve']:
            reviewed_selection = reevaluate_group(old_selection, operation)
            self.assertEqual(reviewed_selection['targetMS'], 5000)
            self.assertTrue(reviewed_selection['passed'])
            self.assertEqual(reviewed_selection['originalTargetMS'], 300)
            self.assertFalse(reviewed_selection['originalPassed'])
            self.assertEqual(reviewed_selection['samples'], old_selection['samples'])
            reviewed_selection['samples'][0]['ok'] = False
            self.assertEqual(old_selection, preserved)
            single = old_selection | {'sessions': 1}
            self.assertEqual(reevaluate_group(single, operation)['targetMS'], 1000)
            self.assertFalse(reevaluate_group(single, operation)['passed'])
        original['permissionNegative'] = True
        self.assertIsNone(reevaluate_group(original, 'parent-create-first')['targetMS'])
        for missing in [dict(completed=False), dict(expectedRequests=201)]:
            with self.subTest(missing=missing), self.assertRaises(AssertionError):
                reevaluate_group(original | missing, 'parent-create-first')


if __name__ == '__main__':
    unittest.main()
