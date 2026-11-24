//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentContextDoesNotLoadDepartmentForest(t *testing.T) {
	s := newDepartmentContract(t)
	lock, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer lock.Rollback(context.Background())
	_, e = lock.Exec(s.f.ctx, `LOCK TABLE organization.department_history,organization.department_shape IN ACCESS EXCLUSIVE MODE`)
	must(t, e)
	for _, b := range []*browser{s.b, s.actor} {
		previous := b.client.Timeout
		b.client.Timeout = 2 * time.Second
		v := b.call("GET", "/api/v1/departments/context", nil, "", 200)
		b.client.Timeout = previous
		if v["currentDate"] != s.day || v["capabilities"].(map[string]any)["create"] != (b == s.b) {
			t.Fatal("wrong context date/capability", v)
		}
	}
	// A create grant in a nonexistent BU must not become a create capability.
	mutateLifeGrants(s.f, s.b.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action='department.create'`, s.b.subject, foundation.NewID())
	v := s.b.call("GET", "/api/v1/departments/context", nil, "", 200)
	if v["capabilities"].(map[string]any)["create"] != false {
		t.Fatal("nonexistent BU scope produced create capability")
	}
}
