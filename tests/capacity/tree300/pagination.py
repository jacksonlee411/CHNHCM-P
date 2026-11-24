#!/usr/bin/env python3
"""Read every synthetic root/wide page both ways; report identity digests only."""
from pathlib import Path
import argparse,hashlib,json,urllib.parse
from matrix import ROOT,MeasuredClient,fixtures,oracle
from measure import server
from policy import sparse_reads

parser=argparse.ArgumentParser()
parser.add_argument('private',type=Path);parser.add_argument('binary',type=Path);parser.add_argument('output',type=Path)
parser.add_argument('--kind',choices=['synthetic','frozen'],default='synthetic')
args=parser.parse_args();private=args.private.resolve();output=args.output
assert not private.is_relative_to(ROOT) and private.stat().st_mode&0o777==0o700
assert (private/('tree300-synthetic.json' if args.kind=='synthetic' else 'tree300-frozen.json')).exists() and not output.exists()
report=dict(completed=False,passed=False,formalCapacity=False,groups=[])
save=lambda:output.write_text(json.dumps(report,indent=2)+'\n')
save()
with server(private,args.binary.resolve(),args.kind,output.stem) as (process,info,diagnostics):
 for role in ['admin','reader','sparse-reader']:
  client=MeasuredClient(info['url']);status,login,_=client.call('POST','/api/v1/session/login',dict(login='reader' if role=='sparse-reader' else role,password='admin1234567'));assert status==200;client.csrf=login['csrf']
  status,context,_=client.call('GET','/api/v1/organization/context');assert status==200
  byid,children,path,branch,wide,deep,large,shape=fixtures(oracle(private,context['currentDate']))
  permitted=({n['id'] for n in sorted(byid.values(),key=lambda n:n['code'])[::100]}|{deep['id'],large['id'],wide,children[wide][-1]['id']}) if role=='sparse-reader' else None
  allowed=lambda nodes:nodes if permitted is None else [n for n in nodes if n['id'] in permitted]
  with sparse_reads(private,login,permitted,output.stem,info['container']):
   for name,parent in [('roots',None),('wide',wide)]:
    wanted=allowed(children[parent]);group=dict(role=role,case=name,expectedNodes=len(wanted),forwardPages=0,reversePages=0,requests=0,passed=False);report['groups'].append(group);save()
    params={'pageSize':50}
    if parent:params['parentId']=parent
    seen=set();collected=[];cursor=None;last=None;pages=[]
    def read(token):
     request=dict(params)
     if token:request['cursor']=token
     status,value,_=client.call('GET','/api/v1/departments/tree/nodes?'+urllib.parse.urlencode(request));group['requests']+=1
     assert status==200 and value['businessAt']==context['currentDate']
     for node in value['items']:
      original=byid[node['id']]
      assert all(node[k]==original[k] for k in ['code','name','parentId'])
      assert node['kind']=='node' and node['hasChildren']==bool(allowed(children.get(node['id'],[])))
     assert bool(value.get('nextCursor'))==value['hasMore'] and bool(value.get('previousCursor'))==value['hasPrevious']
     return value
    while True:
     value=read(cursor);ids=[n['id'] for n in value['items']];assert not (seen&set(ids));assert len(ids)==len(set(ids))
     assert value['hasPrevious']==bool(pages);seen.update(ids);collected.extend(ids);pages.append(ids);group['forwardPages']+=1;last=value
     if not value['hasMore']:break
     assert ids and value['nextCursor']!=cursor;cursor=value['nextCursor']
    assert collected==[n['id'] for n in wanted]
    position=len(pages)-1
    while last['hasPrevious']:
     last=read(last['previousCursor']);position-=1;group['reversePages']+=1
     assert [n['id'] for n in last['items']]==pages[position] and last['hasMore']
     assert last['hasPrevious']==(position>0)
    assert position==0
    group['identitySHA256']=hashlib.sha256('\n'.join(collected).encode()).hexdigest();group['passed']=True;save()
 report['completed']=True;report['passed']=all(g['passed'] for g in report['groups']);save()
