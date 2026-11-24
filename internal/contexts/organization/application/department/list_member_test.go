package department

import (
	"context"
	"testing"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func TestCompactMembersKeepFieldVisibility(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	for _, fields := range [][]string{ReadFields, {"code", "name"}, {"name", "departmentPath", "status"}, {"code", "effectiveDate", "status", "departmentPath"}, {}} {
		var members []listMember
		for _, id := range []string{"b", "a"} {
			timeline := &d.Timeline{ID: id, Code: id, First: id + "-record", Source: id + "-source", Records: []d.Record{{ID: id + "-record", Revision: id + "-revision", Date: "2020-01-01", State: "active", Data: d.Data{Name: "部门 " + id, BU: "bu"}}}, Edges: []d.Edge{{Start: "2020-01-01", BU: "bu", Ref: d.MemberRef{DepartmentID: id, Source: id + "-relation", MemberID: id + "-member", BU: "bu"}}}}
			s := d.Snapshot{Departments: []d.Timeline{*timeline}}.Indexed()
			p := newProjector(context.Background(), projectedReadUnit{guard: pathDigestGuard{}}, s, org.Enterprise{Name: "企业"}, now)
			value, err := p.listPathValue(timeline, &timeline.Records[0], readMask(fields))
			if err != nil {
				t.Fatal(err)
			}
			member := listMember{t: timeline, r: &timeline.Records[0], fields: readMask(fields), status: &timeline.Records[0].State, path: value}
			if member.has("managementBuId") {
				member.bu = &org.DepartmentReference{ID: "bu", Code: ptr("事业部"), Name: ptr("管理单元")}
			}
			members = append(members, member)
		}
		rows := memberSummaries(members)
		if len(rows) != 2 || rows[0].ID != "b" || rows[1].ID != "a" {
			t.Fatal("member identities/order changed")
		}
		for _, row := range rows {
			has := func(field string) bool {
				for _, v := range fields {
					if v == field {
						return true
					}
				}
				return false
			}
			if (row.Code != nil) != has("code") || (row.Name != nil) != has("name") || (row.EffectiveDate != nil) != has("effectiveDate") || (row.Status != nil) != has("status") || (row.ManagementBU != nil) != has("managementBuId") {
				t.Fatal("field visibility", fields, row)
			}
			if (row.Path.Path != nil) != (has("departmentPath") && has("effectiveDate") && has("status")) {
				t.Fatal("path dependency visibility", fields, row)
			}
		}
	}
}
