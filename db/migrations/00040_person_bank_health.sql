-- +goose Up
CREATE TABLE access.person_data_purpose (
 instance_id uuid NOT NULL REFERENCES access.person_policy, purpose text NOT NULL CHECK (purpose IN ('personal-bank','health-summary')),
 version text NOT NULL CHECK (version<>''), valid_until timestamptz NOT NULL,
 ordinary_days int NOT NULL CHECK (ordinary_days>0), evidence_days int NOT NULL CHECK (evidence_days>=ordinary_days),
 PRIMARY KEY(instance_id,purpose)
);
-- A policy change waits for in-flight uses of its customer-specific terms.
-- +goose StatementBegin
CREATE FUNCTION access.person_data_purposes_for_use(customer uuid) RETURNS SETOF access.person_data_purpose
 LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT * FROM access.person_data_purpose WHERE instance_id=customer AND valid_until>clock_timestamp() FOR SHARE
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION access.person_data_purposes_for_use(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION access.person_data_purposes_for_use(uuid) TO chnhcm_app;
CREATE TABLE access.person_sensitive_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,
 PRIMARY KEY(instance_id,principal_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.person_fixture);

ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_hire_action,
 ADD CONSTRAINT person_grant_hire_action CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct','family.read','family.correct','hire.check','qualification.read','qualification.correct','bank.read','bank.correct','health.read','health.correct'));
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_check,
 ADD CHECK(item_id IS NULL OR (denied AND source_version IS NULL AND (action IN ('profile.correct','family.read','family.correct','qualification.read','qualification.correct','bank.read','bank.correct')
 OR (action='profile.read' AND cardinality(fields)>0 AND fields <@ ARRAY['addressType','addressCountry','province','area','street','postalCode','addressPrimary']))));

CREATE TABLE personnel.bank_account (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, account_id uuid NOT NULL,
 PRIMARY KEY(instance_id,account_id), UNIQUE(instance_id,person_id,account_id),
 FOREIGN KEY(instance_id,person_id) REFERENCES personnel.person
);
GRANT SELECT,INSERT ON personnel.bank_account TO chnhcm_app;
ALTER TABLE personnel.profile_item_payload DROP CONSTRAINT profile_item_payload_section_check,
 ADD CHECK(section IN ('addresses','family','education','credentials','bankAccounts','health')),
 ADD COLUMN account_id uuid GENERATED ALWAYS AS (CASE WHEN section='bankAccounts' THEN item_id END) STORED,
 ADD FOREIGN KEY(instance_id,person_id,account_id) REFERENCES personnel.bank_account(instance_id,person_id,account_id);
CREATE UNIQUE INDEX person_bank_number ON personnel.profile_item_payload(instance_id,person_id,version,(fields->>'bankName'),
 (translate(fields->>'accountNumber','abcdefghijklmnopqrstuvwxyz ','ABCDEFGHIJKLMNOPQRSTUVWXYZ'))) WHERE section='bankAccounts';
CREATE UNIQUE INDEX person_bank_primary ON personnel.profile_item_payload(instance_id,person_id,version)
 WHERE section='bankAccounts' AND fields->>'bankPrimary'='yes';
CREATE UNIQUE INDEX person_health_summary ON personnel.profile_item_payload(instance_id,person_id,version) WHERE section='health';
ALTER TABLE personnel.hold DROP CONSTRAINT hold_category_check,
 ADD CHECK(category IN ('identity','evidence','contacts','addresses','family','education','credentials','bankAccounts','health'));
-- +goose Down
SELECT 1/0;
