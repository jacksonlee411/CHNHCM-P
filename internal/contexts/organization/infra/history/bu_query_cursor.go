package history

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
)

// BU has no current name ordering index. Its neighbor read streams the current
// segment of each authorized identity and retains only the facts needed to compare.
// All material comes from one cursor snapshot, without the BU selection digest.
func CaptureBUNavigation(ctx context.Context, tx pgx.Tx, instance, enterprise, day string, ids []string) (out map[string]d.BusinessUnit, err error) {
	const sql = `SELECT jsonb_build_object('id',b.bu_id,'enterprise',b.enterprise_id,'code',b.code,'source',b.head_source_id,'segmentSource',s.source_id,'day',$3::text,'valid',s.source_id=b.head_source_id,'ids',s.ids,
 '_material',jsonb_build_object('words',d.words,'binary',ARRAY(SELECT encode(v,'hex') FROM unnest(d.binaries)v),
 'starts',ARRAY(SELECT coalesce(lower(v)::text,'') FROM unnest(d.ranges)v),'ends',ARRAY(SELECT coalesce(upper(v)::text,'') FROM unnest(d.ranges)v),'parts',ARRAY(SELECT encode(v,'hex') FROM unnest(h.parts)v)))
 FROM organization.business_unit b
 LEFT JOIN organization.bu_read_segment s ON s.instance_id=b.instance_id AND s.object_id=b.bu_id AND s.span @> $3::date
 LEFT JOIN organization.bu_dictionary d ON d.id=s.dictionary_id AND d.instance_id=b.instance_id AND d.enterprise_id=b.enterprise_id
 LEFT JOIN organization.bu_shape h ON h.id=s.shape_id AND h.instance_id=b.instance_id AND h.enterprise_id=b.enterprise_id
 WHERE b.instance_id=$1 AND b.enterprise_id=$2 AND ($4::uuid[] IS NULL OR b.bu_id=ANY($4::uuid[]))`
	if _, err = tx.Exec(ctx, "DECLARE bu_neighbors133 NO SCROLL CURSOR FOR "+sql, instance, enterprise, day, ids); err != nil {
		return
	}
	defer func() {
		_, e := tx.Exec(ctx, "CLOSE bu_neighbors133")
		if err == nil {
			err = e
		}
	}()
	out = map[string]d.BusinessUnit{}
	for {
		rows, e := tx.Query(ctx, "FETCH FORWARD 64 FROM bu_neighbors133")
		if e != nil {
			return nil, e
		}
		n := 0
		for rows.Next() {
			n++
			var raw []byte
			if e = rows.Scan(&raw); e != nil {
				rows.Close()
				return nil, e
			}
			var v capturedBU
			if json.Unmarshal(raw, &v) != nil || !v.Valid || v.Enterprise != enterprise || v.Source != v.SegmentSource {
				rows.Close()
				return nil, queryUnavailable()
			}
			values, data, e := queryValues(v.IDs, v.Material)
			if e != nil {
				rows.Close()
				return nil, errors.Join(queryUnavailable(), e)
			}
			decoded, e := DecodeBURead(ctx, values, data)
			if e != nil {
				rows.Close()
				return nil, errors.Join(queryUnavailable(), e)
			}
			b := decoded.Timeline
			if b.ID != v.ID || b.Source != v.Source || len(b.Records) == 0 {
				rows.Close()
				return nil, queryUnavailable()
			}
			b.Code = v.Code
			out[b.ID] = b
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if n < 64 {
			return out, nil
		}
	}
}
