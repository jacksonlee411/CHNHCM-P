-- +goose Up
-- Go owns every boundary and selected member. SQL only selects a stored range.
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE INDEX bu_list_heads_cover ON organization.business_unit(instance_id,bu_id) INCLUDE(head_source_id);
CREATE TABLE organization.department_read_segment (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, slot integer NOT NULL CHECK(slot>=0),
 source_id uuid NOT NULL, rule integer NOT NULL CHECK(rule=4), span daterange NOT NULL,
 dictionary_id integer NOT NULL REFERENCES organization.department_dictionary(id),
 shape_id integer NOT NULL REFERENCES organization.department_shape(id),
 ids uuid[] NOT NULL, envelope bytea NOT NULL CHECK(octet_length(envelope)=32),
 PRIMARY KEY(instance_id,object_id,slot),
 FOREIGN KEY(instance_id,object_id) REFERENCES organization.department(instance_id,department_id),
 CHECK(NOT isempty(span) AND lower_inc(span)=(NOT lower_inf(span)) AND NOT upper_inc(span)),
 EXCLUDE USING gist(instance_id WITH =,object_id WITH =,span WITH &&)
);
CREATE INDEX department_read_dictionary ON organization.department_read_segment(dictionary_id);
CREATE INDEX department_read_shape ON organization.department_read_segment(shape_id);
CREATE INDEX department_read_seek ON organization.department_read_segment(instance_id,object_id,(coalesce(lower(span),'-infinity'::date)) DESC);
CREATE TABLE organization.bu_read_segment (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, slot integer NOT NULL CHECK(slot>=0),
 source_id uuid NOT NULL, rule integer NOT NULL CHECK(rule=4), span daterange NOT NULL,
 dictionary_id integer NOT NULL REFERENCES organization.bu_dictionary(id),
 shape_id integer NOT NULL REFERENCES organization.bu_shape(id),
 ids uuid[] NOT NULL, envelope bytea NOT NULL CHECK(octet_length(envelope)=32),
 PRIMARY KEY(instance_id,object_id,slot),
 FOREIGN KEY(instance_id,object_id) REFERENCES organization.business_unit(instance_id,bu_id),
 CHECK(NOT isempty(span) AND lower_inc(span)=(NOT lower_inf(span)) AND NOT upper_inc(span)),
 EXCLUDE USING gist(instance_id WITH =,object_id WITH =,span WITH &&)
);
CREATE INDEX bu_read_dictionary ON organization.bu_read_segment(dictionary_id);
CREATE INDEX bu_read_shape ON organization.bu_read_segment(shape_id);
CREATE INDEX bu_read_seek ON organization.bu_read_segment(instance_id,object_id,(coalesce(lower(span),'-infinity'::date)) DESC);
GRANT SELECT,INSERT,UPDATE,DELETE ON organization.department_read_segment,organization.bu_read_segment TO chnhcm_app;

-- A storage directory only. Dates and rights remain in Go. The streaming and
-- materialized adapters consume exactly this same statement.
-- +goose StatementBegin
CREATE FUNCTION organization.list_source_heads(wanted uuid) RETURNS TABLE(kind smallint,identity bytea) LANGUAGE sql STABLE PARALLEL SAFE AS $$
 SELECT 1::smallint,(uuid_send(d.department_id)||uuid_send(d.head_source_id)||uuid_send(h.head_source_id))::bytea
 FROM organization.department d LEFT JOIN organization.department_hierarchy_relation h USING(instance_id,enterprise_id,department_id) WHERE d.instance_id=wanted
 UNION ALL SELECT 2::smallint,(uuid_send(b.bu_id)||uuid_send(b.head_source_id))::bytea FROM organization.business_unit b WHERE b.instance_id=wanted
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION organization.list_source_head_bytes(wanted uuid) RETURNS bytea LANGUAGE sql STABLE PARALLEL SAFE AS $$
 SELECT coalesce(string_agg(set_byte(decode('00','hex'),0,kind)||identity,''::bytea),''::bytea) FROM organization.list_source_heads(wanted)
$$;
-- +goose StatementEnd

-- Go-derived query facts and storage-only framing; installed with the current model.
ALTER TABLE organization.department_read_segment
 ADD COLUMN q_code_key text NOT NULL, ADD COLUMN q_path_date date,
 ADD COLUMN q_name text NOT NULL, ADD COLUMN q_name_key text NOT NULL,
 ADD COLUMN q_short_name text, ADD COLUMN q_short_key text NOT NULL,
 ADD COLUMN q_bu uuid NOT NULL, ADD COLUMN q_parent uuid,
 ADD COLUMN q_record uuid NOT NULL, ADD COLUMN q_revision uuid NOT NULL,
 ADD COLUMN q_date date NOT NULL, ADD COLUMN q_end date, ADD COLUMN q_first_date date NOT NULL,
 ADD COLUMN q_state text NOT NULL, ADD COLUMN q_relation uuid NOT NULL,
 ADD COLUMN q_owner_seal bytea NOT NULL CHECK(octet_length(q_owner_seal)=32),
 ADD COLUMN q_dictionary_seal bytea NOT NULL CHECK(octet_length(q_dictionary_seal)=32),
 ADD COLUMN q_shape_seal bytea NOT NULL CHECK(octet_length(q_shape_seal)=32),
 ADD COLUMN q_seal bytea NOT NULL CHECK(octet_length(q_seal)=32);

-- Date lookup can read the list's narrow scope without fetching every wide
-- historical tuple. Go still owns these facts and the final page material.
CREATE INDEX department_list_span_cover ON organization.department_read_segment USING gist(span)
 INCLUDE(instance_id,object_id,q_name,q_name_key,q_short_key,q_bu,q_date,q_state,q_code_key,q_path_date,source_id,dictionary_id,shape_id,ids);
CREATE INDEX department_parent_sparse_r104 ON organization.department_read_segment(instance_id,object_id) INCLUDE(span,q_parent) WHERE q_parent IS NOT NULL;

-- The seal binds an ordered UUID sequence, while PostgreSQL arrays also carry
-- dimensions and lower bounds. Keep that storage representation canonical.
-- CASE is required: array_position raises for multidimensional input, and
-- boolean AND evaluation order must not decide whether corruption is handled.
-- +goose StatementBegin
CREATE FUNCTION organization.q131_ids_valid(directory uuid[])
RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT CASE WHEN array_ndims(directory)=1 AND array_lower(directory,1)=1 AND cardinality(directory)>0
  THEN array_position(directory,NULL::uuid) IS NULL ELSE false END
$$;
-- +goose StatementEnd
ALTER TABLE organization.department_read_segment
 ADD CONSTRAINT department_query131_ids_valid CHECK(organization.q131_ids_valid(ids));
ALTER TABLE organization.bu_read_segment
 ADD CONSTRAINT bu_read_ids_valid CHECK(organization.q131_ids_valid(ids));

-- Storage-only checksums: generated values are recomputed on every SQL write,
-- including writes with ordinary triggers disabled. Go still owns every fact.
-- +goose StatementBegin
CREATE FUNCTION organization.q131_owner_seal(ent uuid, code text, first_id uuid, relation uuid)
RETURNS bytea LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT organization.history_parts_hash(ARRAY[convert_to('department-query-owner/1','UTF8'),convert_to(ent::text,'UTF8'),convert_to(code,'UTF8'),convert_to(first_id::text,'UTF8'),convert_to(relation::text,'UTF8')])
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION organization.q131_local_seal(
 p_instance_id uuid,
 p_object_id uuid,
 p_source_id uuid,
 p_slot integer,
 p_rule integer,
 p_span daterange,
 p_dictionary_id integer,
 p_shape_id integer,
 p_envelope bytea,
 p_q_owner_seal bytea,
 p_q_dictionary_seal bytea,
 p_q_shape_seal bytea,
 p_q_name text,
 p_q_name_key text,
 p_q_short_name text,
 p_q_short_key text,
 p_q_bu uuid,
 p_q_parent uuid,
 p_q_record uuid,
 p_q_revision uuid,
 p_q_date date,
 p_q_end date,
 p_q_first_date date,
 p_q_state text,
 p_q_relation uuid,
 p_q_code_key text,
 p_q_path_date date,
 p_ids uuid[]
) RETURNS bytea LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT organization.history_parts_hash(ARRAY[
  convert_to('department-query-131/3','UTF8'),
  convert_to(p_instance_id::text,'UTF8'),
  convert_to(p_object_id::text,'UTF8'),
  convert_to(p_source_id::text,'UTF8'),
  convert_to(p_slot::text,'UTF8'),
  convert_to(p_rule::text,'UTF8'),
  convert_to(coalesce(lower(p_span)::text,''),'UTF8'),
  convert_to(coalesce(upper(p_span)::text,''),'UTF8'),
  convert_to(p_dictionary_id::text,'UTF8'),
  convert_to(p_shape_id::text,'UTF8'),
  p_envelope,
  p_q_owner_seal,
  p_q_dictionary_seal,
  p_q_shape_seal,
  convert_to(p_q_name::text,'UTF8'),
  convert_to(p_q_name_key::text,'UTF8'),
  CASE WHEN p_q_short_name IS NULL THEN '\x00'::bytea ELSE '\x01'::bytea||convert_to(p_q_short_name,'UTF8') END,
  convert_to(p_q_short_key::text,'UTF8'),
  convert_to(p_q_bu::text,'UTF8'),
  convert_to(coalesce(p_q_parent::text,''),'UTF8'),
  convert_to(p_q_record::text,'UTF8'),
  convert_to(p_q_revision::text,'UTF8'),
  convert_to(p_q_date::text,'UTF8'),
  convert_to(coalesce(p_q_end::text,''),'UTF8'),
  convert_to(p_q_first_date::text,'UTF8'),
  convert_to(p_q_state::text,'UTF8'),
  convert_to(p_q_relation::text,'UTF8'),
  convert_to(p_q_code_key::text,'UTF8'),
  convert_to(coalesce(p_q_path_date::text,''),'UTF8')
 ] || ARRAY(SELECT convert_to(v::text,'UTF8') FROM unnest(p_ids) WITH ORDINALITY AS u(v,n) ORDER BY n))
$$;
-- +goose StatementEnd
ALTER TABLE organization.department_read_segment
 ADD COLUMN q_stored_seal bytea GENERATED ALWAYS AS (organization.q131_local_seal(instance_id, object_id, source_id, slot, rule, span, dictionary_id, shape_id, envelope, q_owner_seal, q_dictionary_seal, q_shape_seal, q_name, q_name_key, q_short_name, q_short_key, q_bu, q_parent, q_record, q_revision, q_date, q_end, q_first_date, q_state, q_relation, q_code_key, q_path_date, ids)) STORED;

-- Explicit full binding is retained for diagnostics and source-seal equivalence.
-- +goose StatementBegin
CREATE FUNCTION organization.q131_seal(
 s organization.department_read_segment, ent uuid, code text, first_id uuid,
 relation uuid, dictionary_hash bytea, shape_hash bytea
) RETURNS bytea LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT organization.q131_local_seal((s).instance_id, (s).object_id, (s).source_id, (s).slot, (s).rule, (s).span, (s).dictionary_id, (s).shape_id, (s).envelope, organization.q131_owner_seal(ent,code,first_id,relation), dictionary_hash, shape_hash, (s).q_name, (s).q_name_key, (s).q_short_name, (s).q_short_key, (s).q_bu, (s).q_parent, (s).q_record, (s).q_revision, (s).q_date, (s).q_end, (s).q_first_date, (s).q_state, (s).q_relation, (s).q_code_key, (s).q_path_date, (s).ids)
$$;
-- +goose StatementEnd

-- Execute only X03-compiled identity sets; never inspect grants here.
-- +goose StatementBegin
CREATE FUNCTION organization.q131_set(s jsonb, bu uuid, department uuid)
RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT coalesce((s->>'all')::boolean,false) OR coalesce((s->'bus') ? bu::text,false)
  OR coalesce((s->'departments') ? department::text,false)
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION organization.list_source_head_bytes(uuid);
DROP FUNCTION organization.list_source_heads(uuid);
DROP FUNCTION organization.q131_set(jsonb,uuid,uuid);
DROP FUNCTION organization.q131_seal(organization.department_read_segment,uuid,text,uuid,uuid,bytea,bytea);
DROP TABLE organization.bu_read_segment;
DROP TABLE organization.department_read_segment;
DROP FUNCTION organization.q131_local_seal(uuid,uuid,uuid,integer,integer,daterange,integer,integer,bytea,bytea,bytea,bytea,text,text,text,text,uuid,uuid,uuid,uuid,date,date,date,text,uuid,text,date,uuid[]);
DROP FUNCTION organization.q131_owner_seal(uuid,text,uuid,uuid);
DROP FUNCTION organization.q131_ids_valid(uuid[]);
DROP INDEX organization.bu_list_heads_cover;
