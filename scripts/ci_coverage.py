"""Match actual test inventories and successful jobs across candidates of one PR."""
from datetime import datetime, timedelta, timezone
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
from urllib.parse import urlencode
import zipfile

from ci_impact import select

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = '.github/workflows/ci.yml'
POLICY = '.github/ci-scopes.json'
SHA = re.compile(r'^[0-9a-f]{40}$')


def run(*args, cwd=ROOT):
    return subprocess.check_output(args, cwd=cwd, stderr=subprocess.PIPE)


def git(*args, root=ROOT):
    return run('git', *args, cwd=root).decode().strip()


def changed_paths(base, head, root=ROOT):
    if not SHA.fullmatch(base) or not SHA.fullmatch(head):
        raise ValueError('invalid diff endpoints')
    # Disabling rename detection deliberately includes BOTH old and new names.
    raw = run('git', 'diff', '--no-renames', '--name-only', '-z', base, head, '--', cwd=root)
    paths = raw.decode().split('\0')[:-1]
    for ref in [base, head]:
        entries = run('git', 'ls-tree', '-r', '-z', ref, cwd=root).decode().split('\0')
        if any(e.startswith(('120000 ', '160000 ')) and e.split('\t', 1)[-1] in paths for e in entries):
            return paths, False
    return paths, True


def selection_for(base, checkout, root=ROOT):
    policy = json.loads((root / POLICY).read_text())
    paths, regular = changed_paths(base, checkout, root)
    return select(paths, policy, regular, root)


def latest_jobs(jobs):
    latest = {}
    for job in jobs:
        name = job['name']
        if name not in latest or (job['run_attempt'], job['id']) > (latest[name]['run_attempt'], latest[name]['id']):
            latest[name] = job
    return latest


def unpack_evidence(data, filename='ci-evidence.json'):
    limit = 524288 if filename == 'ci-coverage.json' else 65536
    if len(data) > limit * 2:
        raise ValueError('oversize CI evidence')
    with zipfile.ZipFile(io.BytesIO(data)) as z:
        items = z.infolist()
        if len(items) != 1 or items[0].filename != filename or items[0].file_size > limit:
            raise ValueError('invalid CI evidence archive')
        return json.loads(z.read(items[0]))


SUITES = {'integration': 8, 'browser': 11}
READ_ERRORS = (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError, zipfile.BadZipFile)


def inventory_part(identities, members):
    data = json.dumps(sorted(identities), separators=(',', ':')).encode()
    return dict(digest=hashlib.sha256(data).hexdigest(), members=sorted(set(members)))


def inventory_units(inventories):
    units = {'preflight': {}}
    for suite, count in SUITES.items():
        parts = inventories[suite]
        if len(parts) != count + 1:
            raise ValueError('incomplete full shard inventory')
        for number, part in enumerate(parts):
            if not part['members'] or not re.fullmatch('[0-9a-f]{64}', part['digest']):
                raise ValueError('missing shard members or digest')
            name = f'{suite} shard {number}/{count}' if number else 'preflight'
            units.setdefault(name, {})[suite] = part
    return units


def affected(parts, delta):
    if delta is None:
        return False
    if 'integration' in parts and delta['integration_shards']:
        pattern = re.compile(delta['integration_filter'])
        if any(pattern.search(name) for name in parts['integration']['members']):
            return True
    return ('browser' in parts and bool(delta['browser_shards'])
            and (not delta['browser_files'] or
                 not set(delta['browser_files']).isdisjoint(parts['browser']['members'])))


def delta_from(prior, current, root):
    if prior['tree'] == current['tree']:
        return None
    # Superseded GitHub merge refs need not survive. The parents normally remain
    # reachable; accept a reconstructed merge tree only if its object ID matches.
    try:
        kind = git('cat-file', '-t', prior['tree'], root=root)
    except subprocess.CalledProcessError:
        rebuilt = git('merge-tree', '--write-tree', prior['base'], prior['head'], root=root)
        if rebuilt.partition('\n')[0] != prior['tree']:
            raise ValueError('previous merge tree cannot be reconstructed')
        kind = 'tree'
    if kind != 'tree':
        raise ValueError('invalid previous merge tree')
    return selection_for(prior['tree'], current['tree'], root)


def checkpoint(api, rid, attempt):
    artifacts = api.pages(f'actions/runs/{rid}/artifacts', 'artifacts')
    matches = [a for a in artifacts if a['name'] == f'ci-coverage-{attempt}' and not a['expired']]
    if len(matches) != 1:
        return None
    return unpack_evidence(api.raw(f'actions/artifacts/{matches[0]["id"]}/zip'), 'ci-coverage.json')


def source_jobs(api, rid):
    return latest_jobs(api.pages(f'actions/runs/{rid}/jobs?filter=all', 'jobs'))


def same_pr_run(info, plan):
    return (info['event'] == 'pull_request' and info['path'] == WORKFLOW
            and (info.get('head_repository') or {}).get('full_name') == plan['repo']
            and 0 < int(info['id']) < int(plan['run_id'])
            and (not info['pull_requests'] or any(p['number'] == plan['pr'] for p in info['pull_requests'])))


def source_checkpoint(api, info, jobs, plan):
    quality = jobs.get('quality', {})
    if quality.get('status') != 'completed' or quality.get('conclusion') != 'success' or not quality.get('runner_id'):
        return None
    attempt = quality['run_attempt']
    record = checkpoint(api, info['id'], attempt)
    if not record or record['attempt'] != attempt:
        return None
    prior = record['plan']
    if (prior.get('coverage_attempt') != attempt or prior['run_id'] != str(info['id']) or prior['head'] != info['head_sha']
            or prior['scope'] != 'full' or prior['event'] != 'pull_request'
            or any(prior[k] != plan[k] for k in ('repo', 'pr', 'ref', 'contract'))):
        return None
    return record


def documentation_checkpoint(api, plan, previous_head, root):
    if not previous_head or not SHA.fullmatch(previous_head):
        return None
    if previous_head != plan['head'] and selection_for(previous_head, plan['head'], root)['scope'] != 'docs':
        return None
    runs = api.pages(f'actions/workflows/ci.yml/runs?event=pull_request&head_sha={previous_head}', 'workflow_runs')
    for info in sorted(runs, key=lambda r: r['id'], reverse=True):
        if info['head_sha'] != previous_head or not same_pr_run(info, plan):
            continue
        jobs = source_jobs(api, info['id'])
        record = source_checkpoint(api, info, jobs, plan)
        inherited = record['plan'].get('quality_source') if record else None
        if inherited:
            info = api.json(f'actions/runs/{inherited["run"]}')
            if not same_pr_run(info, plan):
                continue
            jobs = source_jobs(api, info['id'])
            record = source_checkpoint(api, info, jobs, plan)
            if not record or record['plan'].get('quality_source') or jobs['quality']['id'] != inherited['job']:
                continue
        if record:
            # The head diff is only a lookup hint. Include every base change
            # and conflict resolution by comparing the actual two merge trees.
            delta = delta_from(record['plan'], plan, root)
            if delta is None or delta['scope'] == 'docs':
                inventory_units(record['inventories'])
                return record, {'run': info['id'], 'job': jobs['quality']['id']}
    return None


def matching_jobs(record, info, jobs, units, names, delta, remote_browser):
    """Only a job that actually ran can issue a receipt for its native shard."""
    prior, attempt = record['plan'], record['attempt']
    original = inventory_units(record['inventories'])
    credits = {}
    for name in names:
        if remote_browser and 'browser' in units[name] and not prior['remote_browser']:
            continue
        job = jobs[name]
        if (name not in prior.get('carried', {}) and job['run_attempt'] >= attempt and job['conclusion'] == 'success'
                and job['status'] == 'completed' and job.get('runner_id')
                and units[name] == original[name] and not affected(units[name], delta)):
            credits[name] = {'run': info['id'], 'job': job['id']}
    return credits


def find_carried(api, plan, branch, inventories, root=ROOT):
    units = inventory_units(inventories)
    pending = {name for name, parts in units.items() if plan['remote_browser'] or 'integration' in parts}
    credits = {}
    since = (datetime.now(timezone.utc) - timedelta(days=7)).strftime('%Y-%m-%dT%H:%M:%SZ')
    query = urlencode(dict(branch=branch, event='pull_request', created='>=' + since))
    runs = api.pages('actions/workflows/ci.yml/runs?' + query, 'workflow_runs')
    for info in sorted(runs, key=lambda r: r['id'], reverse=True):
        if not same_pr_run(info, plan):
            continue
        try:
            jobs = source_jobs(api, info['id'])
        except READ_ERRORS:
            # An unreadable more recent run may have superseded older results.
            # Keep the credits already proved by newer runs.
            break
        # A failed smaller matrix has different slot numbers. Without a
        # full inventory for that run, its execution layer needs a fresh check.
        failed_layers = {name.split()[0] for name, job in jobs.items()
                         if re.fullmatch(r'(integration|browser) shard \d+/\d+', name)
                         and name not in units and job['conclusion'] == 'failure'}
        if failed_layers:
            pending = {name for name in pending if units[name].keys().isdisjoint(failed_layers)}
        # A more recent failure supersedes an older green. An intentionally
        # skipped job has no result and lets us find the actual original job.
        seen = {n for n in pending if n in jobs and jobs[n]['conclusion'] != 'skipped'}
        if seen:
            try:
                record = source_checkpoint(api, info, jobs, plan)
                if record:
                    if any(jobs[n]['conclusion'] == 'failure' for n in seen):
                        # A failed test can move into a previously inherited slot.
                        original = inventory_units(record['inventories'])
                        pending = {n for n in pending if units[n] == original[n]}
                        seen &= pending
                    delta = delta_from(record['plan'], plan, root)
                    credits.update(matching_jobs(record, info, jobs, units, seen, delta, plan['remote_browser']))
            except READ_ERRORS:
                pass  # These slots stay scheduled; independently proven slots still count.
            pending -= seen
        if not pending:
            break
    return credits


def verify_carried(api, plan, names, root=ROOT, *, verify_quality=False):
    carried = {n: ref for n, ref in plan.get('carried', {}).items() if n in names}
    quality = plan.get('quality_source') if verify_quality else None
    if not carried and not quality:
        return
    current = checkpoint(api, plan['run_id'], plan['coverage_attempt'])
    if not current or current['plan'] != plan or current['attempt'] != plan['coverage_attempt']:
        raise ValueError('current quality coverage record does not match the plan')
    units = inventory_units(current['inventories'])
    sources = {ref['run'] for ref in carried.values()}
    if quality:
        sources.add(quality['run'])
    for rid in sources:
        info = api.json(f'actions/runs/{rid}')
        jobs = source_jobs(api, rid)
        record = source_checkpoint(api, info, jobs, plan) if same_pr_run(info, plan) else None
        if not record:
            raise ValueError('previous PR quality record is unavailable')
        delta = delta_from(record['plan'], plan, root)
        if quality and quality['run'] == rid:
            if (jobs['quality']['id'] != quality['job'] or record['plan'].get('quality_source')
                    or (delta is not None and delta['scope'] != 'docs')
                    or record['inventories'] != current['inventories']):
                raise ValueError('previous runtime quality does not cover this candidate')
        expected = {n: ref for n, ref in carried.items() if ref['run'] == rid}
        if (not expected.keys() <= jobs.keys() or
                matching_jobs(record, info, jobs, units, expected, delta, plan['remote_browser']) != expected):
            raise ValueError('previous PR jobs do not cover the claimed shards')
