#!/usr/bin/env python3
"""Attribute sampled events to requests; never estimate a wait duration."""
import argparse
from collections import Counter
import json
from pathlib import Path


def attribute(data):
    observations = {(o['diagnosticRun'], str(o['Request'])): o
                    for o in data['observations']}
    counts = {key: Counter() for key in observations}
    unmatched = ambiguous = correlated = 0
    for batch in data['sqlWaitObservations']:
        run = batch['diagnosticRun']
        for sample in batch['samples']:
            owners = [key for key, o in observations.items()
                      if key[0] == run and o.get('WallClockConsistent')
                      and any(w['PID'] == sample['pid']
                              and w['StartedUnixNS'] <= sample['observedStartUnixNS']
                              <= sample['observedEndUnixNS'] <= w['FinishedUnixNS']
                              for w in o.get('SQLWindows', []))]
            if len(owners) == 1:
                counts[owners[0]][(sample['type'], sample['event'],
                                   bool(sample['shapeInsert']), bool(sample['blockers']))] += 1
                correlated += 1
            elif owners:
                ambiguous += 1
            else:
                unmatched += 1
    requests = []
    for group, values in data['groups'].items():
        for sample in values['samples']:
            key = (sample['diagnosticRun'], str(sample['request']))
            o = observations[key]
            requests.append(dict(group=group, request=key[1], diagnosticRun=key[0],
                backendPIDs=o.get('BackendPIDs', []), wallClockConsistent=o.get('WallClockConsistent'),
                stages=o['Stages'], processAllocatedBytes=o['ProcessAllocatedBytes'],
                processGCCycles=o['ProcessGCCycles'], processGCPauseNS=o['ProcessGCPauseNS'],
                concurrentMemoryWindow=o['ConcurrentMemoryWindow'],
                waitSamples=[dict(type=k[0], event=k[1], shapeInsert=k[2],
                                  blockingPIDObserved=k[3], samples=n)
                             for k, n in sorted(counts[key].items())]))
    return dict(diagnosticOnly=True, capacityAcceptance=False,
                scope='Monotonic request stages; SQL durations include execution and waiting. '
                      'Wait samples are counts, never duration. Nested stages and overlapping '
                      'process memory windows must not be added. Wall timestamps only correlate '
                      'samples; unstable clock intervals and ambiguous ownership are excluded.',
                correlatedSamples=correlated, unmatchedSamples=unmatched,
                ambiguousSamples=ambiguous, timedRequests=requests)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('input', type=Path)
    p.add_argument('output', type=Path)
    a = p.parse_args()
    assert not a.output.exists()
    a.output.write_text(json.dumps(attribute(json.loads(a.input.read_text())), indent=2)+'\n')
