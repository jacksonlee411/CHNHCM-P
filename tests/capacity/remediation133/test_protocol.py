"""Protect the original business denominator and failure/queue timing evidence."""
import json
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import protocol

class ProtocolEvidence(unittest.TestCase):
 def fixture(self,fail=None):
  clock=[0];calls=[]
  rows=[{'id':'a'},{'id':'b'}]
  def page(value,expected,index=1):
   assert value['items']==rows and value['page']==index
  owner=SimpleNamespace(ordered=rows,rows=rows,page=page)
  class Client:
   def call(self,method,path,body):
    calls.append((path,body));clock[0]+=700_000_000 # queue, transport and JSON
    if fail and len(calls)==fail:
     e=RuntimeError('controlled failure');e.status=503;e.code='M02_DEPENDENCY_UNAVAILABLE';e.elapsed_ms=700;raise e
    return dict(page=dict(items=rows,page=body['page'],queryKey='q'),summary=dict(queryKey='q',selectedCount=2,canExport=True,draftId='a'*36,revision=1)),700
  return owner,Client(),clock,calls
 def test_required_capture_stays_inside_resolve_timer(self):
  owner,client,clock,calls=self.fixture()
  with patch.object(protocol.time,'monotonic_ns',side_effect=lambda:clock[0]):v=protocol.operation(owner,'admin','resolve',client)
  self.assertTrue(v['ok']);self.assertEqual(v['ms'],1400);self.assertEqual(v['httpRequests'],2)
  self.assertTrue(calls[0][0].endswith('/start'));self.assertTrue(calls[1][0].endswith('/resolve'))
  self.assertEqual(calls[1][1]['draftId'],'a'*36)
 def test_first_and_second_failures_are_not_retried_or_omitted(self):
  for failure in [1,2]:
   owner,client,clock,calls=self.fixture(failure)
   with patch.object(protocol.time,'monotonic_ns',side_effect=lambda:clock[0]):v=protocol.operation(owner,'admin','resolve',client)
   self.assertFalse(v['ok']);self.assertEqual(v['status'],503);self.assertEqual(v['httpRequests'],failure)
   self.assertEqual(len(calls),failure);self.assertEqual(v['ms'],700*failure)
   self.assertEqual(v['subrequests'][-1]['status'],503)
 def test_original_32_groups_and_required_http_count(self):
  root=Path(__file__).resolve().parents[3]
  groups=json.loads((root/'docs/verification/data/131-t04-capacity-and-handoff/selected-stage-capacity-boundary-r153.json').read_text())['groupResults']
  self.assertEqual(len(groups),32);self.assertEqual(sum(g['requests'] for g in groups),6400)
  for sessions in ['1','20']:self.assertEqual(sum(g['requests'] for g in groups if g['tag'].endswith('/'+sessions)),3200)
  self.assertEqual(sum(g['requests']*(2 if g['tag'].split('/')[1] in ['resolve','parent-create-second','parent-maintain-second'] else 1) for g in groups),8000)
if __name__=='__main__':unittest.main()
