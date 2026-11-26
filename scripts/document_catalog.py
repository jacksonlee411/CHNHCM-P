"""Build the task-oriented document catalog from the sole governance registry."""
import argparse
import json
import os
from pathlib import Path

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


def link(label, path):
    relative = Path(os.path.relpath(ROOT / path, CATALOG.parent)).as_posix()
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
        if not groups[key]:
            lines.append("当前登记为零。")
            continue
        lines.extend(["| 编号与正文 | 文档状态 | 执行状态 | 保存位置 |", "| --- | --- | --- | --- |"])
        for doc in groups[key]:
            label = f"{doc['id'][-3:]} {doc['title']}"
            label = link(label, doc["path"]) if doc["path"] and (ROOT / doc["path"]).is_file() else label
            location = "物理归档" if doc["archive"] else ("已预留" if doc["status"] == "planned" else "原位")
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


def catalog_matches(registry):
    return CATALOG.is_file() and CATALOG.read_text(encoding="utf-8") == render(registry)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    registry = json.loads(REGISTRY.read_text(encoding="utf-8"))
    if args.check:
        if not catalog_matches(registry):
            parser.exit(1, "文档分类目录与登记不一致，请运行 python3 -B scripts/document_catalog.py。\n")
        return
    CATALOG.parent.mkdir(parents=True, exist_ok=True)
    CATALOG.write_text(render(registry), encoding="utf-8")


if __name__ == "__main__":
    main()
