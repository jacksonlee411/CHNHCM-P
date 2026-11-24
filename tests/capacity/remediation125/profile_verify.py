#!/usr/bin/env python3
"""Verify only the new retained model-3 candidate; never switch old entries."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import uuid

ROOT=Path(__file__).resolve().parents[3]
PROFILE='m02-current-vanke-b'
sys.path.insert(0,str(ROOT/'scripts'))

def run(handoff,output):
    import local_runtime as runtime
    import importlib.util
    spec=importlib.util.spec_from_file_location('compact_http',ROOT/'tests/capacity/storage126/measure.py')
    case=importlib.util.module_from_spec(spec);spec.loader.exec_module(case)
    info=json.loads(handoff.read_text());assert info['profile']==PROFILE and info['allStoredRowsPreserved']
    assert runtime.PROFILE==PROFILE
    assert info['url']==runtime.connection()['webOrigin']
    private=Path(info['privateDirectory']);assert ROOT not in private.parents
    runtime.verify_target(runtime.connection())
    owned=runtime.database_container();assert owned and owned['running']
    container=owned['id']
    def sql(statement):
        p=subprocess.run(['docker','exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],input=statement,text=True,capture_output=True)
        if p.returncode:raise RuntimeError('owned verification SQL failed')
        return p.stdout.strip()
    def counts():
        return json.loads(sql("SELECT json_build_object('model',(SELECT contract FROM organization.storage_contract),'migrations',(SELECT count(*) FROM public.goose_db_version WHERE is_applied AND version_id>0),'BU',(SELECT count(*) FROM organization.business_unit),'department',(SELECT count(*) FROM organization.department),'buSources',(SELECT sum(cardinality(source_refs)) FROM organization.bu_history),'departmentSources',(SELECT sum(cardinality(source_refs)) FROM organization.department_history),'securityEvents',(SELECT count(*) FROM access.security_event),'databaseBytes',pg_database_size(current_database()))"))
    def clients():
        values={}
        for role in ['admin','reader']:
            c=case.Client();c.base=info['url'];login,_=c.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'));c.csrf=login['csrf'];values[role]=c
        return values
    def build():
        directory=runtime.BIN;manifest=json.loads((directory/'fixed-build.json').read_text())
        assert hashlib.sha256((directory/'server').read_bytes()).hexdigest()==manifest['apiSHA256']
        assert all(hashlib.sha256((directory/'web'/name).read_bytes()).hexdigest()==digest for name,digest in manifest['webSHA256'].items())
        return dict(apiSHA256=manifest['apiSHA256'],webFiles=manifest['webSHA256'],baseCommit=manifest['baseCommit'])
    result=dict(completed=False,profile=PROFILE,url=info['url'],oldFixedEntriesChanged=False,checks=[],menuEntries=[])
    save=lambda:output.write_text(json.dumps(result,indent=2)+'\n')
    try:
        result['before']=counts();assert result['before']['model']=='m02-compact-3' and result['before']['migrations']==7
        result['fixedBuildBefore']=build()
        c=clients();baseline=counts()
        # Original submission identities come from this profile's private copy
        # of the acknowledged import log, independently of the live response.
        sys.path.insert(0,str(ROOT/'tests/capacity/vanke127'))
        from verify import Oracle
        oracle=Oracle(private)
        codes=sorted(code for code,value in oracle.objects.items() if value['kind']=='departments')
        sample=codes[0];original=oracle.frames[oracle.events[sample][0][1]]
        def original_result(client):
            v,_=client.call('POST','/api/v1/departments/submissions/'+original['submission']+'/resolve')
            assert v['outcome']=='accepted' and v['sourceVersion']==original['source'] and v['targetRecordId']==original['target']
        for role,client in c.items():
            context,_=client.call('GET','/api/v1/departments/context');assert context['capabilities']['create']==(role=='admin')
            for segment,count in [('business-units',1),('departments',87343)]:
                for _ in range(3):
                    v,_=client.call('GET','/api/v1/'+segment);assert v['count']==count and len(v['items'])==min(count,20)
                    if segment=='business-units': assert len(v['selectionVersion'])==64
                    else: assert 'selectionVersion' not in v
            old,_=client.call('GET','/api/v1/departments/'+oracle.objects[sample]['id']+'?sourceVersion='+original['source'])
            oracle.check_detail(sample,old,original['source'])
        original_result(c['admin'])
        after=counts();assert after['securityEvents']==baseline['securityEvents']
        result['readSecurityEventGrowth']=0
        result['checks'].append(dict(kind='both-role-current-source-context-original-result',passed=True))
        parents={row[2] for frame in oracle.frames.values() for row in frame['records'] if row[2]}
        selected=next(code for code in codes if code not in parents)
        path='/api/v1/departments/'+oracle.objects[selected]['id']
        before,_=c['admin'].call('GET',path);record=before['selectedRecord'];key=str(uuid.uuid4())
        changed,_=c['admin'].call('POST',path+'/records/'+record['recordId']+'/correct',dict(expectedSourceVersion=before['headSourceVersion'],expectedRecordRevision=record['revision'],changes=dict(shortName='125交接核验')),key)
        assert changed['outcome']=='accepted'
        current,_=c['admin'].call('GET',path)
        assert current['selectedRecord']['shortName']=='125交接核验' and current['selectedRecord']['name']==record['name']
        restored,_=c['admin'].call('POST',path+'/records/'+record['recordId']+'/correct',dict(expectedSourceVersion=changed['sourceVersion'],expectedRecordRevision=changed['targetRevision'],changes=dict(shortName=record.get('shortName'))),str(uuid.uuid4()))
        assert restored['outcome']=='accepted'
        receipt_path=private/'125-handoff-receipt.json'
        fd=os.open(receipt_path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
        with os.fdopen(fd,'w') as f:json.dump(dict(key=key,receipt=changed,restored=restored),f)
        result['checks'].append(dict(kind='real-save-readback-restore',passed=True))
        before_restart=counts()
        log=private/'125-profile-restart.log'
        with log.open('x') as f:
            log.chmod(0o600)
            for command in ['down','up']:
                subprocess.run([sys.executable,str(ROOT/'scripts/local-bu.py'),command]+(['--fixed-web'] if command=='up' else []),cwd=ROOT,check=True,stdout=f,stderr=subprocess.STDOUT)
        runtime.verify_target(runtime.connection());c=clients();after_restart=counts()
        assert all(before_restart[k]==after_restart[k] for k in ['model','migrations','BU','department','buSources','departmentSources'])
        recovered,_=c['admin'].call('POST','/api/v1/departments/submissions/'+key+'/resolve')
        assert all(recovered.get(k)==v for k,v in changed.items() if k!='currentChanged') and recovered['currentChanged'] is True
        original_result(c['admin'])
        assert counts()['securityEvents']==after_restart['securityEvents']
        result['fixedBuildAfter']=build();assert result['fixedBuildBefore']==result['fixedBuildAfter']
        result['checks'].append(dict(kind='whole-profile-restart-same-business-sources-original-results-fixed-build',passed=True))
        for role in ['admin','reader']:
            for segment,count in [('business-units',1),('departments',87343)]:
                request=dict(url=info['url'],role=role,segment=segment,count=count,screenshot=str(private/f'125-fixed-menu-{role}-{segment}.png'))
                p=subprocess.run(['node',str(ROOT/'tests/capacity/remediation125/menu.mjs')],input=json.dumps(request),text=True,capture_output=True)
                if p.returncode:
                    log=private/f'125-fixed-menu-{role}-{segment}-failure.txt';log.write_text(p.stderr);log.chmod(0o600)
                    result['menuEntries'].append(dict(role=role,segment=segment,passed=False))
                else:result['menuEntries'].append(json.loads(p.stdout))
                save()
        result['after']=counts();result['completed']=True
    finally:
        result['passed']=result['completed'] and all(c['passed'] for c in result['checks']+result['menuEntries']);save()
    if not result['passed']:raise SystemExit(1)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('handoff',type=Path);p.add_argument('output',type=Path);a=p.parse_args()
    assert os.environ.get('CHNHCM_LOCAL_PROFILE')==PROFILE and not a.output.exists()
    run(a.handoff,a.output)
