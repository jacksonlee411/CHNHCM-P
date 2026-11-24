(SELECT jsonb_build_object(
 'id',p.enterprise_id,'source',p.head_source_id,
 'ids',organization.project_ids(h.ids,p.current_refs),
 '_material',jsonb_build_object('words',d.words,
  'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries) v),
  'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges) v),
  'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges) v),
  'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(s.parts) v))
 ) FROM organization.enterprise_profile p
 JOIN organization.enterprise_history h ON (h.instance_id,h.object_id,h.chunk_no)=(p.instance_id,p.enterprise_id,p.head_chunk)
 JOIN organization.enterprise_dictionary d ON (d.instance_id,d.catalog_id,d.id)=(p.instance_id,p.catalog_id,h.dictionary_id)
 JOIN organization.enterprise_shape s ON (s.instance_id,s.catalog_id,s.id)=(p.instance_id,p.catalog_id,p.current_shape)
 WHERE p.instance_id=$1 AND p.enterprise_id=$2)
