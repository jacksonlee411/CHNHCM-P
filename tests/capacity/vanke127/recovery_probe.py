#!/usr/bin/env python3
"""Discard one real success reply, restart the API, and recover the exact batch."""
import argparse
import json
from pathlib import Path
import signal
import subprocess
import uuid

from load import Client, acknowledged_record, acknowledged_records, prepare_history, source_db
from parallel_history import dispatch
from runtime import BIN, ROOT, cli


class InjectedLostReply(ConnectionError):
    pass


class LoseSuccessfulReply:
    def __init__(self, client): self.client = client; self.discarded = False
    def call(self, *args):
        value = self.client.call(*args)
        if value[0] in (200, 201):
            self.discarded = True
            raise InjectedLostReply()
        return value


def run(private, source, evidence):
    db = source_db(private, source)
    assert not db.execute('SELECT jobs FROM history_batch WHERE id=1').fetchone(), 'existing batch must be recovered first'
    order = json.loads((private/'history-order.json').read_text())
    latest, _ = acknowledged_records(db, order)
    pending = {(p,n):(p,n,c,json.loads(v)) for p,n,c,v in db.execute("SELECT phase,row_no,code,payload FROM rows WHERE phase<>'create' AND result IS NULL")}
    rows = [pending[p,n] for p,n in order if (p,n) in pending]
    parents = {x[0] for x in db.execute("SELECT json_extract(payload,'$.parent_org_code') FROM rows WHERE phase='create' UNION SELECT json_extract(payload,'$.new_parent_org_code') FROM rows WHERE phase='parent-change'")}
    log = (private/'recovery-probe-server.log').open('a'); server = None
    def start():
        process = subprocess.Popen([str(BIN),'-mode','serve-import','-private',str(private)],cwd=ROOT,stdout=subprocess.PIPE,stderr=log,text=True)
        return process, json.loads(process.stdout.readline())['url']
    def stop():
        server.send_signal(signal.SIGTERM); server.wait(timeout=10)
    count = lambda: cli('size',private)['counts']['departmentSources']
    day = lambda r:r.get('target_effective_date') or r['effective_date']
    setup = 0
    try:
        server, base = start(); clients=[]
        for _ in range(2):
            c=Client(base);status,result,_=c.call('POST','/api/v1/session/login',dict(login='admin',password='admin1234567'));assert status==200;c.csrf=result['csrf'];clients.append(c)
        while rows:
            first=rows[0];selected=[first]
            if len(rows)>1:
                second=rows[1]
                if first[2] not in parents and second[2] not in parents and first[2]!=second[2] and first[0]==second[0] and day(first[3])==day(second[3]):selected.append(second)
            jobs=[]
            for phase,no,code,row in selected:
                path,body,adaptations=prepare_history(db,phase,code,row,latest[code])
                key=str(uuid.uuid5(uuid.NAMESPACE_URL,'chnhcm-vanke127/history-v1/'+phase+'/'+str(no)))
                jobs.append(dict(phase=phase,row=no,code=code,path=path,body=body,key=key,adaptations=adaptations,previous=latest[code]))
            db.execute('INSERT INTO history_batch VALUES(1,?)',(json.dumps(jobs,ensure_ascii=False),));db.commit()
            if len(jobs)==2:break
            results=dispatch(db,jobs,clients)
            latest[first[2]]=acknowledged_record(latest[first[2]],first[3],results[0][1]);setup+=1;rows=rows[1:]
        assert len(jobs)==2,'no eligible pair remained'
        before=count();lost=LoseSuccessfulReply(clients[0])
        try:
            dispatch(db,jobs,[lost,clients[1]])
        except RuntimeError:
            assert lost.discarded
        else:raise AssertionError('lost success did not retain journal')
        assert db.execute('SELECT count(*) FROM history_batch').fetchone()[0]==1
        assert all(db.execute('SELECT result FROM rows WHERE phase=? AND row_no=?',(j['phase'],j['row'])).fetchone()[0] is None for j in jobs)
        after_lost=count();stop();server,base=start()
        for c in clients:c.base=base
        results=dispatch(db,jobs,clients,parallel=False)
        after_recovery=count();assert after_lost==after_recovery
        accepted=sum(r[1]['outcome']=='accepted' for r in results);assert after_lost-before==accepted
        for job,(_,expected,_) in zip(jobs,results):
            status,value,_=clients[0].call('POST','/api/v1/departments/submissions/'+job['key']+'/resolve')
            assert status==200 and all(value.get(k)==expected.get(k) for k in ['submissionId','outcome','sourceVersion','observedSourceVersion','selectedRecordId'])
        assert not db.execute('SELECT jobs FROM history_batch WHERE id=1').fetchone()
        acknowledged_records(db,order)
        out=dict(scope='intentional client-side discard of one received successful reply',setupRows=setup,batchRows=2,lostSuccessInjected=True,otherSuccessDurablyRetained=True,apiRestarted=True,originalKeysAndBodiesReused=True,sourcesBefore=before,sourcesAfterLostReply=after_lost,sourcesAfterRecovery=after_recovery,duplicateSources=0,originalResultsResolved=2,chronologicalPrefixRestored=True,passed=True)
        evidence.write_text(json.dumps(out,indent=2)+'\n');print(json.dumps(out))
    finally:
        if server and server.poll() is None:stop()
        log.close();db.close()


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('source',type=Path);p.add_argument('evidence',type=Path);a=p.parse_args();run(a.private,a.source,a.evidence)
