#!/usr/bin/env python3
"""R133 final affected matrix after explicit preview fixture writes.

The unchanged 125 oracle still checks every original record and revision. Only
BU 00000's current source identity comes from the retained add/delete receipt;
its original source queries remain pinned to the original frozen input.
"""
import argparse,json,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/remediation125'))
from measure import Lane

def run(a):
 assert not a.output.exists()
 receipts=json.loads((a.private/'synthetic-preview-diagnostic-receipts.json').read_text())
 assert len(receipts)==2 and all(r['outcome']=='accepted' for r in receipts)
 class FinalLane(Lane):
  def detail_check(self,code,source):
   check=super().detail_check(code,source)
   if code=='R6_B_00000' and source is None:
    assert self.oracle.bycode[code][1]['bu_id']==receipts[-1]['buId']
    self.detailExpected[(code,source)]['source']=receipts[-1]['sourceVersion']
   return check
 lane=FinalLane(a.private,a.output,a.binary.resolve())
 lane.result['fixtureSourceExpectation']='retained accepted add/delete receipt; all original record values and revisions still checked'
 try:
  if a.phase=='mixed':lane.matrix(a.phase)
  else:lane.matrix(a.phase,actions=['save','pair'],modes=['BU','Department','mixed'])
  lane.result['completed']=(lane.result['allScheduledGroupsAttempted'] and lane.result['actualTimedRequests']==lane.result['expectedTimedRequests'] and not lane.result['unattemptedRequests'])
 finally:lane.close()
 if not lane.result['passed']:raise SystemExit(1)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);p.add_argument('--binary',type=Path,required=True);p.add_argument('--phase',choices=['warm','cold','mixed'],required=True);run(p.parse_args())
