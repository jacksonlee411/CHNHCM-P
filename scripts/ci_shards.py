"""Allocate sequential jobs by their measured business time on hosted runners."""
import json
import math
import re
from pathlib import Path

FULL_BROWSER = 16
FULL_INTEGRATION = 4
TIMINGS = '.github/ci-shard-seconds.json'
SECONDS = json.loads((Path(__file__).resolve().parents[1] / TIMINGS).read_text())


def browser_seconds(entry):
    # New files participate immediately. Recalibrate the estimate after a full run.
    return SECONDS['browser'].get(entry, 120)


def selected_browser_count(files):
    entries = [f for f in files if f != 'enterprise304.spec.ts']
    if not entries:
        return int(bool(files))
    units = sum(2 if f in SECONDS['browser_halves'] else 1 for f in entries)
    return min(units, FULL_BROWSER, math.ceil(sum(map(browser_seconds, entries)) / 480))


def selected_integration_count(pattern):
    estimate = sum(seconds for name, seconds in SECONDS['integration'].items() if re.search(pattern, name))
    return min(FULL_INTEGRATION, max(1, math.ceil(estimate / 650)))


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
