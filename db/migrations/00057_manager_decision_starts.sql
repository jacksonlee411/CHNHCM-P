-- +goose Up
-- A sourced exit and an explicit appointment may take effect on the same
-- day in different publications. The graph guard enforces uniqueness within
-- the proposed timeline, including explicit decisions to end reporting.
DROP INDEX staffing.manager_start_per_purpose;
CREATE INDEX manager_start_per_purpose ON staffing.manager_agreement(instance_id,assignment_id,purpose,lower(period));

-- +goose StatementBegin
CREATE FUNCTION staffing.check_reporting_starts() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM staffing.assignment_member m
        JOIN staffing.manager_agreement g USING(instance_id,assignment_id,action_id,revision)
        WHERE (m.instance_id,m.assignment_id,m.version)=(NEW.instance_id,NEW.assignment_id,NEW.version)
        GROUP BY g.purpose,lower(g.period) HAVING count(*)>1
    ) THEN
        RAISE EXCEPTION 'multiple reporting decisions at one effective start in a publication' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER check_reporting_starts BEFORE INSERT ON staffing.assignment_publication
    FOR EACH ROW EXECUTE FUNCTION staffing.check_reporting_starts();

-- +goose Down
SELECT 1/0;
