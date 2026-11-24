package history

import (
	"strconv"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5/pgtype"
)

// These are Go-derived facts for one lookup span, never permissions or a
// second timeline. A comparable value makes every query fact part of merging.
type departmentQueryFacts struct {
	codeKey, name, nameKey, shortName, shortKey, bu, parent           string
	record, revision, date, end, firstDate, state, relation, pathDate string
	hasShortName                                                      bool
}

func departmentQuery(t d.Timeline, start string) (departmentQueryFacts, error) {
	if len(t.Records) == 0 || t.First != t.Records[0].ID {
		return departmentQueryFacts{}, compact.ErrCorrupt
	}
	day := start
	if day == "" {
		day = t.Records[0].Date
	}
	r := d.ScopeRecord(t, day)
	if r == nil {
		return departmentQueryFacts{}, compact.ErrCorrupt
	}
	q := departmentQueryFacts{
		codeKey: strings.ToLower(t.Code), name: r.Data.Name, nameKey: strings.ToLower(r.Data.Name),
		bu: r.Data.BU, parent: r.Data.Parent, record: r.ID, revision: r.Revision,
		date: r.Date, end: r.End, firstDate: t.Records[0].Date, relation: t.RelationSource,
		state: d.Status(t, day),
	}
	if r.Data.ShortName != nil {
		q.hasShortName, q.shortName, q.shortKey = true, *r.Data.ShortName, strings.ToLower(*r.Data.ShortName)
	}
	if start == "" {
		q.state = "not_effective"
	}
	switch q.state {
	case "inactive":
		q.pathDate = dated.NextDate(r.Date, -1)
	case "not_effective":
		q.pathDate = q.firstDate
	}
	return q, nil
}

type departmentQuerySeals struct{ owner, dictionary, shape, content []byte }

func departmentQueryDigest(c Chunk, source, code, first string, slot, shape int32, s encodedReadSegment, envelope, dictionaryHash, shapeHash []byte) departmentQuerySeals {
	q := s.query
	owner := hashParts([][]byte{[]byte("department-query-owner/1"), []byte(c.Enterprise), []byte(code), []byte(first), []byte(q.relation)})
	parts := [][]byte{}
	for _, v := range []string{"department-query-131/3", c.Instance, c.Object, source, strconv.Itoa(int(slot)), strconv.Itoa(ReadRule), s.start, s.end, strconv.Itoa(int(c.Dictionary)), strconv.Itoa(int(shape))} {
		parts = append(parts, []byte(v))
	}
	parts = append(parts, envelope, owner, dictionaryHash, shapeHash, []byte(q.name), []byte(q.nameKey))
	short := []byte{0}
	if q.hasShortName {
		short = append([]byte{1}, []byte(q.shortName)...)
	}
	parts = append(parts, short)
	for _, v := range []string{q.shortKey, q.bu, q.parent, q.record, q.revision, q.date, q.end, q.firstDate, q.state, q.relation, q.codeKey, q.pathDate} {
		parts = append(parts, []byte(v))
	}
	for _, id := range s.ids {
		parts = append(parts, []byte(id))
	}
	return departmentQuerySeals{owner: owner, dictionary: dictionaryHash, shape: shapeHash, content: hashParts(parts)}
}

func (q departmentQueryFacts) values(seal departmentQuerySeals) ([]any, error) {
	uuid := func(s string) (pgtype.UUID, error) {
		v := pgtype.UUID{}
		if s == "" {
			return v, nil
		}
		err := v.Scan(s)
		return v, err
	}
	date := func(s string) (pgtype.Date, error) {
		if s == "" {
			return pgtype.Date{}, nil
		}
		t, err := time.Parse("2006-01-02", s)
		return pgtype.Date{Time: t, Valid: err == nil}, err
	}
	ids := []pgtype.UUID{}
	for _, s := range []string{q.bu, q.parent, q.record, q.revision, q.relation} {
		v, err := uuid(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, v)
	}
	dates := []pgtype.Date{}
	for _, s := range []string{q.date, q.end, q.firstDate, q.pathDate} {
		v, err := date(s)
		if err != nil {
			return nil, err
		}
		dates = append(dates, v)
	}
	var short any
	if q.hasShortName {
		short = q.shortName
	}
	return []any{q.codeKey, dates[3], q.name, q.nameKey, short, q.shortKey,
		ids[0], ids[1], ids[2], ids[3], dates[0], dates[1], dates[2], q.state, ids[4], seal.owner, seal.dictionary, seal.shape, seal.content}, nil
}

var departmentQueryColumns = []string{"q_code_key", "q_path_date", "q_name", "q_name_key", "q_short_name", "q_short_key", "q_bu", "q_parent", "q_record", "q_revision", "q_date", "q_end", "q_first_date", "q_state", "q_relation", "q_owner_seal", "q_dictionary_seal", "q_shape_seal", "q_seal"}
