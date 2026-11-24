package public

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

type ReplenishmentRecord struct {
	Date       string    `json:"date"`
	Paused     bool      `json:"paused"`
	Reason     string    `json:"reason"`
	Revision   string    `json:"revision"`
	RecordedAt time.Time `json:"recordedAt"`
}
type ReplenishmentChange struct {
	Date     string `json:"date"`
	Paused   bool   `json:"paused"`
	Reason   string `json:"reason"`
	Expected string `json:"expected"`
}
type ReplenishmentView struct {
	Covered   bool                  `json:"covered"`
	CanChange bool                  `json:"canChange"`
	Paused    bool                  `json:"paused"`
	Version   string                `json:"version"`
	Exempt    *int                  `json:"exempt,omitempty"`
	Records   []ReplenishmentRecord `json:"records"`
}
type Replenishments interface {
	ResolveReplenishment(context.Context, f.Credentials, string, string) (f.RuleReceipt, error)
	Replenishment(context.Context, f.Credentials, string, string) (ReplenishmentView, error)
	ChangeReplenishment(context.Context, f.Credentials, string, string, ReplenishmentChange) (ReplenishmentView, error)
}
