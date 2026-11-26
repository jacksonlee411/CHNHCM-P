import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

import document_catalog

from document_catalog import categories


class DocumentCatalogTests(unittest.TestCase):
    def test_review_and_closed_are_independent(self):
        doc = {"type": "plan", "status": "in_review", "execution_status": "completed", "archive": None}
        self.assertEqual(categories(doc), ["closed", "review"])
        self.assertEqual(categories(dict(doc, status="active")), ["closed"])
        self.assertEqual(categories(dict(doc, status="active", execution_status="in_progress")), ["in-progress"])

    def test_archived_and_replaced_do_not_become_current_standards(self):
        doc = {"type": "business_spec", "status": "active", "archive": None}
        self.assertEqual(categories(doc), ["standards"])
        self.assertEqual(categories(dict(doc, status="superseded")), ["history"])
        self.assertEqual(categories(dict(doc, type="plan", status="superseded",
                                         execution_status="completed", archive={"date": "2026-11-26"})), ["closed"])

    def test_verification_title_changes_invalidate_the_rendered_catalog(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            catalog = root / "docs/governance/catalog/README.md"
            catalog.parent.mkdir(parents=True)
            evidence = root / "docs/verification/record.md"
            evidence.parent.mkdir()
            evidence.write_text("# 初次评审\n", encoding="utf-8")
            registry = {"documents": [], "governance": {"verification_records": [
                {"path": "docs/verification/record.md", "related_documents": ["CHNHCM-511"],
                 "date": "2026-11-26", "status": "final"}
            ]}}
            with patch.multiple(document_catalog, ROOT=root, CATALOG=catalog):
                self.assertFalse(document_catalog.catalog_matches(registry))
                catalog.write_text(document_catalog.render(registry), encoding="utf-8")
                self.assertTrue(document_catalog.catalog_matches(registry))
                evidence.write_text("# 实施复验\n", encoding="utf-8")
                self.assertFalse(document_catalog.catalog_matches(registry))
