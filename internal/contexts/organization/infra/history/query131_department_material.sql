jsonb_build_object(
    'id',department_id,'code',code,'codeKey',(segment).q_code_key,'pathDay',(segment).q_path_date,'source',head_source_id,'relation',relation_head,
    'firstRecord',first_record_id,'firstDate',(segment).q_first_date,
    'record',(segment).q_record,'revision',(segment).q_revision,
    'date',(segment).q_date,'end',(segment).q_end,'status',(segment).q_state,
    'name',(segment).q_name,'shortName',(segment).q_short_name,
    'bu',(segment).q_bu,'parent',(segment).q_parent,
    'dictionary',(segment).dictionary_id,'shape',(segment).shape_id,
    'ids',(segment).ids,'envelope',encode((segment).envelope,'hex'),
    'start',coalesce(lower((segment).span)::text,''),'finish',coalesce(upper((segment).span)::text,''),
    '_material',jsonb_build_object('words',d.words,
      'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries) v),
      'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges) v),
      'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges) v),
      'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(h.parts) v),
      'dictionaryDigest',encode(d.digest,'hex'),'shapeDigest',encode(h.digest,'hex'))
   )
