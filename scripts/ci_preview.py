"""Preview a whole candidate or replay a committed group, independently of GitHub gates."""
import argparse
import json
from pathlib import Path
import re
import signal
import subprocess
import tarfile
import tempfile

from ci_shards import FULL_BROWSER, FULL_INTEGRATION
from ci_impact import full, select
from ci_policy import POLICY, ROOT, changed_paths, git, run, validate_mapping


def names(*args, root):
    return run('git', *args, cwd=root).decode().split('\0')[:-1]


def working_changes(base, root):
    tracked = names('diff', '--no-renames', '--name-only', '-z', base, '--', root=root)
    untracked = names('ls-files', '--others', '--exclude-standard', '-z', root=root)
    paths = sorted(set(tracked + untracked))
    special = {e.split('\t', 1)[1] for e in names('ls-tree', '-rz', base, root=root)
               if e.startswith(('120000 ', '160000 '))}
    regular = all(path not in special and not (root / path).is_symlink()
                  and (not (root / path).exists() or (root / path).is_file()) for path in paths)
    return paths, regular


API_SPEC = 'api/openapi.yaml'
API_CLIENT = 'packages/api-client/src/schema.d.ts'


def response_texts(document):
    # Only the literal 200 response text of two known reads. All other bytes
    # (including refs, other responses, writes, shared schemas) must match.
    texts = {}
    for route, operation in [('/api/v1/jobs', 'listJob'), ('/api/v1/jobs/{id}', 'getJob')]:
        heading = re.escape(f'  {route}:\n    get:\n      operationId: {operation}\n')
        pattern = (r'^(' + heading + r'(?:      (?!responses:)[^\n]*\n)*'
                   r"      responses:\n        '200':\n          description: )([\w][\w .,-]*)$")
        matches = list(re.finditer(pattern, document, re.M))
        if len(matches) == 1:
            texts[operation] = matches[0][2]
            document = re.sub(pattern, r'\g<1><response text>', document, flags=re.M)
    return document, texts


def api_pair(base, paths, tree, repo):
    if API_SPEC not in paths or not (tree / API_SPEC).is_file():
        return []
    present = names('ls-tree', '-rz', '--name-only', base, '--', API_SPEC, API_CLIENT, root=repo)
    if API_SPEC not in present:
        return []
    old_shape, old_texts = response_texts(run('git', 'show', f'{base}:{API_SPEC}', cwd=repo).decode())
    new_shape, new_texts = response_texts((tree / API_SPEC).read_bytes().decode())
    if old_shape != new_shape or old_texts.keys() != new_texts.keys() or old_texts == new_texts:
        return []
    # openapi-typescript mirrors these response descriptions in exactly one
    # JSDoc per operation. Any other generated change keeps the full plan.
    if API_CLIENT not in present or API_CLIENT not in paths or not (tree / API_CLIENT).is_file():
        return []
    client = run('git', 'show', f'{base}:{API_CLIENT}', cwd=repo).decode()
    for operation, old in old_texts.items():
        if old == new_texts[operation]:
            continue
        pattern = (r'(^    ' + re.escape(operation) + r': \{\n(?:        [^\n]*\n)*?'
                   r'        responses: \{\n            /\*\* @description )' + re.escape(old)
                   + r'( \*/\n            200: \{)')
        client, n = re.subn(pattern, lambda m: m[1] + new_texts[operation] + m[2], client, flags=re.M)
        if n != 1:
            return []
    return [API_SPEC, API_CLIENT] if client == (tree / API_CLIENT).read_bytes().decode() else []


INDEX_MIGRATION = re.compile(
    r'-- \+goose Up\s+CREATE INDEX (?P<index>job_read_[a-z0-9_]+) ON staffing\.job_read\s*'
    r'\((?:instance_id|enterprise_id|object_id|start_date|end_date|active|family_id)'
    r'(?:,\s*(?:instance_id|enterprise_id|object_id|start_date|end_date|active|family_id))*\);\s+'
    r'-- \+goose Down\s+DROP INDEX staffing\.(?P=index);\s*'
)
MIGRATION_PATH = re.compile(r'db/migrations/([0-9]{5})_[a-z0-9_]+\.sql')


def shadow_plan(base, paths, regular, current, policy, tree, repo):
    if not regular or current['scope'] != 'full':
        return current, []
    remaining = []
    admitted = api_pair(base, paths, tree, repo)
    migration = False
    base_sql = names('ls-tree', '-rz', '--name-only', base, '--', 'db/migrations/', root=repo)
    base_numbers = [int(Path(p).name.split('_', 1)[0]) for p in base_sql
                    if re.fullmatch(r'db/migrations/[0-9]{5}_.+\.sql', p)]
    for path in paths:
        if path in admitted:
            continue
        source = tree / path
        match = MIGRATION_PATH.fullmatch(path)
        if (match and not migration and base_numbers and int(match[1]) == max(base_numbers) + 1 and source.is_file()
                and INDEX_MIGRATION.fullmatch(source.read_text())):
            admitted.append(path)
            migration = True
            continue
        remaining.append(path)
    if not admitted:
        return current, []
    result = select(remaining, policy, root=tree, backend_impacts=('staffing',))
    if result['browser_shards'] == FULL_BROWSER and not result['browser_files']:
        return full(result['reason']), admitted
    if migration:
        result.update(integration_filter='^Test', integration_shards=FULL_INTEGRATION)
    return result, admitted


def report(base, paths, regular, tree, repo):
    policy = json.loads((tree / POLICY).read_text())
    validate_mapping(policy, tree)
    if not paths:
        empty = dict(scope='none', reason='无候选差异', integration_filter='', integration_shards=0,
                     browser_files=[], browser_shards=0)
        return dict(changed_files=0, current=empty, shadow=empty, shadow_inputs=[])
    current = select(paths, policy, regular, tree)
    shadow, admitted = shadow_plan(base, paths, regular, current, policy, tree, repo)
    return dict(changed_files=len(paths), current=current, shadow=shadow, shadow_inputs=admitted)


def preview(base_ref='origin/main', head_ref=None, repo=ROOT):
    def resolve(ref):
        return git('rev-parse', '--verify', '--end-of-options', ref + '^{commit}', root=repo)
    target, head = resolve(base_ref), resolve(head_ref or 'HEAD')
    base = git('merge-base', target, head, root=repo)
    if head_ref is None:
        paths, regular = working_changes(base, repo)
        result = report(base, paths, regular, repo, repo)
    else:
        paths, regular = changed_paths(base, head, repo)
        # Replay the candidate's mappings and consumer source, including files
        # since removed or renamed. No branch or worktree registration is needed.
        folders = {'apps', 'packages', 'cmd', 'internal', 'api', 'deploy', 'tests', 'db', '.github'}
        entries = names('ls-tree', '-z', '--name-only', head, root=repo)
        with tempfile.TemporaryDirectory() as folder:
            tree = Path(folder)
            with tempfile.TemporaryFile() as archive:
                subprocess.run(['git', 'archive', head, '--', *sorted(folders.intersection(entries))],
                               cwd=repo, stdout=archive, check=True)
                archive.seek(0)
                with tarfile.open(fileobj=archive) as tar:
                    for entry in tar:
                        if entry.isfile() or entry.isdir():
                            tar.extract(entry, tree, filter='data')
                        else:
                            regular = False
            result = report(base, paths, regular, tree, repo)
    return dict(target=target, merge_base=base, head=head, working_tree=head_ref is None,
                needs_base_sync=base != target, **result)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', default='origin/main')
    parser.add_argument('--head', help='replay a commit instead of including current working changes')
    parser.add_argument('--summary', type=Path, help='append a draft scope table to a GitHub step summary')
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument('--preflight', action='store_true', help='exercise the full early regressions directly')
    mode.add_argument('--ready', action='store_true', help='run local generation and applicable early regressions')
    args = parser.parse_args()
    if (args.ready or args.preflight) and args.head:
        parser.error('local checks execute the current working tree only')
    if args.ready or args.preflight:
        from ci_ready import execute_ready

        def interrupt(signum, frame):
            raise KeyboardInterrupt

        signal.signal(signal.SIGTERM, interrupt)
        try:
            execute_ready(args.base, preflight_only=args.preflight)
        except KeyboardInterrupt:
            raise SystemExit(130)
        return
    result = preview(args.base, args.head)
    print(json.dumps(result, ensure_ascii=False, indent=2), flush=True)
    if args.summary:
        with args.summary.open('a') as out:
            print('### 正式 PR 需要覆盖的范围\n', file=out)
            if not result['changed_files']:
                print('当前比较没有候选差异。转为正式 PR 后按当时的基线计算。', file=out)
                return
            print('| 规则 | 范围 | DB/API 分片 | 浏览器影响范围（分片数） |\n|---|---|---:|---:|', file=out)
            for key, label in [('current', '现行门禁'), ('shadow', '试算')]:
                plan = result[key]
                print(f"| {label} | {plan['scope']} | {plan['integration_shards']} | {plan['browser_shards']} |", file=out)
            print('\n草稿期间可用 `make ci-ready` 本地验证。转为正式 PR 后，quality 核对已有成功分片并显示实际派发数量。', file=out)


if __name__ == '__main__':
    main()
