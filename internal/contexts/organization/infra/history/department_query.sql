), walk(root,id,day,visited,cycle) AS (
 SELECT department_id,department_id,coalesce(q_path_date,$3::date),ARRAY[department_id],false FROM paged WHERE path_ok
 UNION ALL
 SELECT w.root,(c.segment).q_parent,w.day,w.visited||(c.segment).q_parent,(c.segment).q_parent=ANY(w.visited)
 FROM walk w JOIN checked c ON c.department_id=w.id AND (c.segment).span @> w.day
 WHERE NOT w.cycle AND (c.segment).q_parent IS NOT NULL
), need AS MATERIALIZED (
 SELECT department_id AS id,$3::date AS day FROM paged
 UNION SELECT id,day FROM walk
 UNION SELECT id,$3::date FROM walk
), facts AS MATERIALIZED (
 SELECT n.day,c.* FROM need n LEFT JOIN checked c ON c.department_id=n.id AND (c.segment).span @> n.day
), department_material AS MATERIALIZED (
 SELECT day,department_id, /*DEPARTMENT_ITEM*/ || jsonb_build_object('queryDay',day) AS item
 FROM facts
 LEFT JOIN organization.department_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,(segment).dictionary_id)
 LEFT JOIN organization.department_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,(segment).shape_id)
), bu_need AS MATERIALIZED (
 SELECT DISTINCT (segment).q_bu AS id,day FROM facts
), bu_material AS MATERIALIZED (
 SELECT n.id,n.day,jsonb_build_object(
  'id',n.id,'day',n.day,'enterprise',b.enterprise_id,'code',b.code,'source',b.head_source_id,'segmentSource',s.source_id,
  'ids',s.ids,
  'valid',s.source_id=b.head_source_id,
  '_material',jsonb_build_object('words',d.words,
   'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries) v),
   'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges) v),
   'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges) v),
   'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(h.parts) v))
 ) AS item
 FROM bu_need n LEFT JOIN organization.business_unit b ON (b.instance_id,b.enterprise_id,b.bu_id)=($1::uuid,$2::uuid,n.id)
 LEFT JOIN organization.bu_read_segment s ON (s.instance_id,s.object_id)=($1::uuid,n.id) AND s.span @> n.day
 LEFT JOIN organization.bu_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,s.dictionary_id)
 LEFT JOIN organization.bu_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,s.shape_id)
)
SELECT (SELECT count(DISTINCT department_id) FROM facts),(SELECT count(*) FROM facts),0::bigint,
 (SELECT count(DISTINCT (segment).dictionary_id) FROM facts),
 (SELECT count(DISTINCT (segment).shape_id) FROM facts),f.denied,
 (SELECT count(*) FROM matching),jsonb_build_object(
  'enterpriseProfile',/*ENTERPRISE_ITEM*/,
  'cycle',EXISTS(SELECT 1 FROM walk WHERE cycle),
  'missing',EXISTS(SELECT 1 FROM facts WHERE department_id IS NULL),
  'page',coalesce((SELECT jsonb_agg(department_id ORDER BY ordinal) FROM paged),'[]'::jsonb),
  'departments',coalesce((SELECT jsonb_agg(item) FROM department_material),'[]'::jsonb),
  'bus',coalesce((SELECT jsonb_agg(item) FROM bu_material),'[]'::jsonb))
FROM forbidden f
