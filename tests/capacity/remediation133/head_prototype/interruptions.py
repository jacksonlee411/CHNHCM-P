from pathlib import Path
import subprocess,sys,os,json,uuid,threading,time,signal
os.umask(0o077);p=Path(sys.argv[1]).resolve();s=p/'long-input';root=Path(__file__).resolve().parents[4];binary=Path(sys.argv[2]).resolve()
sys.path.insert(0,str(root/'tests/capacity/storage130'));from growth import case,resource,sql
label=sys.argv[3] if len(sys.argv)>3 else 'prototype-interruptions';assert not (p/(label+'.json')).exists()
name=resource(s);assert name=='chnhcm-storage126-product';meta=json.loads((s/'input.json').read_text());obj=str(uuid.UUID(meta['object']));case.BIN=binary;server=case.Server(s,s);receipts=[];barrier=None;thread=None;out=dict(prototypeOnly=True,completed=False,checks=[],failures=[])
def mode(which):
 return json.loads(subprocess.check_output([str(binary),'-mode',which,'-input',str(s)]))
def snapshot():
 tables=sql(name,"SELECT tablename FROM pg_tables WHERE schemaname='organization' ORDER BY tablename").splitlines()
 return {t:sql(name,f"SELECT encode(sha256(convert_to(coalesce(string_agg(h,'' ORDER BY h),''),'UTF8')),'hex') FROM (SELECT encode(sha256(convert_to(row_to_json(t)::text,'UTF8')),'hex') h FROM organization.{t} t) q") for t in tables}
def reject(label,q):
 r=subprocess.run(['docker','exec','-i',name,'psql','-X','-U','postgres','-d','storage126','-Atq','-v','ON_ERROR_STOP=1','-v','VERBOSITY=sqlstate'],input='BEGIN; '+q+'; COMMIT;',text=True,capture_output=True)
 assert r.returncode!=0,label
 out['checks'].append(dict(check=label,ordinarySQLRejected=True,sqlstate=r.stderr.strip().split(':')[-1].strip()))
def source():return sql(name,f"SELECT head_source_id::text FROM organization.business_unit WHERE bu_id='{obj}'")
def body(label):
 current,_=client.call('GET',path+'?selectedRecordId='+meta['members'][0]['recordId'])
 r=next(r for r in current['timeline'] if r['recordId']==meta['members'][0]['recordId'])
 return dict(expectedSourceVersion=current['headSourceVersion'],expectedRecordRevision=r['revision'],changes=dict(name=label))
def save(label,want=200):
 key=str(uuid.uuid4());reply,ms=client.call('POST',path+'/records/'+meta['members'][0]['recordId']+'/correct',body(label),key,want)
 if want==200:
  assert reply['outcome']=='accepted';receipts.append(dict(receipt=reply,route='/api/v1/organization/submissions/'+key+'/resolve',path=path))
 return key,reply
import http.client,socket,urllib.request,urllib.parse
conn=None
try:
 client=case.Client();client.base=server.start();v,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));client.csrf=v['csrf'];path='/api/v1/business-units/'+obj
 route=path+'/records/'+meta['members'][0]['recordId']+'/correct'
 def connection(request,key):
  u=urllib.parse.urlsplit(client.base);c=http.client.HTTPConnection(u.hostname,u.port,timeout=10)
  req=urllib.request.Request(client.base+route);client.jar.add_cookie_header(req)
  headers={'Content-Type':'application/json','Origin':client.base,'X-CSRF-Token':client.csrf,'Idempotency-Key':key,'Cookie':req.get_header('Cookie','')}
  c.request('POST',route,body=json.dumps(request,ensure_ascii=False).encode(),headers=headers);return c
 before=snapshot();request=body('真实头取消中断');key=str(uuid.uuid4());out['cancelSubmission']=key
 sql(name,"CREATE FUNCTION public.proof133_barrier() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock_shared(1330213::bigint);RETURN NEW;END $$; CREATE TRIGGER proof133_barrier BEFORE UPDATE ON organization.business_unit FOR EACH ROW WHEN(NEW.proof133_root IS DISTINCT FROM OLD.proof133_root) EXECUTE FUNCTION public.proof133_barrier()")
 barrier=subprocess.Popen(['docker','exec','-i',name,'psql','-X','-U','postgres','-d','storage126','-Atq','-v','ON_ERROR_STOP=1'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
 barrier.stdin.write('SELECT pg_advisory_lock(1330213::bigint);SELECT 1;\n');barrier.stdin.flush()
 while barrier.stdout.readline().strip()!='1':pass
 conn=connection(request,key);errors=[]
 def receive():
  try:conn.getresponse().read()
  except Exception as e:errors.append(type(e).__name__)
 thread=threading.Thread(target=receive);thread.start();start=time.monotonic()
 while int(sql(name,"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='chnhcm_api' AND wait_event='advisory'"))==0:
  assert time.monotonic()-start<3;time.sleep(.01)
 conn.sock.shutdown(socket.SHUT_RDWR);conn.close();thread.join(timeout=10);assert not thread.is_alive() and errors
 start=time.monotonic()
 while int(sql(name,"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='chnhcm_api' AND wait_event='advisory'")):
  assert time.monotonic()-start<5;time.sleep(.01)
 barrier.stdin.write('SELECT pg_advisory_unlock_all();\\q\n');barrier.stdin.flush();barrier.stdin.close();barrier.wait(timeout=5)
 sql(name,'DROP TRIGGER proof133_barrier ON organization.business_unit;DROP FUNCTION public.proof133_barrier()')
 assert snapshot()==before
 reply,_=client.call('POST','/api/v1/organization/submissions/'+key+'/resolve');assert reply['outcome']=='not_applied'
 out['checks'].append(dict(check='client cancellation at actual root UPDATE',originalSubmission=key,allPhysicalOrganizationRowsUnchanged=True,outcome='not_applied',transportError=errors))
 request=body('真实头响应丢失');key=str(uuid.uuid4());out['lostResponseSubmission']=key
 old=source();conn=connection(request,key);response=conn.getresponse();assert response.status==200
 conn.close() # discard the successful response body; recover by original ID
 reply,_=client.call('POST','/api/v1/organization/submissions/'+key+'/resolve');assert reply['outcome']=='accepted' and reply['sourceVersion']==source()!=old
 receipts.append(dict(receipt=reply,route='/api/v1/organization/submissions/'+key+'/resolve',path=path))
 after=snapshot();again,_=client.call('POST',receipts[-1]['route']);assert again['sourceVersion']==reply['sourceVersion'] and snapshot()==after
 out['checks'].append(dict(check='discarded response body recovered exactly once',originalSubmission=key,newSource=reply['sourceVersion'],duplicateBusinessSource=False,notDatabaseCommitAcknowledgmentFault=True))
 out['fullProof']=mode('headproof-check');out['completed']=True
except Exception as e:
 out['failures'].append(dict(type=type(e).__name__,message=str(e)));raise
finally:
 if conn is not None:conn.close()
 if barrier is not None and barrier.poll() is None:
  try:barrier.stdin.write('SELECT pg_advisory_unlock_all();\\q\n');barrier.stdin.flush();barrier.stdin.close();barrier.wait(timeout=5)
  except Exception:barrier.terminate();barrier.wait(timeout=5)
 if thread is not None:thread.join(timeout=10)
 try:sql(name,'DROP TRIGGER IF EXISTS proof133_barrier ON organization.business_unit;DROP FUNCTION IF EXISTS public.proof133_barrier()')
 except Exception as e:out['failures'].append(dict(cleanupError=type(e).__name__))
 server.close();(p/(label+'.json')).write_text(json.dumps(out,ensure_ascii=False,indent=2));(s/('receipts-'+label+'.json')).write_text(json.dumps(receipts,ensure_ascii=False,indent=2))
