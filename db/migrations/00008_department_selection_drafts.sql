-- +goose Up
-- Short-lived identity intent only. No business data, source or authorization
-- result belongs here. Department owns this table group independently of BU.
CREATE TABLE organization.department_selection_draft (
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
CREATE INDEX department_selection_expiry ON organization.department_selection_draft(instance_id,expires_at);
CREATE TABLE organization.department_selection_block (
 instance_id uuid NOT NULL, draft_id uuid NOT NULL, first_id uuid NOT NULL,
 members uuid[] NOT NULL, selected bit varying(1024) NOT NULL,
 PRIMARY KEY(instance_id,draft_id,first_id),
 FOREIGN KEY(instance_id,draft_id) REFERENCES organization.department_selection_draft ON DELETE CASCADE,
 CHECK(cardinality(members) BETWEEN 1 AND 1024),
 CHECK(array_position(members,NULL) IS NULL),
 CHECK(array_lower(members,1)=1 AND first_id=members[1]),
 CHECK(bit_length(selected)=cardinality(members))
);
GRANT SELECT,INSERT,UPDATE,DELETE ON organization.department_selection_draft,organization.department_selection_block TO chnhcm_app;

-- +goose Down
DROP TABLE organization.department_selection_block;
DROP TABLE organization.department_selection_draft;
