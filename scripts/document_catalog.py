"""Build the task-oriented document catalog from the sole governance registry."""
import argparse
import json
import os
from pathlib import Path
from urllib.parse import unquote, urlsplit

from document_governance import link_definitions, link_occurrences, markdown_prose

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


def render(registry):
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
        lines.extend(["| 编号与正文 | 文档状态 | 执行状态 | 保存位置 |", "| --- | --- | --- | --- |"])
        for doc in groups[key]:
            label = f"{doc['id'][-3:]} {doc['title']}"
            label = link(label, doc["path"]) if doc["path"] and (ROOT / doc["path"]).is_file() else label
            location = "物理归档" if doc["archive"] else ("已预留" if doc["status"] == "planned" else "原位")
            if doc.get("archive_deferral"):
                location = "暂缓归档"
            lines.append(f"| {label} | {doc['status']} | {doc['execution_status'] or '—'} | {location} |")
    lines.extend(["", '<a id="catalog-evidence"></a>', "", "## 验收记录", "",
                  "按关联编号查到方案评审、实施复验及其他记录。打开正文核对受测基线、实际结果和限制。", "",
                  "| 关联编号 | 记录 | 日期 | 状态 |", "| --- | --- | --- | --- |"])
    for report in sorted(registry["governance"]["verification_records"], key=lambda item: (item["date"], item["path"]), reverse=True):
        path = report["path"]
        title = (ROOT / path).read_text(encoding="utf-8").splitlines()[0].removeprefix("# ")
        ids = "、".join(n.removeprefix("CHNHCM-") for n in report["related_documents"])
        lines.append(f"| {ids} | {link(title.replace('|', '｜'), path)} | {report['date']} | {report['status']} |")
    return "\n".join(lines) + "\n"


def render_standards(standards, directory):
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

    by_id = {doc["id"]: doc for doc in standards}
    lines = ["# 现行规范", "",
             "按编号、标题或导航中的条款名称查找。检索整份正文时，在仓库根目录运行 `rg -n -F -- '关键词' docs`。", "",
             "| 编号与正文 | 条款入口 | 部分条款的替代去向 |", "| --- | --- | --- |"]
    for doc in standards:
        path = doc["path"]
        title = link(f"{doc['id'][-3:]} {doc['title']}", path, directory)
        entries = [link(label, f"{path}#{fragment}", directory)
                   for fragment, label in clauses[ROOT / path].items()]
        successors = [link(f"{identifier[-3:]} {by_id[identifier]['title']}", by_id[identifier]["path"], directory)
                      for identifier in doc["superseded_by"] if identifier in by_id]
        if len(successors) < len(doc["superseded_by"]):
            successors.append("其余替代范围见正文" if successors else "见正文的替代说明")
        lines.append(f"| {title} | {' · '.join(entries) or '—'} | {' · '.join(successors) or '—'} |")
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
    return {CATALOG: render(registry),
            directory / "README.md": render_standards(standards, directory),
            ROOT / "docs/.ignore": render_search_filter(standards)}


def catalog_matches(registry):
    return all(path.is_file() and path.read_text(encoding="utf-8") == content
               for path, content in catalog_files(registry).items())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    registry = json.loads(REGISTRY.read_text(encoding="utf-8"))
    if args.check:
        if not catalog_matches(registry):
            parser.exit(1, "文档目录或默认检索范围已过期，请运行 python3 -B scripts/document_catalog.py。\n")
        return
    for path, content in catalog_files(registry).items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")


if __name__ == "__main__":
    main()
