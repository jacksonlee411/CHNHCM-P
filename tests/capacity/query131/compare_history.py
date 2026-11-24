#!/usr/bin/env python3
"""Compare every immutable full-source digest against the retained independent run."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path


def run(args):
    assert not args.output.exists()
    info={}
    def rows(path,label):
        digest=hashlib.sha256();previous=None;count=0
        with gzip.open(path,'rb') as stream:
            for line in stream:
                digest.update(line);row=json.loads(line);key=(row[0],row[1],row[3])
                assert previous is None or previous<key,'source order or identity duplicated'
                previous=key;count+=1;yield key,row
        info[label]=dict(rows=count,uncompressedSHA256=digest.hexdigest())
    candidate=iter(rows(args.candidate,'candidate'));current=next(candidate,None);matched=extra=0
    for key,row in rows(args.original,'original'):
        while current is not None and current[0]<key:
            extra+=1;current=next(candidate,None)
        assert current is not None and current[0]==key,'original source missing'
        assert current[1]==row,'original source values changed'
        matched+=1;current=next(candidate,None)
    while current is not None:
        extra+=1;current=next(candidate,None)
    assert matched==args.original_count and extra==args.extra
    out=dict(passed=True,originalSourcesMatched=matched,newSources=extra,allOriginalFullFactsAndFrozenPathsAndTerminalResultsUnchanged=True,inputs=info)
    args.output.write_text(json.dumps(out,indent=2)+'\n');print(json.dumps(out))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('original',type=Path);p.add_argument('candidate',type=Path);p.add_argument('output',type=Path);p.add_argument('--original-count',type=int,required=True);p.add_argument('--extra',type=int,required=True);run(p.parse_args())
