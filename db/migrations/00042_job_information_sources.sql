-- +goose Up
-- A business action keeps its original identity, author and registration time.
-- Corrections have their own revision and recording metadata.
ALTER TABLE staffing.employment_action
    DROP CONSTRAINT employment_action_instance_id_employment_id_kind_key,
    DROP CONSTRAINT employment_action_kind_check,
    DROP CONSTRAINT employment_action_check,
    ADD UNIQUE (instance_id, assignment_id, id),
    ADD CHECK (kind IN ('Hire','CancelHire','DataChange')),
    ADD CHECK ((kind='CancelHire')=(original_id IS NOT NULL));
CREATE UNIQUE INDEX employment_first_hire ON staffing.employment_action(instance_id,employment_id,kind)
    WHERE kind IN ('Hire','CancelHire');
CREATE TABLE staffing.employment_action_revision (
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, action_id uuid NOT NULL, revision uuid NOT NULL,
    previous_revision uuid, business_date date NOT NULL, notice text NOT NULL,
    actor uuid NOT NULL, operator_snapshot jsonb, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), adoption jsonb NOT NULL,
    PRIMARY KEY (instance_id,action_id,revision),
    UNIQUE (instance_id,assignment_id,action_id,revision),
    FOREIGN KEY (instance_id,assignment_id,action_id) REFERENCES staffing.employment_action(instance_id,assignment_id,id),
    FOREIGN KEY (instance_id,action_id,previous_revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision)
);
INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,business_date,notice,actor,recorded_at,adoption)
SELECT instance_id,assignment_id,id,revision,business_date,notice,actor,recorded_at,adoption FROM staffing.employment_action;
ALTER TABLE staffing.employment_action
    DROP COLUMN business_date, DROP COLUMN notice, DROP COLUMN adoption,
    ADD FOREIGN KEY (instance_id,id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE staffing.assignment_source (
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, version uuid NOT NULL, previous uuid,
    action_id uuid NOT NULL, revision uuid NOT NULL,
    PRIMARY KEY (instance_id,assignment_id,version),
    UNIQUE (instance_id,action_id,revision),
    FOREIGN KEY (instance_id,assignment_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    FOREIGN KEY (instance_id,assignment_id,previous) REFERENCES staffing.assignment_source
);
CREATE TABLE staffing.assignment_member (
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, version uuid NOT NULL,
    action_id uuid NOT NULL, revision uuid NOT NULL,
    PRIMARY KEY (instance_id,assignment_id,version,action_id),
    FOREIGN KEY (instance_id,assignment_id,version) REFERENCES staffing.assignment_source,
    FOREIGN KEY (instance_id,assignment_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision)
);
INSERT INTO staffing.assignment_source
SELECT a.instance_id,a.assignment_id,a.revision,orig.revision,a.id,a.revision
FROM staffing.employment_action a LEFT JOIN staffing.employment_action orig ON (orig.instance_id,orig.id)=(a.instance_id,a.original_id);
INSERT INTO staffing.assignment_member SELECT instance_id,assignment_id,version,action_id,revision FROM staffing.assignment_source;
INSERT INTO staffing.assignment_member
SELECT s.instance_id,s.assignment_id,s.version,p.action_id,p.revision
FROM staffing.assignment_source s JOIN staffing.assignment_source p ON (s.instance_id,s.assignment_id,s.previous)=(p.instance_id,p.assignment_id,p.version);

ALTER TABLE staffing.assignment_segment DROP CONSTRAINT assignment_segment_instance_id_assignment_id_period_excl,
    ADD COLUMN version uuid, ADD COLUMN location_id uuid, ADD COLUMN location_ref jsonb;
ALTER TABLE staffing.assignment_segment DISABLE TRIGGER immutable_fact;
UPDATE staffing.assignment_segment s SET version=h.revision,location_id=t.location_id
FROM staffing.employment_action h JOIN staffing.assignment t ON (t.instance_id,t.id)=(h.instance_id,h.assignment_id)
WHERE h.kind='Hire' AND (s.instance_id,s.assignment_id)=(h.instance_id,h.assignment_id);
ALTER TABLE staffing.assignment_segment ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.assignment_segment ALTER COLUMN version SET NOT NULL,
    ADD CONSTRAINT assignment_segment_source FOREIGN KEY (instance_id,assignment_id,version) REFERENCES staffing.assignment_source,
    ADD CONSTRAINT assignment_source_period EXCLUDE USING gist(instance_id WITH =,assignment_id WITH =,version WITH =,period WITH &&);

CREATE TABLE staffing.assignment_publication (
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, version uuid NOT NULL,
    PRIMARY KEY(instance_id,assignment_id,version),
    FOREIGN KEY(instance_id,assignment_id,version) REFERENCES staffing.assignment_source
);
INSERT INTO staffing.assignment_publication SELECT instance_id,assignment_id,version FROM staffing.assignment_source;
CREATE TABLE staffing.assignment_head (
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, version uuid NOT NULL,
    PRIMARY KEY(instance_id,assignment_id),
    FOREIGN KEY(instance_id,assignment_id,version) REFERENCES staffing.assignment_publication
);
INSERT INTO staffing.assignment_head
SELECT s.instance_id,s.assignment_id,s.version FROM staffing.assignment_source s
WHERE NOT EXISTS (SELECT FROM staffing.assignment_source n WHERE (n.instance_id,n.assignment_id,n.previous)=(s.instance_id,s.assignment_id,s.version));

ALTER TABLE staffing.employment_outbox DROP CONSTRAINT employment_outbox_instance_id_action_id_key, ADD COLUMN revision uuid;
ALTER TABLE staffing.employment_outbox DISABLE TRIGGER immutable_fact;
UPDATE staffing.employment_outbox o SET revision=a.revision FROM staffing.employment_action a WHERE (a.instance_id,a.id)=(o.instance_id,o.action_id);
ALTER TABLE staffing.employment_outbox ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.employment_outbox ALTER COLUMN revision SET NOT NULL, ADD UNIQUE(instance_id,action_id,revision),
    ADD FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision);
ALTER TABLE staffing.hire_terminal ADD COLUMN revision uuid;
ALTER TABLE staffing.hire_terminal DISABLE TRIGGER immutable_fact;
UPDATE staffing.hire_terminal t SET revision=a.revision FROM staffing.employment_action a WHERE (a.instance_id,a.id)=(t.instance_id,t.action_id);
ALTER TABLE staffing.hire_terminal ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.hire_terminal ADD CHECK ((action_id IS NULL)=(revision IS NULL)),
    ADD FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision);

-- +goose StatementBegin
DO $$ DECLARE rel text; BEGIN
    FOREACH rel IN ARRAY ARRAY['employment_action_revision','assignment_source','assignment_member','assignment_publication'] LOOP
        EXECUTE format('CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.%I FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact()',rel);
    END LOOP;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION staffing.sealed_assignment_source() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT FROM staffing.assignment_publication p WHERE (p.instance_id,p.assignment_id,p.version)=(NEW.instance_id,NEW.assignment_id,NEW.version)) THEN
        RAISE EXCEPTION 'assignment source already published' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER sealed_source BEFORE INSERT ON staffing.assignment_member FOR EACH ROW EXECUTE FUNCTION staffing.sealed_assignment_source();
CREATE TRIGGER sealed_source BEFORE INSERT ON staffing.assignment_segment FOR EACH ROW EXECUTE FUNCTION staffing.sealed_assignment_source();
GRANT SELECT,INSERT ON staffing.employment_action_revision,staffing.assignment_source,staffing.assignment_member,staffing.assignment_publication,staffing.assignment_head TO chnhcm_app;
GRANT UPDATE(version) ON staffing.assignment_head TO chnhcm_app;

ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK (action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory'));
CREATE TABLE access.employment_location_fixture (
    instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);
-- +goose Down
SELECT 1/0;
