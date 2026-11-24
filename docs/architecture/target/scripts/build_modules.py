#!/usr/bin/env python3
"""Build the authored module documentation and Mermaid diagrams, offline.

Source facts and design decisions live in data/*-modules.json. This script only
formats those reviewed descriptions; it does not infer product completeness.
"""
import argparse
import csv
import io
import json
import re
import hashlib
import posixpath
from datetime import date
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[1]
HISTORY = "../archive/2026-10-29-prototype-eb8a10d8/module-comparison.md"

def safe(value):
    return str(value).replace('"', "'").replace("|", "／").replace("\n", " ")

def validate_naming_review(data, groups, terms):
    """Validate optional naming decisions before formatting any output."""
    if "naming_review" not in data:
        return
    review = data["naming_review"]
    if not isinstance(review, dict):
        raise ValueError("input validation: naming_review must be an object")

    def required_text(row, key):
        value = row.get(key)
        if not isinstance(value, str) or not value.strip():
            raise ValueError("input validation: missing naming review text: " + key)
        return value

    reviewed_on = required_text(review, "reviewed_on")
    try:
        if not re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}", reviewed_on):
            raise ValueError()
        date.fromisoformat(reviewed_on)
    except ValueError:
        raise ValueError("input validation: invalid naming review date") from None
    required_text(review, "scope")
    source_url = required_text(review, "source_url")
    try:
        if any(c.isspace() or ord(c) < 32 or ord(c) == 127 or c in "[]()<>`\\\"'|" for c in source_url):
            raise ValueError()
        parsed = urlsplit(source_url)
        if parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username is not None or parsed.password is not None:
            raise ValueError()
        parsed.port
    except ValueError:
        raise ValueError("input validation: naming review source_url must be a complete HTTP(S) URL without whitespace, credentials or Markdown delimiters") from None

    principles = review.get("principles")
    if not isinstance(principles, list) or not principles or any(not isinstance(p, str) or not p.strip() for p in principles):
        raise ValueError("input validation: naming review principles must contain text")
    titles = review.get("titles")
    if not isinstance(titles, list) or not titles:
        raise ValueError("input validation: naming review titles must be a nonempty list")
    seen = set()
    for row in titles:
        if not isinstance(row, dict):
            raise ValueError("input validation: naming review title must be an object")
        scope = required_text(row, "scope")
        if scope not in groups or scope in seen:
            raise ValueError("catalog mismatch: unknown or duplicate naming review title scope: " + scope)
        seen.add(scope)
        required_text(row, "reason")
    decisions = review.get("decisions")
    if not isinstance(decisions, list) or not decisions:
        raise ValueError("input validation: naming review decisions must be a nonempty list")
    for row in decisions:
        if not isinstance(row, dict):
            raise ValueError("input validation: naming review decision must be an object")
        ids = row.get("terms")
        if not isinstance(ids, list) or not ids or any(not isinstance(tid, str) for tid in ids):
            raise ValueError("input validation: naming review decision terms must be a nonempty string list")
        if len(ids) != len(set(ids)) or set(ids) - terms.keys():
            raise ValueError("catalog mismatch: unknown or duplicate naming review term")
        required_text(row, "decision")
        required_text(row, "reason")

def validate_material_review(data, groups, terms):
    """Keep the authored chapter review complete without reading local materials."""
    review = data.get("material_review")
    refs = {ref for term in terms.values() for ref in term.get("material_refs", [])}
    if review is None:
        if refs:
            raise ValueError("catalog mismatch: material references require review metadata")
        return
    if not isinstance(review, dict) or not re.fullmatch(r"[0-9a-f]{64}", review.get("source_sha256", "")):
        raise ValueError("input validation: invalid material source identity")
    if not isinstance(review.get("page_count"), int) or review["page_count"] < 1:
        raise ValueError("input validation: invalid material page count")
    if [row.get("scope") for row in review.get("chapters", [])] != groups:
        raise ValueError("catalog mismatch: material review must cover every glossary chapter in order")
    evidence = review.get("evidence", {})
    if not evidence or refs - evidence.keys():
        raise ValueError("catalog mismatch: unresolved material evidence")
    for rid, row in evidence.items():
        if (not re.fullmatch(r"p[0-9]{4}-t[0-9]{2}-r[0-9]{3}", rid) or
                not isinstance(row.get("pdf_page"), int) or
                not 1 <= row["pdf_page"] <= review["page_count"] or row["pdf_page"] != int(rid[1:5])):
            raise ValueError("input validation: invalid material row locator")
    for row in review["chapters"]:
        if any(not isinstance(row.get(k), str) or not row[k].strip() for k in ("adoption", "boundary")):
            raise ValueError("input validation: missing material chapter assessment")
        if set(row.get("terms", [])) - terms.keys() or set(row.get("evidence", [])) - evidence.keys():
            raise ValueError("catalog mismatch: unknown material chapter term or evidence")
        if any(not isinstance(page, int) or not 1 <= page <= review["page_count"] for page in row.get("pages", [])):
            raise ValueError("input validation: invalid material chapter page")


def load_glossary(modules, concepts):
    """Fail closed on incomplete or ambiguous authored terminology crosswalks."""
    data = json.loads((ROOT / "data/terminology-glossary.json").read_text())
    if data.get("schema_version") != 1 or data.get("title") != "China HCM 术语、中文名称与定义":
        raise ValueError("input validation: unsupported terminology glossary contract")
    groups = [g["id"] for g in data["groups"]]
    if groups != [m["id"] for m in modules]:
        raise ValueError("catalog mismatch: terminology groups must cover all modules in order")
    terms, scoped_names = {}, {}
    sources = {s["id"]: s for s in data["sources"]}
    if len(sources) != len(data["sources"]) or any(not s.get("title") or not s.get("url") for s in sources.values()):
        raise ValueError("input validation: duplicate or incomplete terminology sources")
    for entry in data["entries"]:
        tid = entry["id"]
        if tid in terms or not re.fullmatch(r"[a-z][a-z0-9-]*", tid):
            raise ValueError("input validation: invalid or duplicate term ID: " + tid)
        if entry["scope"] not in groups or entry["kind"] not in {"project", "reference"}:
            raise ValueError("input validation: invalid term context: " + tid)
        if any(not isinstance(entry.get(k), str) or not entry[k].strip() for k in ("term", "zh", "definition")):
            raise ValueError("input validation: missing terminology text: " + tid)
        if not re.search(r"[\u4e00-\u9fff]", entry["zh"]) or not re.search(r"[\u4e00-\u9fff]", entry["definition"]):
            raise ValueError("input validation: Chinese name/definition required: " + tid)
        if not entry["sources"] or set(entry["sources"]) - sources.keys():
            raise ValueError("input validation: missing terminology evidence: " + tid)
        for name in [entry["term"]] + entry["aliases"]:
            key = (entry["scope"], name.casefold())
            if key in scoped_names and scoped_names[key] != tid:
                raise ValueError("catalog mismatch: ambiguous scoped term: " + str(key))
            scoped_names[key] = tid
        terms[tid] = entry
    if {e["scope"] for e in terms.values()} != set(groups):
        raise ValueError("catalog mismatch: empty terminology domain")

    objects = {}
    expected_objects = {(m["id"], e[0]) for m in modules for e in m["entities"]}
    for binding in data["object_bindings"]:
        key = (binding["module"], binding["object_group"])
        if key in objects or key not in expected_objects:
            raise ValueError("catalog mismatch: duplicate or stale terminology object: " + str(key))
        names = binding["object_group"].split(" / ")
        expected = [scoped_names.get((binding["module"], n.casefold())) for n in names]
        if not binding["terms"] or None in expected or binding["terms"] != expected:
            raise ValueError("catalog mismatch: incomplete or wrong-context object terms: " + str(key))
        objects[key] = binding["terms"]
    if objects.keys() != expected_objects:
        raise ValueError("catalog mismatch: uncovered terminology objects: " + str(expected_objects - objects.keys()))

    references = {}
    expected_concepts = {c["id"]: c["concept"] for c in concepts}
    for binding in data["concept_bindings"]:
        cid = binding["concept_id"]
        ids = binding["terms"]
        if cid in references or binding["concept"] != expected_concepts.get(cid):
            raise ValueError("catalog mismatch: duplicate or stale terminology concept: " + cid)
        if not ids or len(ids) != len(set(ids)) or set(ids) - terms.keys():
            raise ValueError("catalog mismatch: invalid concept terms: " + cid)
        references[cid] = ids
    if references.keys() != expected_concepts.keys():
        raise ValueError("catalog mismatch: uncovered reference concepts")
    validate_naming_review(data, groups, terms)
    validate_material_review(data, groups, terms)
    return data, terms, objects, references


def term_links(ids, terms, prefix="", labels=None):
    labels = labels or [terms[tid]["term"] for tid in ids]
    return " / ".join("[" + safe(label) + "（" + safe(terms[tid]["zh"]) + "）](" +
                      prefix + "17-terminology-glossary.md#term-" + tid + ")"
                      for tid, label in zip(ids, labels))


def build_glossary(data, terms, objects, references):
    out = ["# " + data["title"], "",
           "[返回总览](README.md) · [概念映射](07-concept-mapping.md) · [聚合目录](14-aggregate-catalog.md)", "",
           "本表是 China HCM 当前目标设计的统一术语入口，按20个业务模块和6项横向能力分组。中文名称为本项目首选用语；定义说明业务含义及相邻概念边界，详细规则、生命周期和事务约束以模块说明及[DDD契约](10-ddd-and-state-contracts.md)为准。", "",
           f"当前共 **{len(terms)} 个独立词条**，逐项关联[概念映射](07-concept-mapping.md)的 **{len(references)} 条参考条目**及[聚合目录](14-aggregate-catalog.md)的 **{len(objects)} 组对象**。组合对象分别链接各词条，同一来源概念被多模块使用时不按使用方重复定义。词条数不是聚合根数或功能完成率。", "",
           "范围包含当前参考概念、模块对象及理解设计所需的关键术语；不把每个命令名、字段名和状态码都当作业务概念，命令导航见[16](16-command-acceptance-index.md)。标记“参考概念”的条目用于解释对标来源，采纳、替代或不采纳以07的逐项处置为准，不能据词条存在推定目标实现。", "",
           "同名词按语境解释：M03的Job是职务，X04的Job是技术作业；M04的Assignment是任职，M18的LearningAssignment是" + terms["m18-learningassignment"]["zh"] + "；人才档案与工时处理方案分别定义。别名含原文简称、历史称呼及检索名称，不表示所有别名都适合在新正文继续使用。", "",
           "## 分领域索引", "", "| 领域 | 词条数 |", "| --- | --- |"]
    for group in data["groups"]:
        count = sum(e["scope"] == group["id"] for e in terms.values())
        out.append("| [" + group["id"] + " " + group["title"] + "](#domain-" + group["id"].lower() + ") | " + str(count) + " |")
    for group in data["groups"]:
        mid = group["id"].lower()
        out += ["", '<a id="domain-' + mid + '"></a>', "", "## " + group["id"] + " " + group["title"], "",
                "业务依据与详细契约：[" + group["id"] + "模块说明](modules/" + mid + ".md)。", "",
                "| 术语 | 中文名称 | 定义 |", "| --- | --- | --- |"]
        for e in data["entries"]:
            if e["scope"] != group["id"]:
                continue
            label = '<a id="term-' + e["id"] + '"></a>' + safe(e["term"])
            if e["aliases"]:
                label += "<br>别名／原称：" + "、".join(safe(a) for a in e["aliases"])
            definition = ("**参考概念。** " if e["kind"] == "reference" else "") + safe(e["definition"])
            if e.get("source_note"):
                definition += "<br>来源说明：" + safe(e["source_note"])
            extra = [s for s in e["sources"] if not s.startswith("local-")]
            if extra:
                definition += " 依据：" + "、".join("[" + s + "](#source-" + s + ")" for s in extra) + "。"
            if e.get("material_refs"):
                evidence = data["material_review"]["evidence"]
                definition += " 原文定位：" + "；".join(
                    "华润PDF第" + str(evidence[rid]["pdf_page"]) + "页（" +
                    safe(evidence[rid]["standard_code"] or evidence[rid]["code_id"]) + "）"
                    for rid in e["material_refs"]) + "。"
            out.append("| " + label + " | " + safe(e["zh"]) + " | " + definition + " |")
    out += ["", "## 命名依据与版本边界", "",
            "定义首先依据当前模块和目标契约，参考概念通过07保留与来源模型的差异。Oracle中国官网已核对的中文名称与项目根据业务语境选用的中文分别记录；当前英文概念文档不构成官方中文UI译名证明。本表未核验完整的26D简体中文界面术语包，也未把旧PeopleSoft译名作为Fusion当前界面标准。", "",
            "2026-10-30复核官方Readiness及当前书目，相关HR、Payroll、Time/Absence、Talent/Learning发布入口显示26D；概念页来自当前累计文档渠道。项目定义不宣称与Oracle数据库对象或全部行为等价，中国政策适用证据另见[04](04-china-localization.md)。", ""]
    for source in data["sources"]:
        if source["id"].startswith("local-"):
            continue
        out += ['<a id="source-' + source["id"] + '"></a>', "",
                "- [" + source["title"] + "](" + source["url"] + ")：" + source.get("note", "来源模型及概念导航。") +
                (" 核对日期：" + source["checked_on"] + "。" if source.get("checked_on") else ""), ""]
    review = data.get("naming_review")
    if review:
        out += ["## 中文命名建议的取舍", "",
                review["reviewed_on"] + "对[外部命名建议](" + review["source_url"] + ")逐项评估。" + review["scope"], ""]
        out += ["- " + principle for principle in review["principles"]]
        out += ["", "### 分领域标题", "", "| 领域 | 本表标题 | 处理依据 |", "| --- | --- | --- |"]
        groups = {g["id"]: g for g in data["groups"]}
        for row in review["titles"]:
            group = groups[row["scope"]]
            out.append("| [" + group["id"] + "](#domain-" + group["id"].lower() + ") | " +
                       safe(group["title"]) + " | " + safe(row["reason"]) + " |")
        out += ["", "### 词条建议", "", "| 涉及词条（当前首选中文） | 处理 | 理由 |", "| --- | --- | --- |"]
        for row in review["decisions"]:
            out.append("| " + term_links(row["terms"], terms) + " | " + safe(row["decision"]) + " | " + safe(row["reason"]) + " |")
        out.append("")
    material = data.get("material_review")
    if material:
        out += ['<a id="cr2018-review"></a>', "", "## 华润数据标准逐章评估", "",
                material["reviewed_on"] + "对用户提供的《华润集团数据标准（第一批）》逐章核对。原件编制日期为" +
                material["document_date"] + "，共" + str(material["page_count"]) + "个PDF页；正文印刷页码＝PDF页码－2。", "",
                "原件SHA256：`" + material["source_sha256"] + "`。", "",
                "本地原材料目录：`" + material["local_materials_path"] + "`，含原件、逐页原文、表格坐标、章节索引及SQLite/HTML检索。资料目录不参与本仓库生成或校验；本表保存冻结的来源身份与行定位。", ""]
        out += ["- " + note for note in material["notes"]]
        out += ["", "| 领域 | 采纳与调整 | 保留与适用边界 | 原文位置 |", "| --- | --- | --- | --- |"]
        for row in material["chapters"]:
            adoption = safe(row["adoption"])
            if row["terms"]:
                adoption += "<br>" + term_links(row["terms"], terms)
            locators = []
            for rid in row["evidence"]:
                source = material["evidence"][rid]
                locators.append("PDF第" + str(source["pdf_page"]) + "页：" + safe(source["standard_code"] or source["code_id"]) + "（" + rid + "）")
            locators += ["PDF第" + str(page) + "页（无标准项编码）" for page in row.get("pages", [])]
            out.append("| [" + row["scope"] + "](#domain-" + row["scope"].lower() + ") | " + adoption + " | " +
                       safe(row["boundary"]) + " | " + "<br>".join(locators or ["全文核对，未找到可直接采纳的定义"]) + " |")
        out.append("")
    out += ["## 维护与完整性检查", "",
            "人工维护源为[data/terminology-glossary.json](data/terminology-glossary.json)，由[生成脚本](scripts/build_modules.py)生成本表及其他文档中的双语引用。术语ID及显式锚点在改名时保持不变；组合对象可绑定多个词条，不能将组合字符串当成一个概念。新增模块对象或参考映射时必须同步补齐词条及绑定。", "",
            "每个词条记录所属语境、首选中文、简明定义、别名及来源。对象绑定逐项检查本域名称，参考绑定检查条目身份和词条存在；检查覆盖不代表机器已经证明定义正确。应人工复核语义边界及命名依据，尤其是参考概念的替代关系。历史归档及冻结评审记录保持原有措辞。", "",
            "本仓大陆组织角色修订见[CHNHCM-009](../../adr/009-mainland-organization-terms.md)；只更新本轮明确的语义范围，源R25及后续其他命名增量的评审状态仍见[来源说明](../../reference/README.md)。", "",
            "```bash", "python3 -B docs/architecture/target/scripts/build_modules.py --dry-run",
            "python3 -B docs/architecture/target/scripts/build_modules.py",
            "python3 -B docs/architecture/target/scripts/build_modules.py --check",
            "python3 -B scripts/check-repository.py --check-refs", "```", ""]
    return "\n".join(out)


def module_doc(m, terms, objects):
    mid = m["id"].lower()
    out = [
        "# " + m["id"] + " " + m["title"], "",
        "[返回总览](../README.md) · [完整图集](../diagrams/README.md)", "",
        "**China HCM目标设计 [D]。** 按HCM业务概念、唯一事实主责和DDD合同定义；范围与规则由目标架构决定。", "",
        "业务边界：" + m["contract"], "",
        "![" + m["title"] + "架构](../diagrams/" + mid + ".svg)", "",
        "[Mermaid源文件](../diagrams/" + mid + ".mmd)", "",
        "## 对象、身份与职责", "",
        "| 对象 | 身份或粒度 | 聚合归属及写端口 | 责任及关系 |", "| --- | --- | --- | --- |"
    ]
    if len(m["entities"]) != len(m["aggregate_ownership"]):
        raise ValueError("Missing aggregate ownership: " + m["id"])
    out += ["| " + term_links(objects[(m["id"], row[0])], terms, "../", row[0].split(" / ")) + " | " +
            " | ".join(safe(x) for x in [row[1], owner, row[2]]) + " |"
            for row, owner in zip(m["entities"], m["aggregate_ownership"])]
    out += ["", "## 关键设计规则 [D]", ""]
    out += [str(i + 1) + ". " + rule for i, rule in enumerate(m["rules"])]
    out += ["", "## 跨模块合同与中国场景 [D]", "", m["contract"], "", m["china"],
            "", "日期、修订、幂等及授权共同约束见[一致性合同](../03-consistency-contracts.md)；上下文、聚合、事务及状态闭合以[DDD合同](../10-ddd-and-state-contracts.md)为准。政策证据和适用边界见[中国大陆设计](../04-china-localization.md)。",
            "", "## 设计反例与后续验收问题", ""]
    out += ["- " + c for c in m["checks"]]
    out += ["", "以上是目标合同的验收要求；设计图不代替后续实现、并发和外部接口验证。",
            "", "历史原型比较已移入[独立归档](" + HISTORY + "#" + mid + ")，不作为本模块目标合同。", ""]
    return "\n".join(out)

def module_diagram(m):
    own = [n for n in m["nodes"] if n[2] in ("own", "process")]
    outside = [n for n in m["nodes"] if n[2] not in ("own", "process")]
    ids = [n[0] for n in m["nodes"]]
    if len(ids) != len(set(ids)):
        raise ValueError("Duplicate node: " + m["id"])
    out = ["---", "title: " + m["id"] + " " + m["title"] + " · 目标设计", "---", "flowchart TB",
           '  subgraph domain["' + safe(m["id"] + " " + m["title"]) + '"]', "    direction TB"]
    out += ['    ' + n[0] + '["' + safe(n[1]) + '"]:::' + n[2] for n in own]
    out += ["  end"]
    out += ['  ' + n[0] + '["' + safe(n[1]) + '"]:::' + n[2] for n in outside]
    for a, b, label in m["edges"]:
        if a not in ids or b not in ids:
            raise ValueError("Missing endpoint: " + m["id"])
        out.append("  " + a + " -->|" + safe(label) + "| " + b)
    out += [
        "  classDef own fill:#e9f4ff,stroke:#3c6f99,color:#17324d;",
        "  classDef process fill:#fff1d9,stroke:#9e6b20,color:#51390f;",
        "  classDef external fill:#f4f5f7,stroke:#7e8996,color:#354150;",
        "  classDef shared fill:#e7f5ed,stroke:#478565,color:#214d34;", ""
    ]
    return "\n".join(out)

def build_mapping(terms, references):
    rows = json.loads((ROOT / "data/concept-mapping.json").read_text())
    sections = {1: "人员、工作关系与任职", 2: "组织、职位与共享数据", 3: "法定边界与薪资", 4: "排班、实际工时与缺勤", 5: "招聘、人才与学习", 6: "有效日期与历史", 7: "适用、权限与执行许可"}
    out = ["# HCM核心概念逐项映射", "", f"本表将[参考文档](../hcm/02-domain-model.md)第1—7节映射为{len(rows)}条对标条目（部分为组合概念）。条目数不是功能数、完成率或128计划编号。", "",
           "目标处置是China HCM独立架构决定[D]，按HCM概念问题、中国业务边界与DDD原则逐项说明。原型观察只作历史参考；采纳不表示已实现。本仓采用的来源提交及初始摘要见[基线记录](../../reference/baselines.json)，后续变更见[本仓修订链](../../reference/local-revisions.json)；[来源证据记录](06-evidence-and-validation.md)仅作历史路由，不作为当前全部设计的验收结论。", "",
           "各行的主责是逻辑设计，组合主责表示引用/生产消费关系；事实的唯一维护域以[归属表](02-target-architecture.md#2-事实归属与依赖)为准。", ""]
    for section, title in sections.items():
        out += [f"## {section}. {title}", "", "| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |", "| --- | --- | --- | --- | --- | --- |"]
        for row in rows:
            if row["reference_section"] != section:
                continue
            owners = "、".join(f"[{m}](modules/{m.lower()}.md)" for m in row["owners"])
            views = "；".join(f'[{v["diagram"]}](diagrams/{v["diagram"]}.svg)：' + ", ".join(v["nodes"]) for v in row["diagram_refs"])
            concept = row["concept"] + "<br>" + term_links(references[row["id"]], terms)
            out.append("| " + " | ".join([row["id"], concept, row["decision"], owners, row["target"], views]) + " |")
        out.append("")
    out += ["## 关键差异的判断", "", "- Person独立登记服务身份核对与分阶段办理；这是China HCM的选择，不声称与Oracle创建路径相同。",
            "- LDG/PSU/TRU与PayrollRelationship按法规上下文及责任边界吸收；中国法定口径仍靠官方政策和独立案例验证。",
            "- 共享引用问题由对象适用策略及明确采用承接，不复刻SetID/TableSet。日级人事不采用业务EFFSEQ；真实动作与数据更正分别留证，审计修订不能等价替代日内任职状态，当前不支持日内状态查询/计量。",
            "- 招聘、人才、学习、福利和员工服务均纳入目标设计，设计完整性不受原型阶段限制。",
            "- 图节点定位检查证明引用存在；关键关系、语义替代和主责正确性仍需独立人工评审。",
            "- 技能目录、执行模式和页面占位不等于业务模块实现或正式接口可用。", ""]
    stream = io.StringIO()
    writer = csv.writer(stream, lineterminator="\n")
    writer.writerow(["id", "reference_section", "concept", "owners", "decision", "target", "diagram_nodes"])
    for row in rows:
        views = ";".join(v["diagram"] + ":" + ",".join(v["nodes"]) for v in row["diagram_refs"])
        writer.writerow([row["id"], row["reference_section"], row["concept"], ",".join(row["owners"]), row["decision"], row["target"], views])
    return {"07-concept-mapping.md": "\n".join(out), "data/concept-mapping.csv": stream.getvalue()}

def build_command_index(modules, terms, objects):
    """Derive object ownership from module data; index data contains navigation only."""
    rows = json.loads((ROOT / "data/command-contract-index.json").read_text())["entries"]
    owners = {(m["id"], e[0]): a for m in modules
              for e, a in zip(m["entities"], m["aggregate_ownership"])}
    out = ["# 命令、事务与验收索引", "",
           "本页由模块对象归属与[data/command-contract-index.json](data/command-contract-index.json)生成，仅提供导航。业务规则以[DDD合同](10-ddd-and-state-contracts.md)及模块说明为准；不复制另一套金额、状态或事务算法。", "",
           "接口名称表达目标命令/查询族，不表示已有实现；QueryHistory、ConfirmTime、AssessImpact、ResolveImpact、ApplyRecovery是跨具体端口的导航名称。所有AC场景均为后续产品验收要求；来源设计阶段的文档/图纸检查不等于本仓验收。本仓已执行范围见[验证记录入口](../../verification/README.md)，来源增量边界见[来源说明](../../reference/README.md)。", ""]
    for row in rows:
        out += ["## " + row["id"] + " " + row["title"], "",
                "命令/查询：" + "、".join(row["commands"]) + "。", "",
                "| 对象及所属域 | 同源聚合/写端口 |", "| --- | --- |"]
        for ref in row["objects"]:
            key = (ref["module"], ref["entity"])
            if key not in owners:
                raise ValueError("Unknown command-index object: " + str(key))
            out.append("| [" + ref["module"] + "](modules/" + ref["module"].lower() + ".md) " +
                       term_links(objects[key], terms, labels=ref["entity"].split(" / ")) + " | " + safe(owners[key]) + " |")
        out += ["", "提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#" + row["contract_anchor"] + ")；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#" + row["recovery_anchor"] + ")；验收：" +
                "、".join("[" + a + "](10-ddd-and-state-contracts.md#" + a.lower() + ")" for a in row["acceptance"]) + "。", ""]
    return "\n".join(out)


def build():
    modules = []
    for name in ("business-modules.json", "crosscutting-modules.json"):
        modules += json.loads((ROOT / "data" / name).read_text())
    assert [m["id"] for m in modules] == [f"M{i:02}" for i in range(1, 21)] + [f"X{i:02}" for i in range(1, 7)]
    glossary, terms, objects, references = load_glossary(modules, json.loads((ROOT / "data/concept-mapping.json").read_text()))
    outputs = {}
    for m in modules:
        mid = m["id"].lower()
        outputs["modules/" + mid + ".md"] = module_doc(m, terms, objects)
        outputs["diagrams/" + mid + ".mmd"] = module_diagram(m)
    outputs.update(build_mapping(terms, references))
    catalog = ["# 目标对象、聚合归属与写端口目录", "",
               "本目录与各模块对象表从同一份设计数据生成。根、根内实体、不可变事实、值类型、投影及技术记录分别标明；概念连线不推导同一聚合。", "",
               "[DDD合同](10-ddd-and-state-contracts.md)规定事务和公开端口。业务DecisionRecord归目标根或本域决定根，X04不持有业务决定。", ""]
    for m in modules:
        catalog += ["## " + m["id"] + " " + m["title"], "", "| 目标对象 | 聚合归属及唯一写端口 |", "| --- | --- |"]
        catalog += ["| " + term_links(objects[(m["id"], e[0])], terms, labels=e[0].split(" / ")) + " | " + safe(a) + " |"
                    for e,a in zip(m["entities"],m["aggregate_ownership"])]
        catalog.append("")
    outputs["14-aggregate-catalog.md"] = "\n".join(catalog)
    outputs["16-command-acceptance-index.md"] = build_command_index(modules, terms, objects)
    outputs["17-terminology-glossary.md"] = build_glossary(glossary, terms, objects, references)
    return {name: route_local_links(name, content) for name, content in outputs.items()}


def route_local_links(name, content):
    """Preserve adopted historical link routing without a sibling checkout."""
    origin = posixpath.join("docs/china-hcm", name)
    page = ROOT / name
    index = ROOT.parents[1] / "reference/not-imported.md"
    known_anchors = set(re.findall(r'id="(source-[a-f0-9]+)"', index.read_text()))

    def rewrite(match):
        value = match.group(1)
        if value.startswith(("#", "http://", "https://", "mailto:", "data:")):
            return match.group(0)
        address, sep, anchor = value.partition("#")
        if (page.parent / address).exists():
            return match.group(0)
        original = posixpath.normpath(posixpath.join(posixpath.dirname(origin), address)) + sep + anchor
        key = "source-" + hashlib.sha256(original.encode()).hexdigest()[:12]
        if key not in known_anchors:
            raise ValueError("unresolved local link: " + name + " -> " + value)
        relative = posixpath.relpath(index.as_posix(), page.parent.as_posix())
        return "](" + relative + "#" + key + ")"

    return re.sub(r'\]\(([^)\s]+)\)', rewrite, content)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true", help="Validate authored data and list outputs without writing")
    parser.add_argument("--check", action="store_true", help="Fail if generated files differ; never write")
    args = parser.parse_args()
    outputs = build()
    if args.check:
        changed = [name for name, content in outputs.items()
                   if not (ROOT / name).is_file() or (ROOT / name).read_text() != content.rstrip() + "\n"]
        print(json.dumps({"files": len(outputs), "writes": False, "different": changed}, ensure_ascii=False))
        if changed:
            raise SystemExit(1)
        return
    if args.dry_run:
        print(json.dumps({"files": len(outputs), "modules": 26, "writes": False}, ensure_ascii=False))
        return
    for name, content in outputs.items():
        path = ROOT / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content.rstrip() + "\n")
    print(json.dumps({"files": len(outputs), "modules": 26, "writes": True}, ensure_ascii=False))

if __name__ == "__main__":
    main()
