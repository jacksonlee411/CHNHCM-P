#!/usr/bin/env python3
"""Verify retained same-input A/B archives without copying or emitting secrets.

Input is the retained external storage130 task directory. Only hashes/counts and
build identities are published. No database or fixed runtime is opened.
"""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
BASE = '3591b0313303ea9bc32579ab0f17759e23e6f011'


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def run(source, output):
    source = source.resolve()
    if source == ROOT or ROOT in source.parents:
        raise ValueError('source archives must remain outside checkout')
    output.mkdir(parents=True, exist_ok=False)
    result = dict(baselineCommit=BASE, baselineModel='m02-compact-4', baselineReadRule=3,
                  candidateModel='m02-compact-5', candidateReadRule=4,
                  candidateBuild='T01-T03 final build must be frozen before T04; not built by T00',
                  candidateQuerySeal='department-query-131/3', shapes={})
    for name, relative in [('vanke', 'source'), ('synthetic', 'synthetic/source')]:
        folder = source / relative
        manifest = json.loads((folder / 'snapshot.json').read_text())
        tables = []
        for table in manifest['Tables']:
            path = folder / table['File']
            # storage130 hashes the canonical uncompressed JSONL bytes.
            raw_hash = hashlib.sha256()
            with gzip.open(path, 'rb') as stream:
                count = 0
                for line in stream:
                    raw_hash.update(line)
                    count += 1
            digest = raw_hash.hexdigest()
            if digest != table['SHA256']:
                raise ValueError('archive hash mismatch: ' + table['Name'])
            if count != table['Rows']:
                raise ValueError('archive row mismatch: ' + table['Name'])
            tables.append(dict(table=table['Schema']+'.'+table['Name'], rows=count,
                               bytes=path.stat().st_size, sha256=digest,
                               compressedSHA256=sha(path)))
        result['shapes'][name] = dict(manifestSHA256=sha(folder/'snapshot.json'),
                                     originalModel=manifest['Model'], tables=tables)
    # Extract only the self-contained Go build inputs from the committed A tree.
    archive = subprocess.check_output(['git','archive',BASE,'go.mod','go.sum','cmd','internal','db','api'],cwd=ROOT)
    result['baselineBuildInputSHA256'] = hashlib.sha256(archive).hexdigest()
    tree = output/'baseline-source'
    tree.mkdir()
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(tree, filter='data')
    binary = output.resolve()/'baseline-server'
    completed = subprocess.run(['go','build','-trimpath','-o',str(binary),'./cmd/server'],cwd=tree,
                               env=dict(os.environ,GOMAXPROCS='2'),capture_output=True,text=True)
    (output/'build.txt').write_text(completed.stdout+completed.stderr)
    completed.check_returncode()
    result['baselineBinarySHA256'] = sha(binary)
    result['scriptSHA256'] = sha(Path(__file__).resolve())
    result['fixedDatabaseConnections'] = 0
    result['sourceReadOnly'] = True
    (output/'manifest.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(passed=True,shapes={k:len(v['tables']) for k,v in result['shapes'].items()},
                         baselineBinarySHA256=result['baselineBinarySHA256'])))


if __name__ == '__main__':
    p=argparse.ArgumentParser()
    p.add_argument('source',type=Path)
    p.add_argument('output',type=Path)
    a=p.parse_args()
    run(a.source,a.output)
