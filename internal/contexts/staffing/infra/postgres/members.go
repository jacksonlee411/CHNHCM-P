package postgres

import (
	"bytes"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"slices"
	"strings"
	"time"
)

func memberDigest(m d.GradeMembers) []byte {
	if m == nil {
		m = d.GradeMembers{}
	}
	b, _ := json.Marshal(m)
	sum := sha256.Sum256(b)
	return sum[:]
}

// hydrateMembers has one authoritative format for pre-G2 and new sources.
// Missing seals, counts, order or evidence fail closed, including empty Jobs.
func (u *unit) hydrateMembers(ctx context.Context, t *d.Timeline) error {
	if !d.HasGrades(t.Kind) || t.ID == "" {
		return nil
	}
	return u.hydrateTimelines(ctx, t.Kind, []*d.Timeline{t})
}

// Validate the complete immutable inventory before using a reverse index. Batch
// transport keeps this check bounded by rows, rather than one query per object.
func (u *unit) hydrateTimelines(ctx context.Context, kind d.Kind, timelines []*d.Timeline) error {
	if len(timelines) == 0 {
		return nil
	}
	if kind == d.Job {
		if err := u.hydrateFunctions(ctx, timelines); err != nil {
			return err
		}
	}
	type sourceKey struct{ object, source string }
	type recordKey struct {
		sourceKey
		record string
	}
	type seal struct {
		count  int
		digest []byte
	}
	objects, sources := make([]string, len(timelines)), make([]string, len(timelines))
	bySource := make(map[sourceKey]*d.Timeline, len(timelines))
	seals := map[recordKey]seal{}
	for i, t := range timelines {
		objects[i], sources[i] = t.ID, t.Source
		bySource[sourceKey{t.ID, t.Source}] = t
		for j := range t.Records {
			t.Records[j].Grades = d.GradeMembers{}
		}
	}
	// Typed arrays are query transport only: the authoritative rows and seals
	// stay unchanged. One row per manifest avoids retransmitting its keys and
	// digest for every member, including in the complete reverse inventory.
	rows, e := u.tx.Query(ctx, `SELECT s.object_id::text,s.source_id::text,s.record_id::text,s.member_count,s.digest,
 COALESCE(array_agg(m.grade_id::text ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::text[]),
 COALESCE(array_agg(m.position ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::integer[]),
 COALESCE(array_agg(m.business_at::text ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::text[]),
 COALESCE(array_agg(m.adopted_source::text ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::text[]),
 COALESCE(array_agg(m.adopted_record::text ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::text[]),
 COALESCE(array_agg(m.adopted_revision::text ORDER BY m.position) FILTER(WHERE m.grade_id IS NOT NULL),ARRAY[]::text[])
 FROM `+table(kind)+`_grade_manifest s JOIN unnest($2::uuid[],$3::uuid[]) AS requested(object_id,source_id) USING(object_id,source_id)
 LEFT JOIN `+table(kind)+`_grade_member m USING(instance_id,object_id,source_id,record_id)
 WHERE s.instance_id=$1 GROUP BY s.instance_id,s.object_id,s.source_id,s.record_id ORDER BY s.object_id,s.source_id,s.record_id`, u.store.Instance, objects, sources)
	if e != nil {
		return e
	}
	defer rows.Close()
	// Repeated immutable identifiers/dates are still validated, once per
	// request inventory; this cache cannot outlive the transaction read.
	checkedIDs, checkedDates := map[string]bool{}, map[string]bool{}
	validID := func(v string) bool {
		if checkedIDs[v] {
			return true
		}
		ok := foundation.ValidID(v)
		if ok {
			checkedIDs[v] = true
		}
		return ok
	}
	validDate := func(v string) bool {
		if checkedDates[v] {
			return true
		}
		ok := foundation.ValidDate(v)
		if ok {
			checkedDates[v] = true
		}
		return ok
	}
	for rows.Next() {
		var object, sourceID, id string
		var count int
		var digest []byte
		var grades, dates, adoptedSources, adoptedRecords, revisions []string
		var positions []int32
		if e = rows.Scan(&object, &sourceID, &id, &count, &digest, &grades, &positions, &dates, &adoptedSources, &adoptedRecords, &revisions); e != nil {
			return e
		}
		key := sourceKey{object, sourceID}
		t := bySource[key]
		if t == nil {
			return errors.New("unexpected membership source")
		}
		seals[recordKey{key, id}] = seal{count, digest}
		i := d.Index(*t, id)
		if i < 0 {
			return errors.New("unexpected membership record")
		}
		if len(grades) != count || len(positions) != count || len(dates) != count || len(adoptedSources) != count || len(adoptedRecords) != count || len(revisions) != count {
			return errors.New("membership manifest missing or corrupt")
		}
		t.Records[i].Grades = make(d.GradeMembers, count)
		for j, grade := range grades {
			if positions[j] != int32(j+1) || !validDate(dates[j]) || !validID(grade) || !validID(adoptedSources[j]) || !validID(adoptedRecords[j]) || !validID(revisions[j]) {
				return errors.New("invalid membership evidence")
			}
			t.Records[i].Grades[j] = d.GradeMember{GradeID: grade, Adopted: d.ReferenceAdoption{BusinessAt: dates[j], SourceVersion: adoptedSources[j], RecordID: adoptedRecords[j], Revision: revisions[j]}}
		}
	}
	if e = rows.Err(); e != nil {
		return e
	}
	verifiedCollections := map[string]d.GradeMembers{}
	for _, t := range timelines {
		for _, r := range t.Records {
			seal, ok := seals[recordKey{sourceKey{t.ID, t.Source}, r.ID}]
			if !ok || seal.count != len(r.Grades) {
				return errors.New("membership manifest missing or corrupt")
			}
			// A digest alone is never a cache hit: compare every actual typed
			// member/evidence field before reusing a previously verified seal.
			key := string(seal.digest)
			if prior, ok := verifiedCollections[key]; ok && prior.EqualRevision(r.Grades) {
				continue
			}
			if !bytes.Equal(seal.digest, memberDigest(r.Grades)) {
				return errors.New("membership manifest missing or corrupt")
			}
			normalized, e := (d.MembershipPatch{Present: true, IDs: r.Grades.IDs()}).Apply(t.Kind, r.Grades)
			if e != nil || !normalized.EqualRevision(r.Grades) {
				return errors.New("invalid membership ordering")
			}
			verifiedCollections[key] = r.Grades
		}
	}
	return nil
}
func (u *unit) queueMemberCopies(batch *pgx.Batch, t, before d.Timeline) []string {
	if !d.HasGrades(t.Kind) {
		return nil
	}
	// The locked, fully validated predecessor is immutable. Copy unchanged
	// typed facts inside this transaction, retaining every original source and
	// adoption, rather than round-tripping the same evidence through Go/COPY.
	unchanged := []string{}
	for _, r := range t.Records {
		if i := d.Index(before, r.ID); i >= 0 && r.Grades.EqualRevision(before.Records[i].Grades) {
			unchanged = append(unchanged, r.ID)
		}
	}
	if len(unchanged) > 0 {
		name := table(t.Kind)
		batch.Queue(`INSERT INTO `+name+`_grade_manifest
 (instance_id,object_id,source_id,record_id,member_count,digest)
 SELECT instance_id,object_id,$4,record_id,member_count,digest FROM `+name+`_grade_manifest
 WHERE instance_id=$1 AND object_id=$2 AND source_id=$3 AND record_id=ANY($5::uuid[])`, u.store.Instance, t.ID, before.Source, t.Source, unchanged)
		batch.Queue(`INSERT INTO `+name+`_grade_member
 (instance_id,object_id,source_id,record_id,grade_id,position,business_at,adopted_source,adopted_record,adopted_revision)
 SELECT instance_id,object_id,$4,record_id,grade_id,position,business_at,adopted_source,adopted_record,adopted_revision FROM `+name+`_grade_member
 WHERE instance_id=$1 AND object_id=$2 AND source_id=$3 AND record_id=ANY($5::uuid[])`, u.store.Instance, t.ID, before.Source, t.Source, unchanged)
	}
	return unchanged
}

func (u *unit) saveMembers(ctx context.Context, t d.Timeline, unchanged []string) error {
	if !d.HasGrades(t.Kind) {
		return nil
	}
	// pgx COPY needs binary UUID values. Passing strings invokes its failed
	// binary-encode / text-scan fallback for every cell of every member.
	uuids := map[string]pgtype.UUID{}
	var uuidError error
	uuid := func(id string) pgtype.UUID {
		if v, ok := uuids[id]; ok {
			return v
		}
		var v pgtype.UUID
		if e := v.Scan(id); e != nil {
			uuidError = e
		}
		uuids[id] = v
		return v
	}
	manifests, members := [][]any{}, [][]any{}
	for _, r := range t.Records {
		if slices.Contains(unchanged, r.ID) {
			continue
		}
		manifests = append(manifests, []any{uuid(u.store.Instance), uuid(t.ID), uuid(t.Source), uuid(r.ID), len(r.Grades), memberDigest(r.Grades)})
		for i, m := range r.Grades {
			a := m.Adopted
			date, e := time.Parse("2006-01-02", a.BusinessAt)
			if e != nil {
				return e
			}
			members = append(members, []any{uuid(u.store.Instance), uuid(t.ID), uuid(t.Source), uuid(r.ID), uuid(m.GradeID), i + 1, pgtype.Date{Time: date, Valid: true}, uuid(a.SourceVersion), uuid(a.RecordID), uuid(a.Revision)})
		}
	}
	if uuidError != nil {
		return uuidError
	}
	// COPY changes only transport. All typed rows, immutable seals and structural
	// reference checks are committed in the same business transaction as sources and receipts.
	if len(manifests) == 0 {
		return nil
	}
	name := strings.TrimPrefix(table(t.Kind), "staffing.")
	if _, e := u.tx.CopyFrom(ctx, pgx.Identifier{"staffing", name + "_grade_manifest"}, []string{"instance_id", "object_id", "source_id", "record_id", "member_count", "digest"}, pgx.CopyFromRows(manifests)); e != nil {
		return e
	}
	if len(members) == 0 {
		return nil
	}
	_, e := u.tx.CopyFrom(ctx, pgx.Identifier{"staffing", name + "_grade_member"}, []string{"instance_id", "object_id", "source_id", "record_id", "grade_id", "position", "business_at", "adopted_source", "adopted_record", "adopted_revision"}, pgx.CopyFromRows(members))
	return e
}
func (u *unit) GradeConsumerHeads(ctx context.Context) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	out := []d.Timeline{}
	for _, k := range []d.Kind{d.Job, d.Ladder} {
		// Derive reverse obligations from complete, sealed current heads.
		rows, e := u.tx.Query(ctx, "SELECT head FROM "+table(k)+" WHERE instance_id=$1 AND enterprise_id=$2 ORDER BY id", u.store.Instance, ent.ID)
		if e != nil {
			return nil, e
		}
		current := []*d.Timeline{}
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e != nil {
				rows.Close()
				return nil, e
			}
			t, e := decode(b, k)
			if e != nil {
				rows.Close()
				return nil, e
			}
			current = append(current, &t)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if e = u.hydrateTimelines(ctx, k, current); e != nil {
			return nil, e
		}
		for _, t := range current {
			out = append(out, *t)
		}
	}
	return out, nil
}
