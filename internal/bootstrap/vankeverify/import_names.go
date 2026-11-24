package vankeverify

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"chnhcm.local/chnhcm/internal/bootstrap"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	domain "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sourceName struct {
	phase string
	row   map[string]string
}
type importNames struct {
	org.DepartmentService
	service  app.Service
	pool     *pgxpool.Pool
	instance string
	rootCode string
	approved map[string]sourceName
	admitted map[string]bool
}

func importKey(phase string, row int) string {
	// RFC 4122 UUIDv5 URL namespace; exactly matches the resumable importer.
	ns := []byte{0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	h := sha1.New()
	h.Write(ns)
	h.Write([]byte("chnhcm-vanke127/history-v1/" + phase + "/" + strconv.Itoa(row)))
	v := h.Sum(nil)[:16]
	v[6] = (v[6] & 15) | 0x50
	v[8] = (v[8] & 63) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", v[:4], v[4:6], v[6:8], v[8:10], v[10:])
}

func installImportNames(ctx context.Context, r *bootstrap.Runtime, pool *pgxpool.Pool) error {
	a := &accessdb.Store{Pool: r.Pool, Instance: r.Instance}
	s := &orgdb.Store{Pool: r.Pool, Instance: r.Instance, Guard: a.Guard}
	adapter := &importNames{DepartmentService: r.Department, service: app.Service{Store: s}, pool: pool, instance: r.Instance, approved: map[string]sourceName{}, admitted: map[string]bool{}}
	rows, e := pool.Query(ctx, "SELECT phase,row_no,payload FROM verification.source_row WHERE phase IN ('create','version','parent-change')")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var phase string
		var no int
		var raw []byte
		if e = rows.Scan(&phase, &no, &raw); e != nil {
			return e
		}
		adapter.admitted[importKey(phase, no)] = true
		row := map[string]string{}
		if e = json.Unmarshal(raw, &row); e != nil {
			return e
		}
		if phase == "create" && row["is_business_unit"] == "true" {
			if adapter.rootCode != "" {
				return errors.New("historical input requires one BU root")
			}
			adapter.rootCode = row["org_code"]
		}
		name := row["name"]
		if phase == "version" {
			name = row["core.name"]
		}
		if name == "" {
			continue
		}
		if _, e = domain.NormalizeName(name, true); e == nil {
			continue
		}
		if _, e = domain.HistoricalName(name); e != nil {
			return e
		}
		adapter.approved[importKey(phase, no)] = sourceName{phase, row}
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if adapter.rootCode == "" {
		return errors.New("historical input has no BU root")
	}
	r.Department = adapter
	return nil
}

func (s *importNames) Execute(ctx context.Context, c foundation.Credentials, cmd org.DepartmentCommand) (org.DepartmentResult, error) {
	if s.admitted[cmd.SubmissionID] {
		ctx = foundation.WithOfflineImportGuardBudget(ctx)
	}
	approved, ok := s.approved[cmd.SubmissionID]
	if !ok {
		return s.DepartmentService.Execute(ctx, c, cmd)
	}
	r := approved.row
	name := r["name"]
	fail := func() (org.DepartmentResult, error) {
		return org.DepartmentResult{}, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "历史导入与已核对来源不符")
	}
	if approved.phase == "create" {
		if cmd.Action != "CREATE" || cmd.Create.Code != r["org_code"] || cmd.Create.Name != name || cmd.Create.EffectiveDate != r["effective_date"] || cmd.Create.ShortName != nil || cmd.Create.OperationType != "CREATE" || cmd.Create.OperationReason != "NEW_ORGANIZATION" {
			return fail()
		}
		var root string
		if e := s.pool.QueryRow(ctx, "SELECT bu_id::text FROM organization.business_unit WHERE instance_id=$1 AND code=$2", s.instance, s.rootCode).Scan(&root); e != nil {
			return org.DepartmentResult{}, e
		}
		if cmd.Create.ManagementBUID != root {
			return fail()
		}
		parent := ""
		if r["parent_org_code"] != s.rootCode {
			if e := s.pool.QueryRow(ctx, "SELECT department_id::text FROM organization.department WHERE instance_id=$1 AND code=$2", s.instance, r["parent_org_code"]).Scan(&parent); e != nil {
				return org.DepartmentResult{}, e
			}
		}
		got := ""
		if cmd.Create.ParentDepartmentID != nil {
			got = *cmd.Create.ParentDepartmentID
		}
		if got != parent {
			return fail()
		}
	} else {
		name = r["core.name"]
		if cmd.Action != "ADD" || cmd.Add.EffectiveDate != r["effective_date"] || len(cmd.Add.Changes) != 1 || cmd.Add.Changes["name"] == nil || *cmd.Add.Changes["name"] != name || cmd.Add.OperationType != "CHANGE" || cmd.Add.OperationReason != "NAME_ADJUSTMENT" {
			return fail()
		}
		var code string
		if e := s.pool.QueryRow(ctx, "SELECT code FROM organization.department WHERE instance_id=$1 AND department_id=$2", s.instance, cmd.DepartmentID).Scan(&code); e != nil {
			return org.DepartmentResult{}, e
		}
		if code != r["org_code"] {
			return fail()
		}
	}
	return s.service.ExecuteHistoricalName(ctx, c, cmd, name)
}
