package selection

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
)

const DraftMaxMembers = 131072
const DraftEditLimit = 1024

// Draft is identity intent, never a reusable read or export authorization.
type Draft struct {
	ID, Enterprise, Day, QueryKey, Mode string
	Revision                            int64
	Members, Selected                   int
}

type DraftPageSelection struct{ Members, Selected []string }

type DraftEdit struct{ Add, Remove, Restore []string }

type Drafts interface {
	CreateDraft(context.Context, Draft, []string) (Draft, error)
	ReadDraft(context.Context, string, int64, string, string) (Draft, error)
	EditDraft(context.Context, Draft, DraftEdit) (Draft, error)
	DraftPage(context.Context, Draft, []string) (DraftPageSelection, error)
	DraftMembers(context.Context, Draft) ([]string, error)
	CancelDraft(context.Context, string, int64) error
}

func DraftUnavailable() error {
	return foundation.Fail(409, "M02_SELECTION_CHANGED", "选择已失效，请重新选择")
}

func DraftConflict() error {
	return foundation.Fail(409, "M02_SELECTION_CONFLICT", "选择已被修改，请重新核对")
}

func DraftLimit() error {
	return foundation.Fail(429, "M02_SELECTION_RESOURCE_LIMIT", "选择服务暂时不可用，请稍后重试")
}

func DraftMemberLimit() error {
	return foundation.Fail(413, "M02_SELECTION_MEMBER_LIMIT", fmt.Sprintf("最多可选择%d项，请分批操作", DraftMaxMembers))
}
func DraftBatchLimit() error {
	return foundation.Fail(413, "M02_SELECTION_BATCH_LIMIT", "选择请求过大，请减少本次选择项")
}
