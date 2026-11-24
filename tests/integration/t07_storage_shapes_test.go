//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"reflect"
	"testing"
	"time"
)

func t07SQLState(t *testing.T, e error, code string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != code {
		t.Fatalf("want SQLSTATE %s got %v", code, e)
	}
	t.Logf("SQLSTATE=%s constraint=%s", pg.Code, pg.ConstraintName)
}

func TestT07TerminalAndEventStorageShapes(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	key := foundation.NewID()
	created := b.call("POST", "/api/v1/business-units", createBody("T07_SHAPES", "2026-01-01"), key, 201)
	source := created["sourceVersion"].(string)
	original := f.buSource(source)
	before := businessSnapshot(t, f)
	cases := []struct {
		name string
		edit func(*bootstrap.BUHistoryFact)
	}{
		{"accepted_missing_target", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.Result.TargetRecordID = nil }},
		{"accepted_missing_source", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.Result.SourceVersion = nil }},
		{"rejected_with_success", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.Outcome = "rejected" }},
		{"no_change_with_success", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.Outcome = "no_change" }},
		{"not_applied_with_request", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.Outcome = "not_applied" }},
		{"half_evaluation", func(v *bootstrap.BUHistoryFact) { v.Operation.Terminal.EvaluatedAt = nil }},
		{"missing_terminal_close_time", func(v *bootstrap.BUHistoryFact) { v.Operation.TerminalClosedAt = time.Time{} }},
		{"missing_event", func(v *bootstrap.BUHistoryFact) { v.Operation.EventID = "" }},
		{"empty_data_and_structure", func(v *bootstrap.BUHistoryFact) {
			v.Operation.Impact.DataAffectedRanges = nil
			v.Operation.Impact.StructuralChanges = nil
			v.Operation.Terminal.Result.Impact = &v.Operation.Impact
		}},
		{"event_result_disagree", func(v *bootstrap.BUHistoryFact) { v.Operation.Impact.AffectedRecordIDs = []string{foundation.NewID()} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f.editBU(source, tc.edit)
			r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 503)
			if r["code"] != "M02_RESULT_UNAVAILABLE" {
				t.Fatal("wrong corrupt-result boundary", r)
			}
			f.editBU(source, func(v *bootstrap.BUHistoryFact) { *v = original })
			b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		})
	}
	t.Run("unsupported_physical_format", func(t *testing.T) {
		tx, e := f.admin.Begin(f.ctx)
		must(t, e)
		defer tx.Rollback(f.ctx)
		_, e = tx.Exec(f.ctx, `ALTER TABLE organization.bu_history DISABLE TRIGGER protect_history`)
		must(t, e)
		_, e = tx.Exec(f.ctx, `UPDATE organization.bu_history SET format=99`)
		t07SQLState(t, e, "23514")
	})
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("negative fixtures changed restored history")
	}
}

func TestT07InstanceOnlyMemberMismatchAndImmutablePrivileges(t *testing.T) {
	f := fresh(t)
	g := t07Instance(t, f, "2020-01-01")
	b, other := f.login("admin", f.password), g.login("admin", g.password)
	created := b.call("POST", "/api/v1/business-units", createBody("T07_ROLE", "2026-01-01"), foundation.NewID(), 201)
	key := foundation.NewID()
	b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	var enterprise string
	must(t, g.admin.QueryRow(g.ctx, `SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1`, g.instance).Scan(&enterprise))
	tx, e := g.runtime.Pool.Begin(g.ctx)
	must(t, e)
	defer tx.Rollback(g.ctx)
	sid := foundation.NewID()
	_, e = tx.Exec(g.ctx, `SELECT pg_advisory_xact_lock(organization.submission_key('bu',$1,$2,$3))`, g.instance, other.subject, sid)
	must(t, e)
	// The actor and enterprise are valid in the second instance; only the shared
	// dictionary/content roots belong to the first one.
	_, e = tx.Exec(g.ctx, `INSERT INTO organization.bu_terminal(instance_id,enterprise_id,subject_id,submission_id,request_object_id,outcome,ids,dictionary_id,operation,envelope) SELECT $1,$2,$3,$4,NULL,outcome,ids,dictionary_id,operation,envelope FROM organization.bu_terminal WHERE instance_id=$5 AND submission_id=$6`, g.instance, enterprise, other.subject, sid, f.instance, key)
	t07SQLState(t, e, "23503")
	must(t, tx.Rollback(g.ctx))
	for _, sql := range []string{
		`UPDATE organization.bu_shape SET parts=parts`, `DELETE FROM organization.bu_shape`,
		`UPDATE organization.bu_dictionary SET words=words`, `DELETE FROM organization.bu_history`,
		`UPDATE organization.bu_terminal SET envelope=envelope`, `DELETE FROM organization.bu_terminal`,

		`UPDATE access.security_event SET event_id=event_id`, `DELETE FROM access.security_event`,
		`UPDATE organization.business_unit SET code='CHANGED',normalized_code='changed'`,
		`UPDATE organization.business_unit SET first_record_id=gen_random_uuid()`,
		`DELETE FROM organization.business_unit`, `TRUNCATE organization.bu_history CASCADE`,
		`ALTER TABLE organization.bu_history DISABLE TRIGGER ALL`, `SET session_replication_role=replica`,
	} {
		if _, e = f.runtime.Pool.Exec(f.ctx, sql); e == nil {
			t.Fatalf("API role unexpectedly allowed %s", sql)
		}
	}
	before := f.buSource(created["sourceVersion"].(string))
	_, e = f.runtime.Pool.Exec(f.ctx, `UPDATE organization.business_unit SET delivered_no=head_no WHERE instance_id=$1`, f.instance)
	must(t, e)
	if !reflect.DeepEqual(before, f.buSource(created["sourceVersion"].(string))) {
		t.Fatal("delivery acknowledgement changed event content")
	}
	_, e = f.runtime.Pool.Exec(f.ctx, `UPDATE organization.business_unit SET delivered_no=0 WHERE instance_id=$1`, f.instance)
	t07SQLState(t, e, "23514")
}

func TestLifecyclePairSealedSourceAndDirectItemRelations(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	made := b.call("POST", "/api/v1/business-units", createBody("PAIR_SEAL", "2026-01-01"), foundation.NewID(), 201)
	id := made["buId"].(string)
	key := foundation.NewID()
	d := detail(b, id)
	result := b.call("POST", "/api/v1/business-units/"+id+"/lifecycle-pairs", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": selected(d)["recordId"], "deactivationDate": "2026-06-01", "reactivationDate": "2026-07-01", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}, key, 200)
	source := result["sourceVersion"].(string)
	original := f.buSource(source)
	for _, tc := range []struct {
		name string
		edit func(*bootstrap.IntegrationBUTerminal)
	}{
		{"wrong_before_revision", func(v *bootstrap.IntegrationBUTerminal) { v.Items[0].ExpectedRecordRevision = foundation.NewID() }},
		{"duplicate_item_identity", func(v *bootstrap.IntegrationBUTerminal) { v.Items[1].ItemID = v.Items[0].ItemID }},
		{"wrong_after_revision", func(v *bootstrap.IntegrationBUTerminal) {
			r := foundation.NewID()
			v.ItemOutcomes[0].AfterRevisionID = &r
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.editBUResult(b.subject, key, tc.edit)
			b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 503)
			f.editBU(source, func(v *bootstrap.BUHistoryFact) { *v = original })
		})
	}
	// A structurally appendable directory must still fail at commit without its
	// matching head. This tests the deferred closure, not an old-source seal.
	tx, e := f.runtime.Pool.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	next, submission := foundation.NewID(), foundation.NewID()
	_, e = tx.Exec(f.ctx, `SELECT pg_advisory_xact_lock(organization.submission_key('bu',$1,$2,$3))`, f.instance, b.subject, submission)
	must(t, e)
	_, e = tx.Exec(f.ctx, `UPDATE organization.bu_history SET ids=ids||ARRAY[$3::uuid,$4::uuid],source_refs=array_append(source_refs,cardinality(ids)+1),subject_refs=array_append(subject_refs,subject_refs[1]),submission_refs=array_append(submission_refs,cardinality(ids)+2),states=array_append(states,states[1]),operations=array_append(operations,operations[1]),evidence=array_append(evidence,evidence[1]),encoded_bytes=encoded_bytes+32,written_xid='1'::xid8 WHERE instance_id=$1 AND object_id=$2`, f.instance, id, next, submission)
	must(t, e)
	var stamped bool
	must(t, tx.QueryRow(f.ctx, `SELECT written_xid=pg_current_xact_id() FROM organization.bu_history WHERE instance_id=$1 AND object_id=$2`, f.instance, id).Scan(&stamped))
	if !stamped {
		t.Fatal("caller forged append transaction stamp")
	}
	_, e = tx.Exec(f.ctx, "SET CONSTRAINTS ALL IMMEDIATE")
	t07SQLState(t, e, "23514")
	must(t, tx.Rollback(f.ctx))
	if !reflect.DeepEqual(original, f.buSource(source)) {
		t.Fatal("failed append changed previous complete source")
	}
}
