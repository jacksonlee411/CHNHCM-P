-- +goose Up
-- CHNHCM-126: one current model, initialized only in an isolated fresh database.
CREATE SCHEMA organization;
CREATE TABLE organization.enterprise (
 instance_id uuid NOT NULL UNIQUE, enterprise_id uuid NOT NULL, code text NOT NULL, name text NOT NULL,
 valid_range daterange NOT NULL CHECK(NOT isempty(valid_range) AND NOT lower_inf(valid_range) AND upper_inf(valid_range)),
 initialization_digest bytea NOT NULL CHECK(octet_length(initialization_digest)=32), PRIMARY KEY(instance_id,enterprise_id)
);
-- +goose StatementBegin
CREATE FUNCTION organization.immutable_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'committed history is immutable' USING ERRCODE='23514'; END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_enterprise BEFORE UPDATE OR DELETE ON organization.enterprise FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- +goose StatementBegin
CREATE FUNCTION organization.history_parts_hash(parts bytea[]) RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT AS $$
DECLARE raw bytea; invalid boolean; p bytea;
BEGIN
 IF cardinality(parts)=0 THEN RETURN sha256(int4send(0)); END IF;
 -- Read shapes normally contain one part. Keep identical framing without
 -- starting an aggregate executor for every small shape and conflict probe.
 IF cardinality(parts)=1 AND array_ndims(parts)=1 THEN
  p:=parts[array_lower(parts,1)];
  RETURN sha256(int4send(1)||int4send(octet_length(p))||p);
 END IF;
 SELECT bool_or(v IS NULL),string_agg(int4send(octet_length(v))||v,''::bytea ORDER BY n)
 INTO invalid,raw FROM unnest(parts) WITH ORDINALITY AS t(v,n);
 IF invalid THEN RETURN NULL; END IF;
 RETURN sha256(int4send(cardinality(parts))||coalesce(raw,''::bytea));
END
$$;
-- +goose StatementEnd
-- +goose StatementBegin
-- Preserve the exact count/length framing; aggregate once instead of repeatedly
-- copying growing arrays and byte strings for long current timelines.
CREATE FUNCTION organization.history_dictionary_hash(words text[], binaries bytea[], ranges daterange[]) RETURNS bytea LANGUAGE sql IMMUTABLE STRICT AS $$
 SELECT organization.history_parts_hash(ARRAY(
  SELECT p FROM (
   SELECT 1 AS g,0::bigint AS n,int4send(cardinality(words)) AS p
   UNION ALL SELECT 1,n,convert_to(word,'UTF8') FROM unnest(words) WITH ORDINALITY AS v(word,n)
   UNION ALL SELECT 2,0,int4send(cardinality(binaries))
   UNION ALL SELECT 2,n,b FROM unnest(binaries) WITH ORDINALITY AS v(b,n)
   UNION ALL SELECT 3,0,int4send(cardinality(ranges))
   UNION ALL SELECT 3,2*n-1,convert_to(coalesce(lower(r)::text,''),'UTF8') FROM unnest(ranges) WITH ORDINALITY AS v(r,n)
   UNION ALL SELECT 3,2*n,convert_to(coalesce(upper(r)::text,''),'UTF8') FROM unnest(ranges) WITH ORDINALITY AS v(r,n)
  ) parts ORDER BY g,n
 ))
$$;
-- +goose StatementEnd
CREATE FUNCTION organization.submission_tag(id uuid) RETURNS integer LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS $$ SELECT get_byte(sha256(uuid_send(id)),0)*256+get_byte(sha256(uuid_send(id)),1) $$;
CREATE FUNCTION organization.submission_tags(ids uuid[],refs integer[]) RETURNS integer[] LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS $$ SELECT coalesce(array_agg(DISTINCT organization.submission_tag(ids[r]) ORDER BY organization.submission_tag(ids[r])),'{}') FROM unnest(refs) r $$;
CREATE FUNCTION organization.submission_key(kind text,instance_id uuid,subject_id uuid,submission_id uuid) RETURNS bigint LANGUAGE sql IMMUTABLE STRICT AS $$ SELECT ('x'||substr(encode(sha256(convert_to(CASE WHEN kind='department' THEN '["organization.department",' ELSE '[' END || '"'||instance_id::text||'","'||subject_id::text||'","'||submission_id::text||'"]','UTF8')),'hex'),1,16))::bit(64)::bigint $$;
-- +goose StatementBegin
CREATE FUNCTION organization.require_submission_lock(kind text,instance_id uuid,subject_id uuid,submission_id uuid) RETURNS void LANGUAGE plpgsql AS $$
DECLARE k bigint:=organization.submission_key(kind,instance_id,subject_id,submission_id);
BEGIN
 IF pg_has_role(current_user,'chnhcm_app','member') AND NOT EXISTS(
  SELECT FROM pg_locks WHERE locktype='advisory' AND pid=pg_backend_pid() AND granted AND objsubid=1
   AND classid=((k>>32)&4294967295)::oid AND objid=(k&4294967295)::oid AND mode='ExclusiveLock')
 THEN RAISE EXCEPTION 'complete submission identity must be guarded before storage' USING ERRCODE='23514'; END IF;
END $$;
-- +goose StatementEnd

-- These checks describe physical arrays, not business date or hierarchy rules.
-- +goose StatementBegin
CREATE FUNCTION organization.history_parts_valid(parts bytea[]) RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT AS $$
BEGIN
 IF array_ndims(parts) IS DISTINCT FROM 1 OR array_lower(parts,1)<>1 OR cardinality(parts) NOT BETWEEN 1 AND 256 THEN RETURN false; END IF;
 IF cardinality(parts)=1 THEN RETURN parts[1] IS NOT NULL AND octet_length(parts[1]) BETWEEN 1 AND 262144; END IF;
 RETURN NOT EXISTS(SELECT FROM unnest(parts) p WHERE p IS NULL OR octet_length(p) NOT BETWEEN 1 AND 262144);
END
$$;
-- +goose StatementEnd
CREATE FUNCTION organization.history_ranges_valid(ranges daterange[]) RETURNS boolean LANGUAGE sql IMMUTABLE STRICT AS $$
 SELECT (cardinality(ranges)=0 OR (array_ndims(ranges)=1 AND array_lower(ranges,1)=1))
 AND NOT EXISTS(SELECT FROM unnest(ranges) r WHERE r IS NULL OR (NOT isempty(r) AND lower_inf(r)))
$$;

-- +goose StatementBegin
CREATE FUNCTION organization.submission_slot(ids uuid[], submissions integer[], subjects integer[], actor uuid, submission uuid) RETURNS integer LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE AS $$
DECLARE slot integer;
BEGIN
 FOREACH slot IN ARRAY array_positions(submissions,array_position(ids,submission)) LOOP
  IF ids[subjects[slot]]=actor THEN RETURN slot; END IF;
 END LOOP;
 RETURN NULL;
END $$;
-- +goose StatementEnd

-- Materialize a toasted UUID pool once before selecting its current references.
-- This is an array projection only; no dates, states or authorization run in SQL.
-- +goose StatementBegin
CREATE FUNCTION organization.project_ids(ids uuid[], refs integer[]) RETURNS uuid[] LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE AS $$
DECLARE values uuid[] := array_cat(ids,'{}'::uuid[]); projected uuid[] := '{}'; r integer;
BEGIN
 FOREACH r IN ARRAY refs LOOP projected:=array_append(projected,values[r]); END LOOP;
 RETURN projected;
END $$;
-- +goose StatementEnd

CREATE TABLE organization.business_unit (
 instance_id uuid NOT NULL, bu_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 code text NOT NULL, normalized_code text NOT NULL, first_record_id uuid NOT NULL, head_source_id uuid NOT NULL,
 head_no bigint NOT NULL CHECK(head_no>0), head_chunk integer NOT NULL CHECK(head_chunk>0),
 head_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 current_shape integer, current_refs integer[], current_envelope bytea,
 delivered_no bigint NOT NULL DEFAULT 0 CHECK(delivered_no>=0 AND delivered_no<=head_no),
 PRIMARY KEY(instance_id,bu_id), UNIQUE(instance_id,enterprise_id,bu_id), UNIQUE(instance_id,enterprise_id,normalized_code),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(normalized_code=lower(code))
);
-- +goose StatementBegin
CREATE FUNCTION organization.bu_stamp_head() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.head_no>OLD.head_no THEN NEW.head_xid:=pg_current_xact_id(); ELSE NEW.head_xid:=OLD.head_xid; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER stamp_head BEFORE UPDATE ON organization.business_unit FOR EACH ROW EXECUTE FUNCTION organization.bu_stamp_head();
CREATE INDEX bu_pending_events ON organization.business_unit(instance_id,bu_id) WHERE delivered_no<head_no;
CREATE TABLE organization.bu_dictionary (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 words text[] NOT NULL, binaries bytea[] NOT NULL, ranges daterange[] NOT NULL,
 digest bytea GENERATED ALWAYS AS (organization.history_dictionary_hash(words,binaries,ranges)) STORED,
 UNIQUE(instance_id,enterprise_id,digest), UNIQUE(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(array_position(words,NULL) IS NULL AND array_position(binaries,NULL) IS NULL AND array_position(ranges,NULL) IS NULL),
 CHECK(cardinality(words)<=1048576 AND cardinality(binaries)<=1048576 AND cardinality(ranges)<=1048576),
 CHECK((cardinality(words)=0 OR (array_ndims(words)=1 AND array_lower(words,1)=1))
 AND (cardinality(binaries)=0 OR (array_ndims(binaries)=1 AND array_lower(binaries,1)=1))
 AND organization.history_ranges_valid(ranges))
);
CREATE TRIGGER immutable_dictionary BEFORE UPDATE ON organization.bu_dictionary FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.bu_shape (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 parts bytea[] NOT NULL, digest bytea GENERATED ALWAYS AS (organization.history_parts_hash(parts)) STORED,
 UNIQUE(instance_id,enterprise_id,digest), UNIQUE(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(organization.history_parts_valid(parts))
);
CREATE TRIGGER immutable_shape BEFORE UPDATE OR DELETE ON organization.bu_shape FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.bu_history (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, chunk_no integer NOT NULL CHECK(chunk_no>0),
 first_no bigint NOT NULL CHECK(first_no>0), format smallint NOT NULL CHECK(format=1),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL,
 source_refs integer[] NOT NULL, subject_refs integer[] NOT NULL, submission_refs integer[] NOT NULL,
 states integer[] NOT NULL, operations integer[] NOT NULL, evidence integer[] NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), sealed boolean NOT NULL DEFAULT false,
 encoded_bytes integer NOT NULL CHECK(encoded_bytes>0 AND (encoded_bytes<=262144 OR cardinality(source_refs)=1)),
 written_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 submission_tags integer[] GENERATED ALWAYS AS (organization.submission_tags(ids,submission_refs)) STORED,
 PRIMARY KEY(instance_id,object_id,chunk_no),
 FOREIGN KEY(instance_id,enterprise_id,object_id) REFERENCES organization.business_unit(instance_id,enterprise_id,bu_id),
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.bu_dictionary(instance_id,enterprise_id,id),
 CHECK(cardinality(source_refs) BETWEEN 1 AND 64 AND cardinality(source_refs)=cardinality(submission_refs)
 AND cardinality(source_refs)=cardinality(subject_refs) AND cardinality(source_refs)=cardinality(states)
 AND cardinality(source_refs)=cardinality(operations) AND cardinality(source_refs)=cardinality(evidence)),
 CHECK(array_position(ids,NULL) IS NULL AND cardinality(ids)<=1048576 AND array_ndims(ids)=1 AND array_lower(ids,1)=1),
 CHECK(array_ndims(source_refs)=1 AND array_lower(source_refs,1)=1 AND array_ndims(subject_refs)=1 AND array_lower(subject_refs,1)=1
 AND array_ndims(submission_refs)=1 AND array_lower(submission_refs,1)=1 AND array_ndims(states)=1 AND array_lower(states,1)=1
 AND array_ndims(operations)=1 AND array_lower(operations,1)=1 AND array_ndims(evidence)=1 AND array_lower(evidence,1)=1)
);
ALTER TABLE organization.business_unit ADD FOREIGN KEY(instance_id,bu_id,head_chunk) REFERENCES organization.bu_history(instance_id,object_id,chunk_no) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX bu_submission_lookup ON organization.bu_history USING gin(submission_tags) WITH(fastupdate=off);

CREATE TABLE organization.bu_terminal (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 request_object_id uuid, outcome text NOT NULL CHECK(outcome IN ('rejected','no_change','not_applied')),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL, operation integer NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), closed_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,subject_id) REFERENCES access.principal(instance_id,principal_id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.bu_dictionary(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id,operation) REFERENCES organization.bu_shape(instance_id,enterprise_id,id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON organization.bu_terminal FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- +goose StatementBegin
CREATE FUNCTION organization.bu_append_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE n integer; oldn integer; i integer; olddict organization.bu_dictionary; newdict organization.bu_dictionary;
 previous organization.bu_history; sid uuid; actor uuid; s uuid; count_shapes integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'history cannot be deleted' USING ERRCODE='23514'; END IF;
 n:=cardinality(NEW.source_refs);
 IF n NOT BETWEEN 1 AND 64 THEN RAISE EXCEPTION 'bounded history directory required' USING ERRCODE='23514'; END IF;
 NEW.written_xid:=pg_current_xact_id();
 SELECT * INTO STRICT newdict FROM organization.bu_dictionary WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=NEW.dictionary_id;
 IF TG_OP='UPDATE' THEN
  oldn:=cardinality(OLD.source_refs);
  IF OLD.sealed OR (NEW.instance_id,NEW.enterprise_id,NEW.object_id,NEW.chunk_no,NEW.first_no,NEW.format) IS DISTINCT FROM (OLD.instance_id,OLD.enterprise_id,OLD.object_id,OLD.chunk_no,OLD.first_no,OLD.format)
   OR NEW.ids[1:cardinality(OLD.ids)] IS DISTINCT FROM OLD.ids
   OR NEW.source_refs[1:oldn] IS DISTINCT FROM OLD.source_refs OR NEW.subject_refs[1:oldn] IS DISTINCT FROM OLD.subject_refs
   OR NEW.submission_refs[1:oldn] IS DISTINCT FROM OLD.submission_refs OR NEW.states[1:oldn] IS DISTINCT FROM OLD.states
   OR NEW.operations[1:oldn] IS DISTINCT FROM OLD.operations OR NEW.evidence[1:oldn] IS DISTINCT FROM OLD.evidence
   OR NOT (n=oldn+1 OR (n=oldn AND NEW.sealed AND NEW.ids=OLD.ids AND NEW.dictionary_id=OLD.dictionary_id AND NEW.envelope=OLD.envelope AND NEW.encoded_bytes=OLD.encoded_bytes))
  THEN RAISE EXCEPTION 'history prefixes cannot change' USING ERRCODE='23514'; END IF;
  SELECT * INTO STRICT olddict FROM organization.bu_dictionary WHERE id=OLD.dictionary_id;
  IF coalesce(newdict.words[1:cardinality(olddict.words)],'{}') IS DISTINCT FROM olddict.words
   OR coalesce(newdict.binaries[1:cardinality(olddict.binaries)],'{}') IS DISTINCT FROM olddict.binaries
   OR coalesce(newdict.ranges[1:cardinality(olddict.ranges)],'{}') IS DISTINCT FROM olddict.ranges
  THEN RAISE EXCEPTION 'dictionary prefix cannot change' USING ERRCODE='23514'; END IF;
 ELSE
  oldn:=0;
  IF NEW.chunk_no=1 THEN
   IF NEW.first_no<>1 THEN RAISE EXCEPTION 'first source number required' USING ERRCODE='23514'; END IF;
  ELSE
   SELECT * INTO STRICT previous FROM organization.bu_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no-1;
   IF NOT previous.sealed OR NEW.first_no<>previous.first_no+cardinality(previous.source_refs)
   THEN RAISE EXCEPTION 'preceding chunk must be sealed and contiguous' USING ERRCODE='23514'; END IF;
  END IF;
 END IF;
 -- The prefix above is unchanged and was checked at its original append.
 -- Shape rows are immutable and cannot be deleted. Only new references need
 -- existence/ownership checks; sealing a full block adds no references.
 FOR i IN oldn+1..n LOOP
  IF NEW.source_refs[i] IS NULL OR NEW.subject_refs[i] IS NULL OR NEW.submission_refs[i] IS NULL
   OR NEW.source_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids) OR NEW.subject_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids) OR NEW.submission_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids)
   OR NEW.states[i] IS NULL OR NEW.operations[i] IS NULL OR NEW.evidence[i] IS NULL
  THEN RAISE EXCEPTION 'invalid source directory' USING ERRCODE='23514'; END IF;
  SELECT count(*) INTO count_shapes FROM organization.bu_shape WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=ANY(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]);
  IF count_shapes<>(SELECT count(DISTINCT x) FROM unnest(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]) x)
  THEN RAISE EXCEPTION 'missing or foreign history root' USING ERRCODE='23503'; END IF;
 END LOOP;
 FOR i IN oldn+1..n LOOP
  actor:=NEW.ids[NEW.subject_refs[i]]; sid:=NEW.ids[NEW.submission_refs[i]]; s:=NEW.ids[NEW.source_refs[i]];
  PERFORM organization.require_submission_lock('bu',NEW.instance_id,actor,sid);
  IF NOT EXISTS(SELECT FROM access.principal WHERE instance_id=NEW.instance_id AND principal_id=actor)
  THEN RAISE EXCEPTION 'missing source actor' USING ERRCODE='23503'; END IF;
  IF EXISTS(SELECT FROM organization.bu_terminal WHERE instance_id=NEW.instance_id AND subject_id=actor AND submission_id=sid)
   OR EXISTS(SELECT FROM organization.bu_history h WHERE h.instance_id=NEW.instance_id
    AND h.submission_tags @> ARRAY[organization.submission_tag(sid)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,actor,sid) IS NOT NULL)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.subject_refs[j]]=actor AND NEW.ids[NEW.submission_refs[j]]=sid)
  THEN RAISE EXCEPTION 'submission already has a terminal' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.source_refs) r WHERE h.instance_id=NEW.instance_id AND h.object_id=NEW.object_id AND h.ids[r]=s)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.source_refs[j]]=s)
  THEN RAISE EXCEPTION 'source identity already exists' USING ERRCODE='23505'; END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_history BEFORE INSERT OR UPDATE OR DELETE ON organization.bu_history FOR EACH ROW EXECUTE FUNCTION organization.bu_append_guard();

-- +goose StatementBegin
CREATE FUNCTION organization.bu_head_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.bu_history; current_head organization.business_unit; id_count integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'object identity cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND ((to_jsonb(NEW)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) OR NEW.head_no<OLD.head_no OR NEW.delivered_no<OLD.delivered_no)
 THEN RAISE EXCEPTION 'object identity is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT current_head FROM organization.business_unit WHERE instance_id=NEW.instance_id AND bu_id=NEW.bu_id;
 SELECT * INTO STRICT h FROM organization.bu_history WHERE instance_id=current_head.instance_id AND object_id=current_head.bu_id AND chunk_no=current_head.head_chunk;
 -- Read the large directory's bound once. A cached generic subquery plan must
 -- not detoast that UUID array again for every current reference.
 id_count:=cardinality(h.ids);
 IF current_head.head_no<>h.first_no+cardinality(h.source_refs)-1 OR current_head.head_source_id<>h.ids[h.source_refs[cardinality(h.source_refs)]]
 THEN RAISE EXCEPTION 'head must point to complete final source' USING ERRCODE='23514'; END IF;
 IF current_head.current_shape IS NULL OR current_head.current_refs IS NULL OR cardinality(current_head.current_refs)=0
  OR array_ndims(current_head.current_refs)<>1 OR array_lower(current_head.current_refs,1)<>1
  OR current_head.current_envelope IS NULL OR octet_length(current_head.current_envelope)<>32
  OR EXISTS(SELECT FROM unnest(current_head.current_refs) r WHERE r IS NULL OR r NOT BETWEEN 1 AND id_count)
  OR NOT EXISTS(SELECT FROM organization.bu_shape s WHERE s.instance_id=current_head.instance_id AND s.enterprise_id=current_head.enterprise_id AND s.id=current_head.current_shape)
  OR EXISTS(SELECT FROM organization.bu_history newer WHERE newer.instance_id=current_head.instance_id AND newer.object_id=current_head.bu_id AND newer.chunk_no>current_head.head_chunk)
 THEN RAISE EXCEPTION 'current projection must reference the final owned source' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND NEW.head_no=OLD.head_no
  AND (NEW.current_shape,NEW.current_refs,NEW.current_envelope) IS DISTINCT FROM (OLD.current_shape,OLD.current_refs,OLD.current_envelope)
  AND current_head.head_xid<>pg_current_xact_id()
 THEN RAISE EXCEPTION 'current projection changes require an appended source' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER protect_head AFTER INSERT OR UPDATE OR DELETE ON organization.business_unit DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.bu_head_guard();
-- A chunk append cannot commit without advancing its owning head in the same transaction.
-- +goose StatementBegin
CREATE FUNCTION organization.bu_closure_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.bu_history; owner organization.business_unit;
BEGIN
 SELECT * INTO STRICT h FROM organization.bu_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no;
 SELECT * INTO STRICT owner FROM organization.business_unit WHERE instance_id=h.instance_id AND bu_id=h.object_id;
 IF owner.head_no<h.first_no+cardinality(h.source_refs)-1 OR owner.head_chunk<h.chunk_no
  OR (owner.head_chunk>h.chunk_no AND NOT h.sealed)
 THEN RAISE EXCEPTION 'source directory and current head must commit together' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER complete_history AFTER INSERT OR UPDATE ON organization.bu_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.bu_closure_guard();


-- +goose StatementBegin
CREATE FUNCTION organization.bu_terminal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM organization.require_submission_lock('bu',NEW.instance_id,NEW.subject_id,NEW.submission_id);
 IF EXISTS(SELECT FROM organization.bu_history h WHERE h.instance_id=NEW.instance_id AND h.submission_tags @> ARRAY[organization.submission_tag(NEW.submission_id)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,NEW.subject_id,NEW.submission_id) IS NOT NULL)
 THEN RAISE EXCEPTION 'submission already accepted' USING ERRCODE='23505'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_terminal BEFORE INSERT ON organization.bu_terminal FOR EACH ROW EXECUTE FUNCTION organization.bu_terminal_guard();
GRANT SELECT,INSERT ON organization.business_unit,organization.bu_dictionary,organization.bu_shape,organization.bu_history,organization.bu_terminal TO chnhcm_app;
GRANT UPDATE ON organization.bu_history TO chnhcm_app;
GRANT DELETE ON organization.bu_dictionary TO chnhcm_app;
-- Row-lock privilege only; the immutable trigger still rejects every UPDATE.
GRANT UPDATE(id) ON organization.bu_dictionary TO chnhcm_app;
GRANT UPDATE(head_source_id,head_no,head_chunk,delivered_no,current_shape,current_refs,current_envelope) ON organization.business_unit TO chnhcm_app;

CREATE TABLE organization.department (
 instance_id uuid NOT NULL, department_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 code text NOT NULL, normalized_code text NOT NULL, first_record_id uuid NOT NULL, head_source_id uuid NOT NULL,
 head_no bigint NOT NULL CHECK(head_no>0), head_chunk integer NOT NULL CHECK(head_chunk>0),
 head_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 current_shape integer, current_refs integer[], current_envelope bytea,
 delivered_no bigint NOT NULL DEFAULT 0 CHECK(delivered_no>=0 AND delivered_no<=head_no),
 PRIMARY KEY(instance_id,department_id) INCLUDE(enterprise_id,head_source_id), UNIQUE(instance_id,enterprise_id,department_id), UNIQUE(instance_id,enterprise_id,normalized_code),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(normalized_code=lower(code))
);
-- +goose StatementBegin
CREATE FUNCTION organization.department_stamp_head() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.head_no>OLD.head_no THEN NEW.head_xid:=pg_current_xact_id(); ELSE NEW.head_xid:=OLD.head_xid; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER stamp_head BEFORE UPDATE ON organization.department FOR EACH ROW EXECUTE FUNCTION organization.department_stamp_head();
CREATE INDEX department_pending_events ON organization.department(instance_id,department_id) WHERE delivered_no<head_no;
CREATE TABLE organization.department_dictionary (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 words text[] NOT NULL, binaries bytea[] NOT NULL, ranges daterange[] NOT NULL,
 digest bytea GENERATED ALWAYS AS (organization.history_dictionary_hash(words,binaries,ranges)) STORED,
 UNIQUE(instance_id,enterprise_id,digest), UNIQUE(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(array_position(words,NULL) IS NULL AND array_position(binaries,NULL) IS NULL AND array_position(ranges,NULL) IS NULL),
 CHECK(cardinality(words)<=1048576 AND cardinality(binaries)<=1048576 AND cardinality(ranges)<=1048576),
 CHECK((cardinality(words)=0 OR (array_ndims(words)=1 AND array_lower(words,1)=1))
 AND (cardinality(binaries)=0 OR (array_ndims(binaries)=1 AND array_lower(binaries,1)=1))
 AND organization.history_ranges_valid(ranges))
);
CREATE TRIGGER immutable_dictionary BEFORE UPDATE ON organization.department_dictionary FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.department_shape (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 parts bytea[] NOT NULL, digest bytea GENERATED ALWAYS AS (organization.history_parts_hash(parts)) STORED,
 UNIQUE(instance_id,enterprise_id,digest), UNIQUE(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(organization.history_parts_valid(parts))
);
CREATE TRIGGER immutable_shape BEFORE UPDATE OR DELETE ON organization.department_shape FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.department_history (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, chunk_no integer NOT NULL CHECK(chunk_no>0),
 first_no bigint NOT NULL CHECK(first_no>0), format smallint NOT NULL CHECK(format=1),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL,
 source_refs integer[] NOT NULL, subject_refs integer[] NOT NULL, submission_refs integer[] NOT NULL,
 states integer[] NOT NULL, operations integer[] NOT NULL, evidence integer[] NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), sealed boolean NOT NULL DEFAULT false,
 encoded_bytes integer NOT NULL CHECK(encoded_bytes>0 AND (encoded_bytes<=262144 OR cardinality(source_refs)=1)),
 written_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 submission_tags integer[] GENERATED ALWAYS AS (organization.submission_tags(ids,submission_refs)) STORED,
 PRIMARY KEY(instance_id,object_id,chunk_no),
 FOREIGN KEY(instance_id,enterprise_id,object_id) REFERENCES organization.department(instance_id,enterprise_id,department_id),
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.department_dictionary(instance_id,enterprise_id,id),
 CHECK(cardinality(source_refs) BETWEEN 1 AND 64 AND cardinality(source_refs)=cardinality(submission_refs)
 AND cardinality(source_refs)=cardinality(subject_refs) AND cardinality(source_refs)=cardinality(states)
 AND cardinality(source_refs)=cardinality(operations) AND cardinality(source_refs)=cardinality(evidence)),
 CHECK(array_position(ids,NULL) IS NULL AND cardinality(ids)<=1048576 AND array_ndims(ids)=1 AND array_lower(ids,1)=1),
 CHECK(array_ndims(source_refs)=1 AND array_lower(source_refs,1)=1 AND array_ndims(subject_refs)=1 AND array_lower(subject_refs,1)=1
 AND array_ndims(submission_refs)=1 AND array_lower(submission_refs,1)=1 AND array_ndims(states)=1 AND array_lower(states,1)=1
 AND array_ndims(operations)=1 AND array_lower(operations,1)=1 AND array_ndims(evidence)=1 AND array_lower(evidence,1)=1)
);
ALTER TABLE organization.department ADD FOREIGN KEY(instance_id,department_id,head_chunk) REFERENCES organization.department_history(instance_id,object_id,chunk_no) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX department_submission_lookup ON organization.department_history USING gin(submission_tags) WITH(fastupdate=off);

CREATE TABLE organization.department_terminal (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 request_object_id uuid, outcome text NOT NULL CHECK(outcome IN ('rejected','no_change','not_applied')),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL, operation integer NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), closed_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,subject_id) REFERENCES access.principal(instance_id,principal_id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.department_dictionary(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id,operation) REFERENCES organization.department_shape(instance_id,enterprise_id,id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON organization.department_terminal FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- +goose StatementBegin
CREATE FUNCTION organization.department_append_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE n integer; oldn integer; i integer; olddict organization.department_dictionary; newdict organization.department_dictionary;
 previous organization.department_history; sid uuid; actor uuid; s uuid; count_shapes integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'history cannot be deleted' USING ERRCODE='23514'; END IF;
 n:=cardinality(NEW.source_refs);
 IF n NOT BETWEEN 1 AND 64 THEN RAISE EXCEPTION 'bounded history directory required' USING ERRCODE='23514'; END IF;
 NEW.written_xid:=pg_current_xact_id();
 SELECT * INTO STRICT newdict FROM organization.department_dictionary WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=NEW.dictionary_id;
 IF TG_OP='UPDATE' THEN
  oldn:=cardinality(OLD.source_refs);
  IF OLD.sealed OR (NEW.instance_id,NEW.enterprise_id,NEW.object_id,NEW.chunk_no,NEW.first_no,NEW.format) IS DISTINCT FROM (OLD.instance_id,OLD.enterprise_id,OLD.object_id,OLD.chunk_no,OLD.first_no,OLD.format)
   OR NEW.ids[1:cardinality(OLD.ids)] IS DISTINCT FROM OLD.ids
   OR NEW.source_refs[1:oldn] IS DISTINCT FROM OLD.source_refs OR NEW.subject_refs[1:oldn] IS DISTINCT FROM OLD.subject_refs
   OR NEW.submission_refs[1:oldn] IS DISTINCT FROM OLD.submission_refs OR NEW.states[1:oldn] IS DISTINCT FROM OLD.states
   OR NEW.operations[1:oldn] IS DISTINCT FROM OLD.operations OR NEW.evidence[1:oldn] IS DISTINCT FROM OLD.evidence
   OR NOT (n=oldn+1 OR (n=oldn AND NEW.sealed AND NEW.ids=OLD.ids AND NEW.dictionary_id=OLD.dictionary_id AND NEW.envelope=OLD.envelope AND NEW.encoded_bytes=OLD.encoded_bytes))
  THEN RAISE EXCEPTION 'history prefixes cannot change' USING ERRCODE='23514'; END IF;
  SELECT * INTO STRICT olddict FROM organization.department_dictionary WHERE id=OLD.dictionary_id;
  IF coalesce(newdict.words[1:cardinality(olddict.words)],'{}') IS DISTINCT FROM olddict.words
   OR coalesce(newdict.binaries[1:cardinality(olddict.binaries)],'{}') IS DISTINCT FROM olddict.binaries
   OR coalesce(newdict.ranges[1:cardinality(olddict.ranges)],'{}') IS DISTINCT FROM olddict.ranges
  THEN RAISE EXCEPTION 'dictionary prefix cannot change' USING ERRCODE='23514'; END IF;
 ELSE
  oldn:=0;
  IF NEW.chunk_no=1 THEN
   IF NEW.first_no<>1 THEN RAISE EXCEPTION 'first source number required' USING ERRCODE='23514'; END IF;
  ELSE
   SELECT * INTO STRICT previous FROM organization.department_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no-1;
   IF NOT previous.sealed OR NEW.first_no<>previous.first_no+cardinality(previous.source_refs)
   THEN RAISE EXCEPTION 'preceding chunk must be sealed and contiguous' USING ERRCODE='23514'; END IF;
  END IF;
 END IF;
 -- The prefix above is unchanged and was checked at its original append.
 -- Shape rows are immutable and cannot be deleted. Only new references need
 -- existence/ownership checks; sealing a full block adds no references.
 FOR i IN oldn+1..n LOOP
  IF NEW.source_refs[i] IS NULL OR NEW.subject_refs[i] IS NULL OR NEW.submission_refs[i] IS NULL
   OR NEW.source_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids) OR NEW.subject_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids) OR NEW.submission_refs[i] NOT BETWEEN 1 AND cardinality(NEW.ids)
   OR NEW.states[i] IS NULL OR NEW.operations[i] IS NULL OR NEW.evidence[i] IS NULL
  THEN RAISE EXCEPTION 'invalid source directory' USING ERRCODE='23514'; END IF;
  SELECT count(*) INTO count_shapes FROM organization.department_shape WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=ANY(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]);
  IF count_shapes<>(SELECT count(DISTINCT x) FROM unnest(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]) x)
  THEN RAISE EXCEPTION 'missing or foreign history root' USING ERRCODE='23503'; END IF;
 END LOOP;
 FOR i IN oldn+1..n LOOP
  actor:=NEW.ids[NEW.subject_refs[i]]; sid:=NEW.ids[NEW.submission_refs[i]]; s:=NEW.ids[NEW.source_refs[i]];
  PERFORM organization.require_submission_lock('department',NEW.instance_id,actor,sid);
  IF NOT EXISTS(SELECT FROM access.principal WHERE instance_id=NEW.instance_id AND principal_id=actor)
  THEN RAISE EXCEPTION 'missing source actor' USING ERRCODE='23503'; END IF;
  IF EXISTS(SELECT FROM organization.department_terminal WHERE instance_id=NEW.instance_id AND subject_id=actor AND submission_id=sid)
   OR EXISTS(SELECT FROM organization.department_history h WHERE h.instance_id=NEW.instance_id
    AND h.submission_tags @> ARRAY[organization.submission_tag(sid)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,actor,sid) IS NOT NULL)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.subject_refs[j]]=actor AND NEW.ids[NEW.submission_refs[j]]=sid)
  THEN RAISE EXCEPTION 'submission already has a terminal' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT FROM organization.department_history h CROSS JOIN LATERAL unnest(h.source_refs) r WHERE h.instance_id=NEW.instance_id AND h.object_id=NEW.object_id AND h.ids[r]=s)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.source_refs[j]]=s)
  THEN RAISE EXCEPTION 'source identity already exists' USING ERRCODE='23505'; END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_history BEFORE INSERT OR UPDATE OR DELETE ON organization.department_history FOR EACH ROW EXECUTE FUNCTION organization.department_append_guard();

-- +goose StatementBegin
CREATE FUNCTION organization.department_head_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.department_history; current_head organization.department; id_count integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'object identity cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND ((to_jsonb(NEW)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) OR NEW.head_no<OLD.head_no OR NEW.delivered_no<OLD.delivered_no)
 THEN RAISE EXCEPTION 'object identity is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT current_head FROM organization.department WHERE instance_id=NEW.instance_id AND department_id=NEW.department_id;
 SELECT * INTO STRICT h FROM organization.department_history WHERE instance_id=current_head.instance_id AND object_id=current_head.department_id AND chunk_no=current_head.head_chunk;
 id_count:=cardinality(h.ids);
 IF current_head.head_no<>h.first_no+cardinality(h.source_refs)-1 OR current_head.head_source_id<>h.ids[h.source_refs[cardinality(h.source_refs)]]
 THEN RAISE EXCEPTION 'head must point to complete final source' USING ERRCODE='23514'; END IF;
 IF current_head.current_shape IS NULL OR current_head.current_refs IS NULL OR cardinality(current_head.current_refs)=0
  OR array_ndims(current_head.current_refs)<>1 OR array_lower(current_head.current_refs,1)<>1
  OR current_head.current_envelope IS NULL OR octet_length(current_head.current_envelope)<>32
  OR EXISTS(SELECT FROM unnest(current_head.current_refs) r WHERE r IS NULL OR r NOT BETWEEN 1 AND id_count)
  OR NOT EXISTS(SELECT FROM organization.department_shape s WHERE s.instance_id=current_head.instance_id AND s.enterprise_id=current_head.enterprise_id AND s.id=current_head.current_shape)
  OR EXISTS(SELECT FROM organization.department_history newer WHERE newer.instance_id=current_head.instance_id AND newer.object_id=current_head.department_id AND newer.chunk_no>current_head.head_chunk)
 THEN RAISE EXCEPTION 'current projection must reference the final owned source' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND NEW.head_no=OLD.head_no
  AND (NEW.current_shape,NEW.current_refs,NEW.current_envelope,NEW.management_bus,NEW.parents) IS DISTINCT FROM (OLD.current_shape,OLD.current_refs,OLD.current_envelope,OLD.management_bus,OLD.parents)
  AND current_head.head_xid<>pg_current_xact_id()
 THEN RAISE EXCEPTION 'current projection changes require an appended source' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER protect_head AFTER INSERT OR UPDATE OR DELETE ON organization.department DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.department_head_guard();
-- A chunk append cannot commit without advancing its owning head in the same transaction.
-- +goose StatementBegin
CREATE FUNCTION organization.department_closure_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.department_history; owner organization.department;
BEGIN
 SELECT * INTO STRICT h FROM organization.department_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no;
 SELECT * INTO STRICT owner FROM organization.department WHERE instance_id=h.instance_id AND department_id=h.object_id;
 IF owner.head_no<h.first_no+cardinality(h.source_refs)-1 OR owner.head_chunk<h.chunk_no
  OR (owner.head_chunk>h.chunk_no AND NOT h.sealed)
 THEN RAISE EXCEPTION 'source directory and current head must commit together' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER complete_history AFTER INSERT OR UPDATE ON organization.department_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.department_closure_guard();


-- +goose StatementBegin
CREATE FUNCTION organization.department_terminal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM organization.require_submission_lock('department',NEW.instance_id,NEW.subject_id,NEW.submission_id);
 IF EXISTS(SELECT FROM organization.department_history h WHERE h.instance_id=NEW.instance_id AND h.submission_tags @> ARRAY[organization.submission_tag(NEW.submission_id)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,NEW.subject_id,NEW.submission_id) IS NOT NULL)
 THEN RAISE EXCEPTION 'submission already accepted' USING ERRCODE='23505'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_terminal BEFORE INSERT ON organization.department_terminal FOR EACH ROW EXECUTE FUNCTION organization.department_terminal_guard();
GRANT SELECT,INSERT ON organization.department,organization.department_dictionary,organization.department_shape,organization.department_history,organization.department_terminal TO chnhcm_app;
GRANT UPDATE ON organization.department_history TO chnhcm_app;
GRANT DELETE ON organization.department_dictionary TO chnhcm_app;
-- Row-lock privilege only; the immutable trigger still rejects every UPDATE.
GRANT UPDATE(id) ON organization.department_dictionary TO chnhcm_app;
GRANT UPDATE(head_source_id,head_no,head_chunk,delivered_no,current_shape,current_refs,current_envelope) ON organization.department TO chnhcm_app;

-- Relation heads are object-owned; shared BU guards continue to serialize qualification.
ALTER TABLE organization.department ADD COLUMN management_bus uuid[] NOT NULL DEFAULT '{}', ADD COLUMN parents uuid[] NOT NULL DEFAULT '{}';
GRANT UPDATE(management_bus,parents) ON organization.department TO chnhcm_app;
CREATE INDEX department_bu_dependency ON organization.department USING gin(management_bus);
CREATE INDEX department_parent_dependency ON organization.department USING gin(parents);
CREATE TABLE organization.department_hierarchy_relation (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, department_id uuid NOT NULL, head_source_id uuid NOT NULL,
 PRIMARY KEY(instance_id,enterprise_id,department_id) INCLUDE(head_source_id),
 FOREIGN KEY(instance_id,enterprise_id,department_id) REFERENCES organization.department(instance_id,enterprise_id,department_id)
);
-- +goose StatementBegin
CREATE FUNCTION organization.department_relation_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE owner organization.department;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'relation identity is immutable' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (NEW.instance_id,NEW.enterprise_id,NEW.department_id) IS DISTINCT FROM (OLD.instance_id,OLD.enterprise_id,OLD.department_id)
 THEN RAISE EXCEPTION 'relation owner is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT owner FROM organization.department WHERE instance_id=NEW.instance_id AND department_id=NEW.department_id;
 IF owner.head_xid<>pg_current_xact_id() OR NOT EXISTS(SELECT FROM organization.department_history h WHERE h.instance_id=owner.instance_id AND h.object_id=owner.department_id AND h.chunk_no=owner.head_chunk AND NEW.head_source_id=ANY(h.ids))
 THEN RAISE EXCEPTION 'relation head requires the owning source transaction and exact identity' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER protect_relation AFTER INSERT OR UPDATE OR DELETE ON organization.department_hierarchy_relation DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.department_relation_guard();
GRANT SELECT,INSERT ON organization.department_hierarchy_relation TO chnhcm_app;
GRANT UPDATE(head_source_id) ON organization.department_hierarchy_relation TO chnhcm_app;
CREATE TABLE organization.storage_contract (singleton boolean PRIMARY KEY CHECK(singleton),contract text NOT NULL);
INSERT INTO organization.storage_contract VALUES(true,'m02-compact-5');
GRANT USAGE ON SCHEMA organization TO chnhcm_app;
GRANT SELECT ON organization.enterprise,organization.storage_contract,public.goose_db_version TO chnhcm_app;
GRANT UPDATE(initialization_digest) ON organization.enterprise TO chnhcm_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA organization TO chnhcm_app;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'compact business history cannot be automatically removed'; END $$;
-- +goose StatementEnd
