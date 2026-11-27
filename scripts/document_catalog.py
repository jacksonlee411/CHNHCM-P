"""Build the task-oriented document catalog from the sole governance registry."""
import argparse
from collections import defaultdict
import json
import os
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit

from document_governance import (EVIDENCE_ACCESS, HEADINGS, REPORT_PURPOSES, link_definitions,
                                 link_occurrences, markdown_anchors, markdown_body, markdown_prose)

ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "docs/governance/registry.json"
CATALOG = ROOT / "docs/governance/catalog/README.md"
GROUPS = (
    ("standards", "现行标准"),
    ("in-progress", "在办计划"),
    ("not-started", "待启动计划"),
    ("review", "待审文稿"),
    ("closed", "已关闭计划"),
    ("history", "已替代、废止或取消的其他文档"),
    ("reserved", "已预留的编号"),
)


def categories(doc):
    status = doc["status"]
    groups = []
    if doc["type"] == "plan":
        delivery = {"in_progress": "in-progress", "not_started": "not-started",
                    "completed": "closed", "cancelled": "closed"}
        groups.append(delivery[doc["execution_status"]])
    elif status == "active" and not doc["archive"]:
        groups.append("standards")
    elif status in {"superseded", "retired", "cancelled"}:
        groups.append("history")
    if status in {"draft", "in_review"}:
        groups.append("review")
    if status == "planned":
        groups.append("reserved")
    return groups


def link(label, path, directory=None):
    relative = Path(os.path.relpath(ROOT / path, directory or CATALOG.parent)).as_posix()
    return f"[{label}]({relative})"


def section_numbers(path):
    text = path.read_text(encoding="utf-8")
    sections = {fragment: " " + fragment.upper() if re.fullmatch(r"[mx]\d\d-(br|ac|scn|tl)-\d{3}", fragment) else ""
                for fragment in markdown_anchors(text)}
    for level, title in HEADINGS.findall(markdown_body(text)):
        # Plain numbered headings have a GitHub fragment. Use an explicit anchor for richer Markdown.
        number = re.match(r"(\d+(?:\.\d+)*) ", title)
        if len(level) > 1 and number and re.fullmatch(r"[\w .、，：/（）()\-]+", title):
            fragment = re.sub(r"[^\w -]", "", title.lower()).replace(" ", "-")
            sections[fragment] = " §" + number[1]
    return sections


def clause_navigation(documents):
    current = {doc["id"]: doc for doc in documents if "standards" in categories(doc)}
    sections = {}
    local, outgoing = defaultdict(list), defaultdict(list)

    def section(doc, fragment):
        path = doc["path"]
        if path not in sections:
            sections[path] = section_numbers(ROOT / path)
        if fragment not in sections[path]:
            raise ValueError(f"{doc['id']} 找不到条款锚点 {fragment!r}（{path}）")
        return sections[path][fragment]

    for doc in documents:
        identifier = doc["id"]
        partial = identifier in current and bool(doc["superseded_by"])
        review = doc["status"] in {"draft", "in_review"}
        routes = doc.get("clause_routes", {})
        if not isinstance(routes, dict) or (routes and not (partial or review)):
            raise ValueError(f"{identifier} 条款导航须登记在部分替代的现行规范或待审文稿上")
        if partial and set(routes) != {"retained", *doc["superseded_by"]}:
            raise ValueError(f"{identifier} 须逐项登记本篇剩余条款及每个后继决定的现行去向")
        for origin, topics in routes.items():
            source_section = (section(doc, origin).strip() or "#" + origin) if review else ""
            if not isinstance(topics, dict) or not topics:
                raise ValueError(f"{identifier} 的 {origin} 缺少条款")
            for topic, reference in topics.items():
                if not isinstance(topic, str) or not topic.strip() or re.search(r"[\r\n|\[\]]", topic) or not isinstance(reference, str):
                    raise ValueError(f"{identifier} 的 {origin} 须提供单行问题名称和条款引用")
                owner, _, fragment = reference.partition("#")
                if not owner and partial:
                    owner = identifier if origin == "retained" else origin
                if owner not in current or not fragment or (origin == "retained" and owner != identifier):
                    raise ValueError(f"{identifier} 的 {topic} 须指向相应现行规范的具体条款：{reference}")
                target = current[owner]
                destination = target["path"] + "#" + fragment
                decision = f"（据 {origin[-3:]}）" if partial and origin not in ("retained", owner) else ""
                label = f"{topic}{decision} → {owner[-3:]}{section(target, fragment)}"
                if review or origin == "retained":
                    local[owner].append((label, destination))
                if origin != "retained":
                    prefix = f"{source_section} " if source_section else ""
                    outgoing[identifier].append((prefix + label, destination))
    return local, outgoing


def clause_links(entries, directory=None):
    return " · ".join(link(label, path, directory) for label, path in dict.fromkeys(entries)) or "—"


def render(registry, outgoing=None):
    if outgoing is None:
        _, outgoing = clause_navigation(registry["documents"])
    lines = ["# 项目文档分类检索", "",
             "本页由[唯一登记表](../registry.json)生成。编号文档按采纳与执行两个维度归类，同一项可以出现在两处。核对执行关闭原因、审阅角色或历史基线时，可按编号读取登记和报告。", ""]
    lines.append(" · ".join(f"[{title}](#catalog-{key})" for key, title in (*GROUPS, ("evidence", "验收记录"))))
    groups = {key: [] for key, _ in GROUPS}
    for doc in sorted(registry["documents"], key=lambda item: item["id"]):
        for key in categories(doc):
            groups[key].append(doc)
    for key, title in GROUPS:
        lines.extend(["", f'<a id="catalog-{key}"></a>', "", f"## {title}", ""])
        if key == "standards":
            lines.append("[按现行规范查标题与条款](standards/README.md)")
            continue
        if not groups[key]:
            lines.append("当前登记为零。")
            continue
        header = "| 编号与正文 | 文档状态 | 执行状态 | 保存位置 |"
        separator = "| --- | --- | --- | --- |"
        if key == "review":
            lines.extend(["条款名称前标出送审稿中的原位置，链接直达承接该项规则的现行合同。空栏表示尚未登记单独采纳的条款。", ""])
            header += " 已单独采用的现行条款 |"
            separator += " --- |"
        lines.extend([header, separator])
        for doc in groups[key]:
            label = f"{doc['id'][-3:]} {doc['title']}"
            label = link(label, doc["path"]) if doc["path"] and (ROOT / doc["path"]).is_file() else label
            location = "物理归档" if doc["archive"] else ("已预留" if doc["status"] == "planned" else "原位")
            if doc.get("archive_deferral"):
                location = "暂缓归档"
            row = f"| {label} | {doc['status']} | {doc['execution_status'] or '—'} | {location} |"
            if key == "review":
                row += f" {clause_links(outgoing.get(doc['id'], []))} |"
            lines.append(row)
    lines.extend(["", '<a id="catalog-evidence"></a>', "", "## 验收记录", "",
                  "按用途进入，或在页内查关联编号、对象及结论。每行只说明该批受测范围。定稿表示记录已写定，首败、复验及未测项可在原记录中查回。", "",
                  "证据位置标明本仓能读到什么。仅有摘要时，公开结论取自登记，逐项核对原报告和附件需要原仓权限。标有公开 CI 的记录可从报告打开运行结果。", ""])
    reports = sorted(registry["governance"]["verification_records"],
                     key=lambda item: (item["date"], item["path"]), reverse=True)
    by_purpose = {key: [] for key in REPORT_PURPOSES}
    for report in reports:
        by_purpose[report["purpose"]].append(report)
    lines.append(" · ".join(f"[{name}（{len(by_purpose[key])}）](#evidence-{key})"
                            for key, name in REPORT_PURPOSES.items() if by_purpose[key]))
    for key, records in by_purpose.items():
        if not records:
            continue
        lines.extend(["", f'<a id="evidence-{key}"></a>', "", f"### {REPORT_PURPOSES[key]}", "",
                      "| 关联编号 | 记录 | 日期与稿态 | 本次结论及限制 | 证据位置 |", "| --- | --- | --- | --- | --- |"])
        for report in records:
            path = report["path"]
            title = (ROOT / path).read_text(encoding="utf-8").splitlines()[0].removeprefix("# ").replace("|", "｜")
            ids = "、".join(n.removeprefix("CHNHCM-") for n in report["related_documents"])
            status = "定稿" if report["status"] == "final" else "草稿"
            conclusion = report["conclusion"].replace("|", "｜")
            if report.get("follow_up"):
                conclusion += " " + link("后续记录", report["follow_up"])
            lines.append(f"| {ids} | {link(title, path)} | {report['date']} {status} | "
                         f"{conclusion} | {EVIDENCE_ACCESS[report['evidence_access']]} |")
    return "\n".join(lines) + "\n"


def render_standards(standards, directory, local, outgoing):
    # Reuse the clause labels maintained by the task and module navigation pages.
    clauses = {ROOT / doc["path"]: {} for doc in standards}
    for source in (ROOT / "docs/governance/README.md", ROOT / "docs/specs/README.md"):
        content = source.read_text(encoding="utf-8")
        for _, _, label, target in link_occurrences(markdown_prose(content), link_definitions(content)):
            url = urlsplit(target)
            if url.fragment and not url.scheme and not url.netloc:
                path = (source.parent / unquote(url.path)).resolve()
                if path in clauses:
                    clauses[path].setdefault(url.fragment, label)

    lines = ["# 现行规范", "",
             "按开发问题找条款。第二列列出本篇现行条款，第三列直达已改由其他条款规定的事项。"
             "检索规范正文可在仓库根目录运行 `rg -n -F -- '关键词' docs`。", "",
             "| 编号与正文 | 本规范的现行条款 | 部分条款的替代去向 |", "| --- | --- | --- |"]
    for doc in standards:
        path = doc["path"]
        title = link(f"{doc['id'][-3:]} {doc['title']}", path, directory)
        entries = local.get(doc["id"], [])
        indexed = {target for _, target in entries}
        entries = entries + [(label, f"{path}#{fragment}") for fragment, label in clauses[ROOT / path].items()
                             if f"{path}#{fragment}" not in indexed]
        successors = outgoing.get(doc["id"], [])
        lines.append(f"| {title} | {clause_links(entries, directory)} | {clause_links(successors, directory)} |")
    return "\n".join(lines) + "\n"


def render_search_filter(standards):
    paths = [Path(doc["path"]).relative_to("docs") for doc in standards]
    parents = {parent for path in paths for parent in path.parents if parent != Path(".")}
    lines = ["# 由 scripts/document_catalog.py 按登记的现行规范生成。", "*"]
    lines.extend(f"!/{parent.as_posix()}/" for parent in sorted(parents))
    lines.extend(f"!/{path.as_posix()}" for path in sorted(paths))
    return "\n".join(lines) + "\n"


def catalog_files(registry):
    standards = sorted((doc for doc in registry["documents"] if "standards" in categories(doc)),
                       key=lambda doc: doc["id"])
    directory = CATALOG.parent / "standards"
    local, outgoing = clause_navigation(registry["documents"])
    return {CATALOG: render(registry, outgoing),
            directory / "README.md": render_standards(standards, directory, local, outgoing),
            ROOT / "docs/.ignore": render_search_filter(standards)}


def catalog_matches(registry):
    return all(path.is_file() and path.read_text(encoding="utf-8") == content
               for path, content in catalog_files(registry).items())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    registry = json.loads(REGISTRY.read_text(encoding="utf-8"))
    try:
        if args.check:
            if not catalog_matches(registry):
                parser.exit(1, "文档目录或默认检索范围已过期，请运行 python3 -B scripts/document_catalog.py。\n")
            return
        files = catalog_files(registry)
    except ValueError as error:
        parser.exit(1, f"{error}\n")
    for path, content in files.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")


if __name__ == "__main__":
    main()
