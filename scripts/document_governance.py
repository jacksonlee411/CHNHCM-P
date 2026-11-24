"""Read-only document governance checks, using only the Python standard library.

This deliberately parses the repository's small Markdown convention, not all of
CommonMark. Code fences are excluded from document headings, links and anchors.
"""

from datetime import date
import os
from pathlib import Path, PurePosixPath
import re
from urllib.parse import unquote, urlsplit
from design_revisions import historical_source_exists


DOCUMENT_ID = re.compile(r'CHNHCM-[0-9]{3}\Z')
STABLE_ID = re.compile(r'(?:M(?:0[1-9]|1[0-9]|20)|X0[1-6])-(?:BR|SCN|AC)-(?!000)[0-9]{3}\Z')
STABLE_ID_MENTION = re.compile(r'\b(?:M\d{2}|X\d{2})-(?:BR|SCN|AC)-\d{3}\b')
STABLE_ANCHOR = re.compile(r'(?:m(?:0[1-9]|1[0-9]|20)|x0[1-6])-(?:br|scn|ac)-(?!000)[0-9]{3}\Z')
STABLE_ANCHOR_CANDIDATE = re.compile(r'[mx][0-9]{2}-(?:br|scn|ac)-[0-9]{3}\Z', re.IGNORECASE)
HEADINGS = re.compile(r'^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$', re.MULTILINE)
ANCHORS = re.compile(r'<a\s+[^>]*?id=["\']([^"\']+)["\'][^>]*>', re.IGNORECASE)
IGNORED_DIRS = {'.git', '.local', 'node_modules', 'dist', '__pycache__'}
TYPE_DIRECTORIES = {
    'governance': 'docs/governance/', 'decision': 'docs/adr/',
    'architecture': 'docs/architecture/', 'preservation': 'docs/assets/',
    'plan': 'docs/plans/', 'business_spec': 'docs/specs/',
    'functional_spec': 'docs/specs/', 'technical_design': 'docs/specs/',
    'guide': 'docs/guides/',
}
STATUSES = {'planned', 'draft', 'in_review', 'active', 'superseded', 'retired', 'cancelled'}
LEGACY_REPORTS = {'docs/verification/bootstrap.md', 'docs/verification/009-m02-terminology.md'}


def markdown_body(text):
    """Remove fenced blocks while preserving line numbers for diagnostics."""
    output = []
    fence = None
    for line in text.splitlines():
        marker = re.match(r'^ {0,3}(`{3,}|~{3,})(.*)$', line)
        if fence is None and marker:
            fence = marker.group(1)
            output.append('')
        elif fence is not None:
            if marker and marker.group(1)[0] == fence[0] and len(marker.group(1)) >= len(fence) and not marker.group(2).strip():
                fence = None
            output.append('')
        else:
            output.append(line)
    return '\n'.join(output)


def link_definitions(text):
    """Resolve the reference labels supported by this repository's convention."""
    definitions = {}
    for match in re.finditer(r'^ {0,3}\[([^\]]+)\]:\s*(?:<([^>]+)>|(\S+))', markdown_prose(text), re.MULTILINE):
        definitions[' '.join(match.group(1).split()).casefold()] = match.group(2) or match.group(3)
    return definitions


def link_occurrences(body, definitions):
    """Return (start, end, label, target) for actual links, not their mentions."""
    links = []
    for match in re.finditer(r'!?\[([^\]\n]*)\]\(\s*(?:<([^>]+)>|([^\s)]+))(?:\s+["\'][^\n]*?["\'])?\s*\)', body):
        links.append((match.start(), match.end(), match.group(1), match.group(2) or match.group(3)))
    for match in re.finditer(r'!?\[([^\]\n]+)\]\[([^\]\n]*)\]', body):
        key = ' '.join((match.group(2) or match.group(1)).split()).casefold()
        if key in definitions and not any(start <= match.start() < end for start, end, _, _ in links):
            links.append((match.start(), match.end(), match.group(1), definitions[key]))
    for match in re.finditer(r'(?<![!\]])\[([^\]\n]+)\](?![\[(:])', body):
        key = ' '.join(match.group(1).split()).casefold()
        if key in definitions and not any(start <= match.start() < end for start, end, _, _ in links):
            links.append((match.start(), match.end(), match.group(1), definitions[key]))
    return sorted(links)


def markdown_links(text):
    """Return simple inline and reference-style destinations, excluding code."""
    return [target for _, _, _, target in link_occurrences(markdown_prose(text), link_definitions(text))]


def markdown_prose(text):
    """Exclude fences and matched inline code, including longer backtick runs."""
    body = markdown_body(text)
    markers = list(re.finditer(r'`+', body))
    output, cursor, index = [], 0, 0
    while index < len(markers):
        marker = markers[index]
        prefix = body[:marker.start()]
        escaped = (len(prefix) - len(prefix.rstrip('\\'))) % 2 == 1
        if escaped:
            index += 1
            continue
        closing = next((other for other in range(index + 1, len(markers))
                        if markers[other].group() == marker.group()), None)
        if closing is None:
            index += 1
            continue
        end = markers[closing].end()
        output.append(body[cursor:marker.start()])
        output.append(re.sub(r'[^\n]', ' ', body[marker.start():end]))
        cursor, index = end, closing + 1
    output.append(body[cursor:])
    return ''.join(output)


def markdown_anchors(text):
    """Return actual explicit anchors; examples in inline/fenced code are inert."""
    return ANCHORS.findall(markdown_prose(text))


def heading_errors(text, label, numbered=True):
    errors = []
    headings = HEADINGS.findall(markdown_body(text))
    if not headings or len(headings[0][0]) != 1 or sum(len(level) == 1 for level, _ in headings) != 1:
        errors.append(f'{label}: exactly one leading H1 required')
    counters = []
    for hashes, title in headings:
        level = len(hashes)
        if level == 1 or not numbered:
            continue
        match = re.match(r'^([1-9][0-9]*(?:\.[1-9][0-9]*)*)\s+\S', title)
        if level > 4 or not match:
            errors.append(f'{label}: numbered H2-H4 required: {hashes} {title}')
            continue
        numbers = [int(value) for value in match.group(1).split('.')]
        depth = level - 1
        if len(numbers) != depth or depth > len(counters) + 1:
            errors.append(f'{label}: heading level/number mismatch: {title}')
            continue
        expected = counters[:depth - 1] + [(counters[depth - 1] + 1) if len(counters) >= depth else 1]
        if numbers != expected:
            errors.append(f'{label}: discontinuous heading number: {title}; expected {".".join(map(str, expected))}')
        counters = numbers
    return errors


def safe_path(root, value):
    if not isinstance(value, str) or not value or '\\' in value:
        return None
    path = PurePosixPath(value)
    if path.is_absolute() or '..' in path.parts or str(path) != value:
        return None
    candidate = root / value
    if not candidate.resolve().is_relative_to(root.resolve()):
        return None
    return candidate


def is_date(value):
    if not isinstance(value, str) or not re.fullmatch(r'\d{4}-\d{2}-\d{2}', value):
        return False
    try:
        date.fromisoformat(value)
        return True
    except ValueError:
        return False


def validate_governance(root, registry):
    """Validate the v2 snapshot; no claim of Git-history transition auditing."""
    root = Path(root)
    errors = []

    def require(condition, message):
        if not condition:
            errors.append(message)

    def nonempty(value):
        return isinstance(value, str) and bool(value.strip())

    def strings(value):
        return isinstance(value, list) and all(nonempty(item) for item in value)

    def existing_file(value, label):
        path = safe_path(root, value)
        require(path is not None and path.is_file(), f'{label}: missing or unsafe evidence path: {value}')
        return path

    if not isinstance(registry, dict):
        return ['registry must be an object']
    require(registry.get('schema_version') == 2, 'registry schema_version must be 2')
    documents = registry.get('documents')
    governance = registry.get('governance')
    branches = registry.get('fixed_branches')
    if not isinstance(documents, list) or not isinstance(governance, dict) or not isinstance(branches, list):
        return errors + ['registry requires documents, governance and fixed_branches']
    ranges = {}
    for branch in branches:
        if not isinstance(branch, dict) or not isinstance(branch.get('document_range'), dict):
            errors.append('invalid fixed branch record')
            continue
        limits = branch['document_range']
        if not isinstance(branch.get('branch'), str) or type(limits.get('start')) is not int or type(limits.get('end')) is not int:
            errors.append('invalid branch document range')
            continue
        ranges[branch['branch']] = range(limits['start'], limits['end'] + 1)

    require(governance.get('legacy_documents') == [], 'legacy_documents must be empty; numbered documents have no format exemption')
    require(nonempty(governance.get('migration_note')), 'governance migration_note required')
    by_id = {}
    registered_paths = {}
    required = {'id', 'title', 'path', 'owner_branch', 'owner_role', 'status', 'type', 'modules', 'scope',
                'owner', 'format', 'review', 'execution_status', 'superseded_by', 'closure', 'archive'}
    for doc in documents:
        if not isinstance(doc, dict):
            errors.append('document record must be an object')
            continue
        label = doc.get('id', '<missing ID>')
        require(required <= doc.keys(), f'{label}: missing metadata fields: {sorted(required - doc.keys())}')
        valid_id = isinstance(label, str) and bool(DOCUMENT_ID.fullmatch(label))
        require(valid_id and label != 'CHNHCM-000', f'invalid document ID: {label}')
        if not valid_id:
            continue
        require(label not in by_id, f'duplicate document ID: {label}')
        by_id[label] = doc
        owner_branch = doc.get('owner_branch')
        require(isinstance(owner_branch, str) and owner_branch in ranges and int(label[-3:]) in ranges[owner_branch],
                f'{label}: document number outside owner_branch range')
        for field in ('title', 'owner_role', 'scope'):
            require(nonempty(doc.get(field)), f'{label}: {field} required')
        require(doc.get('owner') is None or nonempty(doc['owner']), f'{label}: owner must be null or a name')
        modules = doc.get('modules')
        require(strings(modules) and len(modules) == len(set(modules)) and all(re.fullmatch(r'M(?:0[1-9]|1[0-9]|20)|X0[1-6]', module) for module in modules),
                f'{label}: invalid modules')
        status = doc.get('status')
        kind = doc.get('type')
        require(isinstance(status, str) and status in STATUSES, f'{label}: invalid document status: {status}')
        require(isinstance(kind, str) and kind in TYPE_DIRECTORIES, f'{label}: invalid document type: {kind}')
        require(doc.get('format') == 'numbered', f'{label}: document format must be numbered')
        path_value = doc.get('path')
        path = safe_path(root, path_value)
        require(path is not None and path_value.startswith('docs/') and path.suffix == '.md', f'{label}: invalid document path')
        if path is None or not isinstance(path_value, str):
            continue
        require(path_value not in registered_paths, f'duplicate document path: {path_value}')
        registered_paths[path_value] = doc
        require(bool(re.fullmatch(label[-3:] + r'-[a-z0-9]+(?:-[a-z0-9]+)*\.md', path.name)), f'{label}: filename must use its number and a lowercase slug')
        archive = doc.get('archive')
        if archive is None:
            require(isinstance(kind, str) and kind in TYPE_DIRECTORIES and path_value.startswith(TYPE_DIRECTORIES[kind]), f'{label}: path does not match document type')
        else:
            require(isinstance(archive, dict), f'{label}: archive must be null or an object')
            require(path.is_file() and not path.is_symlink(), f'{label}: archive requires an existing regular document file')
            if isinstance(archive, dict):
                require(is_date(archive.get('date')) and nonempty(archive.get('reason')), f'{label}: archive date/reason required')
                previous = archive.get('previous_path')
                previous_file = safe_path(root, previous)
                require(previous_file is not None and isinstance(kind, str) and kind in TYPE_DIRECTORIES
                        and previous.startswith(TYPE_DIRECTORIES[kind])
                        and bool(re.fullmatch(label[-3:] + r'-[a-z0-9]+(?:-[a-z0-9]+)*\.md', previous_file.name)),
                        f'{label}: invalid archive previous_path')
                if isinstance(kind, str) and kind in TYPE_DIRECTORIES and is_date(archive.get('date')):
                    category = PurePosixPath(TYPE_DIRECTORIES[kind]).parts[1]
                    require(path_value.startswith(f'docs/archive/{archive["date"][:4]}/{category}/'),
                            f'{label}: archive path must match archive year and original category')
            require(path_value.startswith('docs/archive/'), f'{label}: archived document must be in docs/archive/')
            require(status in ('superseded', 'retired', 'cancelled') or (kind == 'plan' and doc.get('execution_status') in ('completed', 'cancelled')),
                    f'{label}: effective specification cannot be archived')
        if status == 'planned':
            require(not path.exists(), f'{label}: planned document must not have a file')
        elif status != 'cancelled':
            require(path.is_file(), f'{label}: document file required for {status}')
        if path.exists():
            require(path.is_file() and not path.is_symlink(), f'{label}: document must be a regular local file')
            if path.is_file():
                text = path.read_text(encoding='utf-8')
                lines = text.splitlines()
                require(bool(lines) and lines[0] == f'# {label} {doc.get("title")}', f'{label}: first heading must exactly match ID/title')
                errors.extend(heading_errors(text, path_value))
        review = doc.get('review')
        if review is not None:
            require(isinstance(review, dict), f'{label}: review must be null or an object')
            if isinstance(review, dict):
                require(is_date(review.get('date')) and all(nonempty(review.get(key)) for key in ('reviewer', 'scope')), f'{label}: review date/reviewer/scope required')
                require(review.get('result') in ('accepted', 'changes_requested'), f'{label}: invalid review result')
                existing_file(review.get('evidence'), f'{label} review')
        if status == 'active':
            require(isinstance(review, dict) and review.get('result') == 'accepted', f'{label}: active document requires accepted review')
        execution = doc.get('execution_status')
        require(execution in ('not_started', 'in_progress', 'completed', 'cancelled') if kind == 'plan' else execution is None,
                f'{label}: invalid execution_status for type')
        closure = doc.get('closure')
        if closure is not None:
            require(isinstance(closure, dict) and is_date(closure.get('date')) and nonempty(closure.get('reason')), f'{label}: invalid closure')
        if status in ('superseded', 'retired', 'cancelled') or (kind == 'plan' and execution in ('completed', 'cancelled')):
            require(isinstance(closure, dict), f'{label}: closure required')
        replacements = doc.get('superseded_by')
        require(strings(replacements) and len(replacements) == len(set(replacements)), f'{label}: superseded_by must be a unique ID array')
        if status == 'superseded':
            require(isinstance(replacements, list) and bool(replacements), f'{label}: superseded document requires a replacement')
    require(nonempty(governance.get('policy_document')) and governance['policy_document'] in by_id, 'governance policy_document must be registered')
    for label, doc in by_id.items():
        replacements = doc.get('superseded_by')
        if not strings(replacements):
            continue
        for replacement in replacements:
            require(replacement in by_id and replacement != label, f'{label}: missing or self replacement: {replacement}')
            if replacement in by_id and replacement != label:
                successor = by_id[replacement]
                require(successor.get('status') in ('active', 'superseded', 'retired'),
                        f'{label}: replacement must be an adopted document: {replacement}')
                successor_path = safe_path(root, successor.get('path'))
                require(successor_path is not None and successor_path.is_file(),
                        f'{label}: replacement document file required: {replacement}')
    visiting, visited = set(), set()

    def visit(label):
        if label in visiting:
            errors.append(f'cyclic document replacement: {label}')
            return
        if label in visited:
            return
        visiting.add(label)
        replacements = by_id[label].get('superseded_by')
        for replacement in replacements if strings(replacements) else []:
            if replacement in by_id:
                visit(replacement)
        visiting.remove(label)
        visited.add(label)

    for label in by_id:
        visit(label)

    reports = governance.get('verification_records')
    report_paths = {}
    if not isinstance(reports, list):
        errors.append('governance verification_records must be an array')
        reports = []
    for report in reports:
        if not isinstance(report, dict):
            errors.append('verification record must be an object')
            continue
        value = report.get('path')
        label = f'verification {value}'
        report_fields = {'path', 'related_documents', 'format', 'status', 'date', 'base_commit', 'working_tree',
                         'change_description', 'environment', 'scope', 'results', 'limitations'}
        require(report_fields <= report.keys(), f'{label}: missing report metadata fields: {sorted(report_fields - report.keys())}')
        path = existing_file(value, label)
        if path is None or not isinstance(value, str):
            continue
        require(value.startswith('docs/verification/') and path.suffix == '.md', f'{label}: invalid report location')
        require(value not in report_paths and value not in registered_paths, f'{label}: duplicate report path')
        report_paths[value] = report
        related = report.get('related_documents')
        require(strings(related) and bool(related) and len(related) == len(set(related)) and all(item in by_id for item in related), f'{label}: related document IDs required')
        require(report.get('format') in ('numbered', 'legacy'), f'{label}: invalid format')
        legacy_report = report.get('format') == 'legacy' and value in LEGACY_REPORTS
        require(report.get('format') != 'legacy' or legacy_report, f'{label}: legacy report format is restricted to original evidence')
        require(report.get('status') in ('draft', 'final'), f'{label}: invalid report status')
        require(is_date(report.get('date')), f'{label}: invalid date')
        commit = report.get('base_commit')
        unknown_bootstrap = legacy_report and value == 'docs/verification/bootstrap.md' and commit is None
        require(unknown_bootstrap or (isinstance(commit, str) and bool(re.fullmatch(r'[0-9a-f]{40}', commit))), f'{label}: tested base commit required')
        working_tree = report.get('working_tree')
        require(working_tree in ('clean', 'dirty', 'historical-unspecified'), f'{label}: invalid working_tree')
        require(working_tree != 'historical-unspecified' or legacy_report, f'{label}: new reports must identify working tree state')
        require(not unknown_bootstrap or working_tree == 'historical-unspecified', f'{label}: unknown historical baseline must remain explicit')
        for key in ('change_description', 'environment', 'scope'):
            require(nonempty(report.get(key)), f'{label}: {key} required')
        require(strings(report.get('limitations')), f'{label}: limitations must be a string array')
        results = report.get('results')
        require(isinstance(results, list) and bool(results), f'{label}: results required')
        for result in results if isinstance(results, list) else []:
            if not isinstance(result, dict):
                errors.append(f'{label}: result must be an object')
                continue
            require(nonempty(result.get('check')), f'{label}: check description required')
            require(result.get('result') in ('passed', 'failed', 'interrupted', 'not_run'), f'{label}: invalid check result')
            existing_file(result.get('evidence'), label)
        if path.is_file():
            report_text = path.read_text(encoding='utf-8')
            errors.extend(heading_errors(report_text, value, numbered=not legacy_report))
            for level, title in HEADINGS.findall(markdown_body(report_text)):
                if len(level) == 1:
                    for identifier in re.findall(r'\bCHNHCM-\d{3}\b', title):
                        require(strings(related) and identifier in related, f'{label}: title ID must be in related_documents: {identifier}')

    markdown = {}
    for directory, directories, files in os.walk(root):
        directories[:] = [name for name in directories if name not in IGNORED_DIRS]
        for filename in files:
            if not filename.lower().endswith('.md'):
                continue
            path = Path(directory) / filename
            relative = path.relative_to(root).as_posix()
            if not path.resolve().is_relative_to(root.resolve()):
                errors.append(f'Markdown path escapes repository: {relative}')
                continue
            markdown[relative] = path.read_text(encoding='utf-8')
            require(path.parent != root or filename == 'AGENTS.md', f'root Markdown must be AGENTS.md: {relative}')
            exempt = relative.startswith(('docs/architecture/target/', 'docs/templates/'))
            if relative not in registered_paths and relative not in report_paths:
                numbered_name = re.match(r'\d{3}-', filename)
                numbered_title = any(len(level) == 1 and re.match(r'CHNHCM-\d{3}\b', title) for level, title in HEADINGS.findall(markdown_body(markdown[relative])))
                managed = any(relative.startswith(prefix) for prefix in TYPE_DIRECTORIES.values()) or relative.startswith(('docs/verification/', 'docs/archive/'))
                require(not numbered_name and not numbered_title and (exempt or not managed or filename == 'README.md'), f'unregistered document or report: {relative}')

    graph = {}
    global_stable = {}
    for value, text in markdown.items():
        body = markdown_body(text)
        anchors = markdown_anchors(text)
        if not value.startswith(('docs/architecture/target/', 'docs/templates/')):
            document = registered_paths.get(value)
            historical = document is not None and (
                document.get('archive') is not None or document.get('status') in ('superseded', 'retired', 'cancelled'))
            local_stable = set()
            for anchor in anchors:
                if STABLE_ANCHOR_CANDIDATE.fullmatch(anchor):
                    require(bool(STABLE_ANCHOR.fullmatch(anchor)), f'invalid stable ID anchor: {anchor} ({value}); use M01-M20/X01-X06 and 001-999')
                    require(anchor not in local_stable, f'duplicate stable ID anchor in document: {anchor} ({value})')
                    local_stable.add(anchor)
                    if not historical:
                        require(anchor not in global_stable, f'duplicate stable ID anchor: {anchor} ({value})')
                        global_stable[anchor] = value
            definitions = link_definitions(text)
            for _, title in HEADINGS.findall(body):
                prose = markdown_prose(title)
                unlinked = list(prose)
                for start, end, label, target in link_occurrences(prose, definitions):
                    url = urlsplit(target)
                    for identifier in STABLE_ID_MENTION.findall(label):
                        require(bool(STABLE_ID.fullmatch(identifier)), f'{value}: invalid stable ID: {identifier}')
                        require(not (url.scheme or url.netloc) and unquote(url.fragment) == identifier.lower(),
                                f'{value}: stable ID reference must target its own local anchor: {identifier} -> {target}')
                    unlinked[start:end] = ' ' * (end - start)
                for identifier in STABLE_ID_MENTION.findall(''.join(unlinked)):
                    require(bool(STABLE_ID.fullmatch(identifier)), f'{value}: invalid stable ID: {identifier}')
                    require(identifier.lower() in anchors, f'{value}: stable ID heading requires explicit anchor: {identifier}')
        graph[value] = []
        for target in markdown_links(text):
            url = urlsplit(target)
            if url.scheme or url.netloc:
                continue
            resolved = ((root / value).parent / unquote(url.path)).resolve() if url.path else (root / value).resolve()
            if not resolved.is_relative_to(root.resolve()):
                errors.append(f'{value}: link escapes repository: {target}')
                continue
            relative = resolved.relative_to(root.resolve()).as_posix()
            require(resolved.exists() or historical_source_exists(root, report_paths.get(value), relative),
                    f'{value}: broken local link: {target}')
            if relative in markdown:
                graph[value].append(relative)
                fragment = unquote(url.fragment)
                if STABLE_ANCHOR_CANDIDATE.fullmatch(fragment):
                    require(bool(STABLE_ANCHOR.fullmatch(fragment)), f'{value}: invalid stable ID target: {target}')
                if fragment.startswith(('term-', 'source-')) or STABLE_ANCHOR_CANDIDATE.fullmatch(fragment):
                    require(fragment in markdown_anchors(markdown[relative]), f'{value}: missing explicit target anchor: {target}')
    reachable = set()
    pending = ['docs/index.md']
    while pending:
        value = pending.pop()
        if value not in reachable:
            reachable.add(value)
            pending.extend(graph.get(value, []))
    for value, doc in registered_paths.items():
        if (root / value).is_file():
            require(value in reachable, f'document not reachable from docs/index.md: {value}')
    for value in report_paths:
        require(value in reachable, f'report not reachable from docs/index.md: {value}')
    return errors
