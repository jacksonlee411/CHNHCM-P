package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (r *runner) seed() (any, error) {
	var completed atomic.Int64
	err := parallel(2000, r.clients, func(i int, c *client) error {
		code := fmt.Sprintf("T07_CAP_%04d", i)
		var id string
		var n int
		e := r.db.QueryRow(context.Background(), `SELECT b.bu_id,s.source_no FROM organization.business_unit b JOIN organization.source_version s ON s.instance_id=b.instance_id AND s.bu_id=b.bu_id AND s.source_id=b.head_source_id WHERE b.instance_id=$1 AND b.code=$2`, r.instance, code).Scan(&id, &n)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if errors.Is(e, pgx.ErrNoRows) {
			// Resume only exact completed API sources. A transport ambiguity is never
			// replaced with a new identity in this process: it stops the run.
			start := fmt.Sprintf("%d-01-01", 2024+2*(i%3))
			v, e := c.call("POST", "/api/v1/business-units", object{"code": code, "name": fmt.Sprintf("容量资料%04d", (i*997)%2000), "nameEn": "Initial independent name", "effectiveDate": start, "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, foundation.NewID(), 201)
			if e != nil {
				return e
			}
			id = v["buId"].(string)
			n = 1
		}
		for n < 50 {
			d, e := c.detail(id)
			if e != nil {
				return e
			}
			rs := rows(d)
			if len(rs) < 20 {
				j := len(rs)
				base, err := time.Parse("2006-01-02", rs[0]["effectiveDate"].(string))
				if err != nil {
					return err
				}
				_, e = c.call("POST", "/api/v1/business-units/"+id+"/records", object{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": rs[len(rs)-1]["recordId"], "effectiveDate": base.AddDate(0, 0, j*10).Format("2006-01-02"), "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": object{"name": fmt.Sprintf("独立资料%04d段%02d", (i*313)%2000, j)}}, foundation.NewID(), 201)
			} else {
				_, e = c.correct(id, d, rs[n%len(rs)], object{"name": fmt.Sprintf("更正资料%04d轮%02d", i, n)})
			}
			if e != nil {
				return e
			}
			n++
		}
		if done := completed.Add(1); done%20 == 0 {
			fmt.Printf("seed completed %d/2000 BUs\n", done)
		}
		return nil
	})
	result := object{"completedBU": completed.Load(), "requiredBU": 2000, "sourcesPerBU": 50, "membersAfterSource20": 20, "expectedSourceMembers": 1620000, "businessWritePath": "HTTP current API only"}
	if err != nil {
		return result, err
	}
	var bu, sources, members int
	err = r.db.QueryRow(context.Background(), `SELECT count(DISTINCT b.bu_id),count(DISTINCT s.source_id),count(m.record_id) FROM organization.business_unit b JOIN organization.source_version s USING(instance_id,bu_id) JOIN organization.source_member m USING(instance_id,bu_id,source_id) WHERE b.code LIKE 'T07_CAP_%'`).Scan(&bu, &sources, &members)
	result["observedBU"], result["observedSources"], result["observedMembers"] = bu, sources, members
	if err == nil && (bu != 2000 || sources != 100000 || members != 1620000) {
		err = fmt.Errorf("full capacity fixture count differs")
	}
	return result, err
}
func (r *runner) history() (any, error) {
	c := r.clients[0]
	const code = "T07_LONG_HISTORY"
	var id string
	var n int
	err := r.db.QueryRow(context.Background(), `SELECT b.bu_id,s.source_no FROM organization.business_unit b JOIN organization.source_version s ON s.instance_id=b.instance_id AND s.bu_id=b.bu_id AND s.source_id=b.head_source_id WHERE b.instance_id=$1 AND b.code=$2`, r.instance, code).Scan(&id, &n)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		v, e := c.call("POST", "/api/v1/business-units", object{"code": code, "name": "长历史初始", "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, foundation.NewID(), 201)
		if e != nil {
			return nil, e
		}
		id = v["buId"].(string)
		n = 1
	}
	first, err := c.detail(id)
	if err != nil {
		return nil, err
	}
	original := first["headSourceVersion"].(string)
	for n < 1000 {
		d, e := c.detail(id)
		if e != nil {
			return nil, e
		}
		if _, e = c.correct(id, d, rows(d)[0], object{"name": fmt.Sprintf("长历史修订%04d", n)}); e != nil {
			return nil, e
		}
		n++
		if n%100 == 0 {
			fmt.Printf("history %d/1000\n", n)
		}
	}
	old, err := c.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+original, nil, "", 200)
	if err == nil && rows(old)[0]["name"] != rows(first)[0]["name"] {
		err = fmt.Errorf("original history changed")
	}
	return object{"buId": id, "sources": n, "protectedSource": original, "originalName": rows(first)[0]["name"]}, err
}
