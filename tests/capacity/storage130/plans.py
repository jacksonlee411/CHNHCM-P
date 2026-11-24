#!/usr/bin/env python3
"""Capture real read plans and nonexecuted write plans on explicitly owned A/B."""
import argparse
import json
import re
import uuid
from pathlib import Path

from growth import resource, sql


def run(private,output):
    assert not output.exists()
    plans={}
    for role in ['a','b']:
        directory=private/role;name=resource(directory)
        cfg=json.loads((directory/'candidate-private.json').read_text());instance=str(uuid.UUID(cfg['Instance']))
        owners=json.loads((directory/'owners.json').read_text())
        queries={'complete_source_heads':f"SELECT * FROM organization.list_source_heads('{instance}'::uuid)"}
        for kind,table,key in [('bu','business_unit','bu_id'),('department','department','department_id')]:
            selected=owners['bu' if kind=='bu' else 'dep'][:20]
            body=json.dumps([dict(id=str(uuid.UUID(row[key])),date='2026-11-08') for row in selected]).replace("'","''")
            queries[kind+'_range_read']=f"""SELECT b.{key}::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,coalesce(lower(s.span)::text,''),coalesce(upper(s.span)::text,''),s.rule,s.dictionary_id,s.shape_id,s.ids::text[],s.envelope
 FROM jsonb_to_recordset('{body}'::jsonb) w(id uuid,date date)
 JOIN organization.{table} b ON b.instance_id='{instance}'::uuid AND b.{key}=w.id
 JOIN LATERAL (SELECT s.* FROM organization.{kind}_read_segment s
 WHERE s.instance_id=b.instance_id AND s.object_id=b.{key} AND s.source_id=b.head_source_id
 AND coalesce(lower(s.span),'-infinity'::date)<=w.date AND s.span @> w.date
 ORDER BY coalesce(lower(s.span),'-infinity'::date) DESC LIMIT 1) s ON true"""
        result={}
        for label,query in queries.items():
            raw=sql(name,'EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) '+query)
            result[label]=json.loads(re.sub(r'[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}',':uuid',raw))
        for table,key,label in [('department','department_id','department_head_update'),('department_hierarchy_relation','department_id','relation_head_update')]:
            oid=str(uuid.UUID(owners['dep'][0]['department_id']))
            raw=sql(name,f"EXPLAIN (FORMAT JSON) UPDATE organization.{table} SET head_source_id=head_source_id WHERE instance_id='{instance}'::uuid AND {key}='{oid}'::uuid")
            result[label]=json.loads(re.sub(r'[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}',':uuid',raw))
        plans[role]=result
    output.write_text(json.dumps(dict(roles=plans,writePlansExecuted=False,
        scope='real complete-head and production-shaped range-read SQL; EXPLAIN ANALYZE only on reads; write plans explained without execution; UUID constants redacted'),indent=2)+'\n')


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('output',type=Path);a=p.parse_args();run(a.private,a.output)
