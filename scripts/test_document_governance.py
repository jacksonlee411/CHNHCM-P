"""Behavior checks for document admission, preservation and evidence boundaries."""

from copy import deepcopy
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

from document_governance import heading_errors, markdown_anchors, markdown_links, validate_governance


class DocumentGovernanceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name) / 'repo'
        self.root.mkdir()
        self.registry = {
            'schema_version': 2,
            'fixed_branches': [{'branch': 'wt-dev-a', 'document_range': {'start': 1, 'end': 99}}],
            'documents': [],
            'governance': {
                'policy_document': 'CHNHCM-014', 'legacy_documents': [],
                'migration_note': 'Original documents keep their evidence boundary.',
                'verification_records': [],
            },
        }
        self.write('AGENTS.md', '# Instructions\n')
        self.write('docs/index.md', '# Documentation\n')
        self.policy = self.add_document()

    def write(self, path, text):
        destination = self.root / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(text, encoding='utf-8')

    def link(self, path):
        with (self.root / 'docs/index.md').open('a', encoding='utf-8') as entry:
            entry.write(f'\n[Document]({path.removeprefix("docs/")})\n')

    def add_document(self, number=14, kind='governance', status='draft', directory='governance'):
        identifier = f'CHNHCM-{number:03}'
        path = f'docs/{directory}/{number:03}-sample.md'
        document = {
            'id': identifier, 'title': 'Sample', 'path': path, 'owner_branch': 'wt-dev-a',
            'owner_role': 'Maintainer', 'status': status, 'type': kind, 'modules': [], 'scope': 'Repository',
            'owner': None, 'format': 'numbered', 'review': None,
            'execution_status': 'not_started' if kind == 'plan' else None,
            'superseded_by': [], 'closure': None, 'archive': None,
        }
        self.registry['documents'].append(document)
        if status != 'planned':
            self.write(path, f'# {identifier} Sample\n\n## 1 Purpose\n\nContent.\n')
            self.link(path)
        return document

    def add_report(self, result='passed'):
        path = 'docs/verification/014-sample.md'
        self.write(path, '# CHNHCM-014 Verification\n\n## 1 Results\n\nRecorded scope.\n')
        self.link(path)
        record = {
            'path': path, 'related_documents': ['CHNHCM-014'], 'format': 'numbered', 'status': 'final',
            'date': '2026-10-30', 'base_commit': 'a' * 40, 'working_tree': 'dirty',
            'change_description': 'Base plus documented governance changes.', 'environment': 'Python fixture',
            'scope': 'Document checks only', 'results': [{'check': 'Focused check', 'result': result, 'evidence': path}],
            'limitations': ['No product validation.'],
        }
        self.registry['governance']['verification_records'].append(record)
        return record

    def errors(self):
        return validate_governance(self.root, self.registry)

    def assert_rejected(self, fragment):
        errors = self.errors()
        self.assertTrue(any(fragment in error for error in errors), errors)

    def test_minimal_draft_and_reserved_specification_are_valid(self):
        self.add_document(15, 'business_spec', 'planned', 'specs/m02')
        self.assertEqual([], self.errors())

    def test_reserved_document_must_not_be_presented_as_a_written_specification(self):
        document = self.add_document(15, 'business_spec', 'planned', 'specs/m02')
        self.write(document['path'], '# CHNHCM-015 Sample\n')
        self.assert_rejected('planned document must not have a file')

    def test_unknown_state_and_duplicate_identity_or_path_are_rejected(self):
        self.policy['status'] = 'approved-and-tested'
        self.assert_rejected('invalid document status')
        self.policy['status'] = 'draft'
        self.registry['documents'].append(deepcopy(self.policy))
        self.assert_rejected('duplicate document ID')
        self.assert_rejected('duplicate document path')

    def test_declared_type_number_and_title_must_match_the_document(self):
        self.policy['type'] = 'decision'
        self.assert_rejected('path does not match document type')
        self.policy['type'] = 'governance'
        self.policy['title'] = 'Different title'
        self.assert_rejected('first heading must exactly match')
        self.policy['path'] = 'docs/governance/099-sample.md'
        self.assert_rejected('filename must use its number')

    def test_activation_needs_a_real_review_record_without_promoting_implementation(self):
        self.policy['status'] = 'active'
        self.assert_rejected('active document requires accepted review')
        report = self.add_report(result='not_run')
        self.policy['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Document contract',
                                 'result': 'accepted', 'evidence': report['path']}
        self.assertEqual([], self.errors())
        self.assertEqual('not_run', report['results'][0]['result'])

    def test_documents_and_new_reports_cannot_claim_legacy_exemptions(self):
        self.policy['format'] = 'legacy'
        self.assert_rejected('document format must be numbered')
        self.registry['governance']['legacy_documents'].append('CHNHCM-014')
        self.assert_rejected('legacy_documents must be empty')
        self.policy['format'] = 'numbered'
        self.registry['governance']['legacy_documents'] = []
        report = self.add_report()
        report['format'] = 'legacy'
        self.assert_rejected('legacy report format is restricted')

    def test_original_documents_must_use_numbered_format(self):
        for number in range(1, 10):
            with self.subTest(number=number):
                document = self.add_document(number)
                document['format'] = 'legacy'
                self.registry['governance']['legacy_documents'] = [document['id']]
                self.assert_rejected(f'{document["id"]}: document format must be numbered')
                document['format'] = 'numbered'
                self.registry['governance']['legacy_documents'] = []
                self.assertEqual([], self.errors())

    def test_original_documents_cannot_restore_allowlist(self):
        for number in range(1, 10):
            with self.subTest(number=number):
                document = self.add_document(number)
                self.registry['governance']['legacy_documents'] = [document['id']]
                self.assert_rejected('legacy_documents must be empty')
        self.registry['governance']['legacy_documents'] = []
        self.assertEqual([], self.errors())

    def test_original_active_documents_require_review_and_numbered_headings(self):
        report = self.add_report()
        for number in range(1, 10):
            with self.subTest(number=number):
                document = self.add_document(number, status='active')
                self.assert_rejected(f'{document["id"]}: active document requires accepted review')
                document['format'] = 'legacy'
                self.registry['governance']['legacy_documents'] = [document['id']]
                self.assert_rejected(f'{document["id"]}: active document requires accepted review')
                document['format'] = 'numbered'
                self.registry['governance']['legacy_documents'] = []
                document['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Format migration',
                                      'result': 'accepted', 'evidence': report['path']}
                self.assertEqual([], self.errors())
                self.write(document['path'], f'# {document["id"]} Sample\n\n## Purpose\n\nContent.\n')
                self.assert_rejected('numbered H2-H4 required')
                document['format'] = 'legacy'
                self.registry['governance']['legacy_documents'] = [document['id']]
                self.assert_rejected(f'{document["path"]}: numbered H2-H4 required')
                document['format'] = 'numbered'
                self.registry['governance']['legacy_documents'] = []
                self.write(document['path'], f'# {document["id"]} Sample\n\n## 1 Purpose\n\nContent.\n')
                self.assertEqual([], self.errors())

    def test_original_evidence_can_retain_legacy_format(self):
        for path in ('docs/verification/bootstrap.md', 'docs/verification/009-m02-terminology.md'):
            with self.subTest(path=path):
                record = self.add_report()
                old_path = record['path']
                record.update(path=path, format='legacy', working_tree='historical-unspecified',
                              base_commit=None if path.endswith('/bootstrap.md') else 'b' * 40)
                record['results'][0]['evidence'] = path
                (self.root / old_path).unlink()
                self.write('docs/index.md', '# Documentation\n')
                self.link(self.policy['path'])
                self.write(path, '# Original evidence\n\n## Original results\n\nHistorical scope.\n')
                self.link(path)
                self.assertEqual([], self.errors())
                self.registry['governance']['verification_records'].remove(record)
                (self.root / path).unlink()

    def test_unregistered_body_and_duplicate_numbered_body_are_rejected(self):
        self.write('docs/adr/015-hidden.md', '# CHNHCM-015 Hidden decision\n')
        self.assert_rejected('unregistered document or report')
        (self.root / 'docs/adr/015-hidden.md').unlink()
        self.write('docs/governance/hidden.md', '# Hidden rule\n')
        self.assert_rejected('unregistered document or report')

    def test_templates_and_code_examples_do_not_allocate_document_numbers(self):
        self.write('docs/templates/decision.md', '# Template\n\n```markdown\n# CHNHCM-999 Example\n[Missing](does-not-exist.md)\n```\n')
        self.assertEqual([], self.errors())

    def test_imported_and_template_directories_cannot_hide_numbered_documents(self):
        for directory in ('docs/architecture/target', 'docs/templates'):
            for filename, title in (('999-hidden.md', '# Hidden document'), ('hidden.md', '# CHNHCM-999 Hidden document')):
                with self.subTest(directory=directory, filename=filename):
                    path = f'{directory}/{filename}'
                    self.write(path, title + '\n')
                    self.assert_rejected('unregistered document or report')
                    (self.root / path).unlink()
        self.write('docs/architecture/target/02-source-design.md', '# Original imported design\n\n## Original heading\n')
        self.assertEqual([], self.errors())

    def test_root_markdown_and_path_escape_are_rejected(self):
        self.write('README.md', '# Misplaced entry\n')
        self.assert_rejected('root Markdown must be AGENTS.md')
        (self.root / 'README.md').unlink()
        self.policy['path'] = '../outside.md'
        self.assert_rejected('invalid document path')

    def test_path_symlink_must_not_read_outside_the_repository(self):
        outside = self.root.parent / 'outside.md'
        outside.write_text('# Outside\n', encoding='utf-8')
        destination = self.root / self.policy['path']
        destination.unlink()
        destination.symlink_to(outside)
        self.assert_rejected('invalid document path')

    def test_existing_registered_document_must_be_reachable_from_entry(self):
        self.write('docs/index.md', '# Documentation\n')
        self.assert_rejected('document not reachable')

    def test_completed_plan_requires_closeout_while_effective_spec_cannot_be_archived(self):
        plan = self.add_document(15, 'plan', directory='plans')
        plan['execution_status'] = 'completed'
        self.assert_rejected('CHNHCM-015: closure required')
        plan['closure'] = {'date': '2026-10-30', 'reason': 'Scoped work delivered.'}
        self.assert_rejected('physical archive or substantive handoff deferral required')
        plan['archive_deferral'] = dict(kind='historical_links', reason='Referenced by a final report.',
                                        owner_role='Maintainer', recheck='On next revision')
        self.assert_rejected('deferral requires a pending rule or handoff')
        plan['archive_deferral'].update(kind='rules_pending', reason='Existing contract lacks the import date rule.')
        self.assertEqual([], self.errors())
        plan['archive_deferral']['recheck'] = ''
        self.assert_rejected('deferral requires a pending rule or handoff')

        original = self.policy['path']
        self.policy['path'] = 'docs/archive/governance/014-sample.md'
        self.policy['archive'] = {'date': '2026-10-30', 'reason': 'Attempt to archive a current rule.', 'previous_path': original}
        self.write(self.policy['path'], (self.root / original).read_text())
        (self.root / original).unlink()
        self.assert_rejected('effective specification cannot be archived')

    def test_closed_plan_moves_to_its_mirrored_path_and_keeps_original_number(self):
        plan = self.add_document(15, 'plan', status='active', directory='plans')
        report = self.add_report()
        report['related_documents'].append('CHNHCM-015')
        plan['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Plan and scoped closeout',
                          'result': 'accepted', 'evidence': report['path']}
        plan['execution_status'] = 'completed'
        plan['closure'] = {'date': '2026-10-30', 'reason': 'Scoped work complete; lasting rules extracted.'}
        original = plan['path']
        plan['path'] = 'docs/archive/plans/015-sample.md'
        plan['archive'] = {'date': '2026-10-30', 'reason': 'Closed plan retained.', 'previous_path': original}
        self.write(plan['path'], (self.root / original).read_text())
        (self.root / original).unlink()
        entry = (self.root / 'docs/index.md').read_text()
        self.write('docs/index.md', entry.replace(original.removeprefix('docs/'), plan['path'].removeprefix('docs/')))
        self.assertEqual([], self.errors())
        for previous in ('docs/plans/not-a-document', 'docs/plans/099-sample.md', 'docs/adr/015-sample.md'):
            with self.subTest(previous=previous):
                plan['archive']['previous_path'] = previous
                self.assert_rejected('invalid archive previous_path')
        plan['archive']['previous_path'] = original
        plan['archive']['date'] = '2025-10-30'
        self.assertEqual([], self.errors())  # Dates do not determine the directory.
        self.write(original, '# Historical placeholder')
        self.assert_rejected('original file still exists after physical archival')

    def archive_document(self, document):
        original = document['path']
        target = 'docs/archive/' + original.removeprefix('docs/')
        self.write(target, (self.root / original).read_text())
        (self.root / original).unlink()
        index = (self.root / 'docs/index.md').read_text()
        self.write('docs/index.md', index.replace(original.removeprefix('docs/'), target.removeprefix('docs/')))
        document.update(path=target, archive={'date': '2026-10-30', 'reason': 'Handoff complete.', 'previous_path': original})

    def test_archived_module_spec_preserves_every_directory_and_filename(self):
        spec = self.add_document(15, 'functional_spec', status='retired', directory='specs/m02/detail')
        spec['closure'] = {'date': '2026-10-30', 'reason': 'Rule retired.'}
        self.archive_document(spec)
        self.assertEqual([], self.errors())
        for target in ('docs/archive/2026/specs/m02/detail/015-sample.md',
                       'docs/archive/specs/015-sample.md', 'docs/archive/specs/m02/detail/015-renamed.md'):
            with self.subTest(target=target):
                spec['path'] = target
                self.assert_rejected('archive path must mirror the full original path')

    def test_unwritten_cancelled_document_cannot_claim_an_archive(self):
        plan = self.add_document(15, 'plan', status='planned', directory='plans')
        plan.update(status='cancelled', execution_status='cancelled',
                    closure={'date': '2026-10-30', 'reason': 'Cancelled before drafting.'})
        self.assertEqual([], self.errors())
        original = plan['path']
        plan['path'] = 'docs/archive/plans/015-sample.md'
        plan['archive'] = {'date': '2026-10-30', 'reason': 'Invalid archive claim.', 'previous_path': original}
        self.assert_rejected('archive requires an existing regular document file')

    def test_superseded_documents_keep_a_noncyclic_replacement_route(self):
        second = self.add_document(15, status='active')
        report = self.add_report()
        report['related_documents'].append('CHNHCM-015')
        second['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Replacement contract',
                            'result': 'accepted', 'evidence': report['path']}
        self.policy.update(status='superseded', superseded_by=['CHNHCM-015'],
                           closure={'date': '2026-10-30', 'reason': 'Replaced scope.'})
        self.archive_document(self.policy)
        self.assertEqual([], self.errors())
        second.update(status='superseded', superseded_by=['CHNHCM-014'], closure={'date': '2026-10-30', 'reason': 'Bad cycle.'})
        self.assert_rejected('cyclic document replacement')
        second['superseded_by'] = ['CHNHCM-099']
        self.assert_rejected('missing or self replacement')

    def test_replacement_cannot_be_a_reserved_draft_pending_or_cancelled_proposal(self):
        second = self.add_document(15)
        self.policy.update(status='superseded', superseded_by=['CHNHCM-015'],
                           closure={'date': '2026-10-30', 'reason': 'Attempted replacement.'})
        for status in ('planned', 'draft', 'in_review', 'cancelled'):
            with self.subTest(status=status):
                second['status'] = status
                if status == 'cancelled':
                    second['closure'] = {'date': '2026-10-30', 'reason': 'Proposal cancelled.'}
                self.assert_rejected('replacement must be an adopted document')

    def test_replacement_route_requires_the_successor_body(self):
        second = self.add_document(15, status='retired')
        second['closure'] = {'date': '2026-10-30', 'reason': 'Historical rule retired.'}
        self.policy.update(status='superseded', superseded_by=['CHNHCM-015'],
                           closure={'date': '2026-10-30', 'reason': 'Historical replacement.'})
        self.archive_document(self.policy)
        self.archive_document(second)
        self.assertEqual([], self.errors())
        (self.root / second['path']).unlink()
        self.assert_rejected('replacement document file required')

    def test_stable_rule_ids_require_anchors_and_are_unique_across_documents(self):
        text = '# CHNHCM-014 Sample\n\n## 1 M02-BR-001 Rule\n'
        self.write(self.policy['path'], text)
        self.assert_rejected('stable ID heading requires explicit anchor')
        text += '\n<a id="m02-br-001"></a>\n'
        self.write(self.policy['path'], text)
        self.assertEqual([], self.errors())
        second = self.add_document(15)
        self.write(second['path'], '# CHNHCM-015 Sample\n\n## 1 Rule\n\n<a id="m02-br-001"></a>\n')
        self.assert_rejected('duplicate stable ID anchor')

    def test_rule_extraction_keeps_archived_history_and_current_definition(self):
        plan = self.add_document(15, 'plan', status='active', directory='plans')
        specification = self.add_document(16, 'business_spec', status='active', directory='specs/m02')
        report = self.add_report()
        report['related_documents'].extend(['CHNHCM-015', 'CHNHCM-016'])
        for document in (plan, specification):
            document['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Rule extraction',
                                  'result': 'accepted', 'evidence': report['path']}
        original = plan['path']
        plan.update(execution_status='completed', closure={'date': '2026-10-30', 'reason': 'Lasting rule extracted.'},
                    archive={'date': '2026-10-30', 'reason': 'Retain historical rule.', 'previous_path': original},
                    path='docs/archive/plans/015-sample.md')
        rule = '\n\n## 1 M02-BR-001 Rule\n\n<a id="m02-br-001"></a>\n\nOriginal rule retained.\n'
        self.write(plan['path'], '# CHNHCM-015 Sample' + rule +
                   '\n[Current rule](../../specs/m02/016-sample.md#m02-br-001)\n')
        self.write(specification['path'], '# CHNHCM-016 Sample' + rule)
        (self.root / original).unlink()
        index = (self.root / 'docs/index.md').read_text()
        self.write('docs/index.md', index.replace(original.removeprefix('docs/'), plan['path'].removeprefix('docs/')))
        self.assertEqual([], self.errors())
        self.write(plan['path'], '# CHNHCM-015 Sample' + rule + '\n<a id="m02-br-001"></a>\n')
        self.assert_rejected('duplicate stable ID anchor in document')

    def test_historical_rule_alone_does_not_replace_physical_archival(self):
        current = self.add_document(15, status='active')
        report = self.add_report()
        report['related_documents'].append('CHNHCM-015')
        current['review'] = {'date': '2026-10-30', 'reviewer': 'Reviewer', 'scope': 'Current rule',
                             'result': 'accepted', 'evidence': report['path']}
        rule = '\n\n## 1 M02-BR-001 Rule\n\n<a id="m02-br-001"></a>\n'
        self.write(self.policy['path'], '# CHNHCM-014 Sample' + rule)
        self.write(current['path'], '# CHNHCM-015 Sample' + rule)
        for status in ('superseded', 'retired', 'cancelled'):
            with self.subTest(status=status):
                self.policy.update(status=status, closure={'date': '2026-10-30', 'reason': 'Historical definition retained.'},
                                   superseded_by=['CHNHCM-015'] if status == 'superseded' else [])
                self.assert_rejected('physical archive or substantive handoff deferral required')
        self.policy.update(status='draft', closure=None, superseded_by=[])
        self.assert_rejected('duplicate stable ID anchor')

    def test_heading_rule_reference_does_not_define_a_second_rule(self):
        source = self.add_document(15, 'business_spec', directory='specs/m02')
        consumer = self.add_document(16, 'functional_spec', directory='specs/m02')
        self.write(source['path'], '# CHNHCM-015 Sample\n\n## 1 M02-BR-001 Rule\n\n<a id="m02-br-001"></a>\n')
        for link, definitions in (
                ('[M02-BR-001](015-sample.md#m02-br-001)', ''),
                ('[M02-BR-001][source]', '\n[source]: 015-sample.md#m02-br-001\n'),
                ('[M02-BR-001][]', '\n[M02-BR-001]: 015-sample.md#m02-br-001\n'),
                ('[M02-BR-001]', '\n[M02-BR-001]: 015-sample.md#m02-br-001\n')):
            with self.subTest(link=link):
                self.write(consumer['path'], '# CHNHCM-016 Sample\n\n## 1 Mapping\n\n### 1.1 ' + link + ' UI mapping\n' + definitions)
                self.assertEqual([], self.errors())

    def test_heading_reference_still_checks_its_actual_target(self):
        source = self.add_document(15, 'business_spec', directory='specs/m02')
        consumer = self.add_document(16, 'functional_spec', directory='specs/m02')
        self.write(source['path'], '# CHNHCM-015 Sample\n\n## 1 Rules\n\n<a id="m02-br-002"></a>\n')
        for target, error in (
                ('missing.md#m02-br-001', 'broken local link'),
                ('015-sample.md#m02-br-001', 'missing explicit target anchor'),
                ('015-sample.md#m02-br-002', 'stable ID reference must target its own local anchor'),
                ('015-sample.md', 'stable ID reference must target its own local anchor')):
            with self.subTest(target=target):
                self.write(consumer['path'], '# CHNHCM-016 Sample\n\n## 1 [M02-BR-001](' + target + ') Mapping\n')
                self.assert_rejected(error)

    def test_linked_mention_does_not_hide_an_unlinked_definition_in_same_heading(self):
        source = self.add_document(15, 'business_spec', directory='specs/m02')
        consumer = self.add_document(16, 'functional_spec', directory='specs/m02')
        self.write(source['path'], '# CHNHCM-015 Sample\n\n## 1 Rule\n\n<a id="m02-br-001"></a>\n')
        self.write(consumer['path'], '# CHNHCM-016 Sample\n\n## 1 [M02-BR-001](015-sample.md#m02-br-001) and M02-BR-001\n')
        self.assert_rejected('stable ID heading requires explicit anchor')

    def test_reference_links_are_navigable_and_missing_stable_anchors_fail(self):
        self.write('docs/index.md', '# Documentation\n\n[Policy][policy]\n\n[policy]: governance/014-sample.md\n')
        self.assertEqual([], self.errors())
        self.write('docs/index.md', '# Documentation\n\n[Policy](governance/014-sample.md#m02-br-999)\n')
        self.assert_rejected('missing explicit target anchor')

    def test_inline_anchor_examples_do_not_allocate_stable_ids(self):
        self.write(self.policy['path'], '# CHNHCM-014 Sample\n\n## 1 Example\n\nUse `<a id="m02-br-001"></a>`.\n')
        second = self.add_document(15)
        self.write(second['path'], '# CHNHCM-015 Sample\n\n## 1 M02-BR-001 Actual rule\n\n<a id="m02-br-001"></a>\n')
        self.assertEqual([], self.errors())

    def test_inline_anchor_example_cannot_satisfy_a_link_target(self):
        self.write(self.policy['path'], '# CHNHCM-014 Sample\n\n## 1 Example\n\nUse `<a id="m02-br-001"></a>`.\n')
        self.write('docs/index.md', '# Documentation\n\n[Rule](governance/014-sample.md#m02-br-001)\n')
        self.assert_rejected('missing explicit target anchor')

    def test_stable_identifiers_reject_unknown_modules_and_zero_serial(self):
        for identifier in ('M99-BR-001', 'X99-SCN-001', 'M02-AC-000'):
            with self.subTest(identifier=identifier):
                self.write(self.policy['path'], f'# CHNHCM-014 Sample\n\n## 1 {identifier} Rule\n\n<a id="{identifier.lower()}"></a>\n')
                self.assert_rejected('invalid stable ID anchor')
                self.assert_rejected('invalid stable ID:')

    def test_final_report_can_preserve_failure_and_interruption(self):
        report = self.add_report(result='failed')
        self.assertEqual([], self.errors())
        report['results'][0]['result'] = 'interrupted'
        self.assertEqual([], self.errors())
        report['results'][0]['result'] = 'success-ish'
        self.assert_rejected('invalid check result')

    def test_new_report_requires_a_baseline_worktree_state_and_existing_evidence(self):
        report = self.add_report()
        report['working_tree'] = 'historical-unspecified'
        self.assert_rejected('new reports must identify working tree state')
        report['working_tree'] = 'dirty'
        report['base_commit'] = None
        self.assert_rejected('tested base commit required')
        report['base_commit'] = 'a' * 40
        report['results'][0]['evidence'] = 'docs/verification/missing.log'
        self.assert_rejected('missing or unsafe evidence path')

    def test_report_title_cannot_claim_an_unrelated_document(self):
        report = self.add_report()
        self.write(report['path'], '# CHNHCM-099 Other scope\n\n## 1 Results\n')
        self.assert_rejected('title ID must be in related_documents')

    def test_missing_or_malformed_metadata_returns_errors(self):
        self.policy['modules'] = {'M02': True}
        self.policy['superseded_by'] = ['CHNHCM-015', []]
        self.registry['governance']['policy_document'] = []
        del self.policy['scope']
        self.assert_rejected('missing metadata fields')
        self.assert_rejected('invalid modules')


class MarkdownConventionTests(unittest.TestCase):
    def test_numbering_checks_hierarchy_continuity_and_duplicate_h1(self):
        valid = '# Title\n\n## 1 Scope\n### 1.1 First\n#### 1.1.1 Detail\n### 1.2 Next\n## 2 Results\n'
        self.assertEqual([], heading_errors(valid, 'fixture'))
        self.assertTrue(heading_errors(valid.replace('1.2 Next', '1.4 Next'), 'fixture'))
        self.assertTrue(heading_errors('# Title\n### 1.1 Jump\n', 'fixture'))
        self.assertTrue(heading_errors(valid + '# Another title\n', 'fixture'))

    def test_fences_do_not_create_headings_or_links(self):
        text = '# Title\n\n## 1 Scope\n\n````markdown\n# Duplicate\n```\n[Fake](missing.md)\n````\n'
        self.assertEqual([], heading_errors(text, 'fixture'))
        self.assertEqual([], markdown_links(text))

    def test_inline_reference_and_shortcut_links_are_supported(self):
        text = '[A](a.md)\n[B][b]\n[C][]\n[shortcut]\n\n[b]: b.md\n[C]: <c.md>\n[shortcut]: d.md\n'
        self.assertEqual(['a.md', 'b.md', 'c.md', 'd.md'], markdown_links(text))

    def test_anchor_examples_inside_variable_length_code_spans_are_inert(self):
        text = '`<a id="m02-br-001"></a>` and ``code with ` inside <a id="m02-br-002"></a>``\n<a id="m02-br-003"></a>\n'
        self.assertEqual(['m02-br-003'], markdown_anchors(text))
        self.assertEqual([], markdown_links('``[fake](missing.md) with ` inside``'))


if __name__ == '__main__':
    unittest.main()
