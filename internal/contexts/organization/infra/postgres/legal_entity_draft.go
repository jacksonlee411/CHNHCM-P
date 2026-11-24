package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	"context"
)

func (u *legalentityUnit) CreateDraft(ctx context.Context, draft app.Draft, ids []string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).CreateDraft(ctx, draft, ids)
}
func (u *legalentityUnit) ReadDraft(ctx context.Context, id string, revision int64, day, key string) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).ReadDraft(ctx, id, revision, day, key)
}
func (u *legalentityUnit) EditDraft(ctx context.Context, draft app.Draft, edit app.DraftEdit) (app.Draft, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).EditDraft(ctx, draft, edit)
}
func (u *legalentityUnit) DraftPage(ctx context.Context, draft app.Draft, ids []string) (app.DraftPageSelection, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).DraftPage(ctx, draft, ids)
}
func (u *legalentityUnit) DraftMembers(ctx context.Context, draft app.Draft) ([]string, error) {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).DraftMembers(ctx, draft)
}
func (u *legalentityUnit) CancelDraft(ctx context.Context, id string, revision int64) error {
	return (&selectionDraftUnit{unit: u.unit, kind: "legalentity"}).CancelDraft(ctx, id, revision)
}
