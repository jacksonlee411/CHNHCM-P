package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestQuotaIndependentDatesAndOriginal(t *testing.T) {
	pos := QuotaPeriods{ID: foundation.NewID(), Records: []QuotaPeriod{{Date: "2026-01-01", Active: true}}}
	var q StaffingQuota
	apply := func(action, day string, count int) QuotaRecord {
		t.Helper()
		p, e := ChangeQuota(q, QuotaChange{Action: action, Source: q.Source, Date: day, Amount: &count, Reason: "年度核定"}, pos)
		if e != nil {
			t.Fatal(e)
		}
		q = p.After
		return q.Records[q.At(day)]
	}
	jan := apply("create", "2027-01-01", 2)
	june := apply("adjust", "2027-06-01", 4)
	april := apply("adjust", "2027-04-01", 1)
	before := q
	three := 3
	c := QuotaChange{Action: "correct", Source: q.Source, RecordID: april.ID, Revision: april.Revision, Amount: &three}
	correct, e := ChangeQuota(q, c, pos)
	if e != nil {
		t.Fatal(e)
	}
	q = correct.After
	if *before.Records[1].Amount != 1 || *q.Records[q.At("2027-03-31")].Amount != 2 || *q.Records[q.At("2027-04-01")].Amount != 3 || q.Records[2].Revision != june.Revision || q.Records[0].Revision != jan.Revision || q.Records[1].ID != april.ID || q.Records[1].Revision == april.Revision {
		t.Fatal("更正改变了原件、身份或独立后继")
	}
	if _, e = ChangeQuota(q, c, pos); e == nil {
		t.Fatal("接受了过期来源")
	}
	c.Source = q.Source
	c.Revision = q.Records[1].Revision
	if p, e := ChangeQuota(q, c, pos); e != nil || p.Changed || p.After.Source != q.Source {
		t.Fatal("无变化不应产生来源", e)
	}
	c.Action = "redate"
	c.Date = "2027-04-20"
	c.Amount = nil
	p, e := ChangeQuota(q, c, pos)
	if e != nil || *p.After.Records[p.After.At("2027-04-19")].Amount != 2 || p.After.Records[2].ID != june.ID {
		t.Fatal("改日不应移动后继", e)
	}
	c.Action = "delete"
	p, e = ChangeQuota(q, c, pos)
	if e != nil || len(p.After.Records) != 2 || q.Records[1].ID != april.ID {
		t.Fatal("删除应保留原件", e)
	}
	c.RecordID = jan.ID
	c.Revision = jan.Revision
	if _, e = ChangeQuota(q, c, pos); e == nil {
		t.Fatal("接受删除首条")
	}
}
