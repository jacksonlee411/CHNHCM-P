#!/usr/bin/env python3
"""Exercise growth scheduling after rejected or unconfirmed dependent writes.

The actual scheduling blocks run against synthetic state. No HTTP or database
is used, and the 400 commands are the date/pair subset of a 1000-command cycle.
"""
import ast
import copy
from pathlib import Path
import unittest

SOURCE = Path(__file__).with_name('growth.py')

def exercise(mode):
 tree=ast.parse(SOURCE.read_text());fn=next(n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name=='run')
 main=next(n for n in fn.body if isinstance(n,ast.Try))
 begin=next(i for i,n in enumerate(main.body) if isinstance(n,ast.Assign) and any(isinstance(t,ast.Name) and t.id=='added' for t in n.targets))
 end=next(i for i,n in enumerate(main.body) if isinstance(n,ast.Assign) and any(isinstance(t,ast.Subscript) and isinstance(t.value,ast.Name) and t.value.id=='result' and isinstance(t.slice,ast.Constant) and t.slice.value=='writeSeconds' for t in n.targets))
 nodes=main.body[begin:end]
 objects=[f'synthetic-{n:04d}' for n in range(500)]
 state={oid:dict(headSourceVersion='source-original',timeline=[dict(recordId=oid+'-old-inactive',revision='revision-1',name='original',effectiveDate='2000-01-01'),dict(recordId=oid+'-old-active',revision='revision-2',name='original',effectiveDate='2001-01-01')]) for oid in objects}
 calls=[];wrong=[];result=dict(writes=[],failures=[],skippedDependentCommands=[])
 def detail(oid):return copy.deepcopy(state[oid])
 def write(oid,category,suffix,body,**kwargs):
  calls.append(dict(object=oid,category=category));result['writes'].append(dict(category=category))
  failing=(oid==objects[300] and category=='date-add' and mode in ['add-rejected','add-lost','both-rejected']) or (oid==objects[400] and category=='pair-add' and mode in ['pair-rejected','pair-lost','both-rejected'])
  if category=='date-add' and (not failing or mode=='add-lost'):
   rid=oid+'-new-record';state[oid]['timeline'].append(dict(recordId=rid,revision='revision-new',name='new',effectiveDate=body['effectiveDate']))
  if category=='pair-add' and (not failing or mode=='pair-lost'):
   for suffix_part in ['new-inactive','new-active']:state[oid]['timeline'].append(dict(recordId=oid+'-'+suffix_part,revision='revision-new',name='new',effectiveDate='2051-01-01'))
  if category=='date-move' and (f'{oid}-new-record' not in suffix or (oid==objects[300] and mode=='add-lost')):
   wrong.append(dict(category=category,target=suffix,reason='dependent command without confirmed addition'))
  if category=='pair-restore' and (not body['deactivationRecordId'].endswith('-new-inactive') or (oid==objects[400] and mode=='pair-lost')):
   wrong.append(dict(category=category,target=body['deactivationRecordId'],reason='dependent command without confirmed pair'))
  if failing:result['failures'].append(dict(category=category));return None
  if category=='date-add':return dict(targetRecordId=oid+'-new-record')
  return dict(outcome='accepted')
 namespace=dict(plan={'objects':objects},cycle=1,result=result,detail=detail,write=write)
 exec(compile(ast.Module(body=nodes,type_ignores=[]),str(SOURCE),'exec'),namespace)
 expected_skips=2 if mode=='both-rejected' else 0 if mode=='all-success' else 1
 actual_skips=len(result['skippedDependentCommands'])
 passed=not wrong and len(calls)==400-expected_skips and actual_skips==expected_skips
 return dict(case=mode,passed=passed,plannedCommands=400,actualCommandCalls=len(calls),skippedDependentCommands=actual_skips,expectedSkips=expected_skips,
             priorFailureRecords=len(result['failures']),unsafeDependentTargets=wrong,actualHTTPCalls=0,actualDockerCalls=0)


class GrowthDependencyTests(unittest.TestCase):
    def test_confirmed_additions_keep_the_complete_command_distribution(self):
        result = exercise('all-success')
        self.assertEqual(result['actualCommandCalls'], 400)
        self.assertEqual(result['skippedDependentCommands'], 0)
        self.assertEqual(result['priorFailureRecords'], 0)
        self.assertEqual(result['unsafeDependentTargets'], [])

    def test_unconfirmed_additions_preserve_failures_and_do_not_target_old_records(self):
        for mode in ['add-rejected', 'add-lost', 'pair-rejected', 'pair-lost', 'both-rejected']:
            with self.subTest(mode=mode):
                result = exercise(mode)
                expected = 2 if mode == 'both-rejected' else 1
                self.assertEqual(result['priorFailureRecords'], expected)
                self.assertEqual(result['skippedDependentCommands'], expected)
                self.assertEqual(result['actualCommandCalls'], 400-expected)
                self.assertEqual(result['unsafeDependentTargets'], [])


if __name__ == '__main__':
    unittest.main()
