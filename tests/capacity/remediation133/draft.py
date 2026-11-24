#!/usr/bin/env python3
"""G1 storage-only prototype on the explicitly owned full-size 133 fixture.

No product table, API, authorization decision or history is changed. UUIDs stay
in PostgreSQL. Output contains counts/costs, never business identifiers.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import sys
import subprocess
import threading
import time
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'tests/capacity/storage130'))
from growth import resource,sql

# Fixed before measurement: 20 x 87343 complete sets, 1024 IDs per chunk.
# Full current identities fit; no latency acceptance claim from this prototype.
BUDGET=dict(members=87343,sessions=20,apiConnections=12,chunkMembers=1024,
            maxPhysicalBytes=96*1024*1024,maxCreateWALBytes=160*1024*1024,
            maxPageChunkMembers=20*1024,ttlSeconds=1800,
            maxDraftMembers=131072,maxRequestEditMembers=1024)

def run(private,output):
 assert not output.exists()
 name=resource(private)
 assert sql(name,"SELECT count(*) FROM organization.department")=='87343'
 assert sql(name,"SELECT count(*) FROM pg_namespace WHERE nspname='probe133'")=='0'
 result=dict(budget=BUDGET,scope='isolated PostgreSQL storage prototype; no production selection/authorization delivery',completed=False)
 def save():output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
 save()
 sql(name,"""CREATE SCHEMA probe133;
 CREATE TABLE probe133.draft(id integer PRIMARY KEY,session_id integer NOT NULL,expires_at timestamptz NOT NULL,revision integer NOT NULL DEFAULT 1);
 CREATE TABLE probe133.chunk(draft integer NOT NULL REFERENCES probe133.draft ON DELETE CASCADE,first_id uuid NOT NULL,ids uuid[] NOT NULL,selected bit varying NOT NULL,PRIMARY KEY(draft,first_id));
 """)
 wal=sql(name,'SELECT pg_current_wal_insert_lsn()')
 barrier=threading.Barrier(20);slots=threading.Semaphore(12)
 def build(i):
  barrier.wait();start=time.perf_counter()
  with slots:
   sql(name,f"""BEGIN;INSERT INTO probe133.draft(id,session_id,expires_at)VALUES({i},{i},clock_timestamp()+interval '30 minutes');
   WITH ordered AS (SELECT department_id,row_number() OVER(ORDER BY department_id)-1 AS n FROM organization.department),
   chunks AS (SELECT n/1024 AS chunk,array_agg(department_id ORDER BY department_id) AS ids FROM ordered GROUP BY n/1024)
   INSERT INTO probe133.chunk SELECT {i},ids[1],ids,repeat('1',cardinality(ids))::bit varying FROM chunks;COMMIT;""")
  return (time.perf_counter()-start)*1000
 with ThreadPoolExecutor(max_workers=20) as p:times=list(p.map(build,range(1,21)))
 result['createMSIncludingPoolQueue']=times
 result['createWALBytes']=int(sql(name,f"SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),'{wal}')::bigint"))
 result['physicalBytes']=int(sql(name,"SELECT sum(pg_total_relation_size(oid)) FROM pg_class WHERE relnamespace='probe133'::regnamespace AND relkind='r'"))
 result['members']=int(sql(name,'SELECT sum(cardinality(ids)) FROM probe133.chunk'));save()
 assert result['members']==20*87343
 assert result['physicalBytes']<=BUDGET['maxPhysicalBytes'] and result['createWALBytes']<=BUDGET['maxCreateWALBytes']
 # Page membership locates at most one sorted chunk per page identity; no full
 # array crosses the DB boundary. Include widely separated IDs, not only page 1.
 sql(name,"CREATE TABLE probe133.page AS WITH selected AS (SELECT department_id,row_number()OVER(ORDER BY department_id) n FROM organization.department) SELECT department_id FROM selected WHERE (n-1)%4367=0 LIMIT 20")
 page="""SELECT count(*),sum(cardinality(c.ids)) FROM probe133.page p JOIN LATERAL(SELECT ids,selected FROM probe133.chunk WHERE draft=1 AND first_id<=p.department_id ORDER BY first_id DESC LIMIT 1)c ON true
 WHERE array_position(c.ids,p.department_id)IS NOT NULL AND get_bit(c.selected,array_position(c.ids,p.department_id)-1)=1"""
 start=time.perf_counter();counts=sql(name,page).split('|');result['pageMS']=1000*(time.perf_counter()-start);result['pageSelected']=int(counts[0]);result['pageChunkMembers']=int(counts[1]);assert result['pageSelected']==20 and result['pageChunkMembers']<=BUDGET['maxPageChunkMembers']
 # A normal owned-DB restart preserves logged identity intent, without involving a fixed service.
 subprocess.run(['docker','restart',name],check=True,capture_output=True)
 for attempt in range(50):
  try:
   restored=int(sql(name,'SELECT sum(cardinality(ids)) FROM probe133.chunk'));break
  except RuntimeError:
   time.sleep(.2)
 else:raise RuntimeError('owned prototype database did not recover')
 assert restored==20*87343;result['restartMembers']=restored;save()
 # CAS editing serializes same-draft intent, preserves identity and avoids lost edits.
 wal=sql(name,'SELECT pg_current_wal_insert_lsn()')
 def edit(_):return sql(name,"BEGIN;WITH changed AS(UPDATE probe133.draft SET revision=revision+1 WHERE id=1 AND session_id=1 AND revision=1 AND expires_at>clock_timestamp() RETURNING id) UPDATE probe133.chunk SET selected=set_bit(selected,0,0) WHERE draft IN(SELECT id FROM changed) AND first_id=(SELECT first_id FROM probe133.chunk WHERE draft=1 ORDER BY first_id LIMIT 1) RETURNING cardinality(ids);COMMIT")
 with ThreadPoolExecutor(max_workers=2)as p:edits=list(p.map(edit,range(2)))
 assert sum(bool(x)for x in edits)==1
 result['concurrentEditApplied']=1
 result['selectedAfterEdit']=int(sql(name,'SELECT sum(bit_count(selected)) FROM probe133.chunk'))
 assert result['selectedAfterEdit']==20*87343-1
 result['editWALBytes']=int(sql(name,f"SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),'{wal}')::bigint"))
 assert sql(name,'SELECT count(*) FROM probe133.draft WHERE id=1 AND session_id=2')=='0'
 sql(name,"UPDATE probe133.draft SET expires_at=clock_timestamp()-interval '1 second' WHERE id=2")
 assert sql(name,'SELECT count(*) FROM probe133.draft WHERE id=2 AND expires_at>clock_timestamp()')=='0'
 result['crossSessionAndExpiryRejected']=True
 start=time.perf_counter();sql(name,'DELETE FROM probe133.draft');result['cancelAllMS']=1000*(time.perf_counter()-start)
 assert sql(name,'SELECT count(*) FROM probe133.chunk')=='0'
 # Ordinary vacuum demonstrates bounded reusable space, not filesystem shrink.
 sql(name,'VACUUM probe133.chunk');result['bytesAfterVacuum']=int(sql(name,"SELECT pg_total_relation_size('probe133.chunk')"))
 result['completed']=True;save()
 sql(name,'DROP SCHEMA probe133 CASCADE')
 print(json.dumps({k:v for k,v in result.items()if k!='createMSIncludingPoolQueue'}))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);a=p.parse_args();run(a.private,a.output)
