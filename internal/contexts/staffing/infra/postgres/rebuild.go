package postgres

import (
	"bytes"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

// RebuildReads reconstructs derived indexes from immutable sources. It never
// changes an identity, revision, receipt, source or event. Each current head is
// reread under its ordinary object guard; commits do not expose an empty index.
func (s *Store) RebuildReads(ctx context.Context) error {
	for _, k := range []d.Kind{d.Family, d.JobFunction, d.Job, d.Grade, d.Ladder, d.Position} {
		name := table(k)
		rows, e := s.Pool.Query(ctx, "SELECT id::text FROM "+name+" WHERE instance_id=$1 ORDER BY id", s.Instance)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			if e = s.rebuildOne(ctx, k, id); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Store) rebuildOne(ctx context.Context, k d.Kind, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if e != nil {
		return e
	}
	defer rollback(tx)
	if e = lock(ctx, tx, false, objectGuard(k), s.Instance, id); e != nil {
		return e
	}
	name := table(k)
	var head, source []byte
	e = tx.QueryRow(ctx, "SELECT h.head,s.snapshot FROM "+name+" h JOIN "+name+"_source s ON (s.instance_id,s.object_id,s.source_id)=(h.instance_id,h.id,h.source_id) WHERE h.instance_id=$1 AND h.id=$2", s.Instance, id).Scan(&head, &source)
	if e != nil {
		return e
	}
	if !bytes.Equal(head, source) {
		return errors.New("current staffing head differs from immutable source")
	}
	t, e := decode(source, k)
	if e != nil {
		return e
	}
	u := &unit{store: s, tx: tx}
	if e = u.hydrateMembers(ctx, &t); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM "+name+"_read WHERE instance_id=$1 AND object_id=$2", s.Instance, id); e != nil {
		return e
	}
	batch := &pgx.Batch{}
	for _, r := range t.Records {
		var end, family any
		if r.End != "" {
			end = r.End
		}
		if f := r.Data["jobFamilyId"]; f != "" {
			family = f
		}
		rb, e := json.Marshal(r)
		if e != nil {
			return e
		}
		queueRead(batch, k, s.Instance, t, r, end, family, rb)
	}
	if e = tx.SendBatch(ctx, batch).Close(); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
