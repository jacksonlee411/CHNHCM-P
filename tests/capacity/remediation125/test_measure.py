"""Failure accounting for the capacity driver; no database or HTTP traffic."""
import copy
import types
import unittest
from unittest.mock import patch

import measure


class MatrixFixture:
    def __init__(self, failure=None, recovery_ms=1):
        self.failure = failure
        self.recovery_ms = recovery_ms
        self.result = dict(groups={}, preparation=[], cleanup=[])
        self.output = None
        self.issued = 0
        self.resets = 0
        self.failed_preparation = False

    def login(self):
        return list(range(20))

    def job(self, action):
        def invoke():
            self.issued += 1
            if self.failure == 'http' and self.issued == 1:
                return dict(ok=False, ms=10001, httpError=True, status=503), None
            return dict(ok=True, ms=self.recovery_ms if action == 'resolve' else 1), {}
        return invoke

    def read_job(self, client, kind, action, *_):
        if self.failure == 'all-preparations':
            raise RuntimeError('read preparation failed')
        return self.job(action)

    def write_job(self, client, kind, action, *_):
        if self.failure in ['all-preparations', 'write-preparations']:
            raise RuntimeError('write preparation failed')
        if self.failure == 'one-preparation' and not self.failed_preparation:
            self.failed_preparation = True
            raise RuntimeError('one preparation failed')
        return self.job(action), lambda: None

    def cold(self, *_):
        self.resets += 1
        if self.failure == 'third-reset' and self.resets == 3:
            raise RuntimeError('cold reset failed')


class MatrixAccountingTests(unittest.TestCase):
    def test_department_list_target_keeps_other_reads_separate(self):
        self.assertEqual(measure.operation_target_ms('dep', 'list', sessions=1), 300)
        self.assertEqual(measure.operation_target_ms('dep', 'list', sessions=20), 1000)
        self.assertEqual(measure.operation_target_ms('bu', 'list', sessions=20), 300)
        self.assertEqual(measure.operation_target_ms('dep', 'detail', sessions=20), 300)
        fixture = MatrixFixture()
        self.run_matrix(fixture, 'warm')
        for key, target in [('warm/Department/list', 1000), ('warm/BU/list', 300),
                            ('warm/mixed/dep/list', 1000), ('warm/mixed/bu/list', 300),
                            ('warm/mixed/list', None)]:
            self.assertEqual(fixture.result['groups'][key]['targetMS'], target)

    def run_matrix(self, fixture, phase, **selection):
        with patch.object(measure, 'save'), \
             patch.object(measure, 'parallel', lambda jobs: [job() for job in jobs]), \
             patch.object(measure, 'measured', return_value=(dict(ok=True, ms=1), {})):
            measure.Lane.matrix(fixture, phase, **selection)

    def test_explicit_write_partition_preserves_failures_and_omissions(self):
        for phase, expected, resets in [('warm', 1200, 0), ('cold', 600, 30)]:
            with self.subTest(phase=phase):
                fixture = MatrixFixture('one-preparation')
                self.run_matrix(fixture, phase, actions=['save', 'pair'], modes=['BU', 'Department', 'mixed'])
                result = fixture.result
                self.assertEqual(result['expectedTimedRequests'], expected)
                self.assertEqual(result['actualTimedRequests'], expected-1)
                self.assertEqual(len(result['unattemptedRequests']), 1)
                self.assertEqual(fixture.resets, resets)
                self.assertFalse(result['partition']['fullPhaseAcceptance'])
                self.assertFalse(result['groups'][phase+'/BU/save']['passed'])
                self.assertTrue(all(k.endswith(('/save', '/pair')) for k in result['groups']))

    def test_preparation_failures_are_not_http_requests(self):
        for phase, failure, actual in [
            ('mixed', None, 1200), ('mixed', 'write-preparations', 600),
            ('mixed', 'all-preparations', 0), ('mixed', 'http', 1200),
            ('cold', 'one-preparation', 1799), ('cold', None, 1800),
            ('warm', None, 4800),
        ]:
            with self.subTest(phase=phase, failure=failure):
                fixture = MatrixFixture(failure)
                self.run_matrix(fixture, phase)
                result = fixture.result
                groups = {k: v for k, v in result['groups'].items() if len(k.split('/')) == 3}
                self.assertEqual(fixture.issued, actual)
                self.assertEqual(result['actualTimedRequests'], actual)
                self.assertEqual(sum(v['requests'] for v in groups.values()), actual)
                self.assertEqual(actual + len(result['unattemptedRequests']), result['expectedTimedRequests'])
                self.assertTrue(result['allScheduledGroupsAttempted'])
                for group in result['groups'].values():
                    self.assertEqual(group['requests'], len(group['samples']))
                    self.assertEqual(group['requests'], group['timed'])
                    if not group['completed']:
                        self.assertFalse(group['passed'])
                self.assertEqual(sum(v['httpErrors'] for v in groups.values()), int(failure == 'http'))
                if phase == 'cold':
                    self.assertEqual(fixture.resets, 90)

    def test_reset_failure_retains_earlier_requests(self):
        fixture = MatrixFixture('third-reset')
        with self.assertRaisesRegex(RuntimeError, 'cold reset failed'):
            self.run_matrix(fixture, 'cold')
        self.assertEqual(fixture.result['actualTimedRequests'], 40)
        self.assertEqual(fixture.result['groups']['cold/BU/list']['requests'], 40)
        self.assertFalse(fixture.result['groups']['cold/BU/list']['passed'])
        self.assertFalse(fixture.result['allScheduledGroupsAttempted'])

    def test_recovery_budget_applies_to_each_object_and_phase(self):
        for phase in ['warm', 'cold', 'mixed']:
            for elapsed in [499, 500, 501]:
                with self.subTest(phase=phase, elapsed=elapsed):
                    fixture = MatrixFixture(recovery_ms=elapsed)
                    self.run_matrix(fixture, phase)
                    groups = [v for k, v in fixture.result['groups'].items() if k.endswith('/resolve')]
                    self.assertTrue(groups)
                    for group in groups:
                        self.assertEqual(group['targetMS'], 500)
                        self.assertEqual(group['p95MS'], elapsed)
                        self.assertEqual(group['passed'], elapsed <= 500)


class PreparationRecoveryTests(unittest.TestCase):
    def test_failed_preparation_is_counted_once_by_matrix(self):
        fixture = MatrixFixture()
        before = dict(headSourceVersion='before', departmentId='fixture-department',
                      timeline=[dict(recordId='fixture-record', revision=1, name='initial')])
        client = types.SimpleNamespace(call=lambda *args: (copy.deepcopy(before), 1))
        fixture.login = lambda: [client] * 20
        fixture.oracle = types.SimpleNamespace(path=lambda code: '/fixture')
        fixture.receipts = []
        fixture.exact_receipt = measure.Lane.exact_receipt
        original_job = fixture.write_job
        fixture.write_job = lambda c, k, a, r, w: (
            measure.Lane.write_job(fixture, c, k, a, r, w) if a == 'resolve'
            else original_job(c, k, a, r, w))
        submitted = []

        def measured(client, method, route, body=None, key=None, check=None):
            if key:
                submitted.append(key)
                return dict(ok=False, ms=10, httpError=True, status=503,
                            code='M02_DEPENDENCY_UNAVAILABLE', errorType='HTTPFailure'), None
            return dict(ok=True, ms=1), dict(outcome='not_applied')

        with patch.object(measure, 'save'), \
             patch.object(measure, 'parallel', lambda jobs: [job() for job in jobs]), \
             patch.object(measure, 'measured', measured):
            measure.Lane.matrix(fixture, 'mixed')
        self.assertEqual(len(submitted), 200)
        self.assertEqual(len(set(submitted)), 200)
        failures = [v for v in fixture.result['preparation'] if not v['ok']]
        self.assertEqual(len(failures), len(submitted))
        self.assertTrue(all(v['status'] == 503 and v['code'] == 'M02_DEPENDENCY_UNAVAILABLE' for v in failures))
        self.assertEqual(fixture.result['actualTimedRequests'], 1000)
        self.assertEqual(len(fixture.result['unattemptedRequests']), 200)
        self.assertTrue(all(v['status'] == 503 for v in fixture.result['unattemptedRequests']))

    def test_ambiguous_preparation_resolves_original_key_without_resubmission(self):
        for outcome in ['accepted', 'not_applied', 'http_failure', 'wrong_identity', 'source_http_failure']:
            with self.subTest(outcome=outcome):
                initial = dict(headSourceVersion='before', departmentId='fixture-department',
                    timeline=[dict(recordId='fixture-record', revision=1, name='initial',
                                   effectiveDate='2020-01-01', endExclusive=None)])
                state, calls = {}, []

                def call(method, path):
                    if '?sourceVersion=' in path:
                        if outcome == 'source_http_failure':
                            error = RuntimeError('source request failed')
                            error.status, error.code, error.elapsed_ms = 503, 'M02_DEPENDENCY_UNAVAILABLE', 12
                            raise error
                        return dict(resolvedSourceVersion='after', readOnly=True,
                                    timeline=copy.deepcopy(initial['timeline'])), 1
                    value = copy.deepcopy(initial)
                    if 'body' in state:
                        value['headSourceVersion'] = 'after'
                        value['timeline'][0].update(name=state['body']['changes']['name'], revision=2)
                    return value, 1

                def measured(client, method, route, body=None, key=None, check=None):
                    calls.append((route, key))
                    if key:
                        state.update(body=body, key=key)
                        return dict(ok=False, ms=10, httpError=True, status=503), None
                    self.assertIn(state['key'], route)
                    if outcome == 'http_failure':
                        return dict(ok=False, ms=10, httpError=True, status=503), None
                    value = dict(outcome='not_applied') if outcome == 'not_applied' else dict(
                        outcome='accepted', submissionId=state['key'], sourceVersion='after',
                        departmentId='wrong' if outcome == 'wrong_identity' else 'fixture-department',
                        targetRevision=2, currentChanged=False)
                    if check:
                        check(value)
                    return dict(ok=True, ms=1), value

                fixture = types.SimpleNamespace(oracle=types.SimpleNamespace(path=lambda code: '/fixture'),
                    result=dict(preparation=[], cleanup=[]), receipts=[], exact_receipt=measure.Lane.exact_receipt)
                with patch.object(measure, 'measured', measured):
                    with self.assertRaisesRegex(RuntimeError, 'resolve preparation failed'):
                        measure.Lane.write_job(fixture, types.SimpleNamespace(call=call), 'dep', 'resolve', 0, 0)
                self.assertEqual(sum(key is not None for _, key in calls), 1)
                self.assertGreaterEqual(len(calls), 2)
                self.assertFalse(fixture.result['preparation'][-1]['ok'])
                self.assertEqual(len(fixture.receipts), int(outcome in ['accepted', 'source_http_failure']))
                if outcome == 'accepted':
                    self.assertEqual(fixture.receipts[0]['receipt']['submissionId'], state['key'])
                    self.assertTrue(all(v['ok'] for v in fixture.result['cleanup']))
                if outcome in ['http_failure', 'wrong_identity']:
                    self.assertTrue(any(not v['ok'] for v in fixture.result['cleanup']))
                if outcome == 'source_http_failure':
                    failure = fixture.result['cleanup'][-1]
                    self.assertFalse(failure['ok'])
                    self.assertTrue(failure['httpError'])
                    self.assertFalse(failure['semanticError'])
                    self.assertEqual((failure['status'], failure['code'], failure['ms']),
                                     (503, 'M02_DEPENDENCY_UNAVAILABLE', 12))


if __name__ == '__main__':
    unittest.main()
