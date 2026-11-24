#!/usr/bin/env python3
"""M2 full-input attribution, or the complementary single-session read groups."""
import argparse
import importlib.util
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('capacity125', ROOT/'tests/capacity/remediation125/measure.py')
lane125 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lane125)


def run(private, output, binary, single_formal):
    assert not output.exists(), 'retain the first complete denominator'
    for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
        os.environ.pop(key, None)
    lane = lane125.Lane(private, output, binary)
    try:
        clients = lane.login()
        lane.result['purpose'] = ('M2 supplementary single-session capacity; 125 owns the original twenty-session groups'
                                  if single_formal else 'M2 diagnostic; not formal capacity acceptance')
        lane.result['diagnosticOnly'] = not single_formal
        lane.result['sessions'] = 1 if single_formal else 20
        lane.result['sessionCounts'] = [1] if single_formal else [1, 20]
        lane.result['expectedTimedRequests'] = 1600 if single_formal else 520
        modes = [(1, 200)] if single_formal else [(1, 5), (20, 3)]
        for code in ['R6_DEPTH_099', 'R6_FAN_ROOT', 'R6_LONG_HISTORY', 'R6_D_00000']:
            for source in [None, 1]:
                path = lane.oracle.path(code, source)
                check = lane.detail_check(code, source)
                lane.result['preparation'].append(lane125.measured(clients[0], 'GET', path, check=check)[0])
                for sessions, rounds in modes:
                    samples = []
                    for _ in range(rounds):
                        samples += lane125.parallel([
                            lambda c=c: lane125.measured(c, 'GET', path, check=check)[0]
                            for c in clients[:sessions]])
                    lane.group(f'{code}-source{source}-sessions{sessions}', samples, 500 if source else 300)
        lane.result['actualTimedRequests'] = sum(g['requests'] for g in lane.result['groups'].values())
        lane.result['completed'] = lane.result['actualTimedRequests'] == lane.result['expectedTimedRequests']
    finally:
        lane.close()
    if single_formal and not lane.result['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--private', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--single-formal', action='store_true')
    args = parser.parse_args()
    run(args.private.resolve(), args.output.resolve(), args.binary.resolve(), args.single_formal)
