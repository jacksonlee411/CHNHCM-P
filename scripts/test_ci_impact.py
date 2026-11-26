import copy
import json
from pathlib import Path
import re
import tempfile
import unittest

import ci_impact as impact
import ci_policy as ci


class ImpactTests(unittest.TestCase):
    def setUp(self):
        self.policy = json.loads((ci.ROOT / ci.POLICY).read_text())

    def select(self, *paths, root=ci.ROOT):
        return impact.select(list(paths), self.policy, root=root)

    def test_document_and_passive_evidence_do_not_start_business_suites(self):
        p = self.select('docs/README.md', 'docs/verification/data/306-publication/check-merged.log.gz',
                        'docs/verification/data/306-publication/integration-summary.json')
        self.assertEqual((p['scope'], p['integration_shards'], p['browser_shards']), ('docs', 0, 0))
        for path in ['docs/verification/data/116-r0-review-fixes/contract-fixtures.json',
                     'docs/verification/data/116-r4-review-fixes/review-case-oracles.json',
                     'docs/verification/data/100-ui-baseline/native-zoom.py',
                     'docs/verification/data/new/setup.py.gz', 'docs/verification/data/new/query.sql']:
            with self.subTest(path=path):
                self.assertEqual(self.select(path)['scope'], 'full')

    def test_new_evidence_input_reference_revokes_passive_classification(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = 'docs/verification/data/new/fixture.json'
            self.assertEqual(self.select(path, root=root)['scope'], 'docs')
            consumer = root / 'internal/new/read.go'
            consumer.parent.mkdir(parents=True)
            consumer.write_text('read("docs/verification/data/new/fixture.json")')
            self.assertEqual(self.select(path, root=root)['scope'], 'full')

    def test_single_ui_runs_its_browser_consumers_without_backend_suite(self):
        p = self.select('apps/web/src/features/organization/Countries.tsx')
        self.assertEqual(p['scope'], 'selected')
        self.assertEqual(p['integration_shards'], 0)
        self.assertIn('country200.spec.ts', p['browser_files'])
        self.assertIn('geography205-joint.spec.ts', p['browser_files'])
        self.assertNotIn('department-selection.spec.ts', p['browser_files'])
        self.assertLess(p['browser_shards'], 16)
        mixed = self.select('apps/web/src/features/organization/Countries.tsx',
                            'docs/verification/data/306-publication/integration-summary.json')
        self.assertEqual(mixed['browser_files'], p['browser_files'])

    def test_multi_domain_is_union_not_last_file_or_one_label(self):
        a = self.select('apps/web/src/features/organization/Countries.tsx')
        b = self.select('apps/web/src/features/organization/DepartmentRelations.tsx')
        union = self.select('apps/web/src/features/organization/DepartmentRelations.tsx',
                            'apps/web/src/features/organization/Countries.tsx')
        self.assertEqual(set(union['browser_files']), set(a['browser_files']) | set(b['browser_files']))
        self.assertEqual(union['integration_shards'], 0)

    def test_transitive_unmapped_ui_consumer_widens_browser_only(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            directory = root / 'apps/web/src/features/organization'
            directory.mkdir(parents=True)
            (directory / 'Countries.tsx').write_text('')
            (directory / 'GeoAreas.tsx').write_text("import './Countries';")
            (directory / 'UnknownConsumer.tsx').write_text("import './GeoAreas';")
            p = self.select('apps/web/src/features/organization/Countries.tsx', root=root)
            self.assertEqual((p['integration_shards'], p['browser_shards']), (0, 16))

    def test_new_backend_or_browser_consumers_cannot_disappear_from_coverage(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            spec = root / 'tests/e2e/new-business.spec.ts'
            spec.parent.mkdir(parents=True)
            spec.write_text('test new business')
            p = self.select('apps/web/src/features/organization/Countries.tsx', root=root)
            self.assertEqual((p['integration_shards'], p['browser_shards']), (0, 16))
            p = self.select('tests/e2e/new-business.spec.ts', root=root)
            self.assertEqual(p['browser_files'], ['new-business.spec.ts'])
            consumer = root / 'internal/newbusiness/reader.go'
            consumer.parent.mkdir(parents=True)
            consumer.write_text('import "chnhcm.local/chnhcm/internal/contexts/organization/application/country"')
            self.assertEqual(self.select('internal/contexts/organization/application/country/service.go', root=root)['scope'], 'full')
            consumer.write_text('import "chnhcm.local/chnhcm/internal/contexts/staffing/application"')
            self.assertEqual(self.select('internal/contexts/staffing/application/list.go', root=root)['scope'], 'full')

    def test_document_tooling_uses_complete_quality_without_business_jobs(self):
        p = self.select('scripts/check-repository.py', 'scripts/document_catalog.py', 'scripts/test_ci_policy.py')
        self.assertEqual((p['scope'], p['integration_shards'], p['browser_shards']), ('selected', 0, 0))

    def test_backend_domain_includes_reverse_references_and_shared_roots(self):
        p = self.select('internal/contexts/organization/application/department/list_digest.go')
        for test in ['TestDepartmentPath', 'TestEnterpriseReferences', 'TestBUCoverage', 'TestT07Authorization', 'TestNewSharedRegression']:
            self.assertRegex(test, p['integration_filter'])
        for test in ['TestCountryCatalog', 'TestGeoAreaCreation', 'TestLocationAddress', 'TestLegalEntitySources']:
            self.assertIsNone(re.search(p['integration_filter'], test))
        self.assertIn('enterprise304.spec.ts', p['browser_files'])
        self.assertIn('department-selection.spec.ts', p['browser_files'])
        self.assertNotIn('country200.spec.ts', p['browser_files'])

    def test_backend_country_includes_geo_location_and_legal_consumers(self):
        p = self.select('internal/contexts/organization/domain/country/catalog.go')
        for test in ['TestCountryCatalog', 'TestGeoAreaCreation', 'TestLocationAddress', 'TestLegalEntitySources', 'TestEnterpriseReferences', 'TestGeographyJoint']:
            self.assertRegex(test, p['integration_filter'])
        self.assertIsNone(re.search(p['integration_filter'], 'TestDepartmentExport'))

    def test_staffing_page_contract_and_shared_presentations(self):
        for path in ['apps/web/src/features/staffing/ChangeLog.tsx', 'internal/contexts/staffing/application/changes.go']:
            plan = self.select(path)
            self.assertIn('staffing308.spec.ts', plan['browser_files'])
            self.assertIn('staffing308-zoom.spec.ts', plan['browser_files'])
        plan = self.select('internal/contexts/staffing/application/changes.go')
        self.assertRegex('TestStaffing308LogSourcesPagingAndFilters', plan['integration_filter'])
        for path in ['apps/web/src/shared/dated/ColumnFilter.tsx', 'apps/web/src/shared/dated/OperationLog.tsx']:
            plan = self.select(path)
            self.assertEqual(plan['browser_shards'], 16)
            self.assertEqual(plan['integration_shards'], 0)
            self.assertEqual(plan['browser_files'], [])  # None selector means the complete browser manifest.

    def test_staffing_enterprise_port_and_neutral_frontend_consumers(self):
        own=self.select('apps/web/src/features/staffing/Editor.tsx')
        self.assertIn('staffing307.spec.ts',own['browser_files'])
        self.assertEqual(own['integration_shards'],0)
        enterprise=self.select('internal/contexts/organization/application/enterprise/service.go')
        self.assertRegex('TestStaffing307EnterpriseBothLockOrders',enterprise['integration_filter'])
        self.assertEqual(enterprise['browser_shards'],16)
        staff=self.select('internal/contexts/staffing/application/commands.go')
        self.assertIn('staffing307.spec.ts',staff['browser_files'])
        self.assertIn('enterprise304.spec.ts',staff['browser_files'])
        self.assertRegex('TestEnterprise304References',staff['integration_filter'])
        self.assertEqual(self.select('apps/web/src/shared/dated/submission.ts')['browser_shards'],16)

    def test_tests_select_only_their_execution_layer(self):
        browser = self.select('tests/e2e/department-journeys.spec.ts')
        self.assertEqual((browser['integration_shards'], browser['browser_shards']), (0, 1))
        self.assertEqual(browser['browser_files'], ['department-journeys.spec.ts'])
        integration = self.select('tests/integration/department_contract_test.go')
        self.assertEqual((integration['integration_shards'], integration['browser_shards']), (5, 0))
        unit = self.select('internal/contexts/organization/domain/department/path_test.go')
        self.assertEqual((unit['scope'], unit['integration_shards'], unit['browser_shards']), ('selected', 0, 0))

    def test_shared_frontend_and_frontend_lock_do_not_expand_backend(self):
        for path in ['apps/web/src/app/App.tsx', 'apps/web/src/features/organization/dated/Workflow.tsx', 'pnpm-lock.yaml']:
            p = self.select(path)
            self.assertEqual((p['integration_shards'], p['browser_shards']), (0, 16))

    def test_core_validation_inputs_unknown_and_nonregular_stay_full(self):
        for path in ['scripts/ci_impact.py', '.github/ci-scopes.json', '.github/ci-shard-seconds.json', 'scripts/ci_shards.py', 'go.mod', 'package.json',
                     'internal/contexts/organization/domain/dated/timeline.go', 'migrations/00017.sql',
                     'tests/e2e/fixtures.ts', 'api/openapi.yaml', 'unknown.file']:
            with self.subTest(path=path):
                p = self.select(path)
                self.assertEqual((p['scope'], p['integration_shards'], p['browser_shards']), ('full', 5, 16))
        self.assertEqual(impact.select(['docs/README.md'], self.policy, False, ci.ROOT)['scope'], 'full')

    def test_deleted_frontend_keeps_consumer_tests_and_unknown_test_delete_widens(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            p = self.select('apps/web/src/features/organization/Countries.tsx', root=root)
            self.assertIn('country200.spec.ts', p['browser_files'])
            p = self.select('tests/e2e/removed.spec.ts', root=root)
            self.assertEqual((p['integration_shards'], p['browser_shards']), (0, 16))

    def test_mapping_requires_consumers_to_exist_and_have_one_owner(self):
        impact.validate_mapping(self.policy, ci.ROOT)
        for mutate in [lambda p: p['domains']['country']['browser_files'].append('absent.spec.ts'),
                       lambda p: p['domains']['country']['backend_consumers'].append('missing'),
                       lambda p: p['domains']['country']['frontend'].extend(p['domains']['department']['frontend'])]:
            broken = copy.deepcopy(self.policy)
            mutate(broken)
            with self.assertRaises(ValueError):
                impact.validate_mapping(broken, ci.ROOT)


if __name__ == '__main__':
    unittest.main()
