import unittest

import ci_browser as browser


class BrowserInventoryTests(unittest.TestCase):
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

    def test_files_are_literal_patterns_and_shard_is_bounded(self):
        args = browser.arguments(['country200.spec.ts'], 1, 4)
        self.assertEqual(args, ['(?:^|/)country200\\.spec\\.ts$', '--shard=1/4'])
        with self.assertRaises(ValueError):
            browser.arguments([], 0, 4)
