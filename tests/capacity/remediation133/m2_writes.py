#!/usr/bin/env python3
"""Explicit write-only partitions of the unchanged 125 cases and targets.

Warm: 1200 requests. Cold: 600 requests and 30 owned volume evictions.
The separate mixed phase retains all original 1200 mixed read/write requests.
This never counts omitted groups as passed or replaces their first evidence.
"""
import argparse
import json
import os
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'tests/capacity/remediation125'))
from measure import Lane


def run(args):
    assert not args.output.exists()
    for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
        os.environ.pop(key, None)
    lane = Lane(args.private.resolve(), args.output, args.binary.resolve())
    try:
        lane.matrix(args.phase, actions=['save', 'pair'], modes=['BU', 'Department', 'mixed'])
        lane.result['completed'] = (lane.result['allScheduledGroupsAttempted'] and
            lane.result['actualTimedRequests'] == lane.result['expectedTimedRequests'] and
            not lane.result['unattemptedRequests'])
    finally:
        lane.close()
    print(json.dumps({k:lane.result[k] for k in ['completed', 'passed', 'expectedTimedRequests', 'actualTimedRequests', 'partition']}))
    if not lane.result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('private', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--phase', choices=['warm', 'cold'], required=True)
    run(parser.parse_args())
