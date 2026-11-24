-- Only the narrow ordering keys are sorted. The correlated scalar material
-- subplan runs above that sort and is consumed through a single PG cursor.
WITH RECURSIVE input AS NOT MATERIALIZED (SELECT $5::jsonb q), windows AS MATERIALIZED (
 SELECT (v->>'Start')::date AS start,nullif(v->>'End','')::date AS finish,(v->>'BU')::uuid AS bu
 FROM input,jsonb_array_elements(q->'windows') v
), descendants(id) AS (
 SELECT nullif(q->>'target','')::uuid FROM input WHERE q->>'target'<>''
 UNION
 SELECT d.department_id FROM descendants x JOIN organization.department d
 ON d.instance_id=$1 AND d.enterprise_id=$2 AND d.parents @> ARRAY[x.id]
), common(id) AS (
 SELECT id FROM descendants
 UNION SELECT v::uuid FROM input,jsonb_array_elements_text(q->'parents') v
 UNION SELECT p.id FROM common x CROSS JOIN LATERAL (
 SELECT d.parents FROM organization.department d WHERE d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id=x.id OFFSET 0
 ) d CROSS JOIN LATERAL unnest(d.parents) p(id)
), roots AS (
 SELECT 0 AS kind,''::text AS code,NULL::uuid AS id
 UNION ALL
 -- With full current reference/code rights and no search, segment labels
 -- cannot change membership or the immutable code ordering. Read them only
 -- in the same-snapshot frame below; do not probe every current segment.
 SELECT 1,d.code,d.department_id FROM organization.department d CROSS JOIN input
 WHERE coalesce(($4::jsonb #>> '{allowed,all}')::boolean,false)
 AND coalesce(($4::jsonb #>> '{fields,code,all}')::boolean,false) AND q->>'search'=''
 AND d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id::text<>q->>'target'
 AND d.management_bus && ARRAY(SELECT bu FROM windows)
 UNION ALL
 SELECT 1,CASE WHEN organization.q131_set(($4::jsonb)->'fields'->'code',s.q_bu,d.department_id) THEN d.code ELSE '' END,d.department_id
 FROM organization.department d JOIN organization.department_read_segment s
 ON s.instance_id=d.instance_id AND s.object_id=d.department_id AND s.span @> $3::date
 CROSS JOIN input
 WHERE NOT (coalesce(($4::jsonb #>> '{allowed,all}')::boolean,false)
 AND coalesce(($4::jsonb #>> '{fields,code,all}')::boolean,false) AND q->>'search'='')
 AND d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id::text<>q->>'target'
 AND d.management_bus && ARRAY(SELECT bu FROM windows)
 AND organization.q131_set(($4::jsonb)->'allowed',s.q_bu,d.department_id)
 AND (organization.q131_set(($4::jsonb)->'fields'->'code',s.q_bu,d.department_id) OR organization.q131_set(($4::jsonb)->'fields'->'name',s.q_bu,d.department_id))
 AND (q->>'search'='' OR q->>'search'=d.department_id::text
 OR strpos(CASE WHEN organization.q131_set(($4::jsonb)->'fields'->'code',s.q_bu,d.department_id) THEN s.q_code_key ELSE '' END
 || ' ' || CASE WHEN organization.q131_set(($4::jsonb)->'fields'->'name',s.q_bu,d.department_id) THEN s.q_name_key ELSE '' END,q->>'search')>0)
), positioned AS (
 SELECT r.* FROM roots r CROSS JOIN input
 WHERE r.kind=0 OR q->>'afterId'='' OR (r.code COLLATE "C",r.id)>((q->>'afterCode') COLLATE "C",nullif(q->>'afterId','')::uuid)
)
SELECT r.kind,r.code,coalesce(r.id::text,''),(
 WITH RECURSIVE needed(id) AS (
  SELECT unnest(CASE WHEN r.kind=0 THEN (SELECT array_agg(id) FROM common) ELSE ARRAY[r.id] END)
  UNION
  SELECT s.q_parent FROM needed n CROSS JOIN LATERAL (
   SELECT s.q_parent FROM organization.department_read_segment s WHERE s.instance_id=$1 AND s.object_id=n.id
   AND s.q_parent IS NOT NULL AND s.q_state<>'not_effective'
   AND (r.kind=0 OR EXISTS(SELECT 1 FROM windows w WHERE s.span && daterange(w.start,w.finish,'[)'))) OFFSET 0
  ) s
 ), labels(id) AS (
  SELECT r.id WHERE r.kind=1
  UNION SELECT nullif(q->>'selected','')::uuid FROM input WHERE r.kind=0 AND q->>'selected'<>''
  UNION SELECT nullif(q->>'target','')::uuid FROM input WHERE r.kind=0 AND q->>'target'<>''
 ), rows AS MATERIALIZED (
  SELECT s.*,
   (CASE WHEN EXISTS(SELECT 1 FROM needed n WHERE n.id=x.id) AND s.q_state<>'not_effective' AND (r.kind=0 OR EXISTS(SELECT 1 FROM windows w WHERE s.span && daterange(w.start,w.finish,'[)'))) THEN 3 ELSE 0 END
   + CASE WHEN EXISTS(SELECT 1 FROM labels l WHERE l.id=x.id) AND s.span @> $3::date THEN 1 ELSE 0 END) AS purpose
  FROM (SELECT id FROM needed UNION SELECT id FROM labels) x
  CROSS JOIN LATERAL (
   SELECT s.*,d.code,d.first_record_id,d.head_source_id,h.head_source_id AS relation_head
   FROM organization.department d
   JOIN organization.department_read_segment s ON s.instance_id=$1 AND s.object_id=x.id
   LEFT JOIN organization.department_hierarchy_relation h ON (h.instance_id,h.enterprise_id,h.department_id)=(d.instance_id,d.enterprise_id,d.department_id)
   WHERE d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id=x.id
   AND ((EXISTS(SELECT 1 FROM needed n WHERE n.id=x.id) AND s.q_state<>'not_effective' AND (r.kind=0 OR EXISTS(SELECT 1 FROM windows w WHERE s.span && daterange(w.start,w.finish,'[)'))))
    OR EXISTS(SELECT 1 FROM labels l WHERE l.id=x.id) AND s.span @> $3::date)
   OFFSET 0
  ) s
 ), bus(id) AS (
  SELECT DISTINCT q_bu FROM rows
  UNION SELECT (q->>'bu')::uuid FROM input WHERE r.kind=0
  UNION SELECT v::uuid FROM input,jsonb_array_elements_text(q->'bus') v WHERE r.kind=0
 )
 SELECT jsonb_build_object(
  'missing',EXISTS(SELECT 1 FROM needed n LEFT JOIN organization.department d
    ON d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id=n.id WHERE d.department_id IS NULL OR NOT EXISTS(SELECT 1 FROM organization.department_read_segment s WHERE s.instance_id=$1 AND s.object_id=n.id)),
  'conflict',r.kind=0 AND EXISTS(SELECT 1 FROM input WHERE q->>'target'<>'' AND NOT EXISTS(SELECT 1 FROM organization.department d
    WHERE d.instance_id=$1 AND d.enterprise_id=$2 AND d.department_id=nullif(q->>'target','')::uuid AND d.head_source_id=nullif(q->>'source','')::uuid)),
  'contextMissing',r.kind=0 AND NOT EXISTS(SELECT 1 FROM organization.business_unit b CROSS JOIN input WHERE b.instance_id=$1 AND b.enterprise_id=$2 AND b.bu_id::text=q->>'bu'),
  'departments',coalesce((SELECT jsonb_agg(jsonb_build_object('purpose',purpose,'fields',ARRAY[object_id::text,code,first_record_id::text,
   head_source_id::text,relation_head::text,source_id::text,q_record::text,q_revision::text,q_date::text,q_end::text,q_state,q_name,q_bu::text,q_parent::text])) FROM rows),'[]'::jsonb),
  'bus',coalesce((SELECT jsonb_agg(jsonb_build_object('fields',ARRAY[n.id::text,b.code,b.head_source_id::text,b.first_record_id::text,s.source_id::text],
    'ids',s.ids,'material',jsonb_build_object('words',d.words,'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries)v),
    'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges)v),'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges)v),
    'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(h.parts)v))))
   FROM bus n LEFT JOIN organization.business_unit b ON (b.instance_id,b.enterprise_id,b.bu_id)=($1::uuid,$2::uuid,n.id)
   LEFT JOIN organization.bu_read_segment s ON s.instance_id=$1 AND s.object_id=n.id
   LEFT JOIN organization.bu_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,s.dictionary_id)
   LEFT JOIN organization.bu_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,s.shape_id)),'[]'::jsonb)
 )
) AS material FROM positioned r ORDER BY r.kind,r.code COLLATE "C",r.id
