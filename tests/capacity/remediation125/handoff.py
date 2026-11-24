#!/usr/bin/env python3
"""Prepare exactly one NEW, retained model-3 Vanke review profile.
The two original profiles are never targets. Source identities come from this
task's independently rebuilt disposable case, not from either original DB.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT=Path(__file__).resolve().parents[3]
PROFILE='m02-current-vanke-b'

def main(private,output):
    assert not output.exists()
    assert private.is_absolute() and ROOT not in private.parents and private.stat().st_mode&0o777==0o700
    source=json.loads((private/'database-private.json').read_text())
    labels=subprocess.check_output(['docker','inspect','--format','{{index .Config.Labels "io.chnhcm.task"}}/{{index .Config.Labels "io.chnhcm.temporary"}}','chnhcm-vanke127'],text=True).strip()
    assert labels=='vanke127/true'
    os.environ['CHNHCM_LOCAL_PROFILE']=PROFILE
    sys.path.insert(0,str(ROOT/'scripts'))
    import local_runtime as runtime
    spec=importlib.util.spec_from_file_location('handoff_local',ROOT/'scripts/local-bu.py')
    local=importlib.util.module_from_spec(spec);spec.loader.exec_module(local)
    with runtime.lifecycle_lock():
        assert not runtime.CONFIG.exists() and not runtime.database_container(),'existing retained profile must be preserved'
        runtime.configure(dict(web=4390,api=8390,database=54390))
        runtime.register_profile(temporary=False,purpose='125当前模型万科独立观察候选；容量及交接分别验收，原固定入口未切换；负责人wt-dev-b，2026-11-09复查')
        config=local.configuration(create=True)
        config['instance']=source['Instance']
        config['enterprise']=dict(EnterpriseCode='VANKE127',EnterpriseName='组织历史验证企业',Start='1984-01-01')
        runtime.write_private(runtime.CONFIG,config)
    subprocess.run([sys.executable,str(ROOT/'scripts/local-bu.py'),'up','--no-web'],cwd=ROOT,check=True)
    with runtime.lifecycle_lock():
        connection=runtime.connection();runtime.verify_api(connection);runtime.stop_processes(['api'])
        owned=runtime.database_container()
        assert owned and owned['running']
        container=owned['id']
        raw=subprocess.check_output(['docker','inspect','--format','{{json .Config.Labels}}',container],text=True)
        labels=json.loads(raw);assert labels['io.chnhcm.profile']==PROFILE and labels['io.chnhcm.temporary']=='false'
    def target(sql):
        p=subprocess.run(['docker','exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],input=sql,text=True,capture_output=True)
        if p.returncode:raise RuntimeError('new candidate SQL preparation failed; private runtime retained')
        return p.stdout.strip()
    # Only initializer rows exist in this profile. Refuse to replace any object.
    assert target('SELECT (SELECT count(*) FROM organization.business_unit)+(SELECT count(*) FROM organization.department)')=='0'
    def fingerprints(container,database):
        # Hash every complete stored row, not just heads or row counts. The
        # output contains table names and digests, never customer values.
        statement="""SELECT format($command$SELECT %L AS table_name,count(*) AS rows,md5(coalesce(string_agg(md5(to_jsonb(t)::text),'' ORDER BY md5(to_jsonb(t)::text)),'')) AS digest FROM %I.%I t;$command$,n.nspname||'.'||c.relname,n.nspname,c.relname)
          FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
          WHERE c.relkind='r' AND n.nspname IN ('organization','access')
          ORDER BY n.nspname,c.relname;"""
        command=['docker','exec','-i',container,'psql','-U','postgres','-d',database,'-X','-Atq','-v','ON_ERROR_STOP=1']
        queries=subprocess.check_output(command,input=statement,text=True)
        rows=subprocess.check_output(command,input=queries,text=True).splitlines()
        return [dict(table=parts[0],rows=int(parts[1]),rowDigestMD5=parts[2]) for row in rows if (parts:=row.split('|'))]
    original=fingerprints('chnhcm-vanke127','vanke127')
    archive=private/'125-model3-handoff.dump'
    with os.fdopen(os.open(archive,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'wb') as f:
        subprocess.run(['docker','exec','chnhcm-vanke127','pg_dump','-U','postgres','-d','vanke127','--format=custom','--data-only','--schema=organization','--schema=access'],stdout=f,check=True)
    target("DO $$ DECLARE names text; BEGIN SELECT string_agg(format('%I.%I',n.nspname,c.relname),',') INTO names FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access');EXECUTE 'TRUNCATE TABLE '||names||' CASCADE';END $$")
    with archive.open('rb') as f:
        subprocess.run(['docker','exec','-i',container,'pg_restore','-U','postgres','-d','chnhcm','--data-only','--disable-triggers','--exit-on-error'],stdin=f,check=True,stdout=subprocess.DEVNULL)
    target('VACUUM (ANALYZE)')
    copied=fingerprints(container,'chnhcm')
    assert original==copied,'new candidate did not preserve every stored row'
    assert target('SELECT contract FROM organization.storage_contract')=='m02-compact-3'
    counts=json.loads(target("SELECT json_build_object('BU',(SELECT count(*) FROM organization.business_unit),'department',(SELECT count(*) FROM organization.department),'buSources',(SELECT sum(cardinality(source_refs)) FROM organization.bu_history),'departmentSources',(SELECT sum(cardinality(source_refs)) FROM organization.department_history),'databaseBytes',pg_database_size(current_database()))"))
    assert counts['BU']==1 and counts['department']==87343
    # Retain the mapping inputs beside the new profile, outside the checkout.
    destination=runtime.STATE/'125-vanke'
    destination.mkdir(mode=0o700)
    for name in ['source-rows.jsonl','import.sqlite','history-order.json']:
        shutil.copyfile(private/name,destination/name);(destination/name).chmod(0o600)
    env=local.environment(config,connection)
    runtime.write_private(destination/'database-private.json',dict(AdminDSN=env['CHNHCM_ADMIN_DATABASE_URL'],APIDSN=env['CHNHCM_DATABASE_URL'],Instance=config['instance'],Profile=PROFILE))
    subprocess.run([sys.executable,str(ROOT/'scripts/local-bu.py'),'up','--fixed-web'],cwd=ROOT,check=True)
    with archive.open('rb') as f:archive_digest=hashlib.file_digest(f,'sha256').hexdigest()
    result=dict(profile=PROFILE,project=runtime.PROJECT,ports=connection['ports'],url=connection['webOrigin'],model='m02-compact-3',readRule=2,privateDirectory=str(destination),source='new identities imported from the original full source rows into this task-owned candidate, then copied at the same model',originalPhysicalDatabaseRestored=False,oldFixedEntriesChanged=False,temporary=False,owner='wt-dev-b',reviewDate='2026-11-09',storage=counts,archiveSHA256=archive_digest,allStoredRowsPreserved=original==copied,tableFingerprints=copied)
    output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(profile=PROFILE,url=result['url'],model=result['model'],oldFixedEntriesChanged=False)))

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);args=p.parse_args();main(args.private,args.output)
