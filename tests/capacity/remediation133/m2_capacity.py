#!/usr/bin/env python3
"""Run the unchanged 125 denominator and sample only the owned API process.

The separate resource observation never filters HTTP failures or retries a run.
Detailed SQL/stage observers are disabled for formal timings.
"""
import argparse
import hashlib
import json
import os
import signal
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[3]


def run(args):
    resource_output = args.output.with_name(args.output.stem+'-resources.json')
    assert not args.output.exists() and not resource_output.exists()
    env = os.environ.copy()
    for key in ['CHNHCM_QUERY131_STATS', 'CHNHCM_CAPACITY_DIAGNOSTICS', 'CHNHCM_CAPACITY_PROFILE']:
        env.pop(key, None)
    child = subprocess.Popen(['python3', str(ROOT/'tests/capacity/remediation125/measure.py'),
        str(args.private), str(args.output), '--binary', str(args.binary.resolve()), '--phase', args.phase], env=env)
    result = dict(phase=args.phase, detailedObservers=False, samples=[], sampleIntervalSeconds=1,
                  binarySHA256=hashlib.sha256(args.binary.read_bytes()).hexdigest(),
                  clockTicksPerSecond=os.sysconf('SC_CLK_TCK'),
                  scope='API process only; RSS high water is sampled; CPU ticks include all API threads, not database or client CPU')
    start = time.monotonic()
    try:
        while child.poll() is None:
            try:
                pids = Path(f'/proc/{child.pid}/task/{child.pid}/children').read_text().split()
                for pid in pids:
                    proc = Path('/proc')/pid
                    argv = (proc/'cmdline').read_bytes().split(b'\0')
                    if b'-mode' not in argv or b'serve' not in argv or str(args.binary.resolve()).encode() != argv[0]:
                        continue
                    stat = (proc/'stat').read_text().rsplit(')', 1)[1].split()
                    status = (proc/'status').read_text().splitlines()
                    rss = int(next(v.split()[1] for v in status if v.startswith('VmRSS:')))
                    result['samples'].append(dict(seconds=round(time.monotonic()-start, 3),
                        pid=int(pid), rssKiB=rss, cpuTicks=int(stat[11])+int(stat[12])))
            except (FileNotFoundError, ProcessLookupError, StopIteration):
                pass  # A normal cold reset can end the process during sampling.
            time.sleep(1)
        result['exitCode'] = child.returncode
    finally:
        if child.poll() is None:
            child.send_signal(signal.SIGINT)
            child.wait(timeout=30)
        result['elapsedSeconds'] = time.monotonic()-start
        result['peakRSSKiB'] = max((v['rssKiB'] for v in result['samples']), default=None)
        resource_output.write_text(json.dumps(result, indent=2)+'\n')
    raise SystemExit(child.returncode)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('output', type=Path)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--phase', choices=['warm', 'cold', 'mixed', 'context', 'specials'], required=True)
    run(p.parse_args())
