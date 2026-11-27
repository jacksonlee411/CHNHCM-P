import shutil
import subprocess
import unittest
from copy import deepcopy
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
            (root / "docs/governance/README.md").write_text("# 规范导航\n", encoding="utf-8")
            (root / "docs/specs").mkdir()
            (root / "docs/specs/README.md").write_text("# 规格\n", encoding="utf-8")
            evidence = root / "docs/verification/record.md"
            evidence.parent.mkdir()
            evidence.write_text("# 初次评审\n", encoding="utf-8")
            registry = {"documents": [], "governance": {"verification_records": [
                {"path": "docs/verification/record.md", "related_documents": ["CHNHCM-511"],
                 "date": "2026-11-26", "status": "final", "purpose": "retest",
                 "conclusion": "本地 8 项通过 | 远程未重跑", "evidence_access": "public_ci"}
            ]}}
            with patch.multiple(document_catalog, ROOT=root, CATALOG=catalog):
                self.assertFalse(document_catalog.catalog_matches(registry))
                for path, content in document_catalog.catalog_files(registry).items():
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(content, encoding="utf-8")
                self.assertTrue(document_catalog.catalog_matches(registry))
                evidence.write_text("# 实施复验\n", encoding="utf-8")
                self.assertFalse(document_catalog.catalog_matches(registry))

    def test_evidence_groups_show_the_conclusion_access_and_follow_up_for_each_report(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "docs/verification").mkdir(parents=True)
            review_path = "docs/verification/review.md"
            retest_path = "docs/verification/retest.md"
            (root / review_path).write_text("# 初评\n", encoding="utf-8")
            (root / retest_path).write_text("# 专项复验\n", encoding="utf-8")
            review = dict(path=review_path, related_documents=["CHNHCM-511"], date="2026-11-25", status="draft",
                          purpose="review", conclusion="两项待改", evidence_access="restricted", follow_up=retest_path)
            retest = dict(path=retest_path, related_documents=["CHNHCM-511"], date="2026-11-26", status="final",
                          purpose="retest", conclusion="本地 8 项通过 | 远程未重跑", evidence_access="public_ci")
            registry = {"documents": [], "governance": {"verification_records": [review, retest]}}
            with patch.multiple(document_catalog, ROOT=root, CATALOG=root / "docs/governance/catalog/README.md"):
                page = document_catalog.render(registry)
            self.assertIn("[问题复验（1）](#evidence-retest)", page)
            self.assertIn("[方案评审（1）](#evidence-review)", page)
            self.assertIn("定稿 | 本地 8 项通过 ｜ 远程未重跑 | 本仓报告及公开 CI |", page)
            self.assertIn("草稿 | 两项待改 [后续记录](../../verification/retest.md) | 本仓仅有摘要，原件需权限 |", page)
            self.assertNotIn("#evidence-handoff", page)

    @unittest.skipUnless(shutil.which("rg"), "ripgrep is needed to exercise its default search")
    def test_default_search_uses_the_same_active_documents_as_the_standards_page(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            catalog = root / "docs/governance/catalog/README.md"
            (root / "docs/governance").mkdir(parents=True)
            (root / "docs/specs").mkdir()
            (root / "docs/specs/README.md").write_text("# 规格\n", encoding="utf-8")
            (root / "docs/governance/README.md").write_text(
                "[保存与返回](../specs/m01/001-current.md#save) · "
                "[待审跳转](../specs/m01/002-review.md#save)\n", encoding="utf-8")
            cases = [
                ("001", "specs/m01/001-current.md", "business_spec", "active", None, None),
                ("002", "specs/m01/002-review.md", "business_spec", "in_review", None, None),
                ("003", "plans/003-plan.md", "plan", "active", "in_progress", None),
                ("004", "archive/adr/004-old.md", "decision", "superseded", None, {"date": "2026-11-26"}),
            ]
            documents = []
            for number, path, kind, status, execution, archive in cases:
                relative = "docs/" + path
                documents.append(dict(id="CHNHCM-" + number, title=number, path=relative, type=kind,
                                      status=status, execution_status=execution, archive=archive, superseded_by=[]))
                (root / relative).parent.mkdir(parents=True, exist_ok=True)
                (root / relative).write_text("共同关键词\n\n<a id=\"save\"></a>\n## 1 保存\n\n## 2 固定办理\n",
                                             encoding="utf-8")
            report = root / "docs/verification/report.md"
            report.parent.mkdir()
            report.write_text("# 共同关键词：原验证\n", encoding="utf-8")
            documents[0]["superseded_by"] = ["CHNHCM-004"]
            documents[0]["clause_routes"] = {
                "retained": {"原保存规则": "#1-保存"},
                "CHNHCM-004": {"归档后承接的办理": "CHNHCM-001#2-固定办理"},
            }
            registry = {"documents": documents, "governance": {"verification_records": [
                {"path": "docs/verification/report.md", "related_documents": ["CHNHCM-001"],
                 "date": "2026-11-26", "status": "final", "purpose": "acceptance",
                 "conclusion": "未覆盖故障恢复", "evidence_access": "mixed"}]}}
            with patch.multiple(document_catalog, ROOT=root, CATALOG=catalog):
                for path, content in document_catalog.catalog_files(registry).items():
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(content, encoding="utf-8")
                result = subprocess.check_output(["rg", "--no-config", "-l", "共同关键词", "docs"],
                                                 cwd=root, text=True)
                self.assertEqual(result.splitlines(), [documents[0]["path"]])
                self.assertIn("docs/verification/report.md", subprocess.check_output(
                    ["rg", "--no-config", "--no-ignore-dot", "-l", "共同关键词", "docs/verification"],
                    cwd=root, text=True).splitlines())
                page = (catalog.parent / "standards/README.md").read_text(encoding="utf-8")
                self.assertIn("[保存与返回]", page)
                self.assertIn("[归档后承接的办理（据 004） → 001 §2](../../../specs/m01/001-current.md#2-固定办理)", page)
                self.assertNotIn("待审跳转", page)
                self.assertNotIn("原验证", page)
                self.assertNotIn("未覆盖故障恢复", page)
                self.assertNotIn("004-old.md", page)
                self.assertEqual(subprocess.run(["rg", "--no-config", "-q", "原验证", "docs"], cwd=root).returncode, 1)
                documents[1]["status"] = "active"
                self.assertFalse(document_catalog.catalog_matches(registry))
                self.assertIn("!/specs/m01/002-review.md", document_catalog.catalog_files(registry)[root / "docs/.ignore"])
                documents[1]["status"] = "in_review"
                (root / "docs/.ignore").write_text("", encoding="utf-8")
                self.assertFalse(document_catalog.catalog_matches(registry))

    def test_partial_replacement_and_review_adoption_lead_to_the_current_clause(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            catalog = root / "docs/governance/catalog/README.md"
            (root / "docs/governance").mkdir(parents=True)
            (root / "docs/specs").mkdir()
            for directory in ("governance", "specs"):
                (root / f"docs/{directory}/README.md").write_text("# 导航\n")
            old, current, review = [
                dict(id=f"CHNHCM-{n}", title=title, path=f"docs/specs/{n}.md", type="decision",
                     status=status, execution_status=None, archive=None, superseded_by=[])
                for n, title, status in (("001", "原约定", "active"), ("002", "接收合同", "active"),
                                         ("003", "送审稿", "in_review"))
            ]
            (root / old["path"]).write_text("## 1 保留规则\n")
            (root / current["path"]).write_text("## 1 现行规则\n\n### 1.1 列表、分页\n\n<a id=\"snapshot\"></a>\n")
            (root / review["path"]).write_text("## 1 已采纳部分\n\n## 2 待决建议\n")
            old["superseded_by"] = [current["id"]]
            old["clause_routes"] = {"retained": {"保存": "#1-保留规则"}, current["id"]: {"列表": "#11-列表分页"}}
            review["clause_routes"] = {"1-已采纳部分": {"读取一致性": "CHNHCM-002#snapshot"}}
            documents = [old, current, review]
            registry = {"documents": documents, "governance": {"verification_records": []}}
            with patch.multiple(document_catalog, ROOT=root, CATALOG=catalog):
                pages = document_catalog.catalog_files(registry)
                index = pages[catalog.parent / "standards/README.md"]
                original = next(line for line in index.splitlines() if "[001 " in line)
                receiver = next(line for line in index.splitlines() if "[002 " in line)
                self.assertIn("[保存 → 001 §1](../../../specs/001.md#1-保留规则)", original)
                self.assertIn("[列表 → 002 §1.1](../../../specs/002.md#11-列表分页)", original)
                self.assertIn("[读取一致性 → 002](../../../specs/002.md#snapshot)", receiver)
                self.assertIn("读取一致性 → 002](../../specs/002.md#snapshot)", pages[catalog])
                self.assertNotIn("003.md", index + pages[root / "docs/.ignore"])
                self.assertNotIn("待决建议", index + pages[catalog])

                invalid = [
                    (0, {"retained": {"保存": "#1-保留规则"}}, "每个后继"),
                    (0, {"retained": {"保存": "#1-保留规则"}, current["id"]: {}}, "缺少条款"),
                    (2, {"2-不存在": {"读取": "CHNHCM-002#snapshot"}}, "找不到条款锚点"),
                    (2, {"1-已采纳部分": {"读取": "CHNHCM-002"}}, "具体条款"),
                    (2, {"1-已采纳部分": {"读取": "CHNHCM-002#missing"}}, "找不到条款锚点"),
                    (2, {"1-已采纳部分": {"读取": "CHNHCM-003#2-待决建议"}}, "现行规范"),
                ]
                for source, routes, message in invalid:
                    with self.subTest(routes=routes):
                        changed = deepcopy(documents)
                        changed[source]["clause_routes"] = routes
                        with self.assertRaisesRegex(ValueError, message):
                            document_catalog.clause_navigation(changed)

                for status, kind, archive in (("draft", "decision", None), ("active", "plan", None),
                                              ("superseded", "decision", {"date": "2026-11-26"})):
                    with self.subTest(status=status, kind=kind):
                        changed = deepcopy(documents)
                        changed[1].update(status=status, type=kind, archive=archive,
                                          execution_status="in_progress" if kind == "plan" else None)
                        with self.assertRaisesRegex(ValueError, "现行规范"):
                            document_catalog.clause_navigation(changed)
