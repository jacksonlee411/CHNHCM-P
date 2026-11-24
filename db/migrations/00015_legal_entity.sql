-- +goose Up
-- LegalEntity owns enterprise facts and exact country adoption.
CREATE OR REPLACE FUNCTION organization.submission_key(kind text,instance_id uuid,subject_id uuid,submission_id uuid) RETURNS bigint LANGUAGE sql IMMUTABLE STRICT AS $$ SELECT ('x'||substr(encode(sha256(convert_to(CASE WHEN kind='enterprise' THEN '["organization.enterprise",' WHEN kind='department' THEN '["organization.department",' WHEN kind='legalentity' THEN '["organization.legalentity",' WHEN kind='location' THEN '["organization.location",' WHEN kind='geoarea' THEN '["organization.geoarea",' WHEN kind='country' THEN '["organization.country",' ELSE '[' END || '"'||instance_id::text||'","'||subject_id::text||'","'||submission_id::text||'"]','UTF8')),'hex'),1,16))::bit(64)::bigint $$;
CREATE TABLE organization.legal_entity (
 instance_id uuid NOT NULL, legal_entity_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 code text NOT NULL, normalized_code text NOT NULL, first_record_id uuid NOT NULL, head_source_id uuid NOT NULL,
 head_no bigint NOT NULL CHECK(head_no>0), head_chunk integer NOT NULL CHECK(head_chunk>0),
 head_xid xid8 NOT NULL DEFAULT pg_current_xact_id(),
 current_shape integer, current_refs integer[], current_envelope bytea,
 delivered_no bigint NOT NULL DEFAULT 0 CHECK(delivered_no>=0 AND delivered_no<=head_no),
 PRIMARY KEY(instance_id,legal_entity_id), UNIQUE(instance_id,enterprise_id,legal_entity_id), UNIQUE(instance_id,enterprise_id,normalized_code),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(normalized_code=lower(code)), CHECK(code ~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,39}$')
);
-- +goose StatementBegin
CREATE FUNCTION organization.legal_entity_stamp_head() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.head_no>OLD.head_no THEN NEW.head_xid:=pg_current_xact_id(); ELSE NEW.head_xid:=OLD.head_xid; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER stamp_head BEFORE UPDATE ON organization.legal_entity FOR EACH ROW EXECUTE FUNCTION organization.legal_entity_stamp_head();
CREATE INDEX legal_entity_pending_events ON organization.legal_entity(instance_id,legal_entity_id) WHERE delivered_no<head_no;
CREATE TABLE organization.legal_entity_dictionary (
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
CREATE TRIGGER immutable_dictionary BEFORE UPDATE ON organization.legal_entity_dictionary FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.legal_entity_shape (
 id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 parts bytea[] NOT NULL, digest bytea GENERATED ALWAYS AS (organization.history_parts_hash(parts)) STORED,
 UNIQUE(instance_id,enterprise_id,digest), UNIQUE(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(organization.history_parts_valid(parts))
);
CREATE TRIGGER immutable_shape BEFORE UPDATE OR DELETE ON organization.legal_entity_shape FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE organization.legal_entity_history (
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
 FOREIGN KEY(instance_id,enterprise_id,object_id) REFERENCES organization.legal_entity(instance_id,enterprise_id,legal_entity_id),
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.legal_entity_dictionary(instance_id,enterprise_id,id),
 CHECK(cardinality(source_refs) BETWEEN 1 AND 64 AND cardinality(source_refs)=cardinality(submission_refs)
 AND cardinality(source_refs)=cardinality(subject_refs) AND cardinality(source_refs)=cardinality(states)
 AND cardinality(source_refs)=cardinality(operations) AND cardinality(source_refs)=cardinality(evidence)),
 CHECK(array_position(ids,NULL) IS NULL AND cardinality(ids)<=1048576 AND array_ndims(ids)=1 AND array_lower(ids,1)=1),
 CHECK(array_ndims(source_refs)=1 AND array_lower(source_refs,1)=1 AND array_ndims(subject_refs)=1 AND array_lower(subject_refs,1)=1
 AND array_ndims(submission_refs)=1 AND array_lower(submission_refs,1)=1 AND array_ndims(states)=1 AND array_lower(states,1)=1
 AND array_ndims(operations)=1 AND array_lower(operations,1)=1 AND array_ndims(evidence)=1 AND array_lower(evidence,1)=1)
);
ALTER TABLE organization.legal_entity ADD FOREIGN KEY(instance_id,legal_entity_id,head_chunk) REFERENCES organization.legal_entity_history(instance_id,object_id,chunk_no) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX legal_entity_submission_lookup ON organization.legal_entity_history USING gin(submission_tags) WITH(fastupdate=off);

CREATE TABLE organization.legal_entity_terminal (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 request_object_id uuid, outcome text NOT NULL CHECK(outcome IN ('rejected','no_change','not_applied')),
 ids uuid[] NOT NULL, dictionary_id integer NOT NULL, operation integer NOT NULL,
 envelope bytea NOT NULL CHECK(octet_length(envelope)=32), closed_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,subject_id) REFERENCES access.principal(instance_id,principal_id),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 FOREIGN KEY(instance_id,enterprise_id,dictionary_id) REFERENCES organization.legal_entity_dictionary(instance_id,enterprise_id,id),
 FOREIGN KEY(instance_id,enterprise_id,operation) REFERENCES organization.legal_entity_shape(instance_id,enterprise_id,id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON organization.legal_entity_terminal FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- +goose StatementBegin
CREATE FUNCTION organization.legal_entity_append_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE n integer; oldn integer; i integer; olddict organization.legal_entity_dictionary; newdict organization.legal_entity_dictionary;
 previous organization.legal_entity_history; sid uuid; actor uuid; s uuid; count_shapes integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'history cannot be deleted' USING ERRCODE='23514'; END IF;
 n:=cardinality(NEW.source_refs);
 IF n NOT BETWEEN 1 AND 64 THEN RAISE EXCEPTION 'bounded history directory required' USING ERRCODE='23514'; END IF;
 NEW.written_xid:=pg_current_xact_id();
 SELECT * INTO STRICT newdict FROM organization.legal_entity_dictionary WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=NEW.dictionary_id;
 IF TG_OP='UPDATE' THEN
  oldn:=cardinality(OLD.source_refs);
  IF OLD.sealed OR (NEW.instance_id,NEW.enterprise_id,NEW.object_id,NEW.chunk_no,NEW.first_no,NEW.format) IS DISTINCT FROM (OLD.instance_id,OLD.enterprise_id,OLD.object_id,OLD.chunk_no,OLD.first_no,OLD.format)
   OR NEW.ids[1:cardinality(OLD.ids)] IS DISTINCT FROM OLD.ids
   OR NEW.source_refs[1:oldn] IS DISTINCT FROM OLD.source_refs OR NEW.subject_refs[1:oldn] IS DISTINCT FROM OLD.subject_refs
   OR NEW.submission_refs[1:oldn] IS DISTINCT FROM OLD.submission_refs OR NEW.states[1:oldn] IS DISTINCT FROM OLD.states
   OR NEW.operations[1:oldn] IS DISTINCT FROM OLD.operations OR NEW.evidence[1:oldn] IS DISTINCT FROM OLD.evidence
   OR NOT (n=oldn+1 OR (n=oldn AND NEW.sealed AND NEW.ids=OLD.ids AND NEW.dictionary_id=OLD.dictionary_id AND NEW.envelope=OLD.envelope AND NEW.encoded_bytes=OLD.encoded_bytes))
  THEN RAISE EXCEPTION 'history prefixes cannot change' USING ERRCODE='23514'; END IF;
  SELECT * INTO STRICT olddict FROM organization.legal_entity_dictionary WHERE id=OLD.dictionary_id;
  IF coalesce(newdict.words[1:cardinality(olddict.words)],'{}') IS DISTINCT FROM olddict.words
   OR coalesce(newdict.binaries[1:cardinality(olddict.binaries)],'{}') IS DISTINCT FROM olddict.binaries
   OR coalesce(newdict.ranges[1:cardinality(olddict.ranges)],'{}') IS DISTINCT FROM olddict.ranges
  THEN RAISE EXCEPTION 'dictionary prefix cannot change' USING ERRCODE='23514'; END IF;
 ELSE
  oldn:=0;
  IF NEW.chunk_no=1 THEN
   IF NEW.first_no<>1 THEN RAISE EXCEPTION 'first source number required' USING ERRCODE='23514'; END IF;
  ELSE
   SELECT * INTO STRICT previous FROM organization.legal_entity_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no-1;
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
  SELECT count(*) INTO count_shapes FROM organization.legal_entity_shape WHERE instance_id=NEW.instance_id AND enterprise_id=NEW.enterprise_id AND id=ANY(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]);
  IF count_shapes<>(SELECT count(DISTINCT x) FROM unnest(ARRAY[NEW.states[i],NEW.operations[i],NEW.evidence[i]]) x)
  THEN RAISE EXCEPTION 'missing or foreign history root' USING ERRCODE='23503'; END IF;
 END LOOP;
 FOR i IN oldn+1..n LOOP
  actor:=NEW.ids[NEW.subject_refs[i]]; sid:=NEW.ids[NEW.submission_refs[i]]; s:=NEW.ids[NEW.source_refs[i]];
  PERFORM organization.require_submission_lock('legalentity',NEW.instance_id,actor,sid);
  IF NOT EXISTS(SELECT FROM access.principal WHERE instance_id=NEW.instance_id AND principal_id=actor)
  THEN RAISE EXCEPTION 'missing source actor' USING ERRCODE='23503'; END IF;
  IF EXISTS(SELECT FROM organization.legal_entity_terminal WHERE instance_id=NEW.instance_id AND subject_id=actor AND submission_id=sid)
   OR EXISTS(SELECT FROM organization.legal_entity_history h WHERE h.instance_id=NEW.instance_id
    AND h.submission_tags @> ARRAY[organization.submission_tag(sid)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,actor,sid) IS NOT NULL)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.subject_refs[j]]=actor AND NEW.ids[NEW.submission_refs[j]]=sid)
  THEN RAISE EXCEPTION 'submission already has a terminal' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT FROM organization.legal_entity_history h CROSS JOIN LATERAL unnest(h.source_refs) r WHERE h.instance_id=NEW.instance_id AND h.object_id=NEW.object_id AND h.ids[r]=s)
   OR EXISTS(SELECT FROM generate_series(1,i-1) j WHERE NEW.ids[NEW.source_refs[j]]=s)
  THEN RAISE EXCEPTION 'source identity already exists' USING ERRCODE='23505'; END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_history BEFORE INSERT OR UPDATE OR DELETE ON organization.legal_entity_history FOR EACH ROW EXECUTE FUNCTION organization.legal_entity_append_guard();

-- +goose StatementBegin
CREATE FUNCTION organization.legal_entity_head_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.legal_entity_history; current_head organization.legal_entity; id_count integer;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'object identity cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND ((to_jsonb(NEW)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['head_source_id','head_no','head_chunk','head_xid','delivered_no','current_shape','current_refs','current_envelope','management_bus','parents']) OR NEW.head_no<OLD.head_no OR NEW.delivered_no<OLD.delivered_no)
 THEN RAISE EXCEPTION 'object identity is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT current_head FROM organization.legal_entity WHERE instance_id=NEW.instance_id AND legal_entity_id=NEW.legal_entity_id;
 SELECT * INTO STRICT h FROM organization.legal_entity_history WHERE instance_id=current_head.instance_id AND object_id=current_head.legal_entity_id AND chunk_no=current_head.head_chunk;
 -- Read the large directory's bound once. A cached generic subquery plan must
 -- not detoast that UUID array again for every current reference.
 id_count:=cardinality(h.ids);
 IF current_head.head_no<>h.first_no+cardinality(h.source_refs)-1 OR current_head.head_source_id<>h.ids[h.source_refs[cardinality(h.source_refs)]]
 THEN RAISE EXCEPTION 'head must point to complete final source' USING ERRCODE='23514'; END IF;
 IF current_head.current_shape IS NULL OR current_head.current_refs IS NULL OR cardinality(current_head.current_refs)=0
  OR array_ndims(current_head.current_refs)<>1 OR array_lower(current_head.current_refs,1)<>1
  OR current_head.current_envelope IS NULL OR octet_length(current_head.current_envelope)<>32
  OR EXISTS(SELECT FROM unnest(current_head.current_refs) r WHERE r IS NULL OR r NOT BETWEEN 1 AND id_count)
  OR NOT EXISTS(SELECT FROM organization.legal_entity_shape s WHERE s.instance_id=current_head.instance_id AND s.enterprise_id=current_head.enterprise_id AND s.id=current_head.current_shape)
  OR EXISTS(SELECT FROM organization.legal_entity_history newer WHERE newer.instance_id=current_head.instance_id AND newer.object_id=current_head.legal_entity_id AND newer.chunk_no>current_head.head_chunk)
 THEN RAISE EXCEPTION 'current projection must reference the final owned source' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND NEW.head_no=OLD.head_no
  AND (NEW.current_shape,NEW.current_refs,NEW.current_envelope) IS DISTINCT FROM (OLD.current_shape,OLD.current_refs,OLD.current_envelope)
  AND current_head.head_xid<>pg_current_xact_id()
 THEN RAISE EXCEPTION 'current projection changes require an appended source' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER protect_head AFTER INSERT OR UPDATE OR DELETE ON organization.legal_entity DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.legal_entity_head_guard();
-- A chunk append cannot commit without advancing its owning head in the same transaction.
-- +goose StatementBegin
CREATE FUNCTION organization.legal_entity_closure_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.legal_entity_history; owner organization.legal_entity;
BEGIN
 SELECT * INTO STRICT h FROM organization.legal_entity_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no;
 SELECT * INTO STRICT owner FROM organization.legal_entity WHERE instance_id=h.instance_id AND legal_entity_id=h.object_id;
 IF owner.head_no<h.first_no+cardinality(h.source_refs)-1 OR owner.head_chunk<h.chunk_no
  OR (owner.head_chunk>h.chunk_no AND NOT h.sealed)
 THEN RAISE EXCEPTION 'source directory and current head must commit together' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER complete_history AFTER INSERT OR UPDATE ON organization.legal_entity_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.legal_entity_closure_guard();


-- +goose StatementBegin
CREATE FUNCTION organization.legal_entity_terminal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM organization.require_submission_lock('legalentity',NEW.instance_id,NEW.subject_id,NEW.submission_id);
 IF EXISTS(SELECT FROM organization.legal_entity_history h WHERE h.instance_id=NEW.instance_id AND h.submission_tags @> ARRAY[organization.submission_tag(NEW.submission_id)] AND organization.submission_slot(h.ids,h.submission_refs,h.subject_refs,NEW.subject_id,NEW.submission_id) IS NOT NULL)
 THEN RAISE EXCEPTION 'submission already accepted' USING ERRCODE='23505'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_terminal BEFORE INSERT ON organization.legal_entity_terminal FOR EACH ROW EXECUTE FUNCTION organization.legal_entity_terminal_guard();
GRANT SELECT,INSERT ON organization.legal_entity,organization.legal_entity_dictionary,organization.legal_entity_shape,organization.legal_entity_history,organization.legal_entity_terminal TO chnhcm_app;
GRANT UPDATE ON organization.legal_entity_history TO chnhcm_app;
GRANT DELETE ON organization.legal_entity_dictionary TO chnhcm_app;
-- Row-lock privilege only; the immutable trigger still rejects every UPDATE.
GRANT UPDATE(id) ON organization.legal_entity_dictionary TO chnhcm_app;
GRANT UPDATE(head_source_id,head_no,head_chunk,delivered_no,current_shape,current_refs,current_envelope) ON organization.legal_entity TO chnhcm_app;


CREATE TABLE organization.legal_entity_read (
 instance_id uuid NOT NULL,enterprise_id uuid NOT NULL,object_id uuid NOT NULL,record_id uuid NOT NULL,valid_period daterange NOT NULL,
 name text NOT NULL,name_en text,state text NOT NULL CHECK(state IN ('active','inactive')),effective_date date NOT NULL,
 country_id uuid NOT NULL,subject_type text NOT NULL CHECK(subject_type IN ('legal_person','branch','unincorporated','other')),registration_type text,registration_number text,establishing_organization text,
 CHECK((registration_type IS NULL)=(registration_number IS NULL)),
 PRIMARY KEY(instance_id,object_id,record_id),FOREIGN KEY(instance_id,enterprise_id,object_id) REFERENCES organization.legal_entity(instance_id,enterprise_id,legal_entity_id)
);
CREATE INDEX legal_entity_read_country ON organization.legal_entity_read(instance_id,enterprise_id,country_id,object_id);
GRANT SELECT,INSERT,DELETE ON organization.legal_entity_read TO chnhcm_app;
GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA organization TO chnhcm_app;
CREATE TABLE organization.legal_entity_selection_draft (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL,
 subject_id uuid NOT NULL, session_id uuid NOT NULL, draft_id uuid NOT NULL,
 business_day date NOT NULL, query_key text NOT NULL,
 mode text NOT NULL CHECK(mode IN ('explicit','allMatching')),
 revision bigint NOT NULL CHECK(revision>0),
 member_count integer NOT NULL CHECK(member_count BETWEEN 0 AND 131072),
 selected_count integer NOT NULL CHECK(selected_count BETWEEN 0 AND member_count),
 created_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
 admission_slot smallint NOT NULL CHECK(admission_slot BETWEEN 1 AND 128),
 PRIMARY KEY(instance_id,draft_id), UNIQUE(instance_id,session_id), UNIQUE(instance_id,admission_slot),
 FOREIGN KEY(instance_id,enterprise_id) REFERENCES organization.enterprise,
 CHECK(expires_at=created_at+interval '30 minutes')
);
CREATE INDEX legal_entity_selection_expiry ON organization.legal_entity_selection_draft(instance_id,expires_at);
CREATE TABLE organization.legal_entity_selection_block (
 instance_id uuid NOT NULL, draft_id uuid NOT NULL, first_id uuid NOT NULL,
 members uuid[] NOT NULL, selected bit varying(1024) NOT NULL,
 PRIMARY KEY(instance_id,draft_id,first_id),
 FOREIGN KEY(instance_id,draft_id) REFERENCES organization.legal_entity_selection_draft ON DELETE CASCADE,
 CHECK(cardinality(members) BETWEEN 1 AND 1024),
 CHECK(array_position(members,NULL) IS NULL),
 CHECK(array_lower(members,1)=1 AND first_id=members[1]),
 CHECK(bit_length(selected)=cardinality(members))
);
GRANT SELECT,INSERT,UPDATE,DELETE ON organization.legal_entity_selection_draft,organization.legal_entity_selection_block TO chnhcm_app;


-- +goose StatementBegin
DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) AS def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  IF position('read_fields' IN c.def)>0 OR (position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0) THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action IN ('bu.read','bu.history.read','bu.export','bu.create','bu.record.add','bu.record.correct','bu.record.delete','bu.deactivation.delete','bu.reactivation.delete','bu.record.redate','bu.first.redate','bu.deactivation.redate','bu.reactivation.redate','bu.deactivate','bu.reactivate','bu.timeline.revise','bu.first.reclassify','bu.record.reclassify','bu.deactivation.reclassify','bu.reactivation.reclassify','department.read','department.history.read','department.export','department.create','department.record.add','department.record.correct','department.record.delete','department.deactivation.delete','department.reactivation.delete','department.record.redate','department.first.redate','department.deactivation.redate','department.reactivation.redate','department.deactivate','department.reactivate','department.first.reclassify','department.record.reclassify','department.deactivation.reclassify','department.reactivation.reclassify','department.reassign','department.reference.read','bu.reference.read','country.read','country.history.read','country.create','country.record.add','country.record.correct','country.deactivate','country.reactivate','geoarea.read','geoarea.history.read','geoarea.create','geoarea.record.add','geoarea.record.correct','geoarea.deactivate','geoarea.reactivate','location.read','location.history.read','location.export','location.create','location.record.add','location.record.correct','location.record.delete','location.record.redate','location.deactivate','location.reactivate','location.lifecycle.pair','enterprise.read','enterprise.history.read','enterprise.record.add','enterprise.record.correct','enterprise.record.delete','enterprise.deactivation.delete','enterprise.reactivation.delete','enterprise.record.redate','enterprise.first.redate','enterprise.deactivation.redate','enterprise.reactivation.redate','enterprise.deactivate','enterprise.reactivate','enterprise.first.reclassify','enterprise.record.reclassify','enterprise.deactivation.reclassify','enterprise.reactivation.reclassify','legalentity.read','legalentity.history.read','legalentity.export','legalentity.create','legalentity.record.add','legalentity.record.correct','legalentity.record.delete','legalentity.record.redate','legalentity.deactivate','legalentity.reactivate','legalentity.lifecycle.pair'));
ALTER TABLE access.subject_grant ADD CHECK((action LIKE 'legalentity.%' AND read_fields <@ ARRAY['code','name','nameEn','countryRegionId','subjectType','registrationType','registrationNumber','establishingOrganization','status','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','countryRegionId','subjectType','registrationType','registrationNumber','establishingOrganization','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'bu.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'department.%' AND read_fields <@ ARRAY['code','name','shortName','managementBuId','parentDepartmentId','departmentPath','status','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','shortName','managementBuId','parentDepartmentId','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'geoarea.%' AND read_fields <@ ARRAY['code','scheme','nature','level','name','nameEn','countryRegionId','parentAreaId','geoAreaPath','state','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','scheme','nature','level','name','nameEn','countryRegionId','parentAreaId','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'enterprise.%' AND read_fields <@ ARRAY['code','name','nameEn','state','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['name','nameEn','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'country.%' AND read_fields <@ ARRAY['code','name','nameEn','phoneCode','state','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','phoneCode','effectiveDate','operationType','operationReason']::text[]) OR (action LIKE 'location.%' AND read_fields <@ ARRAY['code','name','nameEn','countryRegionId','geoAreaId','geoAreaPath','addressLine','addressExtra','postalCode','status','effectiveDate','operationType','operationReason']::text[] AND write_fields <@ ARRAY['code','name','nameEn','countryRegionId','geoAreaId','addressLine','addressExtra','postalCode','effectiveDate','operationType','operationReason']::text[]));
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'legalentity.%' OR scope_kind='enterprise');

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'LegalEntity history is retained; destructive rollback is not supported'; END $$;
-- +goose StatementEnd
