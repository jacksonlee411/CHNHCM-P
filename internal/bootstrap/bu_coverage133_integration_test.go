//go:build integration

package bootstrap_test

import (
	"context"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestBUCoveragePreview133CapturedReferenceIntegrity(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("COVERAGE133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	child := b.call("POST", "/api/v1/departments", departmentCreateBody("COVERAGE133_DEP", id), foundation.NewID(), 201)["departmentId"].(string)
	detail := b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	for _, tc := range []struct{ name, sql string }{
		{"source", `UPDATE organization.department_read_segment SET source_id=gen_random_uuid() WHERE object_id=$1`},
		{"relation", `UPDATE organization.department_read_segment SET q_relation=gen_random_uuid() WHERE object_id=$1`},
		{"record", `UPDATE organization.department_read_segment SET q_record=gen_random_uuid() WHERE object_id=$1`},
		{"seal", `UPDATE organization.department_read_segment SET q_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(f.ctx, tc.sql, child)
			must(t, err)
			v, err := history.CaptureBUCoverageReference(f.ctx, tx, f.instance, id, detail["headSourceVersion"].(string), []domain.BUCoverageRange{{Start: "2035-01-01"}})
			if err == nil || v != nil {
				t.Fatal("corrupt candidate manufactured a business conflict", err)
			}
		})
	}
}
