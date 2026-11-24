#!/usr/bin/env python3
"""Apply the approved broad-write stage target to unchanged formal samples."""
import argparse
import hashlib
import json
import math
from pathlib import Path

POLICY = 'department-broad-write-stage-2000ms-20261115'
CONTRACT = 'docs/specs/m02/120-dated-maintenance-technical-contract.md#department-broad-write-stage-target'


def assess(report):
    # This decision covers the measured one-session representative Vanke case,
    # not ordinary saves, other dependency shapes, or twenty-session traffic.
    assert report['completed'] and not report['diagnostic']
    assert report['apiConnectionLimit'] == 12
    assert report['totalDepartments'] == 87343
    assert report['datedDescendantsIncludingTarget'] == 9799
    samples = report['attempts']
    assert len(samples) == 200 and [s['index'] for s in samples] == list(range(200))
    assert all(s['phase'] == ('change' if i % 2 == 0 else 'restore') for i, s in enumerate(samples))
    successful = sorted(s['ms'] for s in samples if s['passed'] and s['status'] == 200)
    failures = len(samples) - len(successful)
    p95 = successful[math.ceil(len(successful) * .95) - 1] if successful else None
    complete = report['passed'] and failures == 0 and len(report['readbacks']) == 200
    return dict(policy=POLICY, contract=CONTRACT, sessions=1, attempts=200,
                failures=failures, successP95MS=p95, maximumMS=max(s['ms'] for s in samples),
                stageTargetMS=2000, stagePassed=complete and p95 is not None and p95 <= 2000,
                longTermTargetMS=500, longTermPassed=complete and p95 is not None and p95 <= 500,
                runtimePreparationMS=report.get('runtimeBudgets', {}).get('departmentPreparationMS', 2000),
                runtimeLockAcquisitionMS=report.get('runtimeBudgets', {}).get('cumulativeLockAcquisitionMS', 2000),
                runtimeGuardChanged=report.get('runtimeBudgets', {}).get('departmentPreparationMS', 2000) != 2000,
                additionalHTTPRequests=0,
                scope='representative broad parent change/restore, 87343 departments and 9799 dated descendants; no other save group is relaxed')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('input', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    raw = args.input.read_bytes()
    result = assess(json.loads(raw))
    result['originalSamplesSHA256'] = hashlib.sha256(raw).hexdigest()
    with args.output.open('x') as stream:
        stream.write(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result), flush=True)
