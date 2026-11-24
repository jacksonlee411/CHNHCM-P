"""Run the candidate's generators and the applicable local regression checks."""
from contextlib import contextmanager
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import stat
import subprocess
import tempfile
from time import monotonic

from ci_browser import PREFLIGHT_CASES, arguments, check_inventory, discover, title_args
from ci_policy import ROOT, git, run


def run_check(command, *, cwd, env=None):
    # Let the smoke driver finish reclaiming its profile even if the caller cancels.
    with subprocess.Popen(command, cwd=cwd, env=env, start_new_session=True) as child:
        try:
            code = child.wait()
        except KeyboardInterrupt:
            if child.poll() is None:
                try:
                    os.killpg(child.pid, signal.SIGTERM)
                except ProcessLookupError:
                    pass
            child.wait()
            raise
        if code:
            raise subprocess.CalledProcessError(code, command)


def candidate(base_ref, root):
    """Build the would-be commit tree without staging files or retaining Git objects."""
    flags = run('git', 'ls-files', '-v', '-z', cwd=root).split(b'\0')
    if any(item and (item[:1].islower() or item[:1] == b'S') for item in flags):
        raise ValueError('索引隐藏了工作树文件，无法确认受测内容')
    if run('git', 'ls-files', '--unmerged', '-z', cwd=root):
        raise ValueError('请先解决候选中的合并冲突')
    status = run('git', '--no-optional-locks', 'status', '--porcelain=v1', '-z', '--no-renames', '--untracked-files=all', cwd=root)
    head = git('rev-parse', 'HEAD', root=root)
    target = git('rev-parse', '--verify', '--end-of-options', base_ref + '^{commit}', root=root)
    tree = git('rev-parse', head + '^{tree}', root=root)
    if status:
        with tempfile.TemporaryDirectory(prefix='chnhcm-ci-index-') as folder:
            index = Path(folder) / 'index'
            objects = Path(folder) / 'objects'
            objects.mkdir()
            source = git('rev-parse', '--path-format=absolute', '--git-path', 'index', root=root)
            shutil.copyfile(source, index)
            alternates = git('rev-parse', '--path-format=absolute', '--git-path', 'objects', root=root)
            if os.environ.get('GIT_ALTERNATE_OBJECT_DIRECTORIES'):
                alternates += os.pathsep + os.environ['GIT_ALTERNATE_OBJECT_DIRECTORIES']
            env = dict(os.environ, GIT_INDEX_FILE=str(index), GIT_OBJECT_DIRECTORY=str(objects),
                       GIT_ALTERNATE_OBJECT_DIRECTORIES=alternates)
            subprocess.run(['git', 'add', '-A', '--', '.'], cwd=root, env=env, check=True)
            tree = subprocess.check_output(['git', 'write-tree'], cwd=root, env=env, text=True).strip()
    return dict(base=target, head=head, tree=tree, dirty=bool(status))


def copy_input(source, dest):
    digest = hashlib.sha256()
    with source.open('rb') as reader, dest.open('xb') as writer:
        while chunk := reader.read(1024 * 1024):
            writer.write(chunk)
            digest.update(chunk)
    shutil.copystat(source, dest)
    return digest.digest()


@contextmanager
def measure(timings, name):
    started = monotonic()
    try:
        yield
    finally:
        timings[name] = round(monotonic() - started, 3)


def check_generated(root=ROOT, timings=None):
    if timings is None:
        timings = {}
    temporary = tempfile.TemporaryDirectory(prefix='chnhcm-ci-generate-')
    tree = Path(temporary.name)
    try:
        with measure(timings, 'prepare'):
            files = run('git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z', cwd=root)
            parents = set()
            copied = {}
            for rel in os.fsdecode(files).split('\0')[:-1]:
                src, dest = root / rel, tree / rel
                try:
                    mode = src.lstat().st_mode
                except FileNotFoundError:
                    continue
                if not stat.S_ISREG(mode):
                    raise ValueError('cannot isolate generation input: ' + rel)
                if src.parent not in parents:
                    if src.parent.resolve() != src.parent:
                        raise ValueError('cannot isolate generation directory: ' + rel)
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    parents.add(src.parent)
                copied[rel] = (copy_input(src, dest), mode & stat.S_IXUSR)
        env = dict(os.environ, PATH=str(root / 'node_modules/.bin') + os.pathsep + os.environ['PATH'])
        with measure(timings, 'generate'):
            run_check(['make', 'generate'], cwd=tree, env=env)
        with measure(timings, 'compare'):
            drift = set()
            for parent, dirs, names in os.walk(tree, onerror=raise_walk_error):
                for name in dirs:
                    path = Path(parent, name)
                    if path.is_symlink():
                        drift.add(path.relative_to(tree).as_posix())
                for name in names:
                    path = Path(parent, name)
                    rel = path.relative_to(tree).as_posix()
                    expected = copied.pop(rel, None)
                    mode = path.lstat().st_mode
                    if expected is None or not stat.S_ISREG(mode):
                        drift.add(rel)
                        continue
                    with path.open('rb') as source:
                        if (hashlib.file_digest(source, 'sha256').digest(), mode & stat.S_IXUSR) != expected:
                            drift.add(rel)
            drift.update(copied)
            if drift:
                raise SystemExit('生成物与候选不一致，请运行 make generate 后复核：\n' + '\n'.join(sorted(drift)))
    finally:
        with measure(timings, 'cleanup'):
            temporary.cleanup()


def raise_walk_error(error):
    raise error


class ReadyRun:
    def __init__(self, root):
        parent = root / '.local/ci-ready'
        parent.mkdir(parents=True, exist_ok=True)
        stamp = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ-')
        self.folder = Path(tempfile.mkdtemp(dir=parent, prefix=stamp))
        self.result = dict(status='running', steps=[])
        print('本轮预检记录', self.folder / 'result.json', flush=True)
        self.save()

    def save(self):
        temporary = self.folder / 'result.tmp'
        temporary.write_text(json.dumps(self.result, ensure_ascii=False, indent=2) + '\n')
        temporary.replace(self.folder / 'result.json')

    @contextmanager
    def step(self, name):
        entry = dict(name=name, status='running')
        self.result['steps'].append(entry)
        self.save()
        started = monotonic()
        try:
            yield entry
        except BaseException as exc:
            entry['status'] = 'interrupted' if isinstance(exc, KeyboardInterrupt) else 'failed'
            if isinstance(exc, subprocess.CalledProcessError):
                entry['exit_code'] = exc.returncode
            self.result['first_failure'] = name
            raise
        else:
            entry['status'] = 'passed'
        finally:
            entry['seconds'] = round(monotonic() - started, 3)
            self.save()


def check_ready(plan, record, root=ROOT, preflight_only=False):
    if plan['scope'] == 'docs':
        with record.step('docs'):
            run_check(['make', 'docs'], cwd=root)
        return
    if not preflight_only:
        with record.step('generated') as step:
            step['timings_seconds'] = {}
            check_generated(root, step['timings_seconds'])
    files = plan['browser_files']
    affected = PREFLIGHT_CASES
    if plan['scope'] == 'full':
        with record.step('db_upgrades'):
            run_check(['python3', 'scripts/integration-shard.py', '--preflight',
                       '--results', str(record.folder / 'db.jsonl')], cwd=root)
    else:
        if not plan['browser_shards']:
            return
        with record.step('browser_inventory') as step:
            step['inventory'] = check_inventory(files, plan['browser_shards'])
        affected = [(file, title) for file, title in PREFLIGHT_CASES if not files or file in files]
    if affected:
        with record.step('browser_regressions') as step:
            titles = [title for _, title in affected]
            selected = [] if plan['scope'] == 'full' else sorted({file for file, _ in affected})
            args = arguments(selected) + title_args(titles)
            inventory = discover(args, titles)
            step['planned_cases'] = len(inventory)
            record.save()
            options = ['--output', str(record.folder / 'browser'),
                       '--reporter', 'list,' + str(root / 'scripts/ci-ready-reporter.cjs')]
            run_check(['python3', 'scripts/local-bu.py', 'smoke', '--', *options, *args], cwd=root)
            rows = read_cases(record.folder / 'browser.jsonl')
            expected = {(project, test_id) for project, _, test_id in inventory}
            actual = {(row['project'], row['id']) for row in rows if row['status'] == 'passed'}
            if len(rows) != len(expected) or actual != expected:
                raise ValueError('浏览器实际通过的用例与前置清单不同')


def read_cases(path):
    return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []


def execute_ready(base_ref, root=ROOT, preflight_only=False):
    from ci_preview import preview
    record = ReadyRun(root)
    result = record.result
    result['command'] = 'ci-preflight' if preflight_only else 'ci-ready'
    started = monotonic()
    try:
        initial = candidate(base_ref, root)
        result['candidate'] = initial
        plan = preview(initial['base'], repo=root)
        result.update(merge_base=plan['merge_base'], needs_base_sync=plan['needs_base_sync'],
                      changed_files=plan['changed_files'], scope=plan['current']['scope'])
        if initial['tree'] == git('rev-parse', plan['merge_base'] + '^{tree}', root=root):
            result.update(scope='none', changed_files=0)
        if preflight_only:
            result['scope'] = 'full'
            check_ready(dict(scope='full', browser_files=[]), record, root, preflight_only=True)
        elif result['changed_files']:
            check_ready(plan['current'], record, root)
        final = candidate(base_ref, root)
        if initial != final or plan['head'] != initial['head']:
            result['candidate_after'] = final
            raise ValueError('预检期间基线或候选已变化，请对新候选重新验证')
        result['status'] = 'passed' if preflight_only or result['changed_files'] else 'no_candidate'
    except BaseException as exc:
        result['status'] = 'interrupted' if isinstance(exc, KeyboardInterrupt) else 'failed'
        result.setdefault('first_failure', 'candidate')
        raise
    finally:
        counts = {}
        for name in ('db', 'browser'):
            cases = read_cases(record.folder / (name + '.jsonl'))
            counts[name] = {s: sum(c['status'] == s for c in cases) for s in sorted({c['status'] for c in cases})}
        result.update(case_results=counts, seconds=round(monotonic() - started, 3))
        record.save()
        identity = result.get('candidate')
        if identity:
            print('基线', identity['base'], ' HEAD', identity['head'], ' 文件树', identity['tree'],
                  '含未提交改动' if identity['dirty'] else '已提交')
        if result['status'] == 'no_candidate':
            print('无候选，执行 0 项。')
        else:
            print('本地预检', result['status'], '范围', result.get('scope', '未确定'))
            for step in result['steps']:
                inventory = step.get('inventory')
                detail = f"核对清单 {inventory['unique_tests']} 项" if inventory else ''
                print(step['name'], step['status'], detail)
            for label, layer in [('DB/API', 'db'), ('浏览器', 'browser')]:
                print(label, '已记录的用例结果', counts[layer] or '0 项')
    return record.folder
