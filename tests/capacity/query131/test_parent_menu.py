import copy
import unittest

from parent_menu import require_original_counts, synthetic_fixture


class Oracle:
    def __init__(self):
        self.changes = {}

    def frame(self, code):
        records = self.changes.get(code, [dict(Date='2020-01-01', State='active', Data=dict(BU='fan-bu'))])
        return dict(State=dict(Timeline=dict(Records=records)))


class ParentMenuFixtureTests(unittest.TestCase):
    def setUp(self):
        self.oracle = Oracle()
        # Reverse insertion order proves that the expected pages use the
        # contract's code/identity ordering, not source-file iteration order.
        self.rows = {f'R6_FAN_{i:04d}': dict(id=f'dep-{i:04d}', name=f'name-{i:04d}',
            managementBu=dict(id='fan-bu', code='FAN_BU')) for i in range(2000, -1, -1)}
        self.rows['OTHER_BU'] = dict(id='unrelated', name='unrelated', managementBu=dict(id='other-bu', code='OTHER'))

    def test_complete_fanout_pages_and_future_first_use_independent_facts(self):
        full = synthetic_fixture(self.rows, self.oracle, '2026-11-11')
        self.assertEqual(full['buCode'], 'FAN_BU')
        self.assertEqual(full['count'], 2001)
        self.assertEqual([r['code'] for r in full['pages'][0]], [f'R6_FAN_{i:04d}' for i in range(20)])
        self.assertEqual([r['id'] for r in full['pages'][1]], [f'dep-{i:04d}' for i in range(20, 40)])
        self.assertEqual(full['pages'][1][0], dict(id='dep-0020', code='R6_FAN_0020', name='name-0020', eligibility='eligible'))
        self.oracle.changes['R6_FAN_0000'] = [dict(Date='2030-01-01', State='active', Data=dict(BU='fan-bu'))]
        future = synthetic_fixture(self.rows, self.oracle, '2026-11-11')
        self.assertEqual(future['count'], 2000)
        self.assertEqual(future['pages'][0][0]['code'], 'R6_FAN_0001')

    def test_future_ineligibility_and_reduced_fanout_are_rejected(self):
        for change in (dict(Date='2030-01-01', State='inactive', Data=dict(BU='fan-bu')),
                       dict(Date='2030-01-01', State='active', Data=dict(BU='different-bu'))):
            with self.subTest(change=change):
                self.oracle.changes['R6_FAN_0020'] = [dict(Date='2020-01-01', State='active', Data=dict(BU='fan-bu')), change]
                with self.assertRaises(AssertionError):
                    synthetic_fixture(self.rows, self.oracle, '2026-11-11')
        self.oracle.changes.clear()
        self.rows.pop('R6_FAN_2000')
        with self.assertRaises(AssertionError):
            synthetic_fixture(self.rows, self.oracle, '2026-11-11')

    def test_full_original_count_guard_rejects_growth_and_missing_input(self):
        counts = dict(BU=2023, department=22104, buSources=100000, departmentSources=1004206)
        require_original_counts('synthetic', counts)
        require_original_counts('vanke', dict(BU=1, department=87343, buSources=1, departmentSources=132289))
        for field, delta in [('BU', -1), ('department', -1), ('departmentSources', -1), ('departmentSources', 1)]:
            with self.subTest(field=field, delta=delta):
                wrong = copy.copy(counts)
                wrong[field] += delta
                with self.assertRaises(AssertionError):
                    require_original_counts('synthetic', wrong)


if __name__ == '__main__':
    unittest.main()
