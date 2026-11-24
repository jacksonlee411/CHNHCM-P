//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentBroadDependenciesRetainDeepGuardAndFrozenPath(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("BROAD_PARENT", "大分支原名", "BU_A", "", "2026-01-01", s.b, 201)
	for i := 0; i < 130; i++ {
		s.create(fmt.Sprintf("BROAD_CHILD_%03d", i), "子部门", "BU_A", "BROAD_PARENT", "2026-01-01", s.b, 201)
	}
	leaf := s.create("BROAD_LEAF", "末级部门", "BU_A", "BROAD_CHILD_129", "2026-01-01", s.b, 201)
	detail, record := s.target("BROAD_PARENT", "BROAD_PARENT_FIRST")
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "大分支新名"}}
	path := "/api/v1/departments/" + s.aliases["BROAD_PARENT"] + "/records/" + record["recordId"].(string) + "/correct"
	tx, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer tx.Rollback(context.Background())
	_, e = tx.Exec(s.f.ctx, "SELECT FROM organization.department WHERE department_id=$1 FOR UPDATE", s.aliases["BROAD_LEAF"])
	must(t, e)
	// Unchanged coverage must not lock a descendant, even in a broad hierarchy.
	s.b.call("POST", path, body, foundation.NewID(), 200)
	// A true relation change must still include the grandchild beyond level one.
	detail, record = s.target("BROAD_PARENT", "BROAD_PARENT_FIRST")
	change := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"managementBuId": s.aliases["BU_B"]}}
	key := foundation.NewID()
	s.b.call("POST", path, change, key, 503)
	if s.f.scalar("SELECT count(*) FROM organization.department_terminal WHERE submission_id=$1", key) != 0 {
		t.Fatal("dependency timeout became a business terminal")
	}
	must(t, tx.Rollback(s.f.ctx))
	s.b.call("POST", path, change, key, 409)
	base := "/api/v1/departments/" + s.aliases["BROAD_LEAF"]
	current := s.b.call("GET", base, nil, "", 200)
	frozen := s.b.call("GET", base+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 200)
	currentPath, e := json.Marshal(current["path"])
	must(t, e)
	frozenPath, e := json.Marshal(frozen["path"])
	must(t, e)
	if !strings.Contains(string(currentPath), "大分支新名") || !strings.Contains(string(frozenPath), "大分支原名") || strings.Contains(string(frozenPath), "大分支新名") {
		t.Fatal("broad dependency change lost current or frozen path facts")
	}
}

func TestDepartmentBroadMoveLoadsNewAncestorsWithoutLockingTheirSiblings(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("MOVE_ROOT", "原根", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("MOVE_TARGET", "移动部门", "BU_A", "MOVE_ROOT", "2026-01-01", s.b, 201)
	for i := 0; i < 130; i++ {
		s.create(fmt.Sprintf("MOVE_CHILD_%03d", i), "子部门", "BU_A", "MOVE_TARGET", "2026-01-01", s.b, 201)
	}
	leaf := s.create("MOVE_LEAF", "末级", "BU_A", "MOVE_CHILD_129", "2026-01-01", s.b, 201)
	s.create("DEST_ROOT", "新根", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("DEST_PARENT", "新上级", "BU_A", "DEST_ROOT", "2026-01-01", s.b, 201)
	s.create("DEST_SIBLING", "无关兄弟", "BU_A", "DEST_ROOT", "2026-01-01", s.b, 201)
	blocker, err := s.f.admin.Begin(s.f.ctx)
	must(t, err)
	defer blocker.Rollback(context.Background())
	_, err = blocker.Exec(s.f.ctx, "SELECT FROM organization.department WHERE department_id=$1 FOR UPDATE", s.aliases["DEST_SIBLING"])
	must(t, err)
	detail, record := s.target("MOVE_TARGET", "MOVE_TARGET_FIRST")
	s.b.call("POST", "/api/v1/departments/"+s.aliases["MOVE_TARGET"]+"/records/"+record["recordId"].(string)+"/correct",
		map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"],
			"changes": map[string]any{"parentDepartmentId": s.aliases["DEST_PARENT"]}}, foundation.NewID(), 200)
	base := "/api/v1/departments/" + s.aliases["MOVE_LEAF"]
	current := s.b.call("GET", base, nil, "", 200)
	frozen := s.b.call("GET", base+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 200)
	currentPath, err := json.Marshal(current["path"])
	must(t, err)
	frozenPath, err := json.Marshal(frozen["path"])
	must(t, err)
	if !strings.Contains(string(currentPath), "新根") || !strings.Contains(string(currentPath), "新上级") || strings.Contains(string(currentPath), "原根") || !strings.Contains(string(frozenPath), "原根") || strings.Contains(string(frozenPath), "新根") {
		t.Fatal("broad move lost new ancestors or original frozen path")
	}
}
