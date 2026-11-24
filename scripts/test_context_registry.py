import json
from pathlib import Path
import unittest

from context_registry import slice_errors


class SliceEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]
        contexts = json.loads((self.root/'docs/architecture/contexts.json').read_text())['contexts']
        self.staffing = next(c for c in contexts if c['name'] == 'staffing')
        registry = json.loads((self.root/'docs/governance/registry.json').read_text())
        self.reports = {r['path'] for r in registry['governance']['verification_records']}

    def issues(self):
        return slice_errors(self.root, [self.staffing], self.reports)

    def test_no_implicit_m04_acceptance(self):
        self.staffing['slices'] = [s for s in self.staffing['slices'] if s['module'] != 'M04']
        self.assertTrue(self.issues())

    def test_contract_review_alone_cannot_sign_off_a_runtime(self):
        capacity = next(s for s in self.staffing['slices'] if s['capability'] == '岗位核编')
        capacity.update(state='verified_slice', source='internal/contexts/staffing/public/contracts.go')
        self.assertTrue(self.issues())

    def test_verified_slice_requires_its_evidence(self):
        del self.staffing['slices'][0]['evidence']
        self.assertTrue(self.issues())
