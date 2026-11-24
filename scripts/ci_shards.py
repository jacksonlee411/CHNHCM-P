"""Allocate sequential jobs by their measured business time on hosted runners."""
import json
import re
from pathlib import Path

FULL_BROWSER = 16
FULL_INTEGRATION = 2
TIMINGS = '.github/ci-shard-seconds.json'
SECONDS = json.loads((Path(__file__).resolve().parents[1] / TIMINGS).read_text())


def browser_seconds(entry):
    # New files participate immediately. Recalibrate the estimate after a full run.
    return SECONDS['browser'].get(entry, 120)


def selected_browser_count(files):
    work = [browser_seconds(f) for f in files if f != 'enterprise304.spec.ts']
    if not work:
        return int(bool(files))
    # Leave about four minutes per runner for setup and the enterprise dependency.
    return min(len(work), FULL_BROWSER, max(1, round(sum(work) / 630)))


def selected_integration_count(pattern):
    estimate = sum(seconds for name, seconds in SECONDS['integration'].items() if re.search(pattern, name))
    return min(FULL_INTEGRATION, max(1, round(estimate / 840)))


def balanced_groups(items, total, seconds):
    if not 1 <= total <= len(items) or len(set(items)) != len(items):
        raise ValueError('shards require enough distinct work items')
    groups = [[] for _ in range(total)]
    loads = [0] * total
    for item in sorted(items, key=lambda item: (-seconds(item), item)):
        index = min(range(total), key=loads.__getitem__)
        groups[index].append(item)
        loads[index] += max(1, seconds(item))
    return [sorted(group) for group in groups]
