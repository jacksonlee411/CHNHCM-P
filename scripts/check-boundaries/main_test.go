package main

import "testing"

func TestBoundaryViolations(t *testing.T) {
	for _, tc := range []struct {
		name, source, imported string
		blocked                bool
	}{
		{"Staffing shared text", "internal/contexts/staffing/domain/name.go", module + "internal/foundation", false},
		{"Staffing cannot import dated", "internal/contexts/staffing/domain/date.go", module + "internal/contexts/organization/domain/dated", true},
		{"Staffing cannot import submission", "internal/contexts/staffing/application/write.go", module + "internal/contexts/organization/application/submission", true},
		{"public cannot expose transaction", "internal/contexts/staffing/public/port.go", module + "internal/platform/dbtx", true},
		{"Staffing infrastructure transaction", "internal/contexts/staffing/infra/postgres/store.go", module + "internal/platform/dbtx", false},
		{"Organization reverse port", "internal/contexts/organization/infra/postgres/references.go", module + "internal/contexts/staffing/public", false},
		{"Organization cannot scan staffing adapter", "internal/contexts/organization/infra/postgres/references.go", module + "internal/contexts/staffing/infra/postgres", true},
		{"domain storage", "internal/contexts/organization/domain/object.go", "github.com/jackc/pgx/v5", true},
		{"domain HTTP", "internal/contexts/organization/domain/object.go", "net/http", true},
		{"domain SQL", "internal/contexts/organization/domain/object.go", "database/sql", true},
		{"application storage", "internal/contexts/organization/application/save.go", "github.com/jackc/pgx/v5", true},
		{"domain foreign DTO", "internal/contexts/organization/domain/object.go", module + "internal/contexts/access/public", true},
		{"cross-domain aggregate", "internal/contexts/staffing/application/hire.go", module + "internal/contexts/organization/domain", true},
		{"cross-domain public", "internal/contexts/staffing/infra/organization.go", module + "internal/contexts/organization/public", false},
		{"public leaks domain", "internal/contexts/organization/public/view.go", module + "internal/contexts/organization/domain", true},
		{"app depends on infra", "internal/contexts/organization/application/save.go", module + "internal/contexts/organization/infra/postgres", true},
		{"foundation reversed", "internal/foundation/date.go", module + "internal/contexts/organization/public", true},
		{"NFC shared primitive", "internal/foundation/unicode.go", "golang.org/x/text/unicode/norm", false},
		{"NFC does not open other shared files", "internal/foundation/date.go", "golang.org/x/text/unicode/norm", true},
		{"domain cannot import NFC library", "internal/contexts/organization/domain/name.go", "golang.org/x/text/unicode/norm", true},
		{"NFC wrapper cannot import adapter", "internal/foundation/unicode.go", "github.com/jackc/pgx/v5", true},
		{"NFC wrapper cannot use IO", "internal/foundation/unicode.go", "net/http", true},
		{"platform reversed", "internal/platform/job.go", module + "internal/contexts/organization/public", true},
		{"composition root", "internal/bootstrap/wire.go", module + "internal/contexts/organization/infra/postgres", false},
		{"command bypass", "cmd/server/main.go", module + "internal/contexts/organization/domain", true},
		{"pure shared values", "internal/contexts/organization/domain/object.go", module + "internal/foundation", false},
		{"stdlib", "internal/contexts/organization/domain/object.go", "time", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if actual := violation(tc.source, tc.imported) != ""; actual != tc.blocked {
				t.Fatalf("blocked = %t, want %t", actual, tc.blocked)
			}
		})
	}
}

func TestSQLTableOwnership(t *testing.T) {
	for _, tc := range []struct {
		path, query string
		blocked     bool
	}{
		{"db/queries/access/policy.sql", "select * from access.policy_guard", false},
		{"db/queries/staffing/query.sql", "select * from organization.bu", true},
		{"internal/contexts/organization/infra/postgres/query.go", "SELECT policy_no FROM access.policy_guard", true},
		{"internal/contexts/organization/infra/postgres/query.go", "access.Principal", false},
		{"db/queries/organization/query.sql", "-- FROM access.policy_guard\nselect 'from access.session' /* join staffing.job */ from organization.bu", false},
		{"db/queries/organization/query.sql", "FROM\n\"access\" /* lock */ .\n policy_guard", true},
		{"internal/contexts/staffing/infra/postgres/detail.sql", "FROM staffing.job j LEFT JOIN access.session s ON true", true},
		{"db/queries/organization/query.sql", "SELECT a.id, access.name FROM organization.bu AS access", false},
		{"db/queries/organization/query.sql", "SELECT 1 FROM organization.bu b, access.session", true},
		{"db/queries/organization/query.sql", "SELECT b.bu_id FROM (access.policy_guard g JOIN organization.business_unit b ON b.instance_id = g.instance_id)", true},
		{"db/queries/organization/query.sql", "SELECT 1 FROM organization.bu a JOIN ((\"access\".policy_guard g JOIN organization.bu b USING (instance_id))) ON true", true},
		{"db/queries/organization/query.sql", "SELECT a.bu_id FROM organization.business_unit a JOIN organization.business_unit AS access ON a.head_source_id IS DISTINCT FROM access.head_source_id", false},
		{"db/queries/organization/query.sql", "SELECT 1 FROM organization.bu AS staffing WHERE staffing.id IS NOT DISTINCT /* same value */ FROM staffing.id", false},
		{"db/queries/organization/query.sql", "SELECT EXTRACT(year FROM access.effective_date) FROM organization.bu AS access", false},
		{"db/queries/organization/query.sql", "SELECT 1 FROM organization.bu b WHERE b.id IS DISTINCT FROM (SELECT instance_id FROM access.policy_guard)", true},
		{"db/migrations/00001.sql", "select * from access.policy_guard", false},
	} {
		owner := sqlOwner(tc.path)
		if got := owner != "" && len(foreignTables(owner, tc.query)) != 0; got != tc.blocked {
			t.Errorf("%s: %q: blocked=%v", tc.path, tc.query, got)
		}
	}
}
