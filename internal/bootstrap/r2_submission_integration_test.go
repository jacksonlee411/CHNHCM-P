//go:build integration

package bootstrap_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"net/url"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestR2BUAdapterUsesOriginalLockAndIsolatesObjectNamespace(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	u, e := url.Parse(f.server.URL)
	must(t, e)
	creds := access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value, CSRF: b.csrf, Write: true}
	id := foundation.NewID()
	originalJSON, e := json.Marshal([]string{f.instance, b.subject, id})
	must(t, e)
	digest := sha256.Sum256(originalJSON)
	originalKey := int64(binary.BigEndian.Uint64(digest[:8]))
	departmentKey, e := (submission.Identity{Kind: submission.Department, InstanceID: f.instance, SubjectID: b.subject, SubmissionID: id}).LockKey()
	must(t, e)
	probe, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer probe.Rollback(f.ctx)
	svc := f.runtime.Organization.(application.Service)
	must(t, svc.Store.WithinWrite(f.ctx, creds, func(unit application.Unit) error {
		if e := unit.LockSubmission(f.ctx, id); e != nil {
			return e
		}
		var acquired bool
		if e := probe.QueryRow(f.ctx, "SELECT pg_try_advisory_xact_lock($1)", originalKey).Scan(&acquired); e != nil {
			return e
		}
		if acquired {
			t.Error("BU adapter no longer guards the original identity")
		}
		if e := probe.QueryRow(f.ctx, "SELECT pg_try_advisory_xact_lock($1)", departmentKey).Scan(&acquired); e != nil {
			return e
		}
		if !acquired {
			t.Error("different object namespace blocked by BU")
		}
		unit.Confirm()
		return nil
	}))
	var acquired bool
	must(t, probe.QueryRow(f.ctx, "SELECT pg_try_advisory_xact_lock($1)", originalKey).Scan(&acquired))
	if !acquired {
		t.Fatal("confirmed transaction retained its identity lock")
	}
}
