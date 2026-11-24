#!/usr/bin/env python3
"""Group commuting leaf commands without crossing an ancestor's history change."""
import argparse
import collections
import fcntl
import hashlib
import json
from pathlib import Path
import sqlite3

from runtime import ROOT


def plan(private, out, apply=False):
    private = private.resolve()
    assert ROOT not in private.parents and private.stat().st_mode & 0o777 == 0o700
    lock = (private / 'history-run.lock').open('a')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    db = sqlite3.connect('file:' + str(private / 'import.sqlite') + '?mode=ro', uri=True)
    assert db.execute('SELECT count(*) FROM history_batch').fetchone()[0] == 0
    rows = {(phase, no): (code, json.loads(raw), result is not None)
            for phase, no, code, raw, result in db.execute('SELECT phase,row_no,code,payload,result FROM rows')}
    assert len(rows) == 132302
    assert all(done for (phase, _), (_, _, done) in rows.items() if phase == 'create')
    path = private / 'history-order.json'; original = path.read_bytes()
    order = [tuple(key) for key in json.loads(original)]
    assert len(order) == 44958 and len(set(order)) == len(order)
    parents = collections.defaultdict(set)
    for (phase, _), (code, row, _) in rows.items():
        parent = row.get('parent_org_code') if phase == 'create' else row.get('new_parent_org_code') if phase == 'parent-change' else None
        if parent:
            parents[code].add(parent)
    nonleaf = set().union(*parents.values())
    cache = {}

    def ancestors(code):
        if code not in cache:
            seen = set(); todo = list(parents[code])
            while todo:
                item = todo.pop()
                if item in seen:
                    continue
                seen.add(item); todo.extend(parents[item] - seen)
            cache[code] = tuple(sorted(seen))
        return cache[code]

    prefix = next((i for i, key in enumerate(order) if not rows[key][2]), len(order))
    assert all(rows[k][2] for k in order[:prefix]) and all(not rows[k][2] for k in order[prefix:])
    last = {}; barriers = []; groups = collections.defaultdict(list)
    for key in order[prefix:]:
        code = rows[key][0]
        if code in nonleaf:
            barriers.append(key); last[code] = len(barriers)
        else:
            # Move a leaf only into the interval after its most recent ancestor
            # command. Every crossed command is on an unrelated object.
            slot = max((last.get(a, 0) for a in ancestors(code)), default=0)
            groups[slot].append(key)

    def interleave(group):
        per_code = collections.defaultdict(collections.deque)
        for key in group:
            per_code[rows[key][0]].append(key)
        ready = collections.deque(per_code)
        while ready:
            code = ready.popleft()
            yield per_code[code].popleft()
            if per_code[code]:
                ready.append(code)

    candidate = order[:prefix] + list(interleave(groups[0]))
    for index, key in enumerate(barriers, 1):
        candidate.append(key); candidate.extend(interleave(groups[index]))
    assert candidate[:prefix] == order[:prefix]
    assert len(candidate) == len(order) and set(candidate) == set(order)

    def dependency_signature(sequence):
        last = {}; signatures = {}; per_code = collections.defaultdict(list)
        for key in sequence:
            code = rows[key][0]
            signatures[key] = tuple((a, last.get(a)) for a in ancestors(code))
            last[code] = key; per_code[code].append(key)
        return signatures, dict(per_code)

    # Independent full-sequence comparison: every original row sees exactly the
    # same latest operation for every possible ancestor, and each object's own
    # sequence is identical. Thus each descendant also stays in the same interval
    # between its ancestor's operations. No date, input or request key is changed.
    before, before_objects = dependency_signature(order)
    after, after_objects = dependency_signature(candidate)
    assert before == after and before_objects == after_objects
    assert [k for k in candidate if rows[k][0] in nonleaf] == [k for k in order if rows[k][0] in nonleaf]
    encoded = json.dumps(candidate).encode()
    result = dict(applied=False, completedPrefix=prefix, remaining=len(order)-prefix,
                  nonleafBarriers=len(barriers), independentLeafRows=sum(map(len, groups.values())),
                  nonemptyLeafGroups=sum(bool(g) for g in groups.values()),
                  largestLeafGroup=max(map(len, groups.values()), default=0),
                  maxGlobalAncestors=max(map(len, cache.values())),
                  reorderedPositions=sum(a != b for a, b in zip(order, candidate)),
                  unchangedCompletedPrefix=True, unchangedPerObjectOrder=True,
                  sameAncestorOperationAtEveryRow=True, unchangedNonleafOrder=True,
                  beforeOrderSHA256=hashlib.sha256(original).hexdigest(),
                  afterOrderSHA256=hashlib.sha256(encoded).hexdigest())
    if apply:
        backup = private / 'history-order-before-independent-leaves.json'
        with backup.open('xb') as f:
            backup.chmod(0o600); f.write(original)
        replacement = private / 'history-order-next.json'
        with replacement.open('xb') as f:
            replacement.chmod(0o600); f.write(encoded)
        replacement.replace(path)
        result['applied'] = True
    out.write_text(json.dumps(result, indent=2) + '\n'); print(json.dumps(result))
    db.close(); lock.close()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path); p.add_argument('out', type=Path)
    p.add_argument('--apply', action='store_true')
    a = p.parse_args(); plan(a.private, a.out, a.apply)
