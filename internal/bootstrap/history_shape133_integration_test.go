//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"testing"
	"time"
)

// Both transactions discover an absent shared immutable shape before either
// publishes it. The loser must resolve the committed identity in a fresh
// statement snapshot, and both real business histories must remain readable.
func TestHistory133ConcurrentAbsentShapePublication(t *testing.T) {
	f := fresh(t)
	clients := []*browser{f.login("admin", f.password), f.login("admin", f.password)}
	f.exec(`CREATE FUNCTION public.shape133_barrier() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock_shared(1330133::bigint); RETURN NEW; END $$`)
	f.exec(`CREATE TRIGGER shape133_barrier BEFORE INSERT ON organization.bu_shape FOR EACH ROW EXECUTE FUNCTION public.shape133_barrier()`)
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = tx.Exec(f.ctx, `SELECT pg_advisory_xact_lock(1330133::bigint)`)
	must(t, err)
	type reply struct {
		status int
		body   map[string]any
		err    error
	}
	replies := make(chan reply, 2)
	for i, client := range clients {
		go func(i int, client *browser) {
			status, body, err := client.response("POST", "/api/v1/business-units", createBody(fmt.Sprintf("SHAPE133_%d", i), "2026-01-01"), foundation.NewID())
			replies <- reply{status, body, err}
		}(i, client)
	}
	deadline := time.Now().Add(time.Second)
	for {
		var waiting int
		must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND query LIKE 'INSERT INTO organization.bu_shape%' AND wait_event='advisory'`).Scan(&waiting))
		if waiting == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("two absent-shape writers did not reach the real SQL barrier", waiting)
		}
		time.Sleep(5 * time.Millisecond)
	}
	must(t, tx.Commit(f.ctx))
	for range 2 {
		v := <-replies
		must(t, v.err)
		if v.status != 201 || v.body["outcome"] != "accepted" {
			t.Fatal(v.status, v.body)
		}
		id := v.body["buId"].(string)
		source := v.body["sourceVersion"].(string)
		if f.buHead(id).Operation.Source.ID != source || f.buSource(source).State.Timeline.BUID != id {
			t.Fatal("shared shape crossed business ownership")
		}
		restored := clients[0].call("POST", "/api/v1/organization/submissions/"+v.body["submissionId"].(string)+"/resolve", nil, "", 200)
		if restored["sourceVersion"] != source {
			t.Fatal("original receipt lost")
		}
		assert133BUReadSegments(t, f, id)
	}
}
