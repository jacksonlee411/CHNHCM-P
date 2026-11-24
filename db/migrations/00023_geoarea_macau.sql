-- +goose Up
ALTER TABLE organization.geoarea DROP CONSTRAINT geoarea_country_code_check,
  ADD CHECK (country_code IN ('CN','HK','MO','TW'));
ALTER TABLE organization.geoarea_package DROP CONSTRAINT geoarea_package_country_code_check,
  ADD CHECK (country_code IN ('CN','HK','MO','TW'));

-- Sources without an official code retain their unique source_key and object ID.
ALTER TABLE organization.geoarea DROP CONSTRAINT geoarea_instance_id_country_id_scheme_normalized_code_key;
CREATE UNIQUE INDEX geoarea_assigned_code ON organization.geoarea(instance_id,country_id,scheme,normalized_code)
  WHERE normalized_code <> '';
ALTER TABLE organization.geoarea ADD CHECK (country_code = 'MO' OR code <> '');

-- +goose Down
SELECT 1/0;
