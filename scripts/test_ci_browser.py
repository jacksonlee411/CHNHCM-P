import unittest
from collections import Counter, defaultdict
from unittest.mock import patch

import ci_browser as browser


class BrowserInventoryTests(unittest.TestCase):
    def test_large_entry_uses_discovered_declarations_for_full_and_selected_runs(self):
        entry = 'first-hire.spec.ts'
        specs = [dict(file=entry, line=line, id=str(i), tests=[dict(projectName='chromium')])
                 for i, line in enumerate([15, 15, 31, 97, 123, 123])]
        locations = defaultdict(Counter)
        selected = browser.inventory(dict(suites=[dict(file=entry, specs=specs)]), locations=locations)
        self.assertEqual(browser.selected_browser_count([entry]), 2)
        groups = browser.shard_files(selected, 2, [], locations)
        self.assertCountEqual(groups, [[f'{entry}:15', f'{entry}:31'], [f'{entry}:97', f'{entry}:123']])
        self.assertEqual(browser.arguments([f'{entry}:97']), [r'(?:^|/)first\-hire\.spec\.ts$:97'])
        with self.assertRaisesRegex(ValueError, 'shared declarations'):
            browser.inventory(dict(suites=[dict(file=entry, specs=[dict(specs[0], file='shared.ts')])]),
                              locations=locations)

    def test_same_basename_in_distinct_directories_keeps_its_own_entry(self):
        report = {'config': {'rootDir': '/code/tests/e2e'}, 'suites': []}
        for dirname in ['staffing', 'organization']:
            spec = dict(file='journey.ts', id=dirname, tests=[{'projectName': 'chromium'}])
            report['suites'].append(dict(file=f'/code/tests/e2e/{dirname}/edit.spec.ts', specs=[spec]))
        self.assertEqual(set(browser.inventory(report).values()),
                         {'staffing/edit.spec.ts', 'organization/edit.spec.ts'})

    def test_shared_definitions_belong_to_the_selected_entry_suite(self):
        suites = []
        for name in ['grade.spec.ts', 'job.spec.ts']:
            spec = dict(file='shared.ts', id=name, tests=[dict(projectName='chromium')])
            suites.append(dict(file=name, suites=[dict(file='shared.ts', specs=[spec])]))
        tests = browser.inventory(dict(suites=suites))
        self.assertEqual(tests[('chromium', 'shared.ts', 'job.spec.ts')], 'job.spec.ts')
        self.assertEqual(tests[('chromium', 'shared.ts', 'grade.spec.ts')], 'grade.spec.ts')

    def test_preflight_requires_each_business_case_exactly_once(self):
        cases = [dict(file='a.spec.ts', id=str(i), title=title, tests=[dict(projectName='chromium')])
                 for i, title in enumerate(browser.PREFLIGHT_TITLES)]
        dependency = dict(cases[0], tests=[dict(projectName='enterprise-maintenance')])
        self.assertEqual(len(browser.inventory({'specs': cases + [dependency]}, expected_titles=browser.PREFLIGHT_TITLES)), len(cases) + 1)
        for bad in [cases[1:], cases + [dict(cases[0], id='duplicate-title')], cases + [dict(cases[0], id='extra', title='other')]]:
            with self.assertRaisesRegex(ValueError, 'missing or ambiguous'):
                browser.inventory({'specs': bad}, expected_titles=browser.PREFLIGHT_TITLES)

    def test_dependency_repeats_are_explicit_and_other_duplicates_fail(self):
        dependency = ('enterprise-maintenance', 'enterprise.spec.ts', 'setup')
        a, b = ('chromium', 'a.spec.ts', 'a'), ('chromium', 'b.spec.ts', 'b')
        whole = {key: key[1] for key in [dependency, a, b]}
        result = browser.verify_partition(whole, [{dependency: '', a: ''}, {dependency: '', b: ''}], ['enterprise-maintenance'])
        self.assertEqual((result['unique_tests'], result['executions']), (3, 4))
        for shards in [[{a: ''}, {}], [{a: ''}, {a: '', b: '', dependency: ''}], [{a: '', b: ''}],
                       [{**whole, ('unknown', 'x', 'x'): ''}]]:
            with self.subTest(shards=shards), self.assertRaises(ValueError):
                browser.verify_partition(whole, shards, ['enterprise-maintenance'])

    def test_discovery_errors_empty_and_duplicate_are_not_success(self):
        spec = {'file': 'a.spec.ts', 'id': 'id', 'tests': [{'projectName': 'chromium'}]}
        self.assertEqual(len(browser.inventory({'suites': [{'specs': [spec]}]})), 1)
        for report in [{}, {'errors': ['compile failed']}, {'suites': [{'specs': [spec, spec]}]}]:
            with self.subTest(report=report), self.assertRaises(ValueError):
                browser.inventory(report)

    def test_weighted_selection_keeps_the_entry_and_the_enterprise_dependency_together(self):
        dependency = ('enterprise-maintenance', 'enterprise304.spec.ts', 'setup')
        cases = {dependency: 'enterprise304.spec.ts', ('chromium', 'shared.ts', 'slow'): 'staffing310.spec.ts',
                 ('chromium', 'shared.ts', 'other'): 'staffing308.spec.ts',
                 ('chromium', 'small.ts', 'new'): 'new.spec.ts'}
        weights = {'staffing310.spec.ts': 600, 'staffing308.spec.ts': 550, 'new.spec.ts': 40}
        with patch.object(browser, 'browser_seconds', side_effect=weights.__getitem__):
            groups = browser.shard_files(cases, 2, ['enterprise-maintenance'])
            self.assertEqual(groups, [['staffing310.spec.ts'], ['new.spec.ts', 'staffing308.spec.ts']])
            self.assertEqual(groups, browser.shard_files(dict(reversed(cases.items())), 2, ['enterprise-maintenance']))
        self.assertEqual(browser.shard_files({dependency: 'enterprise304.spec.ts'}, 1, ['enterprise-maintenance']),
                         [['enterprise304.spec.ts']])
        self.assertEqual(browser.arguments(['entry.with.dot.spec.ts']), [r'(?:^|/)entry\.with\.dot\.spec\.ts$'])
        with self.assertRaises(ValueError):
            browser.shard_files(cases, 4, ['enterprise-maintenance'])
