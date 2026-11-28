-- +goose Up
ALTER TABLE personnel.address_type
 ADD COLUMN stop_day date,
 ADD COLUMN confirmed_from date,
 ADD COLUMN undated_stop boolean NOT NULL DEFAULT false;
-- A legacy switch proves the cutover state, never the business date of its last change.
UPDATE personnel.address_type SET confirmed_from=(transaction_timestamp() AT TIME ZONE 'Asia/Shanghai')::date,
 undated_stop=NOT selectable;
DROP INDEX personnel.address_type_choices;
ALTER TABLE personnel.address_type DROP COLUMN selectable,
 ADD CONSTRAINT address_type_period_bounds CHECK ((stop_day IS NULL OR stop_day>=first_day)
   AND (stop_day IS NULL OR confirmed_from IS NULL OR stop_day>=confirmed_from)
   AND (NOT undated_stop OR (confirmed_from IS NOT NULL AND stop_day IS NULL))),
 ADD COLUMN valid_during daterange GENERATED ALWAYS AS
 (CASE WHEN undated_stop OR (stop_day IS NOT NULL AND stop_day<=greatest(first_day,confirmed_from))
       THEN 'empty'::daterange ELSE daterange(greatest(first_day,confirmed_from),stop_day,'[)') END) STORED;
CREATE INDEX address_type_choices ON personnel.address_type(instance_id,sort_order,item_id);
GRANT UPDATE(stop_day,undated_stop) ON personnel.address_type TO chnhcm_app;

-- Original revisions and receipt digests stay intact. Their selectable flag is a recorded fact.
ALTER TABLE personnel.address_type_revision
 ADD COLUMN dated boolean NOT NULL DEFAULT false,
 ADD COLUMN stop_day date,
 ADD COLUMN undated_stop boolean NOT NULL DEFAULT false;
ALTER TABLE personnel.address_type_revision ALTER COLUMN dated SET DEFAULT true;
CREATE INDEX address_type_revision_order ON personnel.address_type_revision(instance_id,item_id,recorded_at,revision);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION personnel.initialize_address_types(customer uuid) RETURNS void LANGUAGE sql AS $$
 WITH seeded AS (
  INSERT INTO personnel.address_type(instance_id,code,name,sort_order,first_day,legacy_label)
  SELECT customer,code,label,position,DATE '1900-01-01',label FROM (VALUES
   ('RESIDENTIAL','居住地址',10),('REGISTERED','户籍地址',20),('MAILING','通讯地址',30),('OTHER','其他地址',40)
  ) AS initial(code,label,position) ON CONFLICT(instance_id,code) DO NOTHING RETURNING *
 )
 INSERT INTO personnel.address_type_revision(instance_id,item_id,revision,name,sort_order,selectable)
 SELECT instance_id,item_id,head,name,sort_order,true FROM seeded
$$;
-- +goose StatementEnd

-- +goose Down
SELECT 1/0;
