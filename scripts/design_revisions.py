"""Resolve local design revisions without rewriting the imported baseline."""
import re
import subprocess


def historical_source_exists(root, record, relative):
    """Final evidence may name code removed after its recorded baseline."""
    if not record or record.get('status') != 'final':
        return False
    commit = record.get('base_commit', '')
    parts = relative.split('/')
    if not re.fullmatch(r'[0-9a-f]{40}', commit) or '..' in parts or parts[0] not in {'internal', 'tests', 'scripts', 'api', 'apps', 'packages', 'db', 'cmd'}:
        return False
    def exists(snapshot):
        result = subprocess.run(['git', 'cat-file', '-t', f'{snapshot}:{relative}'], cwd=root, capture_output=True, text=True)
        return result.returncode == 0 and result.stdout.strip() in {'blob', 'tree'}
    kind = subprocess.run(['git', 'cat-file', '-t', commit], cwd=root, capture_output=True, text=True)
    if kind.returncode != 0 or kind.stdout.strip() != 'commit':
        return False
    if exists(commit):
        return True
    # A report made against a dirty tree may describe code committed with the
    # report after base_commit. Use that snapshot only if the report is unchanged.
    path = record.get('path', '')
    if not path.startswith('docs/verification/') or '..' in path.split('/') or not (root / path).is_file():
        return False
    saved = subprocess.run(['git', 'log', '-1', '--format=%H', '--', path], cwd=root, capture_output=True, text=True).stdout.strip()
    if not re.fullmatch(r'[0-9a-f]{40}', saved):
        return False
    content = subprocess.run(['git', 'show', f'{saved}:{path}'], cwd=root, capture_output=True)
    return content.returncode == 0 and content.stdout == (root / path).read_bytes() and exists(saved)


def resolve_hashes(adopted_files, revisions):
    expected = {item['target_path']: item['adopted_sha256'] for item in adopted_files}
    if len(expected) != len(adopted_files):
        raise ValueError('duplicate adopted design path')
    latest_revisions = {}
    for revision in revisions:
        rid = revision.get('id', '')
        number = revision.get('revision', 1)
        if not isinstance(rid, str) or not re.fullmatch(r'CHNHCM-[0-9]{3}', rid):
            raise ValueError('duplicate or invalid local revision ID')
        if type(number) is not int or number < 1:
            raise ValueError('invalid local revision number')
        if number != latest_revisions.get(rid, 0) + 1:
            raise ValueError('duplicate or invalid local revision sequence')
        latest_revisions[rid] = number
        seen_paths = set()
        if not revision.get('files'):
            raise ValueError('empty local design revision')
        for item in revision['files']:
            path = item['path']
            if path not in expected or path in seen_paths:
                raise ValueError('unknown or duplicate revised design path: ' + path)
            seen_paths.add(path)
            if item['before_sha256'] != expected[path]:
                raise ValueError('broken revision chain: ' + path)
            after = item['after_sha256']
            if not re.fullmatch(r'[0-9a-f]{64}', after) or after == expected[path]:
                raise ValueError('invalid or unchanged revision digest: ' + path)
            expected[path] = after
    return expected
