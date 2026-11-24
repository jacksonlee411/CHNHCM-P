"""Approved warm large-subtree targets; cold and original limits stay separate."""
import math

POLICY_ID = 'department-large-subtree-warm-400-2000ms-20261116'
CONTRACT = 'docs/specs/m02/120-dated-maintenance-technical-contract.md#department-large-subtree-stage-target'
CASES = ('scope-large', 'scope-large-last', 'scope-path')


def limits(name, sessions, *, phase):
    if name not in CASES:
        raise KeyError(name)
    if phase not in ('warm', 'cold', 'mixed'):
        raise KeyError(phase)
    original = {1: 300, 20: 1000}[sessions]
    previous = {1: 300, 20: 1500}[sessions]
    target = {1: 400, 20: 2000}[sessions] if phase == 'warm' and name in ('scope-path', 'scope-large-last') else previous
    result = dict(targetPolicy=POLICY_ID, phase=phase, stageTargetMS=target, previousStageTargetMS=previous,
                  originalTargetMS=original, longTermTargetMS=300 if name == 'scope-path' else original)
    return result


def assess(group, *, phase):
    result = limits(group['name'], group['sessions'], phase=phase)
    samples = group['samples']
    successful = sorted(s['ms'] for s in samples if s.get('requestSent', True) and s.get('status') == 200 and 'error' not in s)
    p95 = successful[math.ceil(len(successful) * .95)-1] if successful else None
    complete = (len(samples) == group['attempts'] == group['executedRequests'] == 200 and len(successful) == 200
                and all(math.isfinite(v) and v >= 0 for v in successful))
    result.update(capacityComplete=complete, successP95MS=p95,
                  passed=complete and p95 <= result['stageTargetMS'],
                  previousStagePassed=complete and p95 <= result['previousStageTargetMS'],
                  originalPassed=complete and p95 <= result['originalTargetMS'],
                  longTermPassed=complete and p95 <= result['longTermTargetMS'])
    return result
