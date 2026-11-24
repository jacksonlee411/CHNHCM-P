#!/usr/bin/env python3
"""Inspect external B&B workbooks; write names and paths only outside the repo."""
import argparse
import collections
import hashlib
import json
from pathlib import Path
import xml.etree.ElementTree as ET
import zipfile

NS = {'s': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}


def read_rows(path):
    with zipfile.ZipFile(path) as archive:
        strings = []
        if 'xl/sharedStrings.xml' in archive.namelist():
            strings = [''.join(v.itertext()) for v in ET.fromstring(
                archive.read('xl/sharedStrings.xml')).findall('s:si', NS)]
        rows = ET.fromstring(archive.read('xl/worksheets/sheet1.xml')).findall('s:sheetData/s:row', NS)
        def values(row):
            result = {}
            for cell in row:
                column = ''.join(c for c in cell.attrib['r'] if c.isalpha())
                value = cell.find('s:v', NS)
                if cell.attrib.get('t') == 'inlineStr':
                    result[column] = ''.join(cell.find('s:is', NS).itertext())
                elif cell.attrib.get('t') == 's':
                    result[column] = strings[int(value.text)]
                else:
                    result[column] = value.text if value is not None else ''
            return result
        headers = values(rows[1])
        return [{headers[k]: v for k, v in values(row).items() if k in headers} for row in rows[2:]]


def inspect(source, private, report, as_of):
    repo = Path(__file__).resolve().parents[3]
    assert private.is_absolute() and repo not in private.parents
    assert private.is_dir() and private.stat().st_mode & 0o777 == 0o700
    files = {key: source / ('orgunit-import-' + key + '-from-source.xlsx')
             for key in ['create', 'version', 'parent-change']}
    rows = {key: read_rows(path) for key, path in files.items()}
    nodes, names, parents = {}, {}, {}
    for row in rows['create']:
        code = row['org_code']
        assert code not in nodes, 'duplicate source identity'
        assert not row.get('manager_pernr'), 'personnel data requires a separate scope'
        nodes[code] = {'Code': code, 'BusinessUnit': row.get('is_business_unit') == 'true'}
        names[code] = {row['effective_date']: row['name']}
        parents[code] = {row['effective_date']: row.get('parent_org_code', '')}
    unsupported_only = 0
    for row in rows['version']:
        code = row['org_code']
        assert code in nodes and not row.get('core.manager_pernr')
        assert not row.get('core.status') and not row.get('core.is_business_unit')
        if row.get('core.name'):
            names[code][row['effective_date']] = row['core.name']
        else:
            unsupported_only += 1
    # Corrections target any existing version, including name-only versions.
    # Checking only earlier parent events falsely calls all 6476 targets absent.
    version_dates = {r['org_code']: {r['effective_date']} for r in rows['create']}
    for r in rows['version']:
        version_dates[r['org_code']].add(r['effective_date'])
    missing_correction_targets = 0
    for row in rows['parent-change']:
        code = row['org_code']
        assert code in nodes and row['new_parent_org_code'] in nodes
        day = row['target_effective_date'] if row['operation'] == 'correct_parent' else row['effective_date']
        if row['operation'] == 'correct_parent' and day not in version_dates[code]:
            missing_correction_targets += 1
        parents[code][day] = row['new_parent_org_code']
    for code, node in nodes.items():
        node['Name'] = names[code][max(day for day in names[code] if day <= as_of)]
        node['Parent'] = parents[code][max(day for day in parents[code] if day <= as_of)]
    depths, visiting = {}, set()
    def depth(code):
        if code in depths:
            return depths[code]
        assert code not in visiting, 'cycle in source final hierarchy'
        visiting.add(code)
        parent = nodes[code]['Parent']
        assert not parent or parent in nodes, 'missing source parent'
        value = depth(parent) + 1 if parent else 0
        visiting.remove(code)
        depths[code] = value
        return value
    for code in nodes:
        depth(code)
    roots = [code for code, node in nodes.items() if not node['Parent']]
    business_units = [code for code, node in nodes.items() if node['BusinessUnit']]
    assert len(roots) == 1 and roots == business_units
    ordered = sorted(nodes.values(), key=lambda node: (depths[node['Code']], node['Code']))
    private_data = {'AsOf': as_of, 'Nodes': ordered}
    raw = json.dumps(private_data, ensure_ascii=False, separators=(',', ':')).encode()
    path = private / 'source-snapshot.json'
    path.write_bytes(raw)
    path.chmod(0o600)
    children = collections.Counter(node['Parent'] for node in ordered if node['Parent'])
    summary = {
        'asOf': as_of, 'organizationCount': len(nodes), 'businessUnits': len(business_units),
        'departments': len(nodes) - len(business_units), 'maxDepthIncludingBURoot': max(depths.values()),
        'maxDirectChildren': max(children.values()), 'rootDirectChildren': children[roots[0]],
        'depthCounts': dict(sorted(collections.Counter(depths.values()).items())),
        'workbooks': [{'file': path.name, 'sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
                       'rows': len(rows[key])} for key, path in files.items()],
        'sourceSnapshotSHA256': hashlib.sha256(raw).hexdigest(),
        'parentOperationCounts': dict(collections.Counter(row['operation'] for row in rows['parent-change'])),
        'correctionTargetsMissingSourceVersionDate': missing_correction_targets,
        'versionRowsWithoutNameChange': unsupported_only,
        'scope': 'full final organization snapshot from all three workbooks; names and parent histories evaluated at asOf; not a historical-event migration',
        'excludedFields': ['ext.d_org_type', 'reason', 'empty manager fields'],
        'privateDataWrittenOutsideRepository': True,
    }
    report.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('source', type=Path)
    parser.add_argument('private', type=Path)
    parser.add_argument('report', type=Path)
    parser.add_argument('--as-of', default='2026-11-07')
    args = parser.parse_args()
    inspect(args.source, args.private, args.report, args.as_of)
