//go:build integration

package bootstrap_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// This is a publication-protocol proof, not a replacement codec. Its payloads
// are the SAME complete current-model read rows, including source/first member,
// dictionaries, UUIDs and query seals. Thus a new source still rewrites every
// payload. The experiment cannot claim one-row business-save cost.
func prototype133BindingDesign(t *testing.T, f *fixture, object string) {
	assert133BUReadSegments(t, f, object)
	var source string
	must(t, f.admin.QueryRow(f.ctx, `SELECT head_source_id::text FROM organization.business_unit WHERE bu_id=$1`, object).Scan(&source))
	rows, err := f.admin.Query(f.ctx, `SELECT convert_to(to_jsonb(r)::text,'UTF8') FROM organization.bu_read_segment r WHERE object_id=$1 ORDER BY slot`, object)
	must(t, err)
	payloads := [][]byte{}
	for rows.Next() {
		var raw []byte
		must(t, rows.Scan(&raw))
		payloads = append(payloads, raw)
	}
	must(t, rows.Err())
	rows.Close()
	if len(payloads) != 10000 {
		t.Fatal("complete same-input comparison requires 10000 read members")
	}
	// Test-owned tables deliberately leave member damage possible so the reader
	// must prove rejection, rather than relying on privileged test corruption.
	f.exec(`CREATE TABLE public.r133_publication(instance uuid,object uuid,source uuid,members integer NOT NULL,root bytea NOT NULL,PRIMARY KEY(instance,object,source));
CREATE TRIGGER immutable_r133_publication BEFORE UPDATE OR DELETE ON public.r133_publication FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TABLE public.r133_member(instance uuid,object uuid,source uuid,slot integer,payload bytea NOT NULL,PRIMARY KEY(instance,object,source,slot),FOREIGN KEY(instance,object,source) REFERENCES public.r133_publication DEFERRABLE INITIALLY DEFERRED);
CREATE TABLE public.r133_head(instance uuid,object uuid,source uuid,PRIMARY KEY(instance,object),FOREIGN KEY(instance,object,source) REFERENCES public.r133_publication DEFERRABLE INITIALLY DEFERRED);
CREATE TABLE public.r133_binding(instance uuid,object uuid,source uuid,PRIMARY KEY(instance,object),FOREIGN KEY(instance,object,source) REFERENCES public.r133_publication DEFERRABLE INITIALLY DEFERRED);
GRANT SELECT,INSERT,UPDATE,DELETE ON public.r133_publication,public.r133_member,public.r133_head,public.r133_binding TO chnhcm_app`)
	defer f.exec(`DROP TABLE public.r133_binding,public.r133_head,public.r133_member,public.r133_publication`)
	root := func(owner, version string, members [][]byte) []byte {
		h := sha256.New()
		for _, v := range []string{"r133-proof-1", f.instance, owner, version} {
			h.Write(binary.BigEndian.AppendUint32(nil, uint32(len(v))))
			h.Write([]byte(v))
		}
		h.Write(binary.BigEndian.AppendUint32(nil, uint32(len(members))))
		for i, p := range members {
			h.Write(binary.BigEndian.AppendUint32(nil, uint32(i)))
			digest := sha256.Sum256(p)
			h.Write(digest[:])
		}
		return h.Sum(nil)
	}
	publish := func(tx pgx.Tx, mode, version string) error {
		_, e := tx.Exec(f.ctx, `INSERT INTO public.r133_publication VALUES($1,$2,$3,$4,$5)`, f.instance, object, version, len(payloads), root(object, version, payloads))
		if e != nil {
			return e
		}
		input := make([][]any, len(payloads))
		for i, p := range payloads {
			input[i] = []any{f.instance, object, version, i, p}
		}
		if _, e = tx.CopyFrom(f.ctx, pgx.Identifier{"public", "r133_member"}, []string{"instance", "object", "source", "slot", "payload"}, pgx.CopyFromRows(input)); e != nil {
			return e
		}
		_, e = tx.Exec(f.ctx, `INSERT INTO public.r133_head VALUES($1,$2,$3) ON CONFLICT(instance,object) DO UPDATE SET source=excluded.source`, f.instance, object, version)
		if e == nil && mode == "shared" {
			_, e = tx.Exec(f.ctx, `INSERT INTO public.r133_binding VALUES($1,$2,$3) ON CONFLICT(instance,object) DO UPDATE SET source=excluded.source`, f.instance, object, version)
		}
		return e
	}
	read := func(tx pgx.Tx, mode string) (string, error) {
		bound := "h.source"
		join := ""
		if mode == "shared" {
			bound = "b.source"
			join = " LEFT JOIN public.r133_binding b USING(instance,object)"
		}
		// One statement obtains the head, manifest and full membership snapshot.
		q := `SELECT h.source::text,` + bound + `::text,p.members,p.root,m.slot,m.payload FROM public.r133_head h` + join + ` LEFT JOIN public.r133_publication p ON (p.instance,p.object,p.source)=(h.instance,h.object,` + bound + `) LEFT JOIN public.r133_member m ON (m.instance,m.object,m.source)=(p.instance,p.object,p.source) WHERE h.instance=$1 AND h.object=$2 ORDER BY m.slot`
		rs, e := tx.Query(f.ctx, q, f.instance, object)
		if e != nil {
			return "", e
		}
		defer rs.Close()
		var head, boundSource string
		var count int
		var seal []byte
		got := [][]byte{}
		for rs.Next() {
			var slot int
			var p []byte
			if e = rs.Scan(&head, &boundSource, &count, &seal, &slot, &p); e != nil {
				return "", e
			}
			if head != boundSource || slot != len(got) {
				return "", fmt.Errorf("source or membership mismatch")
			}
			got = append(got, p)
		}
		if e = rs.Err(); e != nil {
			return "", e
		}
		if count != len(got) || count == 0 || !bytes.Equal(seal, root(object, head, got)) {
			return "", fmt.Errorf("manifest mismatch")
		}
		return head, nil
	}
	measurements := []map[string]any{}
	durable := []map[string]any{}
	// Alternate order to avoid attributing first-run cache costs to one design.
	// Every sample starts logically empty and rolls back on identical input.
	for round := 0; round < 5; round++ {
		order := []string{"head", "shared"}
		if round%2 == 1 {
			order[0], order[1] = order[1], order[0]
		}
		for _, mode := range order {
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			var walStart string
			must(t, tx.QueryRow(f.ctx, `SELECT pg_current_wal_insert_lsn()::text`).Scan(&walStart))
			start := time.Now()
			must(t, publish(tx, mode, source))
			writeMS := float64(time.Since(start)) / 1e6
			var walBytes int64
			must(t, tx.QueryRow(f.ctx, `SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),$1::pg_lsn)::bigint`, walStart).Scan(&walBytes))
			start = time.Now()
			got, e := read(tx, mode)
			must(t, e)
			if got != source {
				t.Fatal("wrong publication source")
			}
			measurements = append(measurements, map[string]any{"mode": mode, "round": round, "writeMS": writeMS, "readMS": float64(time.Since(start)) / 1e6, "walBytes": walBytes})
			must(t, tx.Rollback(f.ctx))
		}
	}
	// Clear test garbage before comparing live table/index sizes and durability.
	f.exec(`TRUNCATE public.r133_binding,public.r133_head,public.r133_member,public.r133_publication`)
	for _, mode := range []string{"head", "shared"} {
		tx, e := f.admin.Begin(f.ctx)
		must(t, e)
		publishedAt := time.Now()
		must(t, publish(tx, mode, source))
		commitAt := time.Now()
		must(t, tx.Commit(f.ctx))
		commitMS := float64(time.Since(commitAt)) / 1e6
		publishMS := float64(time.Since(publishedAt)) / 1e6
		var relationBytes, bindingBytes int64
		must(t, f.admin.QueryRow(f.ctx, `SELECT pg_total_relation_size('public.r133_member')+pg_total_relation_size('public.r133_publication')+pg_total_relation_size('public.r133_head'),pg_total_relation_size('public.r133_binding')`).Scan(&relationBytes, &bindingBytes))
		if mode == "head" {
			bindingBytes = 0 // The head design does not own the experiment's empty binding table.
		}
		durable = append(durable, map[string]any{"mode": mode, "publishIncludingCommitMS": publishMS, "commitMS": commitMS, "memberManifestHeadBytes": relationBytes, "extraBindingBytes": bindingBytes})
		t.Run(mode+"_immutable_manifest", func(t *testing.T) {
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer tx.Rollback(f.ctx)
			_, e = tx.Exec(f.ctx, "SET LOCAL ROLE chnhcm_app")
			must(t, e)
			if _, e = tx.Exec(f.ctx, `UPDATE public.r133_publication SET root=sha256('wrong'::bytea)`); e == nil {
				t.Fatal("ordinary SQL changed the manifest seal")
			}
		})
		for _, damage := range []struct{ name, sql string }{
			{"missing", `DELETE FROM public.r133_member WHERE slot=5000`},
			{"extra", `INSERT INTO public.r133_member SELECT instance,object,source,10000,payload FROM public.r133_member WHERE slot=0`},
			{"replace", `UPDATE public.r133_member SET payload=(SELECT payload FROM public.r133_member WHERE slot=7) WHERE slot=5000`},
			{"reorder", `UPDATE public.r133_member SET slot=10001 WHERE slot=0`},
			{"source", `UPDATE public.r133_head SET source=gen_random_uuid()`},
			{"owner", `UPDATE public.r133_member SET object=gen_random_uuid() WHERE slot=0`},
		} {
			t.Run(mode+"_"+damage.name, func(t *testing.T) {
				tx, e := f.admin.Begin(f.ctx)
				must(t, e)
				defer tx.Rollback(f.ctx)
				_, e = tx.Exec(f.ctx, "SET LOCAL ROLE chnhcm_app")
				must(t, e)
				_, e = tx.Exec(f.ctx, damage.sql)
				must(t, e)
				if _, e = read(tx, mode); e == nil {
					t.Fatal("ordinary SQL damage escaped validation")
				}
			})
		}
		// While the writer contains a partial publication, another transaction
		// sees the complete old one. Rollback must preserve that publication.
		writer, e := f.admin.Begin(f.ctx)
		must(t, e)
		_, e = writer.Exec(f.ctx, `UPDATE public.r133_head SET source=$1`, foundation.NewID())
		must(t, e)
		reader, e := f.admin.Begin(f.ctx)
		must(t, e)
		got, e := read(reader, mode)
		must(t, e)
		if got != source {
			t.Fatal("uncommitted partial publication visible")
		}
		must(t, reader.Rollback(f.ctx))
		if _, e = read(writer, mode); e == nil {
			t.Fatal("partial publication accepted")
		}
		must(t, writer.Rollback(f.ctx))
		writer, e = f.admin.Begin(f.ctx)
		must(t, e)
		_, e = writer.Exec(f.ctx, `UPDATE public.r133_head SET source=$1`, foundation.NewID())
		must(t, e)
		if e = writer.Commit(f.ctx); e == nil {
			t.Fatal("head without its owned manifest committed")
		}
		// A second complete logical version is only a protocol simulation: the
		// actual codec still embeds source, so this is not a business correction.
		version := foundation.NewID()
		writer, e = f.admin.Begin(f.ctx)
		must(t, e)
		must(t, publish(writer, mode, version))
		must(t, writer.Commit(f.ctx))
		reader, e = f.admin.Begin(f.ctx)
		must(t, e)
		got, e = read(reader, mode)
		must(t, e)
		if got != version {
			t.Fatal("atomic publication missing")
		}
		must(t, reader.Rollback(f.ctx))
		f.exec(`TRUNCATE public.r133_binding,public.r133_head,public.r133_member,public.r133_publication`)
	}
	raw, e := json.Marshal(map[string]any{"members": len(payloads), "samples": measurements, "durableSamples": durable, "proof": "publication protocol only; actual source-bearing payload codec retained", "productionAdopted": false, "capacityAcceptance": false})
	must(t, e)
	t.Logf("R133_BINDING_DESIGN %s", raw)
}
