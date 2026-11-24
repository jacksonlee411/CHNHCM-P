package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	"context"
)

func (u *locationUnit) CreateDraft(ctx context.Context, draft app.Draft, ids []string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).CreateDraft(ctx, draft, ids)
}
func (u *locationUnit) ReadDraft(ctx context.Context, id string, revision int64, day, key string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).ReadDraft(ctx, id, revision, day, key)
}
func (u *locationUnit) EditDraft(ctx context.Context, draft app.Draft, edit app.DraftEdit) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).EditDraft(ctx, draft, edit)
}
func (u *locationUnit) DraftPage(ctx context.Context, draft app.Draft, ids []string) (app.DraftPageSelection, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).DraftPage(ctx, draft, ids)
}
func (u *locationUnit) DraftMembers(ctx context.Context, draft app.Draft) ([]string, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).DraftMembers(ctx, draft)
}
func (u *locationUnit) CancelDraft(ctx context.Context, id string, revision int64) error {
	return (&selectionDraftUnit{unit: u.unit, kind: "location"}).CancelDraft(ctx, id, revision)
}
