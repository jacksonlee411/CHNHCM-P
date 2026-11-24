WITH owners AS MATERIALIZED (
 SELECT d.instance_id,d.enterprise_id,d.department_id,d.code,d.first_record_id,
        d.head_source_id,h.head_source_id AS relation_head,
        organization.q131_owner_seal(d.enterprise_id,d.code,d.first_record_id,h.head_source_id) AS owner_seal
 FROM organization.department d
 LEFT JOIN organization.department_hierarchy_relation h
   ON (h.instance_id,h.enterprise_id,h.department_id)=(d.instance_id,d.enterprise_id,d.department_id)
 WHERE d.instance_id=$1 AND d.enterprise_id=$2
), proof_rows AS MATERIALIZED (
 -- Materialize the boolean before window ordering. Inlining would carry all
 -- hash/UUID inputs through the sort and spill wide rows for every segment.
 SELECT o.department_id,s.slot,s.span,s.dictionary_id,s.shape_id,
  (s.source_id=o.head_source_id AND s.rule=4 AND s.q_relation=o.relation_head
   AND s.q_owner_seal=o.owner_seal AND s.q_dictionary_seal=d.digest AND s.q_shape_seal=h.digest
   AND s.q_seal=s.q_stored_seal AND organization.q131_ids_valid(s.ids)) IS TRUE AS content_valid
 FROM owners o LEFT JOIN organization.department_read_segment s
  ON (s.instance_id,s.object_id)=(o.instance_id,o.department_id)
 LEFT JOIN organization.department_dictionary d
  ON (d.instance_id,d.enterprise_id,d.id)=(o.instance_id,o.enterprise_id,s.dictionary_id)
 LEFT JOIN organization.department_shape h
  ON (h.instance_id,h.enterprise_id,h.id)=(o.instance_id,o.enterprise_id,s.shape_id)
), framed AS NOT MATERIALIZED (
 SELECT *,row_number() OVER w AS position,
  count(slot) OVER w AS segment_count,
  lag(upper(span)) OVER w AS previous_end
 FROM proof_rows
 WINDOW w AS (PARTITION BY department_id ORDER BY slot ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
), integrity AS MATERIALIZED (
 SELECT count(*) FILTER(WHERE position=1) AS objects,count(slot) AS segments,
  count(*) FILTER(WHERE NOT (
   slot IS NOT NULL AND slot=position-1 AND content_valid
   AND NOT isempty(span)
   AND lower_inc(span)=(NOT lower_inf(span)) AND NOT upper_inc(span)
   AND ((position=1 AND lower_inf(span)) OR
        (position>1 AND NOT lower_inf(span) AND lower(span)=previous_end))
   AND ((position=segment_count AND upper_inf(span)) OR
        (position<segment_count AND NOT upper_inf(span)))
  ) IS TRUE) AS bad,
  count(DISTINCT dictionary_id) AS dictionaries,count(DISTINCT shape_id) AS shapes
 FROM framed
), checked AS NOT MATERIALIZED (
 -- This access path is independent of the proof's scan, but shares its MVCC
 -- snapshot. NOT MATERIALIZED preserves indexed lookup for recursive ancestors.
 SELECT d.instance_id,d.enterprise_id,d.department_id,d.code,d.first_record_id,
        d.head_source_id,h.head_source_id AS relation_head,s AS segment
 FROM organization.department d
 LEFT JOIN organization.department_hierarchy_relation h
  ON (h.instance_id,h.enterprise_id,h.department_id)=(d.instance_id,d.enterprise_id,d.department_id)
 LEFT JOIN organization.department_read_segment s
  ON (s.instance_id,s.object_id)=(d.instance_id,d.department_id)
 WHERE d.instance_id=$1 AND d.enterprise_id=$2
)
