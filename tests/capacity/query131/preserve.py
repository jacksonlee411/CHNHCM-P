#!/usr/bin/env python3
"""Compare every retained original source after real growth, without loading
customer identities into public output. VisitHistory ordering is stable by
object, chunk and source; additions may appear between original objects.
"""
import argparse
import gzip
import hashlib
import json
from pathlib import Path


def checked_rows(stream):
    # VisitHistory emits one instance, BU before department, ordered by object,
    # chunk and source. Keep identities only for the current object, so this
    # does not accumulate a set of all million original source identities.
    order = {'BU': 0, 'department': 1}
    previous_owner = None
    previous_number = None
    sources = set()
    for raw in stream:
        row = json.loads(raw)
        assert isinstance(row, list) and len(row) == 5, 'invalid source digest row'
        assert row[0] in order and isinstance(row[1], str) and isinstance(row[2], str), 'invalid source identity'
        assert type(row[3]) is int, 'invalid source number'
        owner = (order[row[0]], row[1])
        assert previous_owner is None or owner >= previous_owner, 'source objects reordered'
        if owner != previous_owner:
            sources.clear()
            previous_number = None
        assert previous_number is None or row[3] > previous_number, 'source number duplicated or reordered'
        assert row[2] not in sources, 'source identity duplicated'
        sources.add(row[2])
        previous_owner, previous_number = owner, row[3]
        yield raw, row


def compare(original, current):
    original = checked_rows(original)
    previous = next(original, None)
    retained = added = 0
    digest = hashlib.sha256()
    for raw, row in checked_rows(current):
        if previous is not None and row[:4] == previous[1][:4]:
            assert row == previous[1], 'original source content changed'
            digest.update(previous[0])
            retained += 1
            previous = next(original, None)
        else:
            added += 1
    assert previous is None, 'original source disappeared or order changed'
    return dict(originalSourcesPreserved=retained, addedSources=added,
                originalUncompressedSHA256=digest.hexdigest(), passed=True)


def run(args):
    assert not args.output.exists(), 'retain the first comparison'
    root = Path(__file__).resolve().parents[3]
    for path in [args.original, args.current]:
        assert path.is_absolute() and root not in path.parents
    with gzip.open(args.original, 'rb') as original, gzip.open(args.current, 'rb') as current:
        result = compare(iter(original), iter(current))
    assert result['originalSourcesPreserved'] == args.original_count
    assert result['addedSources'] == args.added_count
    result['driverSHA256'] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    args.output.write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps(result))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('original', type=Path)
    parser.add_argument('current', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--original-count', type=int, required=True)
    parser.add_argument('--added-count', type=int, required=True)
    run(parser.parse_args())
