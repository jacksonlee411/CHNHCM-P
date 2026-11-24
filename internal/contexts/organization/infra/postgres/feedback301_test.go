package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"testing"
)

func TestFeedback301HardLimitsRejectBeforeStorageWork(t *testing.T) {
	unit := &departmentUnit{}
	_, memberErr := unit.CreateDraft(context.Background(), app.Draft{}, make([]string, app.DraftMaxMembers+1))
	_, batchErr := unit.EditDraft(context.Background(), app.Draft{}, app.DraftEdit{Add: make([]string, app.DraftEditLimit+1)})
	for _, tc := range []struct {
		err  error
		code string
	}{
		{memberErr, "M02_SELECTION_MEMBER_LIMIT"}, {batchErr, "M02_SELECTION_BATCH_LIMIT"},
	} {
		var fault *foundation.Fault
		if !errors.As(tc.err, &fault) || fault.Code != tc.code || fault.Status != 413 {
			t.Fatalf("unexpected hard limit: %v", tc.err)
		}
	}
}
