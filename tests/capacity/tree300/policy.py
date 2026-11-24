"""Temporary sparse-read fixture, guarded and restored on the owned database."""
from contextlib import contextmanager
import hashlib
import json
import os
import subprocess
import uuid
from pathlib import Path
from frozen import owned_database as frozen_database

ROOT=Path(__file__).resolve().parents[3]

@contextmanager
def sparse_reads(private, session, allowed, label, container=None):
    if allowed is None:
        yield;return
    frozen=frozen_database(private,container)
    if frozen:
        container,database=frozen
        task='storage126-product' if container=='chnhcm-storage126-product' else 'storage130'
        expected={'io.chnhcm.task':task,'io.chnhcm.parent-task':'storage130','io.chnhcm.temporary':'true'}
    elif (private/'tree300-synthetic.json').exists():
        profile='tree300-synthetic-d';project='chnhcm-bu-'+hashlib.sha256((str(ROOT)+':'+profile).encode()).hexdigest()[:12]
        assert container==project+'-postgres-1';database='chnhcm'
        expected={'io.chnhcm.profile':profile,'io.chnhcm.temporary':'true'}
    else:
        assert container in [None,'chnhcm-vanke127'];container='chnhcm-vanke127';database='vanke127'
        expected={'io.chnhcm.task':'vanke127','io.chnhcm.temporary':'true'}
    labels=json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Config.Labels}}',container],text=True))
    assert all(labels.get(k)==v for k,v in expected.items()),'wrong database owner'
    def sql(statement):
        p=subprocess.run(['docker','exec','-i',container,'psql','-U','postgres','-d',database,'-X','-Atq','-v','ON_ERROR_STOP=1'],input=statement,text=True,capture_output=True)
        if p.returncode:raise RuntimeError('private sparse fixture SQL failed')
        return p.stdout.strip()
    instance,subject=str(uuid.UUID(session['instanceId'])),str(uuid.UUID(session['subjectId']))
    where=f"instance_id='{instance}' AND principal_id='{subject}'"
    snapshot=sql("SELECT json_agg(g) FROM access.subject_grant g WHERE "+where+" AND action='department.read'")
    assert len(json.loads(snapshot))==1,'fixture expects the original one-read-grant reader'
    with os.fdopen(os.open(private/(label+'-sparse-grants.json'),os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'w') as f:f.write(snapshot)
    original="json_populate_recordset(NULL::access.subject_grant,'"+snapshot.replace("'","''")+"')"
    ids=','.join("'"+str(uuid.UUID(value))+"'::uuid" for value in sorted(allowed))
    def mutate(statement):sql(f'BEGIN;SELECT principal_id FROM access.policy_guard WHERE {where} FOR UPDATE;{statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE {where};COMMIT;')
    try:
        mutate(f"DELETE FROM access.subject_grant WHERE {where} AND action='department.read'; INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),principal_id,'department',id,action,read_fields,write_fields FROM {original} g CROSS JOIN unnest(ARRAY[{ids}]) id")
        yield
    finally:mutate(f"DELETE FROM access.subject_grant WHERE {where} AND action='department.read'; INSERT INTO access.subject_grant SELECT * FROM {original}")
