-- Full selection transfers native material from one statement.
), walk(id,day) AS (
 SELECT department_id,coalesce(q_path_date,$3::date) FROM paged WHERE path_ok
 UNION
 -- Follow the writer-maintained parent fact at the requested path day.
 -- Missing required facts fail capture; Go checks the resulting path.
 SELECT s.q_parent,w.day FROM walk w
 JOIN organization.department_read_segment s
  ON s.instance_id=$1::uuid AND s.object_id=w.id AND s.span @> w.day
 WHERE s.q_parent IS NOT NULL
), need_current AS MATERIALIZED (
 -- Every path dependency also needs its current authorization fact.
 SELECT id FROM walk
 UNION SELECT department_id FROM paged WHERE NOT path_ok
), need_other AS MATERIALIZED (
 SELECT id,day FROM walk WHERE day<>$3::date
), facts AS MATERIALIZED (
 SELECT $3::date AS day,c.department_id,c.code,c.head_source_id,c.first_record_id,c.relation_head,
 (c.segment).source_id,(c.segment).dictionary_id,(c.segment).ids,(c.segment).q_bu,(c.segment).shape_id
 FROM need_current n LEFT JOIN checked c ON c.department_id=n.id AND (c.segment).span @> $3::date
 UNION ALL
 SELECT n.day,c.department_id,c.code,c.head_source_id,c.first_record_id,c.relation_head,
 (c.segment).source_id,(c.segment).dictionary_id,(c.segment).ids,(c.segment).q_bu,(c.segment).shape_id
 FROM need_other n LEFT JOIN checked c ON c.department_id=n.id AND (c.segment).span @> n.day
), dictionary_use AS MATERIALIZED (
 SELECT dictionary_id,count(*) AS uses FROM facts GROUP BY dictionary_id
), department_material AS NOT MATERIALIZED (SELECT ARRAY[day::text,department_id::text,code,head_source_id::text,
  first_record_id::text,relation_head::text,source_id::text] AS item,
  CASE WHEN u.uses=1 THEN d.words END AS words,
  NULL::bytea[] AS binaries,
  CASE WHEN u.uses=1 THEN d.ranges END AS ranges,
  d.digest AS dictionary_digest,h.digest AS shape_digest,f.ids FROM facts f
 LEFT JOIN dictionary_use u ON u.dictionary_id=f.dictionary_id
 LEFT JOIN organization.department_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,f.dictionary_id)
 LEFT JOIN organization.department_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,f.shape_id)
), dictionaries AS MATERIALIZED (
 SELECT d.words,NULL::bytea[] AS binaries,d.ranges,d.digest
 FROM dictionary_use n
 LEFT JOIN organization.department_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,n.dictionary_id)
 WHERE n.uses>1
), shapes AS MATERIALIZED (
 SELECT h.parts,h.digest FROM (SELECT DISTINCT shape_id FROM facts) n
 LEFT JOIN organization.department_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,n.shape_id)
), bu_need AS MATERIALIZED (
 SELECT DISTINCT q_bu AS id,day FROM facts
), bu_material AS MATERIALIZED (
 SELECT ARRAY[n.id::text,n.day::text,b.enterprise_id::text,b.code,
  b.head_source_id::text,s.source_id::text] AS item,
  d.words,NULL::bytea[] AS binaries,d.ranges,h.parts,s.ids
 FROM bu_need n LEFT JOIN organization.business_unit b ON (b.instance_id,b.enterprise_id,b.bu_id)=($1::uuid,$2::uuid,n.id)
 LEFT JOIN organization.bu_read_segment s ON (s.instance_id,s.object_id)=($1::uuid,n.id) AND s.span @> n.day
 LEFT JOIN organization.bu_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,s.dictionary_id)
 LEFT JOIN organization.bu_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,s.shape_id)
)
SELECT 0 AS kind,ARRAY[json_build_object('objects',(SELECT count(DISTINCT department_id) FROM facts),
'segments',(SELECT count(*) FROM facts),'bad',0,'dictionaries',(SELECT count(DISTINCT dictionary_id) FROM facts),
'shapes',(SELECT count(DISTINCT shape_id) FROM facts),'denied',f.denied,'count',(SELECT count(*) FROM matching),
'enterpriseProfile',/*ENTERPRISE_ITEM*/,'missing',EXISTS(SELECT 1 FROM facts WHERE department_id IS NULL),
'page',coalesce((SELECT json_agg(department_id ORDER BY ordinal) FROM paged),'[]'::json))::text] AS item,
NULL::text[] words,NULL::bytea[] binaries,NULL::daterange[] ranges,NULL::bytea[] parts,NULL::bytea dictionary_digest,NULL::bytea shape_digest,NULL::uuid[] ids
FROM forbidden f
-- Shared immutable material is transferred once per capture.
-- Row order is not a protocol: Go retains any early facts until their material
-- arrives in this same MVCC statement, without a later material lookup.
-- A dictionary used by one fact travels inline exactly once. This avoids
-- retaining every unique dictionary when the data has little content sharing.
UNION ALL SELECT -2,NULL,words,binaries,ranges,NULL,digest,NULL,NULL FROM dictionaries
UNION ALL SELECT -1,NULL,NULL,NULL,NULL,parts,NULL,digest,NULL FROM shapes
-- Go decodes only these necessary source/path facts, without recomputing seals.
UNION ALL SELECT 3,item,words,binaries,ranges,NULL,dictionary_digest,shape_digest,ids FROM department_material
UNION ALL SELECT 2,item,words,binaries,ranges,parts,NULL,NULL,ids FROM bu_material
