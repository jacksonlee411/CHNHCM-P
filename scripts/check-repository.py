#!/usr/bin/env python3
"""Read-only checks for CHNHCM document registration and source preservation."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import unquote, urlsplit
import zipfile
from context_registry import slice_errors
from design_revisions import resolve_hashes, historical_source_exists
from document_catalog import catalog_matches
from document_governance import markdown_anchors, markdown_links, validate_governance
from project_skills import project_skill_errors

ROOT = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--dry-run', action='store_true', help='explicit read-only mode (all checks are read-only)')
parser.add_argument('--check-refs', action='store_true', help='also require main and all six local Git branch refs')
args = parser.parse_args()
errors = []


def require(condition, message):
    if not condition:
        errors.append(message)


def read_json(path):
    return json.loads((ROOT / path).read_text())


def digest(data):
    return hashlib.sha256(data).hexdigest()


def local_file(path):
    p = (ROOT / path).resolve()
    require(p.is_relative_to(ROOT), f'path escapes repository: {path}')
    return p


registry = read_json('docs/governance/registry.json')
errors.extend(project_skill_errors(ROOT))
if errors:
    print('\n'.join(errors))
    raise SystemExit(1)
errors.extend(validate_governance(ROOT, registry))
if errors:
    print('\n'.join(errors))
    raise SystemExit(1)
try:
    require(catalog_matches(registry), '文档目录或默认检索范围已过期，请运行 python3 -B scripts/document_catalog.py。')
except ValueError as error:
    errors.append(str(error))
branches = registry['fixed_branches']
require([b['branch'] for b in branches] == ['wt-dev-' + c for c in 'abcdef'], 'six fixed branches required')
ranges = {}
used = set()
for branch in branches:
    limits = branch['document_range']
    numbers = set(range(limits['start'], limits['end'] + 1))
    require(numbers and not used.intersection(numbers), f'document range overlap: {branch["branch"]}')
    used.update(numbers)
    ranges[branch['branch']] = numbers
ids = {doc['id'] for doc in registry['documents']}
require(registry['first_slice']['module'] == 'M02', 'first slice must be M02')
require({'CHNHCM-010', 'CHNHCM-011', 'CHNHCM-012', 'CHNHCM-013'}.issubset(ids), 'four next documents must be registered')
if args.check_refs:
    refs = subprocess.check_output(['git', 'for-each-ref', '--format=%(refname:short)', 'refs/heads'], cwd=ROOT, text=True).splitlines()
    require(set(['main'] + list(ranges)).issubset(refs), 'missing local branch refs')

contexts = read_json('docs/architecture/contexts.json')
modules = [m for c in contexts['contexts'] for m in c['modules']]
require(len(contexts['contexts']) == 17 and len(modules) == len(set(modules)), 'invalid context map')
require(set(modules) == {f'M{i:02}' for i in range(1, 21)} | {'X02', 'X03', 'X06'}, 'context module coverage')
require(contexts['first_slice'] == 'M02', 'context first slice mismatch')
errors.extend(slice_errors(ROOT, contexts['contexts'], {r['path'] for r in registry['governance']['verification_records']}))

baseline = read_json('docs/reference/baselines.json')
supplemental_sources = read_json('docs/reference/ui-sources.json')['sources']
all_packages = [baseline['architecture'], baseline['cubebox'], *supplemental_sources]
require(len({source['archive'] for source in all_packages}) == len(all_packages), 'duplicate source archive')
for source in all_packages:
    package = local_file(source['archive'])
    if source.get('private_url'):
        require(source['private_url'].startswith('https://github.com/jacksonlee411/CHNHCM/blob/')
                and not package.exists(), 'private source archive must remain outside the public checkout')
        continue
    require(digest(package.read_bytes()) == source['archive_sha256'], f'archive drift: {package}')
    with zipfile.ZipFile(package) as archive:
        members = {m['path']: m['sha256'] for m in source['archive_members']}
        require(set(archive.namelist()) == set(members), f'archive member mismatch: {package}')
        for name, expected in members.items():
            require(not name.startswith('/') and '..' not in Path(name).parts, f'unsafe archive path: {name}')
            require(digest(archive.read(name)) == expected, f'archive content drift: {name}')
local_revisions = read_json('docs/reference/local-revisions.json')['revisions'] if (ROOT / 'docs/reference/local-revisions.json').is_file() else []
for revision in local_revisions:
    require(revision['id'] in ids, f'unregistered local revision: {revision["id"]}')
    require(local_file(revision['decision']).is_file(), f'missing local revision decision: {revision["id"]}')
try:
    expected_design = resolve_hashes(baseline['architecture']['adopted_files'], local_revisions)
except ValueError as error:
    errors.append(str(error))
    expected_design = {}
for path, expected in expected_design.items():
    p = local_file(path)
    require(p.is_file() and digest(p.read_bytes()) == expected, f'adopted design drift outside recorded revisions: {p}')
builder = ROOT / 'docs/architecture/target/scripts/build_modules.py'
if builder.is_file():
    generated = subprocess.run([sys.executable, '-B', str(builder), '--check'], cwd=ROOT, capture_output=True, text=True)
    require(generated.returncode == 0, 'generated design differs: ' + generated.stdout + generated.stderr)
if baseline['architecture']['current_integration_status'] != 'pending':
    evidence = baseline['architecture'].get('integration_evidence', [])
    require(bool(evidence) and all(local_file(p).is_file() for p in evidence), 'integration completion requires new local evidence')

inventory = read_json('docs/assets/inventory.json')
require({a['category'] for a in inventory['assets']} == {'experience', 'business', 'verification'}, 'three asset classes required')
cube_sources = {s['archive']: s for s in [baseline['cubebox'], *supplemental_sources]}
require(all(s.get('repository') == 'CubeBox' for s in supplemental_sources), 'unexpected supplemental source')
asset_ids = set()
for asset in inventory['assets']:
    require(asset['id'] not in asset_ids, 'duplicate asset ID')
    asset_ids.add(asset['id'])
    source = asset['source']
    package = cube_sources.get(asset['preserved_in']['archive'])
    require(package is not None, f'unregistered asset source package: {asset["id"]}')
    if package is None:
        continue
    source_members = {m['path']: m['sha256'] for m in package['archive_members']}
    require(source['commit'] == package['commit'], f'asset baseline mismatch: {asset["id"]}')
    require(source_members.get(source['path']) == source['sha256'], f'asset source hash mismatch: {asset["id"]}')
    require(asset['preserved_in']['member'] == source['path'], f'asset preservation location mismatch: {asset["id"]}')
    require(not asset['state']['verified'] or bool(asset['new_evidence']), f'verified asset without new evidence: {asset["id"]}')

# Native validation assets keep their source snapshot separately from imported
# CubeBox archives; neither an old pass nor a registered file proves adoption.
native_ids = set()
for asset in inventory.get('native_verification_assets', []):
    require(asset['id'] not in native_ids, 'duplicate native verification asset ID')
    native_ids.add(asset['id'])
    require(asset['category'] == 'verification' and asset['source']['repository'] == 'CHNHCM', 'invalid native verification owner')
    preserved = local_file(asset['preserved_in']['file'])
    if 'private_url' in asset['preserved_in']:
        require(not preserved.exists() and asset['preserved_in']['private_url'].startswith('https://github.com/jacksonlee411/CHNHCM/blob/'),
                f'private evidence cannot be versioned: {asset["id"]}')
    else:
        require(preserved.is_file() and digest(preserved.read_bytes()) == asset['source']['sha256'], f'native source hash mismatch: {asset["id"]}')
    require(not asset['state']['verified'] or bool(asset['new_evidence']), f'verified native asset lacks evidence: {asset["id"]}')
    for path in [*asset['target_files'], *asset['new_evidence']]:
        if path.startswith('https://github.com/jacksonlee411/CHNHCM/blob/'):
            continue
        require(local_file(path).is_file(), f'native verification target/evidence missing: {path}')

# Customer workbooks may not be staged, even through git add -f.
private_blobs = {
    '7bd2e09c8ce6e0fe4ee2052eab0d5756c3bd6f2b',
    '5d39d54a0e9ccee87ae75e8c15d988548c13659c',
    '280adc896b4bb78ffa11e4dc14ade378f4db0bfa',
}
entries = subprocess.check_output(['git', 'ls-files', '--stage', '-z'], cwd=ROOT).decode().split('\0')
for entry in filter(None, entries):
    metadata, path = entry.split('\t', 1)
    require(metadata.split()[1] not in private_blobs and not path.startswith('tests/fixtures/vanke/')
            and not re.fullmatch(r'orgunit-import-.*-from-source\.xlsx', Path(path).name, re.I),
            'customer organization workbook may not be versioned')

# Check file targets and explicit glossary anchors; do not claim external URL or all heading-slug validation.
markdown_count = link_count = 0
historical_source_links = 0
reports = {v['path']: v for v in registry['governance']['verification_records']}
for directory, dirs, files in os.walk(ROOT):
    dirs[:] = [d for d in dirs if d not in {'.git', '.local', 'node_modules', 'dist', '__pycache__'}]
    for name in files:
        if not name.endswith('.md'):
            continue
        path = Path(directory) / name
        markdown_count += 1
        content = path.read_text()
        for target in markdown_links(content):
            url = urlsplit(target)
            if url.scheme or url.netloc:
                continue
            resolved = (path.parent / unquote(url.path)).resolve() if url.path else path
            require(resolved.is_relative_to(ROOT), f'external local dependency: {path.relative_to(ROOT)} -> {target}')
            exists = resolved.exists()
            if not exists and resolved.is_relative_to(ROOT):
                exists = historical_source_exists(ROOT, reports.get(str(path.relative_to(ROOT))), str(resolved.relative_to(ROOT)))
                historical_source_links += int(exists)
            require(exists, f'broken link: {path.relative_to(ROOT)} -> {target}')
            if url.fragment.startswith(('term-', 'source-')) and resolved.is_file():
                require(unquote(url.fragment) in markdown_anchors(resolved.read_text()), f'missing explicit anchor: {target}')
            link_count += 1
diagrams = ROOT / 'docs/architecture/target/diagrams'
require(len(list(diagrams.glob('*.svg'))) == 53, '53 source diagrams expected')
for svg in diagrams.glob('*.svg'):
    require(svg.with_suffix('.mmd').is_file(), f'missing Mermaid source: {svg.name}')
for folder in ['apps/web/src', 'packages/ui/src', 'internal', 'cmd']:
    for path in (ROOT / folder).rglob('*'):
        if path.suffix in {'.ts', '.tsx', '.go'}:
            text = path.read_text()
            require('/home/vivobook/projects/CubeBox' not in text and '/home/vivobook/projects/oracle-fscr' not in text, f'source-workspace dependency: {path}')

if errors:
    print('\n'.join(errors))
    raise SystemExit(1)
print(json.dumps({'status': 'passed', 'read_only': True, 'markdown_files': markdown_count, 'local_links': link_count, 'historical_source_links': historical_source_links, 'target_diagrams': 53, 'preserved_assets': len(asset_ids), 'fixed_branch_ranges': 6, 'git_refs_checked': args.check_refs, 'limits': 'Historical source links require the final report baseline or unchanged committed report snapshot; no external URL, full heading slug, domain semantics, database or product acceptance validation'}, ensure_ascii=False))
