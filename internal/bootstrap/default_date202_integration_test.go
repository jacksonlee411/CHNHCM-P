//go:build integration

package bootstrap_test

import (
	"reflect"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestInitialDate202Enterprise(t *testing.T) {
	for _, start := range []string{"", "1880-01-01", "2020-01-01"} {
		t.Run("start="+start, func(t *testing.T) {
			f := freshWithEnterpriseStart(t, start)
			want := start
			if want == "" {
				want = "1900-01-01"
			}
			var before, after string
			query := "SELECT row_to_json(e)::text FROM organization.enterprise e WHERE instance_id=$1"
			must(t, f.admin.QueryRow(f.ctx, query, f.instance).Scan(&before))
			must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Login: "admin", Password: f.password}))
			must(t, f.admin.QueryRow(f.ctx, query, f.instance).Scan(&after))
			if before != after {
				t.Fatal("reopening rewrote enterprise")
			}
			b := f.login("admin", f.password)
			v := b.call("GET", "/api/v1/organization/context", nil, "", 200)
			if v["enterprise"].(map[string]any)["validFrom"] != want {
				t.Fatal("wrong initial enterprise date")
			}
			if e := bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1899-01-01", Login: "admin", Password: f.password}); e == nil {
				t.Fatal("explicit conflicting date overwritten")
			}
		})
	}
}

func TestInitialDate202Writes(t *testing.T) {
	for _, start := range []string{"", "1880-01-01", "2020-01-01"} {
		t.Run("start="+start, func(t *testing.T) {
			f := freshWithEnterpriseStart(t, start)
			b := f.login("admin", f.password)
			reader := f.login("reader", f.readPassword)
			now := time.Date(2026, 6, 14, 15, 59, 59, 0, time.UTC)
			restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return now })
			defer restore()
			now = now.Add(time.Second)
			date := "1900-01-01"
			if start == "1880-01-01" {
				date = "1890-01-01"
			}
			if start == "2020-01-01" {
				v := b.call("POST", "/api/v1/business-units", createBody("BAD_DEFAULT", date), foundation.NewID(), 400)
				if v["field"] != "effectiveDate" {
					t.Fatal("date rejection lost field")
				}
				date = start
			}
			bu := b.call("POST", "/api/v1/business-units", createBody("DATE202_BU", date), foundation.NewID(), 201)["buId"].(string)
			for _, kind := range []string{"business-units", "departments"} {
				body := createBody("DATE202_EXPLICIT", date)
				resolve, idField := "/api/v1/organization/submissions/", "buId"
				if kind == "departments" {
					body = departmentCreateBody("DATE202_DEP", bu)
					body["effectiveDate"] = date
					resolve, idField = "/api/v1/departments/submissions/", "departmentId"
				}
				path := "/api/v1/" + kind
				key := foundation.NewID()
				saved := b.call("POST", path, body, key, 201)
				id := saved[idField].(string)
				d := b.call("GET", path+"/"+id, nil, "", 200)
				if d["selectedRecord"].(map[string]any)["effectiveDate"] != date {
					t.Fatal("explicit date replaced")
				}
				if d["additionContext"].(map[string]any)["defaultEffectiveDate"] != "2026-06-15" {
					t.Fatal("append inherited initial default")
				}
				original := b.call("GET", path+"/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
				recovered := b.call("POST", resolve+key+"/resolve", nil, "", 200)
				if recovered["sourceVersion"] != saved["sourceVersion"] {
					t.Fatal("original result changed")
				}
				for _, bad := range []any{nil, "", "1900-02-29"} {
					body["effectiveDate"] = bad
					b.call("POST", path, body, foundation.NewID(), 400)
				}
				delete(body, "effectiveDate")
				b.call("POST", path, body, foundation.NewID(), 400)
				body["effectiveDate"] = date
				reader.call("POST", path, body, foundation.NewID(), 403)
				action := "bu.create"
				if kind == "departments" {
					action = "department.create"
				}
				f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'effectiveDate') WHERE principal_id=$1 AND action=$2", b.subject, action)
				b.call("POST", path, body, foundation.NewID(), 403)
				after := b.call("GET", path+"/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
				if !reflect.DeepEqual(original["timeline"], after["timeline"]) {
					t.Fatal("failed writes changed original history")
				}
			}
		})
	}
}

func TestInitialDate202Contexts(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	now := time.Date(2026, 6, 14, 15, 59, 59, 0, time.UTC)
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return now })
	defer restore()
	for _, day := range []string{"2026-06-14", "2026-06-15"} {
		for _, path := range []string{"/api/v1/organization/context", "/api/v1/departments/context"} {
			v := b.call("GET", path, nil, "", 200)
			if v["defaultEffectiveDate"] != "1900-01-01" || v["currentDate"] != day {
				t.Fatal("default and current date conflated")
			}
			v = reader.call("GET", path, nil, "", 200)
			if v["capabilities"].(map[string]any)["create"] != false {
				t.Fatal("reader may create")
			}
		}
		now = now.Add(time.Second)
	}
}

func TestInitialDate202Dependencies(t *testing.T) {
	f := freshWithEnterpriseStart(t, "")
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("LATE_BU", "1984-01-01"), foundation.NewID(), 201)["buId"].(string)
	body := departmentCreateBody("LATE_PARENT", bu)
	body["effectiveDate"] = "1900-01-01"
	v := b.call("POST", "/api/v1/departments", body, foundation.NewID(), 409)
	if v["code"] != "M02_DEPARTMENT_BU_CONFLICT" {
		t.Fatal("late BU not identified")
	}
	body["effectiveDate"] = "1990-01-01"
	parent := b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	parentPath := "/api/v1/departments/" + parent
	before := b.call("GET", parentPath, nil, "", 200)
	body = departmentCreateBody("LATE_CHILD", bu)
	body["effectiveDate"], body["parentDepartmentId"] = "1984-01-01", parent
	v = b.call("POST", "/api/v1/departments", body, foundation.NewID(), 409)
	if v["code"] != "M02_DEPARTMENT_PARENT_INVALID" {
		t.Fatal("late parent not identified")
	}
	body["effectiveDate"] = "1990-01-01"
	b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	after := b.call("GET", parentPath, nil, "", 200)
	if !reflect.DeepEqual(before["timeline"], after["timeline"]) || before["headSourceVersion"] != after["headSourceVersion"] {
		t.Fatal("dependent creation changed parent history")
	}
}
