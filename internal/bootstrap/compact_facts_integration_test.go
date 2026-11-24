//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"encoding/json"
	"time"
)

// The oracle reads complete persisted facts through the privileged decoder;
// expected identities and business outcomes remain asserted by each journey.
func (f *fixture) history() bootstrap.HistoryInspection {
	f.t.Helper()
	facts, err := bootstrap.InspectHistory(f.ctx, f.admin, f.instance)
	must(f.t, err)
	return facts
}
func (f *fixture) eventCount() int {
	n := 0
	for _, v := range f.history().BUs {
		if v.Operation.EventID != "" {
			n++
		}
	}
	return n
}
func (f *fixture) revisionCount(deleted bool) int {
	ids := map[string]bool{}
	for _, v := range f.history().BUs {
		for _, r := range v.Operation.Revisions {
			if !deleted || r.Deleted {
				ids[r.Record.Revision] = true
			}
		}
	}
	return len(ids)
}
func (f *fixture) recordCount(object string) int {
	ids := map[string]bool{}
	for _, v := range f.history().BUs {
		if object != "" && v.Object != object {
			continue
		}
		for _, r := range v.Operation.Revisions {
			ids[r.Record.ID] = true
		}
	}
	return len(ids)
}
func (f *fixture) buSource(source string) bootstrap.BUHistoryFact {
	f.t.Helper()
	for _, v := range f.history().BUs {
		if v.Operation.Source.ID == source {
			return v
		}
	}
	f.t.Fatalf("persisted BU source %s missing", source)
	return bootstrap.BUHistoryFact{}
}
func (f *fixture) departmentSource(source string) bootstrap.DepartmentHistoryFact {
	f.t.Helper()
	for _, v := range f.history().Departments {
		if v.Operation.Source.ID == source {
			return v
		}
	}
	f.t.Fatalf("persisted department source %s missing", source)
	return bootstrap.DepartmentHistoryFact{}
}
func logicalBusinessSnapshot(f *fixture) map[string]string {
	f.t.Helper()
	out := map[string]string{}
	raw, e := json.Marshal(f.history().BUs)
	must(f.t, e)
	out["sources"] = string(raw)
	for label, table := range map[string]string{"business_unit": "business_unit", "terminals": "bu_terminal"} {
		var value string
		must(f.t, f.admin.QueryRow(f.ctx, "SELECT COALESCE(jsonb_agg(j ORDER BY j::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(t)-ARRAY['current_shape','current_refs','current_envelope','head_xid'] AS j FROM organization."+table+" t) q").Scan(&value))
		out[label] = value
	}
	return out
}

func (f *fixture) recordedAt(value string) func() {
	at, e := time.Parse(time.RFC3339Nano, value)
	must(f.t, e)
	return bootstrap.SetIntegrationRecordedClock(f.runtime, func() time.Time { return at })
}

func (f *fixture) assertBUInheritance() {
	f.t.Helper()
	for _, v := range f.history().BUs {
		for i, r := range v.State.Timeline.Records {
			if r.State == "inactive" {
				if i == 0 {
					f.t.Fatal("inactive first record")
				}
				previous := v.State.Timeline.Records[i-1]
				if r.InheritedRecord != previous.ID || r.InheritedRevision != previous.Revision || previous.End != r.Date {
					f.t.Fatal("inexact stored inheritance", v.Operation.Source.ID)
				}
			}
		}
	}
}

func (f *fixture) editBU(source string, edit func(*bootstrap.BUHistoryFact)) {
	f.t.Helper()
	var object string
	must(f.t, f.admin.QueryRow(f.ctx, `SELECT object_id::text FROM organization.bu_history h WHERE instance_id=$1 AND EXISTS(SELECT FROM unnest(h.source_refs) r WHERE h.ids[r]=$2::uuid)`, f.instance, source).Scan(&object))
	must(f.t, bootstrap.MutateIntegrationBU(f.ctx, f.admin, f.runtime, object, source, edit))
}
func (f *fixture) editDepartment(source string, edit func(*bootstrap.DepartmentHistoryFact)) {
	f.t.Helper()
	var object string
	must(f.t, f.admin.QueryRow(f.ctx, `SELECT object_id::text FROM organization.department_history h WHERE instance_id=$1 AND EXISTS(SELECT FROM unnest(h.source_refs) r WHERE h.ids[r]=$2::uuid)`, f.instance, source).Scan(&object))
	must(f.t, bootstrap.MutateIntegrationDepartment(f.ctx, f.admin, f.runtime, object, source, edit))
}
func (f *fixture) editBUResult(subject, submission string, edit func(*bootstrap.IntegrationBUTerminal)) {
	f.t.Helper()
	for _, v := range f.history().BUs {
		if v.Operation.Source.Subject == subject && v.Operation.Source.Submission == submission {
			f.editBU(v.Operation.Source.ID, func(fact *bootstrap.BUHistoryFact) { edit(&fact.Operation.Terminal) })
			return
		}
	}
	must(f.t, bootstrap.MutateIntegrationBUResult(f.ctx, f.admin, f.runtime, subject, submission, edit))
}
func (f *fixture) editDepartmentResult(subject, submission string, edit func(*bootstrap.IntegrationDepartmentTerminal)) {
	f.t.Helper()
	for _, v := range f.history().Departments {
		if v.Operation.Source.Subject == subject && v.Operation.Source.Submission == submission {
			f.editDepartment(v.Operation.Source.ID, func(fact *bootstrap.DepartmentHistoryFact) { edit(&fact.Operation.Terminal) })
			return
		}
	}
	must(f.t, bootstrap.MutateIntegrationDepartmentResult(f.ctx, f.admin, f.runtime, subject, submission, edit))
}

func (f *fixture) editBURevision(revision string, edit func(*bootstrap.IntegrationBURevision)) {
	f.t.Helper()
	for _, v := range f.history().BUs {
		found := false
		for _, r := range v.State.Timeline.Records {
			if r.Revision == revision {
				found = true
			}
		}
		for _, r := range v.Operation.Revisions {
			if r.Record.Revision == revision || r.Before != nil && r.Before.Revision == revision {
				found = true
			}
		}
		if !found {
			continue
		}
		f.editBU(v.Operation.Source.ID, func(v *bootstrap.BUHistoryFact) {
			for i := range v.State.Timeline.Records {
				r := &v.State.Timeline.Records[i]
				if r.Revision == revision {
					edit(r)
				}
			}
			for i := range v.Operation.Revisions {
				r := &v.Operation.Revisions[i]
				if r.Record.Revision == revision {
					edit(&r.Record)
				}
				if r.Before != nil && r.Before.Revision == revision {
					edit(r.Before)
				}
			}
		})
	}
}

func (f *fixture) eventsFor(submission string) int {
	n := 0
	for _, v := range f.history().BUs {
		if v.Operation.Source.Submission == submission && v.Operation.EventID != "" {
			n++
		}
	}
	return n
}

func (f *fixture) buHead(id string) bootstrap.BUHistoryFact {
	f.t.Helper()
	var source string
	must(f.t, f.admin.QueryRow(f.ctx, "SELECT head_source_id::text FROM organization.business_unit WHERE bu_id=$1", id).Scan(&source))
	return f.buSource(source)
}
func (f *fixture) departmentHead(id string) bootstrap.DepartmentHistoryFact {
	f.t.Helper()
	var source string
	must(f.t, f.admin.QueryRow(f.ctx, "SELECT head_source_id::text FROM organization.department WHERE department_id=$1", id).Scan(&source))
	return f.departmentSource(source)
}

func (f *fixture) jointItemCount() int {
	n := 0
	for _, v := range f.history().BUs {
		if v.Operation.Source.Action == "REVISE_TIMELINE" {
			for _, it := range v.Operation.Terminal.ItemOutcomes {
				if it.Outcome == "accepted" {
					n++
				}
			}
		}
	}
	return n
}
