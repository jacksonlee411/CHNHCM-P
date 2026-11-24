#!/usr/bin/env python3
"""Run query integrity probes against fresh, task-owned current-model databases.

No DSN/profile input is accepted. Tests use the formal migration and source
writer; historical T00 overlay runs remain in their original evidence records.
"""
import argparse
import hashlib
import json
import os
import pathlib
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run', default='^TestDepartmentQuery131')
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    root = pathlib.Path(__file__).resolve().parents[1]
    out = (root / args.output).resolve()
    out.relative_to(root)
    out.mkdir(parents=True, exist_ok=False)
    inputs = set(['scripts/prototype-query131.py'])
    for directory in ['db/migrations', 'internal']:
        inputs.update(str(p.relative_to(root)) for p in (root / directory).rglob('*')
                      if p.is_file() and p.suffix in {'.go', '.sql'})
    inputs = sorted(inputs)
    input_hashes = {name: hashlib.sha256((root / name).read_bytes()).hexdigest() for name in inputs}
    cmd = ['go', 'test', '-count=1', '-p=1', '-tags=integration,query131',
           '-timeout=12m', '-run', args.run, '-v', './internal/bootstrap']
    start = time.monotonic()
    result = subprocess.run(cmd, cwd=root, env={**os.environ, 'GOMAXPROCS': '2'},
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    elapsed = time.monotonic() - start
    raw = result.stdout
    (out / 'output.txt').write_text('\n'.join(line.rstrip() for line in raw.splitlines()) + '\n')
    (out / 'result.json').write_text(json.dumps({
        'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
        'run': args.run, 'command': cmd, 'model': 'm02-compact-5', 'read_rule': 4,
        'input_sha256': input_hashes,
        'exit_code': result.returncode, 'seconds': round(elapsed, 3), 'stdout': raw,
    }, ensure_ascii=False, indent=2) + '\n')
    assert all(hashlib.sha256((root / name).read_bytes()).hexdigest() == digest
               for name, digest in input_hashes.items()), 'tested inputs changed during probes'
    concise = [line for line in raw.splitlines() if 'Q131_EXPLAIN ' not in line]
    print('\n'.join(concise[-48:]))
    print(json.dumps({'exit_code': result.returncode, 'seconds': round(elapsed, 3)}, ensure_ascii=False))
    return result.returncode


if __name__ == '__main__':
    raise SystemExit(main())
