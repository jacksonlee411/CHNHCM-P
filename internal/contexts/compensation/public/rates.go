// Package public exposes Compensation configuration, without persistence types.
package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type WriteRate struct {
	Action           string             `json:"action"`
	EffectiveDate    string             `json:"effectiveDate,omitempty"`
	Reason           string             `json:"reason,omitempty"`
	GradeID          string             `json:"gradeId,omitempty"`
	Currency         string             `json:"currency,omitempty"`
	Frequency        string             `json:"frequency,omitempty"`
	Unit             string             `json:"unit,omitempty"`
	Precision        *int               `json:"precision,omitempty"`
	RecordID         string             `json:"recordId,omitempty"`
	ExpectedVersion  string             `json:"expectedVersion,omitempty"`
	ExpectedRevision string             `json:"expectedRevision,omitempty"`
	Changes          map[string]*string `json:"changes,omitempty"`
}

type Reference struct {
	Open          bool   `json:"open,omitempty"`
	BusinessAt    string `json:"businessAt"`
	SourceVersion string `json:"sourceVersion"`
	RecordID      string `json:"recordId"`
	Revision      string `json:"revision"`
}
type GradeLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}
type RateRecord struct {
	ID        string      `json:"recordId"`
	Revision  string      `json:"revision"`
	Start     string      `json:"effectiveDate,omitempty"`
	Until     string      `json:"endExclusive,omitempty"`
	Status    string      `json:"status,omitempty"`
	Operation string      `json:"operation,omitempty"`
	Reason    string      `json:"reason,omitempty"`
	Form      string      `json:"form,omitempty"`
	Amount    string      `json:"amount,omitempty"`
	Minimum   string      `json:"minimum,omitempty"`
	Guidance  string      `json:"guidance,omitempty"`
	Maximum   string      `json:"maximum,omitempty"`
	Grade     *GradeLabel `json:"grade,omitempty"`
	Adopted   *Reference  `json:"adopted,omitempty"`
}
type Rate struct {
	ID        string     `json:"id"`
	Version   string     `json:"sourceVersion"`
	Currency  string     `json:"currency,omitempty"`
	Frequency string     `json:"frequency,omitempty"`
	Unit      string     `json:"unit,omitempty"`
	Precision *int       `json:"precision,omitempty"`
	Record    RateRecord `json:"record"`
}
type DateNode struct {
	ID     string `json:"recordId"`
	Start  string `json:"effectiveDate"`
	Active bool   `json:"active,omitempty"`
}
type Window struct {
	Minimum   string `json:"minimum"`
	Maximum   string `json:"maximum,omitempty"`
	Default   string `json:"default,omitempty"`
	Available bool   `json:"available"`
}
type RateDetail struct {
	Rate
	Today   string              `json:"today"`
	Dates   []DateNode          `json:"dates"`
	History bool                `json:"history"`
	Actions map[string][]string `json:"actions"`
	Reasons map[string][]string `json:"reasons"`
	Windows map[string]Window   `json:"windows"`
}
type RateList struct {
	Readable []string `json:"readable"`
	Items    []Rate   `json:"items"`
	Total    int      `json:"total"`
	Today    string   `json:"today"`
	Create   bool     `json:"create"`
}
type RateQuery struct {
	Q, GradeID, Currency, Frequency, Form, Status string
	Page, Size                                    int
}
type Change struct {
	Previous   string      `json:"previousSourceVersion,omitempty"`
	Currency   string      `json:"currency,omitempty"`
	Frequency  string      `json:"frequency,omitempty"`
	Unit       string      `json:"unit,omitempty"`
	Precision  *int        `json:"precision,omitempty"`
	Version    string      `json:"sourceVersion"`
	Action     string      `json:"action"`
	RecordedAt string      `json:"recordedAt"`
	Operator   string      `json:"operator,omitempty"`
	Before     *RateRecord `json:"before,omitempty"`
	After      *RateRecord `json:"after,omitempty"`
}
type Receipt struct {
	SubmissionID string `json:"submissionId"`
	Message      string `json:"message,omitempty"`
	Outcome      string `json:"outcome"`
	ObjectID     string `json:"objectId,omitempty"`
	Version      string `json:"sourceVersion,omitempty"`
	RecordID     string `json:"recordId,omitempty"`
	Revision     string `json:"revision,omitempty"`
}
type RateContext struct {
	Reasons     []string          `json:"reasons"`
	Units       map[string]string `json:"units"`
	DefaultDate string            `json:"defaultEffectiveDate"`
	Writable    []string          `json:"writable"`
}
type MenuAccess struct {
	Plans      bool `json:"plans"`
	GradeRates bool `json:"gradeRates"`
}
type RateService interface {
	Menu(context.Context, foundation.Credentials) (MenuAccess, error)
	RateContext(context.Context, foundation.Credentials) (RateContext, error)
	ListRates(context.Context, foundation.Credentials, RateQuery) (RateList, error)
	ReadRate(context.Context, foundation.Credentials, string, string, string) (RateDetail, error)
	RateHistory(context.Context, foundation.Credentials, string, string) ([]Change, error)
	SaveRate(context.Context, foundation.Credentials, string, string, WriteRate) (Receipt, error)
	RateReceipt(context.Context, foundation.Credentials, string) (Receipt, error)
	ResolveRate(context.Context, foundation.Credentials, string) (Receipt, error)
}
