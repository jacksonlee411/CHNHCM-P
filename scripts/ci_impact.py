"""Explicit impact selection. A missing mapping widens only the affected layer."""
import json
from pathlib import Path, PurePosixPath
import re

from ci_shards import FULL_BROWSER, FULL_INTEGRATION, selected_browser_count, selected_integration_count

SELECTION_KEYS = ('scope', 'reason', 'integration_filter', 'integration_shards',
                  'browser_files', 'browser_shards')


def docs_path(path):
    if path in {'AGENTS.md', 'README.md', '.github/workflows/README.md', 'docs/.ignore',
                'docs/governance/registry.json', 'docs/assets/inventory.json'}:
        return True
    p = PurePosixPath(path)
    return (p.suffix == '.md' and len(p.parts) >= 2 and p.parts[0] == 'docs'
            and (len(p.parts) == 2 or p.parts[1] in {'adr', 'governance', 'guides', 'plans', 'specs', 'templates', 'verification'})
            and 'data' not in p.parts and '..' not in p.parts)


def evidence_inputs(root):
    # Executed code or fixtures that mention evidence keep that directory out of
    # the passive-document class. Scenario capture tools are not CI inputs.
    found = set()
    for folder in ('apps', 'packages', 'cmd', 'internal', 'api', 'deploy', 'tests/integration',
                   'tests/e2e', 'tests/fixtures', 'tests/capacity'):
        for p in (root / folder).rglob('*'):
            if p.is_file() and p.suffix in {'.go', '.ts', '.tsx', '.mjs', '.py', '.json', '.yaml', '.yml'}:
                found.update(re.findall(r'verification/data/([A-Za-z0-9_-]+)', p.read_text()))
    return found


def passive_evidence(path, inputs):
    p = PurePosixPath(path)
    return (len(p.parts) >= 5 and p.parts[:3] == ('docs', 'verification', 'data')
            and p.parts[3] not in inputs and '..' not in p.parts
            and p.suffix in {'.json', '.jsonl', '.png', '.txt', '.csv', '.gz'}
            and (p.suffix == '.txt' or not any(s in {'.py', '.mjs', '.js', '.go', '.sql', '.sh'} for s in p.suffixes)))


def full(reason):
    return dict(scope='full', reason=reason, integration_filter='^Test', integration_shards=FULL_INTEGRATION,
                browser_files=[], browser_shards=FULL_BROWSER)


def journey_consumers(path, root):
    folder = root / 'tests/e2e'
    if not (root / path).is_file():
        return set()
    # Journey and suite modules are imported by file-level Playwright entries.
    # Include re-exports and literal dynamic imports through intermediate helpers.
    imports = {}
    for file in folder.rglob('*'):
        if file.suffix not in {'.ts', '.tsx', '.js', '.mjs', '.cjs', '.mts', '.cts'}:
            continue
        for module in re.findall(r'''(?:\bfrom\s*|\bimport\s*(?:\(\s*)?|\brequire\s*\(\s*)['"](\.[^'"]+)['"]''', file.read_text()):
            target = (file.parent / module).resolve()
            if target.is_dir():
                target /= 'index.ts'
            elif target.suffix in {'.js', '.mjs', '.cjs'}:
                target = target.with_suffix('.ts')
            elif not module.endswith('.ts'):
                target = Path(str(target) + '.ts')
            imports.setdefault(target, set()).add(file.resolve())
    pending, seen, entries = [(root / path).resolve()], set(), set()
    while pending:
        helper = pending.pop()
        if helper in seen:
            continue
        seen.add(helper)
        if helper.name in {'playwright.config.ts', 'runtime-setup.ts'}:
            return set()
        if helper.name.endswith('.spec.ts'):
            entries.add(helper.relative_to(folder.resolve()).as_posix())
        pending.extend(imports.get(helper, ()))
    return entries


def selected(integration, browser, reasons, policy):
    """None means all, an empty set means not applicable, otherwise a union."""
    if integration is None:
        pattern, count = '^Test', FULL_INTEGRATION
    elif not integration:
        pattern, count = '', 0
    else:
        # Unclassified/shared test roots ALWAYS participate in backend changes.
        excluded = [prefix for domain, prefixes in policy['integration_prefixes'].items()
                    if domain not in integration for prefix in prefixes]
        pattern = '^Test' + ('(?!(?:' + '|'.join(sorted(excluded)) + '))' if excluded else '')
        count = selected_integration_count(pattern) if excluded else FULL_INTEGRATION
    if browser is None:
        files, shards = [], FULL_BROWSER
    else:
        files = sorted(browser)
        shards = selected_browser_count(files)
        all_files = {f for spec in policy['domains'].values() for f in spec['browser_files']}
        if files and set(files) == all_files:
            files, shards = [], FULL_BROWSER
    scope = 'full' if pattern == '^Test' and not files and shards == FULL_BROWSER else 'selected' if count or shards else 'selected' if any(r != 'documentation/passive evidence' for r in reasons) else 'docs'
    return dict(scope=scope, reason='; '.join(sorted(set(reasons))), integration_filter=pattern,
                integration_shards=count, browser_files=files, browser_shards=shards)


def select(paths, policy, regular=True, root=Path('.'), *, backend_impacts=()):
    if (not paths and not backend_impacts) or not regular:
        return full('empty diff or non-regular entry')
    inputs = evidence_inputs(root) if any(p.startswith('docs/verification/data/') for p in paths) else set()
    domains = policy['domains']
    owners = {p: d for d, spec in domains.items() for p in spec['frontend']}
    integration, browser, reasons = set(), set(), []
    backend_domains, product_domains = set(), set()

    def add_domain(domain, backend=False):
        nonlocal integration, browser
        product_domains.add(domain)
        consumers = domains[domain]['backend_consumers'] if backend else [domain]
        if backend and integration is not None:
            integration.update(consumers)
        if browser is not None:
            browser_domains = domains[domain].get('backend_browser_domains', consumers) if backend else consumers
            for d in browser_domains:
                browser.update(domains[d]['browser_files'])
            if backend:
                browser.update(domains[domain].get('backend_extra_browser_files', []))
            browser.update(policy['shared_browser_files'])

    for domain in backend_impacts:
        backend_domains.add(domain)
        add_domain(domain, backend=True)
        reasons.append('backend domain and reference consumers: ' + domain)

    for path in paths:
        if docs_path(path) or passive_evidence(path, inputs):
            reasons.append('documentation/passive evidence')
            continue
        if path in {'scripts/check-repository.py', 'scripts/document_catalog.py', 'scripts/document_governance.py', 'scripts/design_revisions.py'} or (
                path.startswith('scripts/test_') and Path(path).suffix in {'.py', '.mjs'}):
            reasons.append('document tooling or tooling unit test: ' + path)
            continue
        if path.startswith('apps/web/src/') and Path(path).suffix in {'.ts', '.tsx', '.css'}:
            domain = owners.get(path)
            if domain:
                add_domain(domain)
                # A new cross-module import is not silently outside the map.
                pending, seen = [path], set()
                sources = [(p.relative_to(root).as_posix(), p.read_text())
                           for p in (root / 'apps/web/src').rglob('*')
                           if p.suffix in {'.ts', '.tsx'}]
                while pending:
                    current = pending.pop()
                    if current in seen:
                        continue
                    seen.add(current)
                    name = Path(current).stem
                    for rel, text in sources:
                        if rel in {current, 'apps/web/src/app/App.tsx'}:
                            continue
                        imports = re.findall(r'''(?:\bfrom\s*|\bimport\s*(?:\(\s*)?|\brequire\s*\(\s*)['"]([^'"]+)['"]''', text)
                        if any(Path(module).stem == name for module in imports):
                            if rel in owners:
                                add_domain(owners[rel])
                                pending.append(rel)
                            else:
                                browser = None
                reasons.append('frontend domain: ' + domain)
            else:
                browser = None
                reasons.append('shared/unmapped frontend: ' + path)
            continue
        if path.startswith('tests/e2e/') and path.endswith('.spec.ts'):
            if (root / path).is_file():
                if browser is not None:
                    browser.add(path.removeprefix('tests/e2e/'))
            else:
                browser = None
            reasons.append('browser test: ' + path)
            continue
        if path.startswith('tests/e2e/') and path.endswith(('-journey.ts', '-suite.ts')):
            consumers = journey_consumers(path, root)
            if not consumers:
                browser = None
            elif browser is not None:
                browser.update(consumers)
            reasons.append('browser journey consumers: ' + path)
            continue
        if path.startswith(('tests/integration/', 'internal/bootstrap/')) and path.endswith('_test.go'):
            # Helpers in a Go test file may serve other roots in the package.
            integration = None
            reasons.append('integration test/package helpers: ' + path)
            continue
        if path.endswith('_test.go') and path.startswith(('internal/', 'cmd/')):
            reasons.append('unit test: ' + path)
            continue
        owner = next((d for d, spec in domains.items() if path.endswith('.go')
                      and any(path.startswith(prefix) for prefix in spec['backend_prefixes'])), None)
        owner = owner or next((d for prefix, d in policy['business_tools'].items() if path.startswith(prefix)), None)
        if owner:
            backend_domains.add(owner)
            add_domain(owner, backend=True)
            reasons.append('backend domain and reference consumers: ' + owner)
            continue
        if path in {'pnpm-lock.yaml', 'apps/web/package.json', 'tsconfig.json'} or path.startswith('packages/ui/'):
            browser = None
            reasons.append('frontend dependencies/components: ' + path)
            continue
        return full('shared/unknown runtime or validation input: ' + path)
    # Discover newly introduced Go consumers instead of silently relying on an
    # old domain list. Bootstrap and infrastructure are the admitted wiring and
    # storage adapters; their own changes remain full through the rule above.
    pending, seen = list(backend_domains), set()
    sources = [(p.as_posix(), p.read_text()) for p in (root / 'internal').rglob('*.go')
               if not p.name.endswith('_test.go')] if pending else []
    while pending:
        domain = pending.pop()
        if domain in seen:
            continue
        seen.add(domain)
        prefixes = domains[domain]['backend_prefixes'] or [
            f'internal/contexts/organization/{layer}/{domain}/' for layer in ('application', 'domain')]
        packages = [re.escape(p.rstrip('/')) for p in prefixes]
        pattern = r'["`]chnhcm\.local/chnhcm/(?:' + '|'.join(packages) + r')(?:/|["`])'
        for file, text in sources:
            if not re.search(pattern, text):
                continue
            path = Path(file).relative_to(root).as_posix()
            owner = next((d for d, spec in domains.items()
                          if any(path.startswith(prefix) for prefix in spec['backend_prefixes'])), None)
            if owner:
                if owner != domain:
                    add_domain(owner, backend=True)
                    pending.append(owner)
            elif not path.startswith(('internal/bootstrap/', 'internal/contexts/organization/infra/')):
                return full('unmapped Go consumer: ' + path)
    mapped_specs = {f for spec in domains.values() for f in spec['browser_files']}
    tests = root / 'tests/e2e'
    unknown_specs = {p.relative_to(tests).as_posix() for p in tests.rglob('*.spec.ts')} - mapped_specs
    if product_domains and unknown_specs:
        browser = None
        reasons.append('new browser consumers require admission: ' + ', '.join(sorted(unknown_specs)))
    return selected(integration, browser, reasons, policy)


def validate_mapping(policy, root):
    domains = policy['domains']
    paths = [p for spec in domains.values() for p in spec['frontend']]
    if len(paths) != len(set(paths)) or set(domains) != set(policy['integration_prefixes']):
        raise ValueError('duplicate source owner or missing integration domain')
    for domain, spec in domains.items():
        if (not spec['browser_files'] or not set(spec['backend_consumers']) <= domains.keys()
                or domain not in spec['backend_consumers']
                or not set(spec.get('backend_browser_domains', [])) <= domains.keys()):
            raise ValueError('invalid domain consumers: ' + domain)
        for file in spec['browser_files'] + spec.get('backend_extra_browser_files', []):
            if (PurePosixPath(file).name != file or not file.endswith('.spec.ts')
                    or not (root / 'tests/e2e' / file).is_file()):
                raise ValueError('missing mapped browser consumer: ' + file)
