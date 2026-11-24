//go:build integration

package bootstrap_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

// Compare every published span with the complete immutable source, independently
// of the segment builder. This catches stale bindings and unchanged rows that a
// partial publication could otherwise leave behind outside today's page.
func assert133BUReadSegments(t *testing.T, f *fixture, object string) {
	t.Helper()
	state := f.buHead(object).State.Timeline
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer func() { must(t, tx.Rollback(f.ctx)) }()
	var number int32
	must(t, tx.QueryRow(f.ctx, `SELECT head_chunk FROM organization.business_unit WHERE bu_id=$1`, object).Scan(&number))
	chunk, err := history.LoadChunk(f.ctx, tx, history.BusinessUnit, f.instance, object, number)
	must(t, err)
	rows, err := tx.Query(f.ctx, `SELECT r.slot,r.source_id::text,coalesce(lower(r.span)::text,''),coalesce(upper(r.span)::text,''),r.dictionary_id,r.ids::text[],r.envelope,s.parts,s.digest,d.digest
 FROM organization.bu_read_segment r JOIN organization.bu_shape s ON s.id=r.shape_id JOIN organization.bu_dictionary d ON d.id=r.dictionary_id
 WHERE r.instance_id=$1 AND r.object_id=$2 ORDER BY r.slot`, f.instance, object)
	must(t, err)
	defer rows.Close()
	count := 0
	for rows.Next() {
		var slot, dictionary int32
		var source, start, end string
		var ids []string
		var envelope, shapeHash, dictionaryHash []byte
		var parts [][]byte
		must(t, rows.Scan(&slot, &source, &start, &end, &dictionary, &ids, &envelope, &parts, &shapeHash, &dictionaryHash))
		if int(slot) != count || count >= len(state.Records) || dictionary != chunk.Dictionary || source != state.Source {
			t.Fatal("incomplete or stale read segment", slot, dictionary, count)
		}
		expectedStart := state.Records[count].Date
		if count == 0 {
			expectedStart = ""
		}
		if start != expectedStart || end != state.Records[count].End {
			t.Fatal("wrong span after publication", count, start, end)
		}
		want := []d.BURecord{}
		indices := []int{0}
		if count > 0 {
			indices = append(indices, count)
		}
		for _, index := range indices {
			r := state.Records[index]
			want = append(want, d.BURecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name})
		}
		values := chunk.Values
		values.IDs = ids
		decoded, e := history.DecodeBURead(f.ctx, values, bytes.Join(parts, nil))
		must(t, e)
		if decoded.Timeline.ID != object || decoded.Timeline.Source != source || decoded.Timeline.First != state.First || !reflect.DeepEqual(decoded.Timeline.Records, want) {
			t.Fatal("read segment differs from original source", slot)
		}
		digest, e := history.ReadSegmentDigest(history.BusinessUnit, f.instance, chunk.Enterprise, object, source, start, end, ids, dictionaryHash, shapeHash)
		must(t, e)
		if !bytes.Equal(digest, envelope) {
			t.Fatal("read segment binding differs", slot)
		}
		count++
	}
	must(t, rows.Err())
	if count != len(state.Records) {
		t.Fatal("missing or extra read spans", count, len(state.Records))
	}
}

func assert133GenericHeadReferenceBounds(t *testing.T, f *fixture, object string) {
	t.Helper()
	for _, refs := range []string{"current_refs", "ARRAY[NULL]::integer[]", "ARRAY[0]", "ARRAY[2147483647]"} {
		tx, err := f.admin.Begin(f.ctx)
		must(t, err)
		defer tx.Rollback(f.ctx)
		_, err = tx.Exec(f.ctx, `SET LOCAL plan_cache_mode=force_generic_plan;SET LOCAL statement_timeout='1s'`)
		must(t, err)
		_, err = tx.Exec(f.ctx, `UPDATE organization.business_unit SET current_refs=`+refs+` WHERE bu_id=$1`, object)
		must(t, err)
		at := time.Now()
		_, err = tx.Exec(f.ctx, `SET CONSTRAINTS ALL IMMEDIATE`)
		if refs == "current_refs" {
			must(t, err)
			t.Logf("generic-plan full head reference check=%s; all original reference bounds retained", time.Since(at))
		} else {
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" || pg.Message != "current projection must reference the final owned source" {
				t.Fatal("invalid reference escaped native head guard", refs, err)
			}
		}
		must(t, tx.Rollback(f.ctx))
	}
}

func TestHistory133BUStableAndChangedReadSpansAreAtomic(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	id := b.call("POST", "/api/v1/business-units", createBody("WRITE133", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	for i, date := range []string{"2026-04-01", "2026-08-01", "2026-12-01"} {
		b.call("POST", path+"/records", addBody(detail(b, id), date, map[string]any{"name": fmt.Sprintf("边界%d", i)}), foundation.NewID(), 201)
		assert133BUReadSegments(t, f, id)
	}
	for _, i := range []int{0, 2, 3} {
		base := detail(b, id)
		r := base["timeline"].([]any)[i].(map[string]any)
		b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": base["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("更正%d", i)}}, foundation.NewID(), 200)
		assert133BUReadSegments(t, f, id)
	}
	for _, failure := range []string{"UPDATE", "INSERT"} {
		t.Run("rollback_"+failure, func(t *testing.T) {
			base := detail(b, id)
			r := base["timeline"].([]any)[2].(map[string]any)
			before := businessSnapshot(t, f)["sources"]
			f.exec(`CREATE TABLE public.read133_before AS SELECT * FROM organization.bu_read_segment`)
			f.exec(`CREATE FUNCTION public.fail_read133() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled read publication failure' USING ERRCODE='23514'; END $$; CREATE TRIGGER fail_read133 BEFORE ` + failure + ` ON organization.bu_read_segment FOR EACH ROW EXECUTE FUNCTION public.fail_read133()`)
			body := map[string]any{"expectedSourceVersion": base["headSourceVersion"], "expectedRecordRevision": r["revision"]}
			action := "correct"
			if failure == "UPDATE" {
				body["changes"] = map[string]any{"name": "回滚更名"}
			} else {
				action = "redate"
				body["newEffectiveDate"] = "2026-09-01"
			}
			key := foundation.NewID()
			b.call("POST", path+"/records/"+r["recordId"].(string)+"/"+action, body, key, 503)
			if detail(reader, id)["headSourceVersion"] != base["headSourceVersion"] || businessSnapshot(t, f)["sources"] != before || f.scalar(`SELECT count(*) FROM ((SELECT * FROM organization.bu_read_segment EXCEPT SELECT * FROM public.read133_before) UNION ALL (SELECT * FROM public.read133_before EXCEPT SELECT * FROM organization.bu_read_segment)) changed`) != 0 {
				t.Fatal("partial read publication escaped rollback")
			}
			resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if resolved["outcome"] != "not_applied" {
				t.Fatal("failed publication not resolved as not_applied")
			}
			f.exec(`DROP TRIGGER fail_read133 ON organization.bu_read_segment;DROP FUNCTION public.fail_read133();DROP TABLE public.read133_before`)
			b.call("POST", path+"/records/"+r["recordId"].(string)+"/"+action, body, foundation.NewID(), 200)
			assert133BUReadSegments(t, f, id)
		})
	}
	// Pause the writer after the head changes but before the derived rows change.
	// A different session must still observe the complete previous commit.
	base := detail(b, id)
	r := base["timeline"].([]any)[0].(map[string]any)
	holder, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer holder.Rollback(f.ctx)
	_, err = holder.Exec(f.ctx, `SELECT pg_advisory_xact_lock(133009)`)
	must(t, err)
	f.exec(`CREATE FUNCTION public.pause_read133() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(133009); RETURN NEW; END $$; CREATE TRIGGER pause_read133 BEFORE UPDATE ON organization.bu_read_segment FOR EACH ROW EXECUTE FUNCTION public.pause_read133()`)
	done := independent207Call(b, path+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": base["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "原子发布完成"}}, foundation.NewID())
	deadline := time.Now().Add(2 * time.Second)
	for f.scalar(`SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=133009 AND NOT granted`) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("writer did not reach publication barrier")
		}
		time.Sleep(time.Millisecond)
	}
	if detail(reader, id)["headSourceVersion"] != base["headSourceVersion"] {
		t.Fatal("reader saw uncommitted head")
	}
	assert133BUReadSegments(t, f, id)
	must(t, holder.Rollback(f.ctx))
	select {
	case reply := <-done:
		if reply.status != 200 || reply.err != nil {
			t.Fatal("publication did not complete", reply)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("publication did not release")
	}
	f.exec(`DROP TRIGGER pause_read133 ON organization.bu_read_segment;DROP FUNCTION public.pause_read133()`)
	assert133BUReadSegments(t, f, id)
	base = detail(b, id)
	r = base["timeline"].([]any)[1].(map[string]any)
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": base["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 200)
	assert133BUReadSegments(t, f, id)
}
