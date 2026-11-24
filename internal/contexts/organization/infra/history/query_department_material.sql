jsonb_build_object(
 'id',department_id,'code',code,'source',head_source_id,'segmentSource',(segment).source_id,
 'relation',relation_head,'firstRecord',first_record_id,'ids',(segment).ids,
 '_material',jsonb_build_object('words',d.words,
  'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries) v),
  'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges) v),
  'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges) v),
  'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(h.parts) v))
)
