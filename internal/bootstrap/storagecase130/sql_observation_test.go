package storagecase130

import "testing"

func TestSQLStageNameUsesOnlyFixedLabels(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{"INSERT INTO organization.department_shape VALUES ('private')", "sql.insert.department_shape"},
		{"SELECT * FROM organization.department_history WHERE secret='private'", "sql.select.department_history"},
		{"copy organization.department_read_segment", "sql.copy.department_read_segment"},
		{"UPDATE access.session SET secret='private'", "sql.update.access"},
		{" COMMIT ", "sql.commit"},
		{"EXPLAIN SELECT 'private'", "sql.other"},
	} {
		if got := sqlStageName(tc.sql); got != tc.want {
			t.Errorf("fixed SQL label: got %q, want %q", got, tc.want)
		}
	}
}
