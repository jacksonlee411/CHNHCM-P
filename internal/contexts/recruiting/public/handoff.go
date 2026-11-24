package public

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type HandoffRef struct {
	Offer   string `json:"offer"`
	Version string `json:"version"`
	Number  string `json:"number"`
}
type HireResult struct {
	Action     string `json:"action"`
	Revision   string `json:"revision"`
	Assignment string `json:"assignment"`
}
type Handoffs interface {
	OfferForHire(context.Context, f.Credentials, string, string, bool) (Offer, error)
	CompleteHire(context.Context, f.Credentials, Offer, HireResult) (string, error)
	CancelHireConversion(context.Context, f.Credentials, HandoffRef, string, string, string) error
}
