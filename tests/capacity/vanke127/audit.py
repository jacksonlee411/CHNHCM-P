#!/usr/bin/env python3
"""Independent full dated-graph oracle; emits aggregate evidence only."""
import argparse
import collections
import hashlib
import json
from pathlib import Path
import sqlite3
import time


def run(private, destination):
    db = sqlite3.connect(private / 'import.sqlite')
    initial, versions, events = {}, {}, collections.defaultdict(dict)
    counts, unsupported, future = collections.Counter(), collections.Counter(), collections.Counter()
    missing = 0
    raw = db.execute('SELECT phase,row_no,payload FROM rows ORDER BY phase,row_no').fetchall()
    for phase, no, payload in raw:
        row = json.loads(payload)
        counts[phase] += 1
        code = row['org_code']
        if phase == 'create':
            assert code not in initial
            initial[code] = row.get('parent_org_code', '')
            versions[code] = {row['effective_date']}
        elif phase == 'version':
            assert code in initial
            assert row['effective_date'] not in versions[code]
            versions[code].add(row['effective_date'])
            unsupported['extensionOnlyVersions'] += not bool(row.get('core.name'))
            unsupported['versionsWithExtension'] += bool(row.get('ext.d_org_type'))
        day = row.get('target_effective_date') or row['effective_date']
        future[phase] += day > '2026-11-07'
    for phase, no, payload in raw:
        if phase != 'parent-change':
            continue
        row = json.loads(payload)
        code = row['org_code']
        assert code in initial and row['new_parent_org_code'] in initial
        day = row.get('target_effective_date') or row['effective_date']
        if row['operation'] == 'correct_parent':
            missing += day not in versions[code]
        else:
            assert day not in versions[code], 'parent addition conflicts with an existing version date'
        assert code not in events[day], 'duplicate parent operation at a source date'
        events[day][code] = row['new_parent_org_code']
    parents = initial.copy()
    maximum, examined, start = 0, 0, time.perf_counter()
    for day, changes in [('1984-01-01', {})] + sorted(events.items()):
        parents.update(changes)
        depths, visiting = {}, set()
        def depth(code):
            if code in depths:
                return depths[code]
            assert code not in visiting, 'cycle in dated source hierarchy'
            visiting.add(code)
            parent = parents[code]
            assert not parent or parent in parents
            value = depth(parent) + 1 if parent else 0
            visiting.remove(code)
            depths[code] = value
            return value
        for code in parents:
            depth(code)
        maximum = max(maximum, max(depths.values()))
        examined += len(parents)
    assert missing == 0
    result = dict(sourceRows=sum(counts.values()), counts=dict(counts), correctionTargetsMissing=missing, sourceDateStatesChecked=len(events)+1, nodeDateStatesChecked=examined, maxHistoricalDepthIncludingBU=maximum, futureRowsIncluded=dict(future), unmappedFields=dict(unsupported), graphCycles=0, missingParents=0, elapsedSeconds=round(time.perf_counter()-start,2), scope='independent complete business-date source hierarchy, including future records; not product import success')
    destination.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('destination', type=Path)
    a = p.parse_args()
    run(a.private, a.destination)
