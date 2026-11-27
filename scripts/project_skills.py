"""Validate the Git-distributed Codex skill files and project automation."""
import os
from pathlib import Path
import re
import subprocess


SKILLS = Path('.agents/skills')
SKILL_NAME = re.compile(r'chnhcm-p-[a-z0-9]+(?:-[a-z0-9]+)*\Z')
PUBLIC_SKILLS = re.compile(r'(?:\.codex|\.agents|/etc/codex)/skills\b')
INSTALL = re.compile(
    r'(?:^|[;&|(])\s*@?(?:cp|ln|install|mkdir|rsync|tee)\s+'
    r'|\.(?:symlink|symlink_to|copytree|copyfile|copy2|write_text|write_bytes|mkdir)\s*\('
    r'|\s>{1,2}\s*["\x27$~/]', re.MULTILINE)


def project_skill_errors(root):
    listing = subprocess.check_output(
        ['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=root)
    paths = {Path(os.fsdecode(name)) for name in listing.split(b'\0') if name}
    errors = []
    skill_dirs = set()
    entries = set()
    for path in sorted(paths):
        file = root / path
        if not os.path.lexists(file):
            continue
        if path in (SKILLS.parent, SKILLS) or SKILLS in path.parents:
            try:
                target = file.resolve(strict=True).relative_to(root)
            except (OSError, RuntimeError, ValueError):
                errors.append(f'技能路径越界或目标不可用: {path}')
                continue
            if target not in paths or not file.is_file() or (file.is_symlink() and file.readlink().is_absolute()):
                errors.append(f'技能须为本仓发布的文件或相对文件链接: {path}')
                continue
            if len(path.parts) < 4 or not SKILL_NAME.fullmatch(path.parts[2]):
                errors.append(f'技能应位于 .agents/skills/chnhcm-p-<名称>/: {path}')
                continue
            skill_dirs.add(SKILLS / path.parts[2])
            if path.name == 'SKILL.md':
                if len(path.parts) != 4:
                    errors.append(f'技能入口目录不能嵌套: {path}')
                    continue
                entries.add(path)
        elif path.name == 'SKILL.md':
            errors.append(f'本项目技能只能从 .agents/skills 发布: {path}')

        automation = path.name in ('Makefile', 'package.json') or path.suffix in (
            '.sh', '.bash', '.py', '.js', '.mjs', '.ts', '.yaml', '.yml', '.ps1')
        if not automation or file.is_symlink() or not file.is_file():
            continue
        source = file.read_text(encoding='utf-8')
        if 'skills' not in source or not INSTALL.search(source):
            continue
        # Match personal/admin roots, including installers which first assign
        # CODEX_HOME to a local variable and append /skills in a later command.
        shared = ('CODEX_HOME' in source and '/skills' in source) or any(
            '/etc/codex/skills' in line or
            (PUBLIC_SKILLS.search(line) and re.search(r'HOME|~/|/home/|/Users/|USERPROFILE', line))
            for line in source.splitlines())
        if shared:
            errors.append(f'项目自动化引用了公共技能目录及写入操作: {path}')

    for directory in sorted(skill_dirs):
        entry = directory / 'SKILL.md'
        if entry not in entries:
            errors.append(f'缺少技能入口: {entry}')
            continue
        lines = (root / entry).read_text().splitlines()
        header = lines[1:lines.index('---', 1)] if lines[:1] == ['---'] and '---' in lines[1:] else []
        if (f'name: {directory.name}' not in header or
                not any(line.startswith('description: ') and line[13:].strip() for line in header)):
            errors.append(f'技能名称或说明与入口不符: {entry}')
    return errors
