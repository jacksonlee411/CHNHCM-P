package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"context"
)

func (u *departmentUnit) CreateDraft(ctx context.Context, draft app.Draft, ids []string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).CreateDraft(ctx, draft, ids)
}
func (u *departmentUnit) ReadDraft(ctx context.Context, id string, revision int64, day, key string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).ReadDraft(ctx, id, revision, day, key)
}
func (u *departmentUnit) EditDraft(ctx context.Context, draft app.Draft, edit app.DraftEdit) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).EditDraft(ctx, draft, edit)
}
func (u *departmentUnit) DraftPage(ctx context.Context, draft app.Draft, ids []string) (app.DraftPageSelection, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).DraftPage(ctx, draft, ids)
}
func (u *departmentUnit) DraftMembers(ctx context.Context, draft app.Draft) ([]string, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).DraftMembers(ctx, draft)
}
func (u *departmentUnit) CancelDraft(ctx context.Context, id string, revision int64) error {
	return (&selectionDraftUnit{unit: u.unit, kind: "department"}).CancelDraft(ctx, id, revision)
}

// Startup invalidates this single API instance's temporary intent. Business
// sources and terminal outcomes are outside these table groups.
func (s *Store) ResetDrafts(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM organization.department_selection_draft WHERE instance_id=$1`, s.Instance)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.location_selection_draft WHERE instance_id=$1`, s.Instance)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.legal_entity_selection_draft WHERE instance_id=$1`, s.Instance)
	return err
}
func (s *Store) DropSessionDraft(ctx context.Context, subject, session string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM organization.department_selection_draft WHERE instance_id=$1 AND subject_id=$2 AND session_id=$3`, s.Instance, subject, session)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.location_selection_draft WHERE instance_id=$1 AND subject_id=$2 AND session_id=$3`, s.Instance, subject, session)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.legal_entity_selection_draft WHERE instance_id=$1 AND subject_id=$2 AND session_id=$3`, s.Instance, subject, session)
	return err
}

func (s *Store) SweepDrafts(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM organization.department_selection_draft WHERE (instance_id,draft_id) IN
 (SELECT instance_id,draft_id FROM organization.department_selection_draft WHERE instance_id=$1 AND expires_at<=clock_timestamp()
 ORDER BY expires_at LIMIT 128 FOR UPDATE SKIP LOCKED)`, s.Instance)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.location_selection_draft WHERE (instance_id,draft_id) IN
 (SELECT instance_id,draft_id FROM organization.location_selection_draft WHERE instance_id=$1 AND expires_at<=clock_timestamp()
 ORDER BY expires_at LIMIT 128 FOR UPDATE SKIP LOCKED)`, s.Instance)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM organization.legal_entity_selection_draft WHERE (instance_id,draft_id) IN
 (SELECT instance_id,draft_id FROM organization.legal_entity_selection_draft WHERE instance_id=$1 AND expires_at<=clock_timestamp()
 ORDER BY expires_at LIMIT 128 FOR UPDATE SKIP LOCKED)`, s.Instance)
	return err
}
