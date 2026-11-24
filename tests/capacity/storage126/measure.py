#!/usr/bin/env python3
"""Task-owned, real HTTP case verification. Never connects to the preserved DB."""
import argparse,base64,concurrent.futures,copy,gzip,hashlib,http.cookiejar,json,math,os,signal,socket,subprocess,sys,time,urllib.error,urllib.request,uuid
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
BIN=ROOT/'.local/storage-126-implementation/storage126'
class Client:
 def __init__(self):
  self.jar=http.cookiejar.CookieJar();self.http=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPCookieProcessor(self.jar));self.csrf='';self.base=''
 def call(self,method,path,body=None,key=None,want=200):
  self.request_id=None
  req=urllib.request.Request(self.base+path,data=None if body is None else json.dumps(body,ensure_ascii=False).encode(),method=method,headers={'Content-Type':'application/json','Origin':self.base,'X-CSRF-Token':self.csrf,**({'Idempotency-Key':key} if key else {})})
  start=time.perf_counter()
  try:
   with self.http.open(req,timeout=20) as r:status=r.status;raw=r.read();v=json.loads(raw) if raw else None;self.request_id=r.headers.get('X-Capacity-Request')
  except urllib.error.HTTPError as e:status=e.code;v=json.load(e);self.request_id=e.headers.get('X-Capacity-Request')
  ms=(time.perf_counter()-start)*1000
  if status!=want:
   error=RuntimeError(f'{method} {path}: status={status}, code={v.get("code")}, ms={ms:.2f}');error.status=status;error.code=v.get('code');error.elapsed_ms=ms;raise error
  return v,ms
class Server:
 def __init__(self,directory,evidence):self.dir=directory;self.log=(evidence/'http-server.log').open('a');self.p=None;self.base=''
 def start(self):
  self.p=subprocess.Popen([str(BIN),'-mode','serve','-input',str(self.dir)],cwd=ROOT,stdout=subprocess.PIPE,stderr=self.log,text=True)
  raw=self.p.stdout.readline()
  if not raw:raise RuntimeError('candidate API failed startup; inspect sanitized log')
  self.info=json.loads(raw);self.base=self.info['url'];return self.base
 def stop(self):
  if self.p and self.p.poll() is None:self.p.send_signal(signal.SIGTERM);self.p.wait(timeout=10)
  self.p=None
 def restart(self,clients):
  self.stop();self.start()
  for c in clients:c.base=self.base
 def close(self):self.stop();self.log.close()
def cli(mode,directory):return json.loads(subprocess.check_output([str(BIN),'-mode',mode,'-input',str(directory)],cwd=ROOT))
def save(path,v):path.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n')
def mapped(prefix,s):
 h=bytearray(hashlib.md5((prefix+s).encode()).digest());h[6]=(h[6]&15)|0x30;h[8]=(h[8]&15)|0x80;return str(uuid.UUID(bytes=bytes(h)))
def background(b,o,proto_bu,index):
 f=copy.deepcopy(b['Frames'][-1]);stable={o['instance_id'],o['enterprise_id']}
 for r in b['Frames']:stable.add(r['Operation']['Source']['Subject'])
 def transform(x):
  if isinstance(x,dict):return {k:transform(v) for k,v in x.items()}
  if isinstance(x,list):return [transform(v) for v in x]
  if isinstance(x,str):
   try:uuid.UUID(x)
   except ValueError:return x
   if x in stable:return x
   if x==proto_bu:return mapped(f'R6_B_{index%2000:05d}:',x)
   return mapped(o['code']+':',x)
  return x
 return transform(f)
class Oracle:
 def __init__(self,d):
  self.owners=json.loads((d/'owners.json').read_text());self.proto={k:json.loads((d/(k+'-prototype.json')).read_text()) for k in ['bu','dep']};self.extra={}
  for k in ['bu','dep']:
   with gzip.open(d/(k+'-typed.jsonl.gz'),'rt') as f:
    for line in f:
     b=json.loads(line);self.extra[b['Owner']['code']]=b
  self.bycode={o['code']:(k,o) for k,rows in self.owners.items() for o in rows}
 def frame(self,code,source=None):
  k,o=self.bycode[code]
  if code in self.extra:
   fs=self.extra[code]['Frames'];return fs[-1] if source is None else next(f for f in fs if f['Operation']['Source']['Number']==source)
  b=self.proto[k]
  if code==b['Owner']['code']:
   fs=b['Frames'];return fs[-1] if source is None else next(f for f in fs if f['Operation']['Source']['Number']==source)
  if source is not None:b=dict(b,Frames=b['Frames'][:source])
  return background(b,o,self.proto['bu']['Owner']['bu_id'],int(code[-5:]))
 def path(self,code,source=None):
  k,o=self.bycode[code];kind='business-units' if k=='bu' else 'departments';oid=o['bu_id' if k=='bu' else 'department_id'];p='/api/v1/'+kind+'/'+oid
  if source is not None:p+='?sourceVersion='+self.frame(code,source)['Operation']['Source']['ID']
  return p
 def check_detail(self,code,v,source=None):
  k,o=self.bycode[code];f=self.frame(code,source);t=f['State']['Timeline'];rows=v['timeline'];assert len(rows)==len(t['Records']),('timeline length',code)
  assert v.get('resolvedSourceVersion')==t['Source'] if source else v['headSourceVersion']==o['head_source_id'],('source',code)
  for a,b in zip(rows,t['Records']):
   assert (a['recordId'],a['revision'],a['effectiveDate'],a['operationType'])==(b['ID'],b['Revision'],b['Date'],b['Type']),('record identities',code,a)
   if 'name' in a:assert a['name']==(b['Name'] if k=='bu' else b['Data']['Name']),('record name',code)
  selected=v['selectedRecord'];original=next(r for r in t['Records'] if r['ID']==selected['recordId'])
  assert selected['revision']==original['Revision'] and selected['name']==(original['Name'] if k=='bu' else original['Data']['Name']),('selected fact',code)
  if source:assert v['readOnly'] is True
  if k=='dep':
   assert selected['managementBu']['id']==original['Data']['BU'],('management BU',code)
   if source:
    segments=[p for p in f['Evidence']['Segments'] if p['RecordID']==original['ID'] and p['Revision']==original['Revision']]
    assert segments,('frozen source path absent',code)
    names=[' / '.join(n['Name'] for n in p['Nodes'] if n['Kind']=='department') for p in segments]
    assert v['path']['departmentPath'] in names,('frozen path display',code)
  return True
 def list_expected(self,k):return sorted(self.owners[k],key=lambda o:o['normalized_code'] if k=='bu' else o['code'])
 def check_list(self,k,v):
  expected=self.list_expected(k);assert v['count']==len(expected),('count',k,v.get('count'),len(expected));assert v['page']==1 and v['pageSize']==20 and len(v['items'])==min(20,len(expected)),('page shape',k)
  assert [x['code'] for x in v['items']]==[x['code'] for x in expected[:len(v['items'])]],('first page order',k)
  for a,o in zip(v['items'],expected):
   assert a['id']==o['bu_id' if k=='bu' else 'department_id']
   t=self.frame(o['code'])['State']['Timeline'];day=v['businessAt'];live=[r for r in t['Records'] if r['Date']<=day and (not r['End'] or day<r['End'])];r=live[0] if live else t['Records'][0];assert a['name']==(r['Name'] if k=='bu' else r['Data']['Name']) and a['effectiveDate']==r['Date'] and a['status']==(r['State'] if live else 'not_effective'),('current values',o['code'])
  return True

def stats(samples,target):
 valid=[x['ms'] for x in samples if x['ok']];vals=sorted(valid);p95=vals[max(0,math.ceil(len(vals)*.95)-1)] if vals else None
 return dict(requests=len(samples),successful=len(vals),failed=len(samples)-len(vals),p95MS=p95,maxMS=max(vals) if vals else None,targetMS=target,passed=len(vals)==len(samples) and p95 is not None and p95<=target,samples=samples)
def parallel_jobs(jobs):
 with concurrent.futures.ThreadPoolExecutor(max_workers=20) as pool:return list(pool.map(lambda fn:fn(),jobs))
def measured(c,method,path,body=None,key=None,check=None):
 start=time.perf_counter()
 try:
  v,ms=c.call(method,path,body,key)
  if check:check(v)
  return dict(ok=True,ms=ms),v
 except Exception as e:return dict(ok=False,ms=(time.perf_counter()-start)*1000,error=str(e)),None

def run(d,out):
 oracle=Oracle(d);result=dict(scope='same complete case; real HTTP including response decode; no original database writes',sessions=20,cacheScope='cold API process only; PostgreSQL and OS caches retain natural state',startedAt=time.strftime('%Y-%m-%dT%H:%M:%S%z'),completed=False,groups={},checks=[])
 result['before']=cli('size',d)
 counts=result['before']['counts'];assert counts['BU']==2023 and counts['department']==22104 and counts['buSources']==101053 and counts['departmentSources']==1003153,counts
 server=Server(d,out.parent);clients=[Client() for _ in range(20)]
 try:
  base=server.start()
  for i,c in enumerate(clients):
   c.base=base;v,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));c.csrf=v['csrf']
   if i<19:time.sleep(3.1)
  # Exact old-source assertions are independent of the candidate decoder.
  codes=['R6_B_00000','R6_D_00000']
  specials=[code for code,b in oracle.extra.items() if len(b['Frames'])>=1000 or code=='R6_DEPTH_099' or 'FAN_ROOT' in code]
  for code in codes+specials:
   for source in [None,1,25] if len(oracle.extra.get(code,oracle.proto[oracle.bycode[code][0]])['Frames'])>=25 else [None,1]:
    s,v=measured(clients[0],'GET',oracle.path(code,source),check=lambda v,code=code,source=source:oracle.check_detail(code,v,source));result['checks'].append(dict(code=code,source=source,**s))
  # Deliberate bad HTTP-200 payload must fail the independent oracle.
  bad=False
  try:oracle.check_list('dep',dict(count=22104,page=1,pageSize=20,businessAt='2026-11-07',items=[]))
  except Exception:bad=True
  assert bad,'oracle accepted fabricated empty HTTP-200 page'
  result['oracleRejectsWrongEmptyPage']=bad
  for k,route in [('bu','business-units'),('dep','departments')]:
   for cache in ['cold-api','warm']:
    if cache=='cold-api':server.restart(clients)
    else:
     try:clients[0].call('GET','/api/v1/'+route)
     except Exception as e:result['checks'].append(dict(warmup=route,error=str(e),ok=False))
    samples=[]
    for repeat in range(1 if cache=='cold-api' else 3):
     jobs=[lambda c=c,k=k,route=route:measured(c,'GET','/api/v1/'+route,check=lambda v:oracle.check_list(k,v))[0] for c in clients]
     samples+=parallel_jobs(jobs)
    result['groups'][k+'-list-'+cache]=stats(samples,300);save(out,result);print(k,cache,result['groups'][k+'-list-'+cache]['p95MS'],flush=True)
  for action,source in [('detail',None),('source',25)]:
   samples=[]
   for repeat in range(3):
    jobs=[]
    for i,c in enumerate(clients):
     code=f'R6_{"B" if i%2==0 else "D"}_{i:05d}'
     jobs.append(lambda c=c,code=code,source=source:measured(c,'GET',oracle.path(code,source),check=lambda v:oracle.check_detail(code,v,source))[0])
    samples+=parallel_jobs(jobs)
   result['groups']['mixed-'+action]=stats(samples,500 if source else 300);save(out,result)
  for route in ['organization','departments']:
   samples=parallel_jobs([lambda c=c,route=route:measured(c,'GET','/api/v1/'+route+'/context',check=lambda v:check_context(v))[0] for c in clients])
   result['groups'][route+'-context']=stats(samples,300);save(out,result)
  result['afterReads']=cli('size',d)
  # Actual independently submitted corrections; all summaries/digests/timestamps differ.
  originals=[];write_samples=[];recover_samples=[];postchecks=[]
  for round in range(3):
   jobs=[]
   for i,c in enumerate(clients):
    code=f'R6_{"B" if i%2==0 else "D"}_{100+i:05d}';path=oracle.path(code);key=str(uuid.uuid4());name='126独立修改'+str(uuid.uuid4())
    try:
     detail,_=c.call('GET',path);row=detail['timeline'][0];body=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=row['revision'],changes=dict(name=name));endpoint=path+'/records/'+row['recordId']+'/correct'
     jobs.append(lambda c=c,code=code,path=path,key=key,name=name,endpoint=endpoint,body=body,row=row:(measured(c,'POST',endpoint,body,key),c,code,path,key,name,row['recordId']))
    except Exception as e:write_samples.append(dict(ok=False,ms=-1,error='preparation: '+str(e)))
   for (sample,v),c,code,path,key,name,record in parallel_jobs(jobs):
    write_samples.append(sample)
    if v is not None:
     assert v['outcome']=='accepted' and v['submissionId']==key
     originals.append(dict(code=code,key=key,source=v['sourceVersion'],record=record,name=name))
     current,_=c.call('GET',path);stored=next(x for x in current['timeline'] if x['recordId']==record);assert stored['name']==name
     route='/api/v1/'+('organization' if oracle.bycode[code][0]=='bu' else 'departments')+'/submissions/'+key+'/resolve'
     sample,_=measured(c,'POST',route,check=lambda got,v=v:check_resolve(got,v));recover_samples.append(sample)
  result['groups']['mixed-save']=stats(write_samples,500);result['groups']['original-result-resolve']=stats(recover_samples,500);result['writeIdentities']=originals;result['afterWrites']=cli('size',d);save(out,result)
  # Fixed lifecycle pairs on independent BUs/departments, with exact deletion.
  pair_samples=[];pair_delete=[];jobs=[]
  for i,c in enumerate(clients):
   code=f'R6_SOLO_{i:02d}' if i<10 else f'R6_D_{500+i:05d}';path=oracle.path(code)
   detail,_=c.call('GET',path);key=str(uuid.uuid4());body=dict(expectedSourceVersion=detail['headSourceVersion'],anchorRecordId=detail['timeline'][-1]['recordId'],deactivationDate='2035-01-01',deactivationReason='ORG_RESTRUCTURING',reactivationDate='2035-02-01',reactivationReason='BUSINESS_RESUMPTION')
   jobs.append(lambda c=c,code=code,path=path,key=key,body=body:(measured(c,'POST',path+'/lifecycle-pairs',body,key),c,code,path,key))
  for (sample,v),c,code,path,key in parallel_jobs(jobs):
   pair_samples.append(sample)
   if v is None:continue
   assert v['outcome']=='accepted' and len(v['items'])==2
   originals.append(dict(code=code,key=key,source=v['sourceVersion'],kind='pair'))
   detail,_=c.call('GET',path);a,z=detail['timeline'][-2:];assert a['effectiveDate']=='2035-01-01' and z['effectiveDate']=='2035-02-01' and a['state']=='inactive' and z['state']=='active'
   body=dict(expectedSourceVersion=detail['headSourceVersion'],deactivationRecordId=a['recordId'],deactivationRevision=a['revision'],reactivationRecordId=z['recordId'],reactivationRevision=z['revision'])
   sample,deleted=measured(c,'POST',path+'/lifecycle-pairs/delete',body,str(uuid.uuid4()));pair_delete.append(sample)
   if deleted is not None:
    now,_=c.call('GET',path);assert len(now['timeline'])==len(detail['timeline'])-2
  result['groups']['mixed-fixed-pair']=stats(pair_samples,500);result['pairDeletionChecks']=pair_delete;save(out,result)
  # Twenty sessions continuously mix distinct endpoint types. Readers use stable
  # objects; each writer has its own object and a genuinely independent digest.
  mixed=[]
  last={i:originals[i%len(originals)] for i in range(20)} if originals else {}
  for turn in range(6):
   jobs=[]
   for i,c in enumerate(clients):
    action=(i+turn)%5;code=f'R6_{"B" if i%2==0 else "D"}_{i:05d}'
    if action==0:
     k='bu' if i%2==0 else 'dep';route='business-units' if k=='bu' else 'departments';jobs.append(lambda c=c,k=k,route=route:measured(c,'GET','/api/v1/'+route,check=lambda v:oracle.check_list(k,v))[0])
    elif action in [1,2]:
     source=None if action==1 else 25;jobs.append(lambda c=c,code=code,source=source:measured(c,'GET',oracle.path(code,source),check=lambda v:oracle.check_detail(code,v,source))[0])
    elif action==3:
     code=f'R6_{"B" if i%2==0 else "D"}_{800+i:05d}';path=oracle.path(code);detail,_=c.call('GET',path);row=detail['timeline'][0];key=str(uuid.uuid4());body=dict(expectedSourceVersion=detail['headSourceVersion'],expectedRecordRevision=row['revision'],changes=dict(name='126混合独立'+str(uuid.uuid4())))
     def write(c=c,path=path,row=row,key=key,body=body):
      sample,v=measured(c,'POST',path+'/records/'+row['recordId']+'/correct',body,key)
      if v is not None:assert v['outcome']=='accepted' and v['sourceVersion']!=body['expectedSourceVersion']
      return sample
     jobs.append(write)
    else:
     item=last.get(i)
     if item is None:
      jobs.append(lambda:dict(ok=False,ms=0,error='no accepted original result to recover'));continue
     route='/api/v1/'+('organization' if oracle.bycode[item['code']][0]=='bu' else 'departments')+'/submissions/'+item['key']+'/resolve';jobs.append(lambda c=c,route=route,item=item:measured(c,'POST',route,check=lambda v:check_resolve(v,dict(sourceVersion=item['source'],submissionId=item['key'])))[0])
   for i,sample in enumerate(parallel_jobs(jobs)):
    sample['action']=['list','detail','source','save','resolve'][(i+turn)%5];sample['targetMS']=300 if sample['action'] in ['list','detail'] else 500;mixed.append(sample)
  result['groups']['continuous-mixed-actions']=stats(mixed,500);result['groups']['continuous-mixed-actions']['perAction']={action:stats([x for x in mixed if x['action']==action],300 if action in ['list','detail'] else 500) for action in ['list','detail','source','save','resolve']};result['groups']['continuous-mixed-actions']['passed']=all(g['passed'] for g in result['groups']['continuous-mixed-actions']['perAction'].values());result['afterMixed']=cli('size',d);save(out,result)
  # Full instance restart then recover the exact original identities, no new write.
  server.stop();assert subprocess.check_output(['docker','inspect','--format','{{index .Config.Labels "io.chnhcm.task"}}','chnhcm-storage126-product'],text=True).strip()=='storage126-product';subprocess.run(['docker','restart','chnhcm-storage126-product'],check=True,stdout=subprocess.DEVNULL)
  for attempt in range(60):
   ready=subprocess.run(['docker','exec','chnhcm-storage126-product','pg_isready','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   if ready.returncode==0:break
   time.sleep(.2)
  server.start()
  for c in clients:c.base=server.base
  for item in originals[:20]+originals[-20:]:
   route='/api/v1/'+('organization' if oracle.bycode[item['code']][0]=='bu' else 'departments')+'/submissions/'+item['key']+'/resolve'
   v,ms=clients[0].call('POST',route);assert v['outcome']=='accepted' and v['sourceVersion']==item['source'];postchecks.append(dict(code=item['code'],source=item['source'],ms=ms,ok=True))
  result['restartRecoveries']=postchecks
  result['afterRestart']=cli('size',d);result['allSemanticChecks']=all(x['ok'] for x in result['checks']) and all(g['failed']==0 for g in result['groups'].values());result['allLatencyTargets']=all(g['passed'] for g in result['groups'].values());result['allSemanticChecks']=result['allSemanticChecks'] and all(x['ok'] for x in pair_delete+postchecks);result['completed']=True;save(out,result)
 except Exception as e:
  result['fatalError']=str(e);raise
 finally:server.close();save(out,result)
 return result

def check_context(v):assert v.get('currentDate') and v.get('enterprise'), 'missing context identity/date'

def check_resolve(got,want):assert got['outcome']=='accepted' and got['sourceVersion']==want['sourceVersion'] and got['submissionId']==want['submissionId']

def resume_restart(directory, output):
 """Keep original timings and writes; continue only the failed restart harness."""
 result=json.loads(output.read_text());assert not result['completed'] and result.get('afterMixed')
 oracle=Oracle(directory);server=Server(directory,output.parent);client=Client()
 result['restartContinuation']={'priorError':result.pop('fatalError',None),'repeatedBusinessWrites':False,'sameCookieAcrossSecondRestart':False}
 try:
  client.base=server.start();v,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));client.csrf=v['csrf']
  server.stop()
  assert subprocess.check_output(['docker','inspect','--format','{{index .Config.Labels "io.chnhcm.task"}}','chnhcm-storage126-product'],text=True).strip()=='storage126-product'
  subprocess.run(['docker','restart','chnhcm-storage126-product'],check=True,stdout=subprocess.DEVNULL)
  for attempt in range(60):
   if subprocess.run(['docker','exec','chnhcm-storage126-product','pg_isready','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:break
   time.sleep(.2)
  client.base=server.start();checks=[]
  for item in result['writeIdentities']:
   route='/api/v1/'+('organization' if oracle.bycode[item['code']][0]=='bu' else 'departments')+'/submissions/'+item['key']+'/resolve'
   got,ms=client.call('POST',route);check_resolve(got,dict(sourceVersion=item['source'],submissionId=item['key']));checks.append(dict(code=item['code'],source=item['source'],ms=ms,ok=True))
  result['restartRecoveries']=checks;result['restartContinuation']['sameCookieAcrossSecondRestart']=True
  result['afterRestart']=cli('size',directory)
  result['allSemanticChecks']=all(x['ok'] for x in result['checks']+result['pairDeletionChecks']+checks) and all(g['failed']==0 for g in result['groups'].values())
  result['allLatencyTargets']=all(g['passed'] for g in result['groups'].values());result['completed']=True
 except Exception as error:
  result['fatalError']=str(error);raise
 finally:server.close();save(output,result)
 return result

if __name__=='__main__':
 ap=argparse.ArgumentParser();ap.add_argument('directory',type=Path);ap.add_argument('output',type=Path);ap.add_argument('--resume-restart',action='store_true');args=ap.parse_args();assert args.directory.is_absolute() and ROOT not in args.directory.parents
 (resume_restart if args.resume_restart else run)(args.directory,args.output)
