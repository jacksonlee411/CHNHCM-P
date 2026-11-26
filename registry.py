#!/usr/bin/env python3
"""Allocate goose migration numbers on the shared fast-forward-only branch."""
import argparse
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile

BRANCH = "migration-registry"
OWNERS = tuple("wt-dev-" + suffix for suffix in "abcdef")
CLAIM_FIELDS = ("number", "branch", "name", "purpose", "depends_on")
NAME = re.compile(r"[a-z][a-z0-9]*(?:_[a-z0-9]+)*")
NUMBER = re.compile(r"[0-9]{5}")
SHA = re.compile(r"[0-9a-f]{40}")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def git(root, *args):
    return subprocess.check_output(["git", "-C", str(root), *args], text=True).strip()


def main_files(root, main):
    files = {}
    for row in git(root, "ls-tree", main + ":db/migrations").splitlines():
        info, name = row.split("\t", 1)
        if not re.match(r"^\d.*\.sql$", name):
            continue
        match = re.fullmatch(r"([0-9]{5})_(.+)\.sql", name)
        require(match and NAME.fullmatch(match[2]), f"主线迁移文件名无效 {name}")
        mode, kind, blob = info.split()
        require(mode == "100644" and kind == "blob", f"主线迁移应为普通 SQL 文件 {name}")
        number, slug = match.groups()
        require(number not in files, f"主线迁移号重复 {number}")
        files[number] = (slug, blob)
    return files


def read_ledger(root):
    base = json.loads((root / "baseline.json").read_text())
    require(SHA.fullmatch(base["main"]), "基线缺少主线 SHA")
    entries = {}
    names = set()
    for row in base["migrations"]:
        number, name = row["number"], row["name"]
        require(number == f"{len(entries) + 1:05}" and name not in names
                and NAME.fullmatch(name) and SHA.fullmatch(row["blob"]), "基线编号或文件无效")
        entries[number] = dict(row, status="merged")
        names.add(name)
    for line in (root / "events.jsonl").read_text().splitlines():
        row = json.loads(line)
        number, name, deps = row["number"], row["name"], row["depends_on"]
        require(NUMBER.fullmatch(number) and NAME.fullmatch(name) and row["purpose"].strip()
                and row["branch"] in OWNERS and SHA.fullmatch(row["main"]), "登记字段无效")
        require(isinstance(deps, list) and deps == sorted(set(deps))
                and all(dep < number and dep in entries and entries[dep]["status"] != "cancelled"
                        for dep in deps), f"{number} 的依赖无效")
        old = entries.get(number)
        if old is None:
            require(number == f"{len(entries) + 1:05}" and name not in names
                    and row["status"] == "reserved", f"重号、跳号或重复用途标识 {number}/{name}")
            names.add(name)
        else:
            require(old["status"] == "reserved" and row["status"] in ("merged", "cancelled")
                    and all(old[k] == row[k] for k in CLAIM_FIELDS), f"禁止修改既有登记 {number}")
            if row["status"] == "cancelled":
                require(not dependents(entries, number), f"{number} 仍有在用的下游登记")
            else:
                require(SHA.fullmatch(row["blob"]), f"{number} 缺少主线 SQL 对象")
        entries[number] = row
    return entries


def dependents(entries, number):
    return [n for n, row in entries.items()
            if row["status"] != "cancelled" and number in row.get("depends_on", [])]


def compare_main(files, entries):
    for number, (name, _) in files.items():
        row = entries.get(number)
        require(row and row["name"] == name and row["status"] != "cancelled",
                f"主线 {number}_{name} 与登记不一致，请先核对")
        require(all(d in files for d in row.get("depends_on", [])), f"主线 {number} 缺少前置迁移")
    for n, row in entries.items():
        if row["status"] == "merged":
            require(files.get(n) == (row["name"], row["blob"]), f"已合入 SQL 缺失或内容改变 {n}")


def next_event(args, entries, files, main):
    if args.command == "reserve":
        require(NAME.fullmatch(args.name) and args.purpose.strip(), "用途不能为空，名称须为小写 SQL 文件后缀")
        deps = sorted(set(args.depends_on))
        require(all(dep in entries and entries[dep]["status"] != "cancelled" for dep in deps),
                "依赖编号须已登记且仍有效")
        fields = dict(branch=args.branch, name=args.name, purpose=args.purpose.strip(), depends_on=deps)
        existing = next((r for r in entries.values() if r["name"] == args.name), None)
        if existing:
            require(existing["status"] != "cancelled" and all(existing.get(k) == v for k, v in fields.items()),
                    "该迁移名称已被另一条登记使用")
            return existing, False
        number = len(entries) + 1
        require(number <= 99999, "五位迁移编号已用尽")
        return dict(number=f"{number:05}", **fields, status="reserved", main=main), True
    old = entries.get(args.number)
    require(old and old.get("branch") == args.branch, "编号不存在或负责分支不符")
    if old["status"] == args.status:
        return old, False
    require(old["status"] == "reserved", "仅可更新已预留的登记")
    row = dict(old, status=args.status, main=main)
    if args.status == "merged":
        require(args.number in files, "对应 SQL 尚未合入远端 main")
        row["blob"] = files[args.number][1]
    else:
        require(args.number not in files and not dependents(entries, args.number),
                "已合入或有下游依赖的登记不可取消")
    return row, True


def transact(args):
    repo = Path(__file__).resolve().parent
    remote = git(repo, "remote", "get-url", "--push", "origin")
    with tempfile.TemporaryDirectory(prefix="migration-registry-") as tmp:
        work = Path(tmp) / "checkout"
        product = Path(tmp) / "product.git"
        subprocess.run(["git", "init", "--bare", "--quiet", str(product)], check=True)
        git(product, "remote", "add", "origin", remote)
        git(repo, "worktree", "add", "--quiet", "--detach", str(work), "HEAD")
        try:
            for _ in range(6):
                git(work, "fetch", "--quiet", "--no-tags", remote, "refs/heads/" + BRANCH)
                base = git(work, "rev-parse", "FETCH_HEAD")
                git(work, "checkout", "--quiet", "--detach", base)
                require((work / "registry.py").read_bytes() == Path(__file__).read_bytes(),
                        "领号工具已更新，请先在登记分支克隆执行 git pull --ff-only")
                git(product, "fetch", "--quiet", "--no-tags", "--depth=1", "--filter=blob:none", "origin", "refs/heads/main")
                main = git(product, "rev-parse", "FETCH_HEAD")
                entries = read_ledger(work)
                files = main_files(product, main)
                compare_main(files, entries)
                if args.command == "check":
                    return {"registry": base, "main": main, "used_through": max(entries),
                            "reserved": [r["number"] for r in entries.values() if r["status"] == "reserved"]}
                row, changed = next_event(args, entries, files, main)
                if not changed:
                    return row
                with (work / "events.jsonl").open("a", encoding="utf-8") as out:
                    out.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")
                read_ledger(work)
                git(work, "add", "events.jsonl")
                git(work, "commit", "--quiet", "-m", f'{row["number"]} {row["branch"]} {row["status"]} {row["name"]}')
                push = subprocess.run(["git", "-C", str(work), "push", "--quiet", remote,
                                       "HEAD:refs/heads/" + BRANCH])
                if push.returncode == 0:
                    return row
                # Also resolves a push accepted by the server whose reply was lost.
                git(work, "fetch", "--quiet", "--no-tags", remote, "refs/heads/" + BRANCH)
                require(git(work, "rev-parse", "FETCH_HEAD") != base,
                        "远端未接受提交，请检查权限或连接，再原样重试")
            raise ValueError("多次遇到并发更新，请原样重试")
        finally:
            git(repo, "worktree", "remove", "--force", str(work))


def cli():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("check")
    reserve = commands.add_parser("reserve")
    reserve.add_argument("--branch", choices=OWNERS, required=True)
    reserve.add_argument("--name", required=True)
    reserve.add_argument("--purpose", required=True)
    reserve.add_argument("--depends-on", nargs="*", default=[])
    mark = commands.add_parser("mark")
    mark.add_argument("--branch", choices=OWNERS, required=True)
    mark.add_argument("--number", required=True)
    mark.add_argument("--status", choices=("merged", "cancelled"), required=True)
    try:
        print(json.dumps(transact(parser.parse_args()), ensure_ascii=False))
    except (ValueError, KeyError, TypeError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    cli()
