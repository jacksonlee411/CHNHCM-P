package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"context"
)

func (u *departmentUnit) CaptureQuery(ctx context.Context, q app.QueryRequest) (app.QueryCapture, error) {
	return history.CaptureDepartmentQuery(ctx, u.tx, u.guard, q)
}

func (u *departmentUnit) WalkParents(ctx context.Context, q app.ParentRequest, visit func(app.ParentFrame) (bool, error)) error {
	return history.WalkDepartmentParents(ctx, u.tx, u.guard, q, visit)
}

func (u *departmentUnit) WalkQuery(ctx context.Context, q app.QueryRequest, visit func(app.QueryFrame) (bool, error)) error {
	return history.WalkDepartmentQuery(ctx, u.tx, u.guard, q, visit)
}
