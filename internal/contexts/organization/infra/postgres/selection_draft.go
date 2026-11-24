package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"slices"
	"strings"

	app "chnhcm.local/chnhcm/internal/contexts/organization/application/selection"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// Serialize edits/replacement/cancel for this authenticated session. The
// slot locks reserve independent admission positions; no instance-wide lock
// serializes identity-block insertion across twenty sessions.
func (u *selectionDraftUnit) lockDraftSession(ctx context.Context) error {
	_, err := u.exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,13302))`, u.kind+":"+u.instance+":"+u.guard.Principal().SessionID)
	return err
}

func (u *selectionDraftUnit) CreateDraft(ctx context.Context, draft app.Draft, ids []string) (app.Draft, error) {
	if len(ids) > app.DraftMaxMembers {
		return draft, app.DraftMemberLimit()
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	for i, id := range ids {
		if !foundation.ValidID(id) || i > 0 && id == ids[i-1] {
			return draft, app.DraftUnavailable()
		}
	}
	if err := u.lockDraftSession(ctx); err != nil {
		return draft, err
	}
	if _, err := u.exec(ctx, `DELETE FROM organization.department_selection_draft WHERE (instance_id,draft_id) IN
 (SELECT instance_id,draft_id FROM organization.department_selection_draft WHERE instance_id=$1 AND expires_at<=clock_timestamp() ORDER BY expires_at LIMIT 128 FOR UPDATE SKIP LOCKED)`, u.instance); err != nil {
		return draft, err
	}
	principal := u.guard.Principal()
	slot, err := u.admitDraftSlot(ctx, principal.SessionID)
	if err != nil {
		return draft, err
	}
	// Capture/authorization must finish before this method. A failed replacement
	// rolls back the transaction, including deletion of the preceding draft.
	if _, err := u.exec(ctx, `DELETE FROM organization.department_selection_draft WHERE instance_id=$1 AND session_id=$2`, u.instance, principal.SessionID); err != nil {
		return draft, err
	}
	draft.ID = foundation.NewID()
	draft.Revision = 1
	draft.Members = len(ids)
	draft.Selected = len(ids)
	if _, err := u.exec(ctx, `INSERT INTO organization.department_selection_draft
 (instance_id,enterprise_id,subject_id,session_id,draft_id,business_day,query_key,mode,revision,member_count,selected_count,created_at,expires_at,admission_slot)
 VALUES($1,$2,$3,$4,$5,$6::text::date,$7,$8,1,$9,$9,statement_timestamp(),statement_timestamp()+interval '30 minutes',$10)`, u.instance, draft.Enterprise, principal.ID, principal.SessionID, draft.ID, draft.Day, draft.QueryKey, draft.Mode, len(ids), slot); err != nil {
		return draft, err
	}
	if len(ids) > 0 {
		_, err := u.exec(ctx, `INSERT INTO organization.department_selection_block(instance_id,draft_id,first_id,members,selected)
 SELECT $1,$2,(array_agg(id ORDER BY n))[1],array_agg(id ORDER BY n),repeat('1',count(*)::int)::bit varying
 FROM unnest($3::uuid[]) WITH ORDINALITY members(id,n) GROUP BY (n-1)/1024`, u.instance, draft.ID, ids)
		if err != nil {
			return draft, err
		}
	}
	return draft, nil
}

// A persisted unique slot is the hard 128-draft cap. Each allocation uses a
// nonblocking transaction lock and checks occupancy after locking, so concurrent
// captures can insert their blocks independently without over-admission.
func (u *selectionDraftUnit) admitDraftSlot(ctx context.Context, session string) (int, error) {
	var slot int
	err := u.queryRow(ctx, `SELECT admission_slot FROM organization.department_selection_draft WHERE instance_id=$1 AND session_id=$2 FOR UPDATE`, u.instance, session).Scan(&slot)
	if err == nil {
		return slot, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	rows, err := u.query(ctx, `SELECT n FROM generate_series(1,128) n WHERE NOT EXISTS(SELECT 1 FROM organization.department_selection_draft d WHERE d.instance_id=$1 AND d.admission_slot=n) ORDER BY n`, u.instance)
	if err != nil {
		return 0, err
	}
	candidates := []int{}
	for rows.Next() {
		var n int
		if err = rows.Scan(&n); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, n)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, n := range candidates {
		var locked bool
		if err = u.queryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1||':'||$2::int::text,13303))`, u.instance, n).Scan(&locked); err != nil {
			return 0, err
		}
		if !locked {
			continue
		}
		var occupied bool
		if err = u.queryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization.department_selection_draft WHERE instance_id=$1 AND admission_slot=$2)`, u.instance, n).Scan(&occupied); err != nil {
			return 0, err
		}
		if !occupied {
			return n, nil
		}
	}
	return 0, app.DraftLimit()
}

func (u *selectionDraftUnit) ReadDraft(ctx context.Context, id string, revision int64, day, key string) (draft app.Draft, err error) {
	if !foundation.ValidID(id) || revision < 1 {
		return draft, app.DraftUnavailable()
	}
	if err = u.lockDraftSession(ctx); err != nil {
		return
	}
	p := u.guard.Principal()
	err = u.queryRow(ctx, `SELECT draft_id::text,enterprise_id::text,business_day::text,query_key,mode,revision,member_count,selected_count
 FROM organization.department_selection_draft WHERE instance_id=$1 AND draft_id=$2 AND subject_id=$3 AND session_id=$4 AND expires_at>clock_timestamp() FOR UPDATE`, u.instance, id, p.ID, p.SessionID).
		Scan(&draft.ID, &draft.Enterprise, &draft.Day, &draft.QueryKey, &draft.Mode, &draft.Revision, &draft.Members, &draft.Selected)
	if errors.Is(err, pgx.ErrNoRows) {
		err = app.DraftUnavailable()
		return
	}
	if err != nil {
		return
	}
	if draft.Day != day || draft.QueryKey != key {
		return draft, app.DraftUnavailable()
	}
	if draft.Revision != revision {
		return draft, app.DraftConflict()
	}
	return
}

func (u *selectionDraftUnit) writeDraftBlock(ctx context.Context, id string, members []string, bits string) error {
	_, err := u.exec(ctx, `INSERT INTO organization.department_selection_block(instance_id,draft_id,first_id,members,selected)
 VALUES($1,$2,$3,$4,$5::text::bit varying) ON CONFLICT(instance_id,draft_id,first_id) DO UPDATE SET members=excluded.members,selected=excluded.selected`, u.instance, id, members[0], members, bits)
	return err
}

type draftBlock struct {
	first string
	ids   []string
	bits  string
}

// Seek only blocks hit by requested identities. A new identity before the first
// block uses that block; an empty explicit draft has no block yet.
func (u *selectionDraftUnit) draftBlocks(ctx context.Context, id string, ids []string) ([]draftBlock, error) {
	rows, err := u.query(ctx, `SELECT DISTINCT b.first_id::text,b.members,b.selected::text FROM unnest($3::uuid[]) wanted(id)
 CROSS JOIN LATERAL (SELECT coalesce(
 (SELECT first_id FROM organization.department_selection_block WHERE instance_id=$1 AND draft_id=$2 AND first_id<=wanted.id ORDER BY first_id DESC LIMIT 1),
 (SELECT first_id FROM organization.department_selection_block WHERE instance_id=$1 AND draft_id=$2 ORDER BY first_id LIMIT 1)) AS first_id) hit
 JOIN organization.department_selection_block b ON b.instance_id=$1 AND b.draft_id=$2 AND b.first_id=hit.first_id`, u.instance, id, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blocks []draftBlock
	for rows.Next() {
		var b draftBlock
		if err = rows.Scan(&b.first, &b.ids, &b.bits); err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

func (u *selectionDraftUnit) DraftPage(ctx context.Context, draft app.Draft, ids []string) (app.DraftPageSelection, error) {
	blocks, err := u.draftBlocks(ctx, draft.ID, ids)
	if err != nil {
		return app.DraftPageSelection{}, err
	}
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	out := app.DraftPageSelection{Members: []string{}, Selected: []string{}}
	for _, b := range blocks {
		for i, id := range b.ids {
			if wanted[id] {
				out.Members = append(out.Members, id)
				if b.bits[i] == '1' {
					out.Selected = append(out.Selected, id)
				}
			}
		}
	}
	return out, nil
}

func (u *selectionDraftUnit) DraftMembers(ctx context.Context, draft app.Draft) ([]string, error) {
	rows, err := u.query(ctx, `SELECT members,selected::text FROM organization.department_selection_block WHERE instance_id=$1 AND draft_id=$2 ORDER BY first_id`, u.instance, draft.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, draft.Selected)
	for rows.Next() {
		var block []string
		var bits string
		if err = rows.Scan(&block, &bits); err != nil {
			return nil, err
		}
		for i, id := range block {
			if bits[i] == '1' {
				ids = append(ids, id)
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) != draft.Selected {
		return nil, app.DraftUnavailable()
	}
	return ids, nil
}

func (u *selectionDraftUnit) EditDraft(ctx context.Context, draft app.Draft, edit app.DraftEdit) (app.Draft, error) {
	ids := append(append(slices.Clone(edit.Add), edit.Remove...), edit.Restore...)
	if len(ids) > app.DraftEditLimit {
		return draft, app.DraftBatchLimit()
	}
	if len(ids) == 0 {
		return draft, nil
	}
	want := map[string]byte{}
	for _, id := range ids {
		if !foundation.ValidID(id) || want[id] != 0 {
			return draft, app.DraftUnavailable()
		}
		want[id] = '0'
	}
	for _, id := range append(slices.Clone(edit.Add), edit.Restore...) {
		want[id] = '1'
	}
	if draft.Mode != "explicit" && len(edit.Add) > 0 {
		return draft, app.DraftUnavailable()
	}
	blocks, err := u.draftBlocks(ctx, draft.ID, ids)
	if err != nil {
		return draft, err
	}
	slices.SortFunc(blocks, func(a, b draftBlock) int { return strings.Compare(a.first, b.first) })
	seen := map[string]bool{}
	for index := range blocks {
		b := &blocks[index]
		bits := []byte(b.bits)
		for i, id := range b.ids {
			if next, ok := want[id]; ok {
				seen[id] = true
				if bits[i] != next {
					if next == '1' {
						draft.Selected++
					} else {
						draft.Selected--
					}
					bits[i] = next
				}
			}
		}
		b.bits = string(bits)
	}
	for _, id := range append(slices.Clone(edit.Remove), edit.Restore...) {
		if !seen[id] {
			return draft, app.DraftUnavailable()
		}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, draftBlock{})
	}
	for _, id := range edit.Add {
		if seen[id] {
			continue
		}
		if draft.Members >= app.DraftMaxMembers {
			return draft, app.DraftMemberLimit()
		}
		index := 0
		for i := 1; i < len(blocks) && blocks[i].first <= id; i++ {
			index = i
		}
		b := &blocks[index]
		position, _ := slices.BinarySearch(b.ids, id)
		b.ids = slices.Insert(b.ids, position, id)
		b.bits = b.bits[:position] + "1" + b.bits[position:]
		draft.Members++
		draft.Selected++
	}
	// Every membership and budget check is complete before the first mutation.
	// Only hit blocks are rewritten, each once; new IDs preserve UUID order.
	for _, b := range blocks {
		if len(b.ids) == 0 {
			continue
		}
		if b.first != "" {
			if _, err = u.exec(ctx, `DELETE FROM organization.department_selection_block WHERE instance_id=$1 AND draft_id=$2 AND first_id=$3`, u.instance, draft.ID, b.first); err != nil {
				return draft, err
			}
		}
		// Balanced splits prevent repeated inserts into a full prefix from
		// creating one-row tails and unbounded block overhead.
		parts := (len(b.ids) + 1023) / 1024
		width := (len(b.ids) + parts - 1) / parts
		for start := 0; start < len(b.ids); start += width {
			end := min(start+width, len(b.ids))
			if err = u.writeDraftBlock(ctx, draft.ID, b.ids[start:end], b.bits[start:end]); err != nil {
				return draft, err
			}
		}
	}
	command, err := u.exec(ctx, `UPDATE organization.department_selection_draft SET revision=revision+1,member_count=$4,selected_count=$5 WHERE instance_id=$1 AND draft_id=$2 AND revision=$3`, u.instance, draft.ID, draft.Revision, draft.Members, draft.Selected)
	if err != nil {
		return draft, err
	}
	if command.RowsAffected() != 1 {
		return draft, app.DraftConflict()
	}
	draft.Revision++
	return draft, nil
}

func (u *selectionDraftUnit) CancelDraft(ctx context.Context, id string, revision int64) error {
	if !foundation.ValidID(id) || revision < 1 {
		return app.DraftUnavailable()
	}
	if err := u.lockDraftSession(ctx); err != nil {
		return err
	}
	p := u.guard.Principal()
	command, err := u.exec(ctx, `DELETE FROM organization.department_selection_draft WHERE instance_id=$1 AND draft_id=$2 AND subject_id=$3 AND session_id=$4 AND revision=$5`, u.instance, id, p.ID, p.SessionID, revision)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return app.DraftConflict()
	}
	return nil
}

var _ app.Drafts = (*selectionDraftUnit)(nil)

// Owner is fixed by the typed adapter. These mechanics store only temporary
// identities; the adapter retains all business projection and authorization.
type selectionDraftUnit struct {
	*unit
	kind string
}

func (u *selectionDraftUnit) sql(q string) string {
	if u.kind == "legalentity" {
		return strings.ReplaceAll(q, "organization.department_selection_", "organization.legal_entity_selection_")
	}
	if u.kind == "location" {
		return strings.ReplaceAll(q, "organization.department_selection_", "organization.location_selection_")
	}
	return q
}
func (u *selectionDraftUnit) exec(ctx context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	return u.tx.Exec(ctx, u.sql(q), args...)
}
func (u *selectionDraftUnit) queryRow(ctx context.Context, q string, args ...any) pgx.Row {
	return u.tx.QueryRow(ctx, u.sql(q), args...)
}
func (u *selectionDraftUnit) query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	return u.tx.Query(ctx, u.sql(q), args...)
}
