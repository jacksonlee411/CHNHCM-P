#!/usr/bin/env python3
"""Measure actual hosted and self-hosted minutes for the runs in a delivery ledger."""
import argparse
from collections import Counter
from datetime import datetime
import json
import math
from pathlib import Path
import subprocess
import sys

from ci_policy import GitHub, WORKFLOW


def instant(value):
    return datetime.fromisoformat(value.replace('Z', '+00:00'))


def job_usage(job):
    if job['status'] != 'completed':
        raise ValueError(f"job {job['id']} is incomplete")
    if not job.get('runner_id'):
        if job['conclusion'] not in ('skipped', 'cancelled'):
            raise ValueError(f"job {job['id']} is missing allocation evidence")
        return 'unallocated', 0
    seconds = (instant(job['completed_at']) - instant(job['started_at'])).total_seconds()
    if seconds < 0:
        raise ValueError(f"job {job['id']} has inconsistent timestamps")
    if 'self-hosted' in {label.lower() for label in job['labels']}:
        group = 'self_hosted'
    elif job.get('runner_group_name') == 'GitHub Actions':
        group = 'github_hosted'
    else:
        group = 'unclassified'
    return group, math.ceil(seconds / 60)


def read_placement_plan(api, job, rid, run, pr):
    if not job.get('runner_id'):
        return None
    try:
        log = api.raw(f"actions/jobs/{job['id']}/logs").decode('utf-8-sig', 'replace')
    except subprocess.CalledProcessError:
        return None  # An expired or unavailable log cannot establish the rollout stage.
    # The initial selection also exists when quality fails before scheduling.
    # Prefer the final placement after successful prior shards were subtracted.
    for line in reversed(log.splitlines()):
        payload = line.partition(' ')[2].strip()
        if not payload.startswith('{"schema"'):
            continue
        try:
            plan = json.loads(payload)
        except ValueError:
            continue
        if (plan.get('repo'), plan.get('run_id'), plan.get('head'), plan.get('event')) != (
                api.repo, str(rid), run['head_sha'], run['event']):
            continue
        if run['event'] == 'pull_request' and plan.get('pr') != pr:
            continue
        counts = plan.get('self_hosted_shards')
        suites = ('browser', 'integration')
        if counts is not None and (not isinstance(counts, dict) or set(counts) != set(suites) or any(
                type(counts[s]) is not int or not 0 <= counts[s] <= plan[s + '_shards'] for s in suites)):
            raise ValueError(f"invalid runner counts in quality job {job['id']}")
        return dict(scope=plan['scope'], self_hosted_shards=counts)
    return None


def attempt_placements(api, rid, run, pr, jobs):
    minutes, quality_jobs = {}, {}
    for job in jobs:
        attempt = job['run_attempt']
        group, cost = job_usage(job)
        minutes.setdefault(attempt, Counter())[group] += cost
        if job['name'] in ('quality', 'diagnostic quality'):
            if attempt in quality_jobs:
                raise ValueError(f'run {rid} attempt {attempt} has ambiguous quality jobs')
            quality_jobs[attempt] = job
    records = []
    source, plan = {}, None
    for attempt, amounts in sorted(minutes.items()):
        # Partial reruns retain the last quality outputs. A new quality job replaces them.
        if attempt in quality_jobs:
            source = quality_jobs[attempt]
            plan = read_placement_plan(api, source, rid, run, pr)
        records.append(dict(attempt=attempt, quality_job=source.get('id'), plan_attempt=source.get('run_attempt'),
                            plan=plan, minutes=dict(amounts)))
    return records


def matrix_starts(jobs):
    # Count a launch once per attempt when GitHub expanded all 19 named slots
    # and actually assigned a runner to at least one of them. A failed-only
    # retry can contain shard 2/11 without ever requesting the full matrix.
    expected = {f'{suite} shard {i}/{size}' for suite, size in [('browser', 11), ('integration', 8)]
                for i in range(1, size + 1)}
    result = []
    for attempt in sorted({job['run_attempt'] for job in jobs}):
        slots = [job for job in jobs if job['run_attempt'] == attempt and job['name'] in expected]
        starts = [instant(job['started_at']) for job in slots if job.get('runner_id')]
        names = {job['name'] for job in slots}
        if len(names) != len(slots):
            raise ValueError('duplicated matrix slot within the same attempt')
        result.append(dict(attempt=attempt, expanded_slots=len(names), allocated_slots=len(starts),
                           first_allocated_at=min(starts).isoformat() if starts else None,
                           full_start=names == expected and bool(starts)))
    return result


def measure(api, delivery, rollout=False):
    minutes, counts, records = Counter(), Counter(), []
    run_ids = delivery['run_ids']
    endpoints = {delivery['final_run'], delivery['main_run']}
    if len(endpoints) != 2 or not endpoints <= set(run_ids) or len(run_ids) != len(set(run_ids)):
        raise ValueError('missing or duplicated delivery run IDs')
    for rid in run_ids:
        run = api.json(f'actions/runs/{rid}')
        if run['path'] != WORKFLOW or run['status'] != 'completed':
            raise ValueError(f'run {rid} is outside the workflow or incomplete')
        if rid == delivery['final_run'] and (run['event'], run['head_sha'], run['conclusion']) != (
                'pull_request', delivery['head'], 'success'):
            raise ValueError(f'final PR run {rid} does not match the delivery')
        if rid == delivery['main_run'] and (run['event'], run['head_branch'], run['head_sha'], run['conclusion']) != (
                'push', 'main', delivery['merge_sha'], 'success'):
            raise ValueError(f'main run {rid} does not match the delivery')
        jobs = api.pages(f'actions/runs/{rid}/jobs?filter=all', 'jobs')
        if len({j['id'] for j in jobs}) != len(jobs) or set(j['run_attempt'] for j in jobs) != set(range(1, run['run_attempt'] + 1)):
            raise ValueError(f'run {rid} has duplicated jobs or missing attempts')
        amounts, job_counts, placements = Counter(), Counter(), []
        for job in jobs:
            group, cost = job_usage(job)
            amounts[group] += cost
            job_counts[group] += 1
            if group in ('self_hosted', 'unclassified'):
                placements.append(dict(job=job['id'], name=job['name'], attempt=job['run_attempt'],
                                       runner_id=job['runner_id'], group=group, conclusion=job['conclusion'],
                                       started_at=job['started_at'], completed_at=job['completed_at'], minutes=cost))
        started = instant(run['created_at'])
        ended = max((instant(j['completed_at']) for j in jobs if j.get('runner_id')), default=instant(run['updated_at']))
        minutes.update(amounts)
        counts.update(job_counts)
        record = dict(run=rid, attempts=run['run_attempt'], event=run['event'], conclusion=run['conclusion'],
                      started_at=started.isoformat(), ended_at=ended.isoformat(), wait_seconds=int((ended - started).total_seconds()),
                      job_counts=dict(job_counts), minutes=dict(amounts))
        if placements:
            record['runner_placements'] = placements
        if run['event'] == 'pull_request':
            record['pr_matrix_attempts'] = matrix_starts(jobs)
        if rollout:
            record['attempt_placement'] = attempt_placements(api, rid, run, delivery['pr'], jobs)
        records.append(record)
    first = min(instant(r['started_at']) for r in records)
    last = max(instant(r['ended_at']) for r in records)
    final_start = next(instant(r['started_at']) for r in records if r['run'] == delivery['final_run'])
    launches = [instant(a['first_allocated_at']) for r in records for a in r.get('pr_matrix_attempts', [])
                if a['full_start']]
    return dict(pr=delivery['pr'], minutes=dict(minutes), job_counts=dict(counts),
                full_pr_matrix_starts=len(launches),
                before_final_pr_run=sum(t < final_start for t in launches),
                delivery_elapsed_seconds=int((last - first).total_seconds()), runs=records)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', required=True, help='owner/repository')
    parser.add_argument('--deliveries', required=True, type=Path, help='136 window-deliveries.json')
    parser.add_argument('--pr', nargs='+', type=int, required=True, help='comparable delivery PRs to include')
    parser.add_argument('--rollout', action='store_true', help='read each quality plan and report costs by attempt')
    args = parser.parse_args()
    rows = [row for row in json.loads(args.deliveries.read_text()) if row['pr'] in args.pr]
    ids = [rid for row in rows for rid in row['run_ids']]
    if sorted(row['pr'] for row in rows) != sorted(args.pr) or len(ids) != len(set(ids)):
        parser.error('missing or duplicated delivery membership')
    api = GitHub(args.repo)
    results = [measure(api, row, args.rollout) for row in rows]
    total, counts = Counter(), Counter()
    for row in results:
        total.update(row['minutes'])
        counts.update(row['job_counts'])
    print(json.dumps(dict(repository=args.repo, deliveries=results, minutes=total, job_counts=counts,
                          hosted_mean=total['github_hosted'] / len(rows) if not counts['unclassified'] else None),
                     ensure_ascii=False, indent=2))
    return 1 if counts['unclassified'] else 0


if __name__ == '__main__':
    sys.exit(main())
