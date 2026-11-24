package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"github.com/jackc/pgx/v5"
)

// Grade has no family relationship column. Both save and rebuild write the same
// projection; PostgreSQL derives the half-open range from Go's decided bounds.
func queueRead(batch *pgx.Batch, k d.Kind, instance string, t d.Timeline, r d.Record, end, family any, fact []byte) {
	if k == d.JobFunction || k == d.Grade || k == d.Ladder || k == d.Position {
		batch.Queue("INSERT INTO "+table(k)+"_read(instance_id,enterprise_id,object_id,record_id,source_id,start_date,end_date,active,fact) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)", instance, t.Enterprise, t.ID, r.ID, t.Source, r.Date, end, r.Active(), fact)
		return
	}
	batch.Queue("INSERT INTO "+table(k)+"_read VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)", instance, t.Enterprise, t.ID, r.ID, t.Source, r.Date, end, r.Active(), family, fact)
}
