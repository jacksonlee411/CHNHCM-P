-- +goose Up
-- Country owns its facts and instance catalog scope. Reuse the fixed history
-- layout/guards; no enterprise or management BU is invented for this catalog.
CREATE OR REPLACE FUNCTION organization.submission_key(kind text,instance_id uuid,subject_id uuid,submission_id uuid) RETURNS bigint LANGUAGE sql IMMUTABLE STRICT AS $$ SELECT ('x'||substr(encode(sha256(convert_to(CASE WHEN kind='department' THEN '["organization.department",' WHEN kind='country' THEN '["organization.country",' ELSE '[' END || '"'||instance_id::text||'","'||subject_id::text||'","'||submission_id::text||'"]','UTF8')),'hex'),1,16))::bit(64)::bigint $$;
CREATE TABLE organization.country_region (
 instance_id uuid NOT NULL, country_region_id uuid NOT NULL, catalog_id uuid NOT NULL,
 code text NOT NULL, normalized_code text NOT NULL, source_key text NOT NULL, first_record_id uuid NOT NULL, head_source_id uuid NOT NULL,
 head_no bigint NOT NULL CHECK(head_no>0), head_chunk integer NOT NULL CHECK(head_chunk>0),
 head_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 current_shape integer, current_refs integer[], current_envelope bytea,
 delivered_no bigint NOT NULL DEFAULT 0 CHECK(delivered_no>=0 AND delivered_no<=head_no),
 PRIMARY KEY(instance_id,country_region_id), UNIQUE(instance_id,catalog_id,country_region_id), UNIQUE(instance_id,catalog_id,normalized_code),
 CHECK(catalog_id=instance_id),
 UNIQUE(instance_id,source_key), CHECK(normalized_code=lower(code)), CHECK(code ~ '^[A-Z]{2}$')
);
-- +goose StatementBegin
CREATE FUNCTION organization.country_stamp_head() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.head_no>OLD.head_no THEN NEW.head_xid:=pg_current_xact_id(); ELSE NEW.head_xid:=OLD.head_xid; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER stamp_head BEFORE UPDATE ON organization.country_region FOR EACH ROW EXECUTE FUNCTION organization.country_stamp_head();
CREATE INDEX country_pending_events ON organization.country_region(instance_id,country_region_id) WHERE delivered_no<head_no;
CREATE TABLE organization.country_dictionary (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, catalog_id uuid NOT NULL,
 words text[] NOT NULL, binaries bytea[] NOT NULL, ranges daterange[] NOT NULL,
 digest bytea GENERATED ALWAYS AS (organization.history_dictionary_hash(words,binaries,ranges)) STORED,
 UNIQUE(instance_id,catalog_id,digest), UNIQUE(instance_id,catalog_id,id),
 CHECK(catalog_id=instance_id),
 CHECK(array_position(words,NULL) IS NULL AND array_position(binaries,NULL) IS NULL AND array_position(ranges,NULL) IS NULL),
 CHECK(cardinality(words)<=1048576 AND cardinality(binaries)<=1048576 AND cardinality(ranges)<=1048576),
 CHECK((cardinality(words)=0 OR (array_ndims(words)=1 AND array_lower(words,1)=1))
 AND (cardinality(binaries)=0 OR (array_ndims(binaries)=1 AND array_lower(binaries,1)=1))
 AND organization.history_ranges_valid(ranges))
);
CREATE TRIGGER immutable_dictionary BEFORE UPDATE ON organization.country_dictionary FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.country_shape (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, catalog_id uuid NOT NULL,
 parts bytea[] NOT NULL, digest bytea GENERATED ALWAYS AS (organization.history_parts_hash(parts)) STORED,
 UNIQUE(instance_id,catalog_id,digest), UNIQUE(instance_id,catalog_id,id),
 CHECK(catalog_id=instance_id),
 CHECK(organization.history_parts_valid(parts))
);
CREATE TRIGGER immutable_shape BEFORE UPDATE OR DELETE ON organization.country_shape FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.country_history (
 instance_id uuid NOT NULL, catalog_id uuid NOT NULL, object_id uuid NOT NULL, chunk_no integer NOT NULL CHECK(chunk_no>0),
 first_no bigint NOT NULL CHECK(first_no>0), format smallint NOT NULL CHECK(format=1),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL,
 source_refs integer[] NOT NULL, subject_refs integer[] NOT NULL, submission_refs integer[] NOT NULL,
 states integer[] NOT NULL, operations integer[] NOT NULL, evidence integer[] NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), sealed boolean NOT NULL DEFAULT false,
 encoded_bytes integer NOT NULL CHECK(encoded_bytes>0 AND (encoded_bytes<=262144 OR cardinality(source_refs)=1)),
 written_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 submission_tags integer[] GENERATED ALWAYS AS (organization.submission_tags(ids,submission_refs)) STORED,
 PRIMARY KEY(instance_id,object_id,chunk_no),
 FOREIGN KEY(instance_id,catalog_id,object_id) REFERENCES organization.country_region(instance_id,catalog_id,country_region_id),
 FOREIGN KEY(instance_id,catalog_id,dictionary_id) REFERENCES organization.country_dictionary(instance_id,catalog_id,id),
 CHECK(cardinality(source_refs) BETWEEN 1 AND 64 AND cardinality(source_refs)=cardinality(submission_refs)
 AND cardinality(source_refs)=cardinality(subject_refs) AND cardinality(source_refs)=cardinality(states)
 AND cardinality(source_refs)=cardinality(operations) AND cardinality(source_refs)=cardinality(evidence)),
 CHECK(array_position(ids,NULL) IS NULL AND cardinality(ids)<=1048576 AND array_ndims(ids)=1 AND array_lower(ids,1)=1),
 CHECK(array_ndims(source_refs)=1 AND array_lower(source_refs,1)=1 AND array_ndims(subject_refs)=1 AND array_lower(subject_refs,1)=1
 AND array_ndims(submission_refs)=1 AND array_lower(submission_refs,1)=1 AND array_ndims(states)=1 AND array_lower(states,1)=1
 AND array_ndims(operations)=1 AND array_lower(operations,1)=1 AND array_ndims(evidence)=1 AND array_lower(evidence,1)=1)
);
ALTER TABLE organization.country_region ADD FOREIGN KEY(instance_id,country_region_id,head_chunk) REFERENCES organization.country_history(instance_id,object_id,chunk_no) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX country_submission_lookup ON organization.country_history USING gin(submission_tags) WITH(fastupdate=off);

CREATE TABLE organization.country_terminal (
 instance_id uuid NOT NULL, catalog_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 request_object_id uuid, outcome text NOT NULL CHECK(outcome IN ('rejected','no_change','not_applied')),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL, operation integer NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), closed_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,subject_id) REFERENCES access.principal(instance_id,principal_id),
 CHECK(catalog_id=instance_id),
 FOREIGN KEY(instance_id,catalog_id,dictionary_id) REFERENCES organization.country_dictionary(instance_id,catalog_id,id),
 FOREIGN KEY(instance_id,catalog_id,operation) REFERENCES organization.country_shape(instance_id,catalog_id,id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON organization.country_terminal FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- +goose StatementBegin
CREATE FUNCTION organization.country_append_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE n integer; oldn integer; i integer; olddict organization.country_dictionary; newdict organization.country_dictionary;
 previous organization.country_history; sid uuid; actor uuid; s uuid; count_shapes integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'history cannot be deleted' USING ERRCODE='23514'; END IF;
 n:=cardinality(NEW.source_refs);
 IF n NOT BETWEEN 1 AND 64 THEN RAISE EXCEPTION 'bounded history directory required' USING ERRCODE='23514'; END IF;
 NEW.written_xid:=pg_current_xact_id();
 SELECT * INTO STRICT newdict FROM organization.country_dictionary WHERE instance_id=NEW.instance_id AND catalog_id=NEW.catalog_id AND id=NEW.dictionary_id;
 IF TG_OP='UPDATE' THEN
  oldn:=cardinality(OLD.source_refs);
  IF OLD.sealed OR (NEW.instance_id,NEW.catalog_id,NEW.object_id,NEW.chunk_no,NEW.first_no,NEW.format) IS DISTINCT FROM (OLD.instance_id,OLD.catalog_id,OLD.object_id,OLD.chunk_no,OLD.first_no,OLD.format)
   OR NEW.ids[1:cardinality(OLD.ids)] IS DISTINCT FROM OLD.ids
   OR NEW.source_refs[1:oldn] IS DISTINCT FROM OLD.source_refs OR NEW.subject_refs[1:oldn] IS DISTINCT FROM OLD.subject_refs
   OR NEW.submission_refs[1:oldn] IS DISTINCT FROM OLD.submission_refs OR NEW.states[1:oldn] IS DISTINCT FROM OLD.states
   OR NEW.operations[1:oldn] IS DISTINCT FROM OLD.operations OR NEW.evidence[1:oldn] IS DISTINCT FROM OLD.evidence
   OR NOT (n=oldn+1 OR (n=oldn AND NEW.sealed AND NEW.ids=OLD.ids AND NEW.dictionary_id=OLD.dictionary_id AND NEW.envelope=OLD.envelope AND NEW.encoded_bytes=OLD.encoded_bytes))
  THEN RAISE EXCEPTION 'history prefixes cannot change' USING ERRCODE='23514'; END IF;
  SELECT * INTO STRICT olddict FROM organization.country_dictionary WHERE id=OLD.dictionary_id;
  IF coalesce(newdict.words[1:cardinality(olddict.words)],'{}') IS DISTINCT FROM olddict.words
   OR coalesce(newdict.binaries[1:cardinality(olddict.binaries)],'{}') IS DISTINCT FROM olddict.binaries
   OR coalesce(newdict.ranges[1:cardinality(olddict.ranges)],'{}') IS DISTINCT FROM olddict.ranges
  THEN RAISE EXCEPTION 'dictionary prefix cannot change' USING ERRCODE='23514'; END IF;
 ELSE
  oldn:=0;
  IF NEW.chunk_no=1 THEN
   IF NEW.first_no<>1 THEN RAISE EXCEPTION 'first source number required' USING ERRCODE='23514'; END IF;
  ELSE
   SELECT * INTO STRICT previous FROM organization.country_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no-1;
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
  SELECT count(*) INTO count_shapes FROM organization.country_shape WHERE instance_id=NEW.instance_id AND catalog_id=NEW.catalog_id AND id=ANY(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]);
  IF count_shapes<>(SELECT count(DISTINCT x) FROM unnest(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]) x)
  THEN RAISE EXCEPTION 'missing or foreign history root' USING ERRCODE='23503'; END IF;
 END LOOP;
 FOR i IN oldn+1..n LOOP
  actor:=NEW.ids[NEW.subject_refs[i]]; sid:=NEW.ids[NEW.submission_refs[i]]; s:=NEW.ids[NEW.source_refs[i]];
  PERFORM organization.require_submission_lock('country',NEW.instance_id,actor,sid);
  IF NOT EXISTS(SELECT FROM access.principal WHERE instance_id=NEW.instance_id AND principal_id=actor)
  THEN RAISE EXCEPTION 'missing source actor' USING ERRCODE='23503'; END IF;
  IF EXISTS(SELECT FROM organization.country_terminal WHERE instance_id=NEW.instance_id AND subject_id=actor AND submission_id=sid)
   OR EXISTS(SELECT FROM organization.country_history h WHERE h.instance_id=NEW.instance_id
    AND h.submission_tags @> ARRAY[organization.submission_tag(sid)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,actor,sid) IS NOT NULL)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.subject_refs[j]]=actor AND NEW.ids[NEW.submission_refs[j]]=sid)
  THEN RAISE EXCEPTION 'submission already has a terminal' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT FROM organization.country_history h CROSS JOIN LATERAL unnest(h.source_refs) r WHERE h.instance_id=NEW.instance_id AND h.object_id=NEW.object_id AND h.ids[r]=s)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.source_refs[j]]=s)
  THEN RAISE EXCEPTION 'source identity already exists' USING ERRCODE='23505'; END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_history BEFORE INSERT OR UPDATE OR DELETE ON organization.country_history FOR EACH ROW EXECUTE FUNCTION organization.country_append_guard();

-- +goose StatementBegin
CREATE FUNCTION organization.country_head_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.country_history; current_head organization.country_region; id_count integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'object identity cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND ((to_jsonb(NEW)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) OR NEW.head_no<OLD.head_no OR NEW.delivered_no<OLD.delivered_no)
 THEN RAISE EXCEPTION 'object identity is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT current_head FROM organization.country_region WHERE instance_id=NEW.instance_id AND country_region_id=NEW.country_region_id;
 SELECT * INTO STRICT h FROM organization.country_history WHERE instance_id=current_head.instance_id AND object_id=current_head.country_region_id AND chunk_no=current_head.head_chunk;
 -- Read the large directory's bound once. A cached generic subquery plan must
 -- not detoast that UUID array again for every current reference.
 id_count:=cardinality(h.ids);
 IF current_head.head_no<>h.first_no+cardinality(h.source_refs)-1 OR current_head.head_source_id<>h.ids[h.source_refs[cardinality(h.source_refs)]]
 THEN RAISE EXCEPTION 'head must point to complete final source' USING ERRCODE='23514'; END IF;
 IF current_head.current_shape IS NULL OR current_head.current_refs IS NULL OR cardinality(current_head.current_refs)=0
  OR array_ndims(current_head.current_refs)<>1 OR array_lower(current_head.current_refs,1)<>1
  OR current_head.current_envelope IS NULL OR octet_length(current_head.current_envelope)<>32
  OR EXISTS(SELECT FROM unnest(current_head.current_refs) r WHERE r IS NULL OR r NOT BETWEEN 1 AND id_count)
  OR NOT EXISTS(SELECT FROM organization.country_shape s WHERE s.instance_id=current_head.instance_id AND s.catalog_id=current_head.catalog_id AND s.id=current_head.current_shape)
  OR EXISTS(SELECT FROM organization.country_history newer WHERE newer.instance_id=current_head.instance_id AND newer.object_id=current_head.country_region_id AND newer.chunk_no>current_head.head_chunk)
 THEN RAISE EXCEPTION 'current projection must reference the final owned source' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND NEW.head_no=OLD.head_no
  AND (NEW.current_shape,NEW.current_refs,NEW.current_envelope) IS DISTINCT FROM (OLD.current_shape,OLD.current_refs,OLD.current_envelope)
  AND current_head.head_xid<>pg_current_xact_id()
 THEN RAISE EXCEPTION 'current projection changes require an appended source' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER protect_head AFTER INSERT OR UPDATE OR DELETE ON organization.country_region DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.country_head_guard();
-- A chunk append cannot commit without advancing its owning head in the same transaction.
-- +goose StatementBegin
CREATE FUNCTION organization.country_closure_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.country_history; owner organization.country_region;
BEGIN
 SELECT * INTO STRICT h FROM organization.country_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no;
 SELECT * INTO STRICT owner FROM organization.country_region WHERE instance_id=h.instance_id AND country_region_id=h.object_id;
 IF owner.head_no<h.first_no+cardinality(h.source_refs)-1 OR owner.head_chunk<h.chunk_no
  OR (owner.head_chunk>h.chunk_no AND NOT h.sealed)
 THEN RAISE EXCEPTION 'source directory and current head must commit together' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER complete_history AFTER INSERT OR UPDATE ON organization.country_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.country_closure_guard();


-- +goose StatementBegin
CREATE FUNCTION organization.country_terminal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM organization.require_submission_lock('country',NEW.instance_id,NEW.subject_id,NEW.submission_id);
 IF EXISTS(SELECT FROM organization.country_history h WHERE h.instance_id=NEW.instance_id AND h.submission_tags @> ARRAY[organization.submission_tag(NEW.submission_id)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,NEW.subject_id,NEW.submission_id) IS NOT NULL)
 THEN RAISE EXCEPTION 'submission already accepted' USING ERRCODE='23505'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_terminal BEFORE INSERT ON organization.country_terminal FOR EACH ROW EXECUTE FUNCTION organization.country_terminal_guard();
GRANT SELECT,INSERT ON organization.country_region,organization.country_dictionary,organization.country_shape,organization.country_history,organization.country_terminal TO chnhcm_app;
GRANT UPDATE ON organization.country_history TO chnhcm_app;
GRANT DELETE ON organization.country_dictionary TO chnhcm_app;
-- Row-lock privilege only; the immutable trigger still rejects every UPDATE.
GRANT UPDATE(id) ON organization.country_dictionary TO chnhcm_app;
GRANT UPDATE(head_source_id,head_no,head_chunk,delivered_no,current_shape,current_refs,current_envelope) ON organization.country_region TO chnhcm_app;

ALTER TABLE access.subject_grant DROP CONSTRAINT subject_grant_scope_kind_check, DROP CONSTRAINT subject_grant_action_check, DROP CONSTRAINT subject_grant_check2;
ALTER TABLE access.subject_grant ADD CHECK(scope_kind IN ('enterprise','bu','department','instance')), ADD CHECK(action IN ('bu.read','bu.history.read','bu.export','bu.create','bu.record.add','bu.record.correct','bu.record.delete','bu.deactivation.delete','bu.reactivation.delete','bu.record.redate','bu.first.redate','bu.deactivation.redate','bu.reactivation.redate','bu.deactivate','bu.reactivate','bu.timeline.revise','bu.first.reclassify','bu.record.reclassify','bu.deactivation.reclassify','bu.reactivation.reclassify','department.read','department.history.read','department.export','department.create','department.record.add','department.record.correct','department.record.delete','department.deactivation.delete','department.reactivation.delete','department.record.redate','department.first.redate','department.deactivation.redate','department.reactivation.redate','department.deactivate','department.reactivate','department.first.reclassify','department.record.reclassify','department.deactivation.reclassify','department.reactivation.reclassify','department.reassign','department.reference.read','bu.reference.read','country.read','country.history.read','country.create','country.record.add','country.record.correct','country.deactivate','country.reactivate')), ADD CHECK((action LIKE 'country.%')=(scope_kind='instance')), ADD CHECK(scope_kind<>'instance' OR scope_id=instance_id);

ALTER TABLE access.subject_grant ADD CHECK((action LIKE 'bu.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'department.%' AND read_fields <@ ARRAY['code','name','shortName','managementBuId','parentDepartmentId','departmentPath','status','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','shortName','managementBuId','parentDepartmentId','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'country.%' AND read_fields <@ ARRAY['code','name','nameEn','phoneCode','source','state','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','phoneCode','source','effectiveDate','operationType','operationReason']::text[]));


CREATE TABLE organization.country_package (
 instance_id uuid PRIMARY KEY, version text NOT NULL, package_digest text NOT NULL CHECK(length(package_digest)=64),
 archive_digest text NOT NULL CHECK(length(archive_digest)=64), country_count integer NOT NULL CHECK(country_count=251),
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER immutable_package BEFORE UPDATE OR DELETE ON organization.country_package FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
GRANT SELECT,INSERT ON organization.country_package TO chnhcm_app;
GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA organization TO chnhcm_app;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'country business history cannot be automatically removed'; END $$;
-- +goose StatementEnd
