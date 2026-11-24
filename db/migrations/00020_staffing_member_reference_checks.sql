-- +goose Up
-- All three referenced fact tables reject UPDATE/DELETE and the application
-- has only SELECT/INSERT rights. Existence is therefore monotonic after commit:
-- structural references can be checked once per inserted batch without taking
-- a KEY SHARE tuple lock for every repeated historical member. Uncommitted
-- sources are not visible/eligible; same-transaction sources precede members.
-- This changes constraints only. No source/member/adoption/receipt is rewritten.
-- +goose StatementBegin
DO $$ BEGIN
 IF NOT (SELECT count(*)=6 FROM (VALUES ('grade_source','immutable_source'),('grade_revision','immutable_revision'),('job_grade_manifest','immutable_manifest'),('grade_ladder_grade_manifest','immutable_manifest'),('job_grade_member','immutable_member'),('grade_ladder_grade_member','immutable_member')) expected(table_name,trigger_name)
 JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.table_name) AND t.tgname=expected.trigger_name
 WHERE t.tgenabled='O' AND t.tgfoid=to_regprocedure('staffing.immutable_fact()')) THEN RAISE EXCEPTION 'immutable Staffing references are required'; END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION staffing.check_grade_member_references() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE invalid boolean;
BEGIN
 SELECT EXISTS(SELECT FROM (SELECT DISTINCT instance_id,grade_id,adopted_source,adopted_record,adopted_revision FROM new_members) m
 LEFT JOIN staffing.grade_source g ON (g.instance_id,g.object_id,g.source_id)=(m.instance_id,m.grade_id,m.adopted_source)
 LEFT JOIN staffing.grade_revision r ON (r.instance_id,r.object_id,r.record_id,r.revision)=(m.instance_id,m.grade_id,m.adopted_record,m.adopted_revision)
 WHERE g.source_id IS NULL OR r.revision IS NULL) INTO invalid;
 IF invalid THEN RAISE EXCEPTION 'membership adoption reference missing' USING ERRCODE='23503'; END IF;
 EXECUTE format('SELECT EXISTS(SELECT FROM (SELECT DISTINCT instance_id,object_id,source_id,record_id FROM new_members) m
 LEFT JOIN staffing.%I s USING(instance_id,object_id,source_id,record_id)
 WHERE s.record_id IS NULL)', replace(TG_TABLE_NAME,'_member','_manifest')) INTO invalid;
 IF invalid THEN RAISE EXCEPTION 'membership reference missing' USING ERRCODE='23503'; END IF;
 RETURN NULL;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ DECLARE r record; BEGIN
 FOR r IN SELECT conrelid::regclass AS tbl, conname FROM pg_constraint WHERE conrelid IN ('staffing.job_grade_member'::regclass,'staffing.grade_ladder_grade_member'::regclass) AND contype='f' LOOP
 EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.tbl,r.conname);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE TRIGGER member_references AFTER INSERT ON staffing.job_grade_member REFERENCING NEW TABLE AS new_members FOR EACH STATEMENT EXECUTE FUNCTION staffing.check_grade_member_references();
CREATE TRIGGER member_references AFTER INSERT ON staffing.grade_ladder_grade_member REFERENCING NEW TABLE AS new_members FOR EACH STATEMENT EXECUTE FUNCTION staffing.check_grade_member_references();

UPDATE staffing.storage_contract SET contract='staffing-grade-ladder-2' WHERE singleton AND contract='staffing-grade-ladder-1';

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Staffing reference constraints require an explicit forward migration; history is retained'; END $$;
-- +goose StatementEnd
