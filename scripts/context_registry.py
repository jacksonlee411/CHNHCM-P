"""Check the evidence and ownership of individually registered implementation slices."""
from pathlib import Path


def slice_errors(root, contexts, verification_paths):
    errors = []
    for context in contexts:
        owner = context['code_path']
        slices = context.get('slices', [])
        if context['state'] == 'planned' and not slices:
            continue
        if context['state'] != 'partially_implemented':
            errors.append(f'{owner}: invalid implementation state')
        if {s.get('module') for s in slices} != set(context['modules']):
            errors.append(f'{owner}: each assigned module needs its own slice status')
        if len({(s.get('module'), s.get('capability')) for s in slices}) != len(slices):
            errors.append(f'{owner}: duplicate slice')
        for s in slices:
            label = f"{owner}/{s.get('capability')}"
            evidence = s.get('evidence', '')
            if not s.get('scope') or not s.get('capability') or not evidence.startswith('docs/') or not (root/evidence).is_file():
                errors.append(f'{label}: missing scope or evidence')
            if s.get('state') in ('contract_only', 'planned'):
                if 'entry' in s or 'source' in s:
                    errors.append(f'{label}: deferred slice claims an implemented entry')
                continue
            if s.get('state') != 'verified_slice' or evidence not in verification_paths:
                errors.append(f'{label}: verified slice requires registered verification')
            source = Path(s.get('source', ''))
            layers = ((root/owner/layer).resolve() for layer in ('application', 'domain', 'infra'))
            if source.suffix != '.go' or not (root/source).is_file() or not any(
                    (root/source).resolve().is_relative_to(layer) for layer in layers):
                errors.append(f'{label}: missing implementation in owning context')
            entry_source = (root/s.get('entry_source', '')).resolve()
            if not entry_source.is_relative_to(root) or not entry_source.is_file() or not s.get('entry') or s['entry'] not in entry_source.read_text():
                errors.append(f'{label}: missing menu or authorization entry')
    return errors
