-- The Go candidate supplies the lost active periods. This statement only
-- selects a stored overlapping reference and captures all its immutable bytes.
-- A confirmed conflict needs one witness; a zero result exhausts the scope.
WITH bu AS MATERIALIZED (
 SELECT enterprise_id,bu_id,head_source_id FROM organization.business_unit
 WHERE instance_id=$1 AND bu_id=$2
), windows AS MATERIALIZED (
 SELECT (v->>'Start')::date AS start,nullif(v->>'End','')::date AS finish
 FROM jsonb_array_elements($3::jsonb) v
)
SELECT b.head_source_id::text, (
 SELECT jsonb_build_object(
  'id',d.department_id,'enterprise',d.enterprise_id,'first',d.first_record_id,
  'source',d.head_source_id,'number',d.head_no,'relation',r.head_source_id,
  'record',s.q_record,'revision',s.q_revision,'start',s.q_date,'end',coalesce(s.q_end::text,''),
  'bu',s.q_bu,'state',s.q_state,
  'valid',(s.source_id=d.head_source_id AND s.rule=4 AND s.q_relation=r.head_source_id
    AND s.q_owner_seal=organization.q131_owner_seal(d.enterprise_id,d.code,d.first_record_id,r.head_source_id)
    AND s.q_dictionary_seal=sd.digest AND s.q_shape_seal=ss.digest AND s.q_seal=s.q_stored_seal) IS TRUE,
  'ids',organization.project_ids(h.ids,d.current_refs),
  'envelope',encode(d.current_envelope,'hex'),
  'dictionaryDigest',encode(cd.digest,'hex'),'shapeDigest',encode(cs.digest,'hex'),
  'material',jsonb_build_object('words',cd.words,'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(cd.binaries) v),
    'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(cd.ranges) v),
    'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(cd.ranges) v),
    'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(cs.parts) v)))
 FROM (
  SELECT s.* FROM organization.department owner
  CROSS JOIN LATERAL (
   SELECT s.* FROM organization.department_read_segment s
   WHERE s.instance_id=owner.instance_id AND s.object_id=owner.department_id
    AND s.q_bu=$2::uuid AND s.q_state='active'
    AND EXISTS(SELECT 1 FROM windows w WHERE s.span && daterange(w.start,w.finish,'[)'))
   LIMIT 1
  ) s
  WHERE owner.instance_id=$1 AND owner.enterprise_id=b.enterprise_id
   AND owner.management_bus @> ARRAY[$2::uuid]
  LIMIT 1
 ) s
 LEFT JOIN organization.department d ON d.instance_id=$1 AND d.enterprise_id=b.enterprise_id AND d.department_id=s.object_id
 LEFT JOIN organization.department_hierarchy_relation r ON (r.instance_id,r.enterprise_id,r.department_id)=(d.instance_id,d.enterprise_id,d.department_id)
 LEFT JOIN organization.department_dictionary sd ON (sd.instance_id,sd.enterprise_id,sd.id)=(d.instance_id,d.enterprise_id,s.dictionary_id)
 LEFT JOIN organization.department_shape ss ON (ss.instance_id,ss.enterprise_id,ss.id)=(d.instance_id,d.enterprise_id,s.shape_id)
 LEFT JOIN organization.department_history h ON (h.instance_id,h.object_id,h.chunk_no)=(d.instance_id,d.department_id,d.head_chunk)
 LEFT JOIN organization.department_dictionary cd ON (cd.instance_id,cd.enterprise_id,cd.id)=(d.instance_id,d.enterprise_id,h.dictionary_id)
 LEFT JOIN organization.department_shape cs ON (cs.instance_id,cs.enterprise_id,cs.id)=(d.instance_id,d.enterprise_id,d.current_shape)
) FROM bu b
