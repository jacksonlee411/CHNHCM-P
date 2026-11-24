from pathlib import Path
import subprocess,sys,os,json,uuid,threading,time,signal
os.umask(0o077);p=Path(sys.argv[1]).resolve();s=p/'long-input';root=Path(__file__).resolve().parents[4];binary=Path(sys.argv[2]).resolve()
sys.path.insert(0,str(root/'tests/capacity/storage130'));from growth import case,resource,sql
label=sys.argv[3] if len(sys.argv)>3 else 'prototype-correctness';assert not (p/(label+'.json')).exists()
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
try:
 client=case.Client();client.base=server.start();v,_=client.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));client.csrf=v['csrf'];path='/api/v1/business-units/'+obj
 before=snapshot()
 reject('missing off-page member',"DELETE FROM organization.proof133_node WHERE left_hash IS NULL AND first_slot>8000")
 reject('replace leaf content',"UPDATE organization.proof133_node SET payload=payload WHERE left_hash IS NULL")
 reject('replace immutable source root',"UPDATE organization.proof133_source SET root=root")
 reject('increase published member count',f"UPDATE organization.business_unit SET proof133_count=proof133_count+1 WHERE bu_id='{obj}'")
 reject('remove current root',f"UPDATE organization.business_unit SET proof133_root=NULL WHERE bu_id='{obj}'")
 reject('wrong current source root',f"UPDATE organization.business_unit b SET proof133_root=s.root,proof133_count=s.members FROM organization.proof133_source s WHERE b.bu_id='{obj}' AND s.object_id=b.bu_id AND s.source_no=1")
 reject('wrong object ownership',"INSERT INTO organization.proof133_node(instance_id,object_id,payload,first_slot,members,first_date,last_date,left_hash,right_hash) SELECT instance_id,'00000000-0000-4000-8000-000000000001',payload,first_slot,members,first_date,last_date,left_hash,right_hash FROM organization.proof133_node LIMIT 1")
 reject('reordered child subtrees',"INSERT INTO organization.proof133_node(instance_id,object_id,payload,first_slot,members,first_date,last_date,left_hash,right_hash) SELECT instance_id,object_id,convert_to(jsonb_set(jsonb_set(convert_from(payload,'UTF8')::jsonb,'{Left}',convert_from(payload,'UTF8')::jsonb->'Right'),'{Right}',convert_from(payload,'UTF8')::jsonb->'Left')::text,'UTF8'),first_slot,members,first_date,last_date,right_hash,left_hash FROM organization.proof133_node WHERE left_hash IS NOT NULL LIMIT 1")
 reject('change real historical UUID',f"UPDATE organization.bu_history SET ids[1]='00000000-0000-4000-8000-000000000002'::uuid WHERE object_id='{obj}'")
 reject('change real current source',f"UPDATE organization.business_unit SET head_source_id='00000000-0000-4000-8000-000000000002'::uuid WHERE bu_id='{obj}'")
 assert snapshot()==before
 out['checks'].append(dict(check='bounded reader after SQL damage attempts',result=mode('headproof-page')))
 for late in [False,True]:
  before=snapshot();key=str(uuid.uuid4());request=body('真实头失败'+str(late))
  sql(name,"CREATE FUNCTION public.proof133_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled prototype publication failure';END $$")
  if late:sql(name,"CREATE CONSTRAINT TRIGGER proof133_fail AFTER UPDATE ON organization.business_unit DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION public.proof133_fail()")
  else:sql(name,"CREATE TRIGGER proof133_fail BEFORE UPDATE ON organization.business_unit FOR EACH ROW WHEN(NEW.proof133_root IS DISTINCT FROM OLD.proof133_root) EXECUTE FUNCTION public.proof133_fail()")
  client.call('POST',path+'/records/'+meta['members'][0]['recordId']+'/correct',request,key,503)
  sql(name,'DROP TRIGGER proof133_fail ON organization.business_unit; DROP FUNCTION public.proof133_fail()')
  assert snapshot()==before
  reply,_=client.call('POST','/api/v1/organization/submissions/'+key+'/resolve');assert reply['outcome']=='not_applied'
  out['checks'].append(dict(check='late commit rollback' if late else 'root publication rollback',allPhysicalOrganizationRowsUnchanged=True,originalSubmission=key,outcome='not_applied'))
 # Hold the real final root UPDATE while the real head/history are uncommitted.
 sql(name,"CREATE FUNCTION public.proof133_barrier() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock_shared(1330213::bigint);RETURN NEW;END $$; CREATE TRIGGER proof133_barrier BEFORE UPDATE ON organization.business_unit FOR EACH ROW WHEN(NEW.proof133_root IS DISTINCT FROM OLD.proof133_root) EXECUTE FUNCTION public.proof133_barrier()")
 barrier=subprocess.Popen(['docker','exec','-i',name,'psql','-X','-U','postgres','-d','storage126','-Atq','-v','ON_ERROR_STOP=1'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
 barrier.stdin.write('SELECT pg_advisory_lock(1330213::bigint); SELECT 1;\n');barrier.stdin.flush()
 while barrier.stdout.readline().strip()!='1':pass
 old=source();reply=[];errors=[]
 def write():
  try:reply.append(save('真实头并发完整新版'))
  except Exception as e:errors.append(type(e).__name__)
 thread=threading.Thread(target=write);thread.start();start=time.monotonic()
 while int(sql(name,"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='chnhcm_api' AND wait_event='advisory'"))==0:
  assert time.monotonic()-start<2;time.sleep(.01)
 page=mode('headproof-page');assert page['source']==old and page['returned']==50
 barrier.stdin.write('SELECT pg_advisory_unlock(1330213::bigint);\\q\n');barrier.stdin.flush();barrier.stdin.close();barrier.wait(timeout=5)
 thread.join(timeout=10);assert not thread.is_alive() and not errors
 sql(name,'DROP TRIGGER proof133_barrier ON organization.business_unit; DROP FUNCTION public.proof133_barrier()')
 assert mode('headproof-page')['source']==reply[0][1]['sourceVersion']!=old
 out['checks'].append(dict(check='concurrent reader',oldSource=old,newSource=reply[0][1]['sourceVersion'],oldAndNewComplete=True))
 out['fullProof']=mode('headproof-check');out['completed']=True
except Exception as e:
 out['failures'].append(dict(type=type(e).__name__,message=str(e)));raise
finally:
 if barrier is not None and barrier.poll() is None:
  try:barrier.stdin.write('SELECT pg_advisory_unlock_all();\\q\n');barrier.stdin.flush();barrier.stdin.close();barrier.wait(timeout=5)
  except Exception:barrier.terminate();barrier.wait(timeout=5)
 if thread is not None:thread.join(timeout=10)
 try:sql(name,'DROP TRIGGER IF EXISTS proof133_fail ON organization.business_unit; DROP TRIGGER IF EXISTS proof133_barrier ON organization.business_unit; DROP FUNCTION IF EXISTS public.proof133_fail(); DROP FUNCTION IF EXISTS public.proof133_barrier()')
 except Exception as e:out['failures'].append(dict(cleanupError=type(e).__name__))
 server.close();(p/(label+'.json')).write_text(json.dumps(out,ensure_ascii=False,indent=2));(s/('receipts-'+label+'.json')).write_text(json.dumps(receipts,ensure_ascii=False,indent=2))
