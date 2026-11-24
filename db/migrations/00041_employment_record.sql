-- +goose Up
CREATE TABLE staffing.employment_record (
    instance_id uuid NOT NULL,
    person_id uuid NOT NULL,
    record_number integer NOT NULL CHECK (record_number >= 0),
    PRIMARY KEY (instance_id, person_id, record_number)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.employment_record
    FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT, INSERT ON staffing.employment_record TO chnhcm_app;

-- Every assignment issued by 00037 was the first assignment of its employment.
-- A withdrawn hire still owns its number. The original actions and adoptions stay intact.
INSERT INTO staffing.employment_record (instance_id, person_id, record_number)
SELECT DISTINCT e.instance_id, e.person_id, 0
FROM staffing.assignment a
JOIN staffing.employment e ON (e.instance_id, e.id) = (a.instance_id, a.employment_id);

ALTER TABLE staffing.employment ADD UNIQUE (instance_id, id, person_id);
ALTER TABLE staffing.assignment ADD COLUMN person_id uuid, ADD COLUMN record_number integer;
ALTER TABLE staffing.assignment DISABLE TRIGGER immutable_fact;
UPDATE staffing.assignment a SET person_id = e.person_id, record_number = 0
FROM staffing.employment e WHERE (e.instance_id, e.id) = (a.instance_id, a.employment_id);
ALTER TABLE staffing.assignment ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.assignment
    ALTER COLUMN person_id SET NOT NULL,
    ALTER COLUMN record_number SET NOT NULL,
    ADD CONSTRAINT assignment_person FOREIGN KEY (instance_id, employment_id, person_id)
        REFERENCES staffing.employment (instance_id, id, person_id),
    ADD CONSTRAINT assignment_record FOREIGN KEY (instance_id, person_id, record_number)
        REFERENCES staffing.employment_record;

-- +goose Down
SELECT 1/0;
