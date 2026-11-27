package public

import (
	"context"
	"time"
)

type Target struct {
	Options []string `json:"options,omitempty"`
	Kind    string   `json:"kind"`
	ID      string   `json:"id"`
	Code    string   `json:"code"`
	Name    string   `json:"name"`
}

// TargetDirectory resolves stable business identities using the owning domains in the current transaction.
// A nil ID set requests the instance's complete candidate directory. An empty set requests no objects.
type TargetDirectory func(context.Context, []string) ([]Target, error)

type Definition struct {
	Options      []string `json:"options,omitempty"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	ID           string   `json:"id,omitempty"`
	DefinitionID string   `json:"definitionId,omitempty"`
	Number       int64    `json:"number,omitempty"`
	Actions      []string `json:"actions,omitempty"`
	TargetKind   string   `json:"targetKind,omitempty"`
	TargetID     string   `json:"targetId,omitempty"`
	Read         []string `json:"readFields,omitempty"`
	Write        []string `json:"writeFields,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Fields       string   `json:"fields,omitempty"`
	Disabled     bool     `json:"disabled,omitempty"`
}

type ManagementLimit struct {
	Owner        string     `json:"owner"`
	Rights       []string   `json:"rights"`
	Role         string     `json:"role"`
	Scope        string     `json:"scope"`
	Fields       string     `json:"fields"`
	MaxProxyDays int        `json:"maxProxyDays"`
	Until        *time.Time `json:"until,omitempty"`
}
type Mandate struct {
	ID      string          `json:"id"`
	Limit   ManagementLimit `json:"limit"`
	Revoked bool            `json:"revoked"`
}

type ManagedAccount struct {
	ID       string   `json:"id"`
	Login    string   `json:"login"`
	Display  string   `json:"displayName"`
	Owner    string   `json:"owner"`
	State    string   `json:"state"`
	Revision int64    `json:"revision"`
	Options  []string `json:"options,omitempty"`
}

type ManagedGrant struct {
	ID       string     `json:"id"`
	Subject  string     `json:"subject"`
	Role     string     `json:"role"`
	Scope    string     `json:"scope"`
	Fields   string     `json:"fields"`
	Revision int64      `json:"revision"`
	Source   string     `json:"source"`
	Purpose  string     `json:"purpose"`
	From     *time.Time `json:"from,omitempty"`
	Until    *time.Time `json:"until,omitempty"`
	State    string     `json:"state"`
}
type CatalogAction struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Menu string `json:"menu"`
	API  string `json:"api"`
}
type ManagementEvent struct {
	PreviousOwner string    `json:"previousOwner,omitempty"`
	AssignedOwner string    `json:"assignedOwner,omitempty"`
	ID            string    `json:"id"`
	Actor         string    `json:"actor"`
	Target        string    `json:"target"`
	Operation     string    `json:"operation"`
	Reason        string    `json:"reason"`
	At            time.Time `json:"at"`
	Request       string    `json:"request"`
}
type ManagementView struct {
	SuggestedProxyDays int               `json:"suggestedProxyDays,omitempty"`
	Total              int               `json:"total"`
	Rights             []string          `json:"rights,omitempty"`
	Owners             []Target          `json:"owners,omitempty"`
	Targets            []Target          `json:"targets,omitempty"`
	Actions            []CatalogAction   `json:"actions,omitempty"`
	Read               []string          `json:"readFields,omitempty"`
	Write              []string          `json:"writeFields,omitempty"`
	Accounts           []ManagedAccount  `json:"accounts,omitempty"`
	Versions           []Definition      `json:"versions,omitempty"`
	Grants             []ManagedGrant    `json:"grants,omitempty"`
	Mandates           []Mandate         `json:"mandates,omitempty"`
	Events             []ManagementEvent `json:"events,omitempty"`
}
type ManagementQuery struct {
	Area, Subject, Search, Owner, State, Object string
	Page, PageSize                              int
}

type ManagementCommand struct {
	Request   string           `json:"requestId"`
	Instance  string           `json:"instanceId"`
	Operation string           `json:"operation"`
	Subject   string           `json:"subject,omitempty"`
	ID        string           `json:"id,omitempty"`
	Revision  int64            `json:"revision,omitempty"`
	Login     string           `json:"login,omitempty"`
	Display   string           `json:"displayName,omitempty"`
	Password  string           `json:"password,omitempty"`
	Owner     string           `json:"owner,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	Version   *Definition      `json:"version,omitempty"`
	NewScope  *Definition      `json:"newScope,omitempty"`
	NewFields *Definition      `json:"newFields,omitempty"`
	Role      string           `json:"role,omitempty"`
	Scope     string           `json:"scope,omitempty"`
	Fields    string           `json:"fields,omitempty"`
	Source    string           `json:"source,omitempty"`
	Purpose   string           `json:"purpose,omitempty"`
	From      *time.Time       `json:"from,omitempty"`
	Until     *time.Time       `json:"until,omitempty"`
	Limit     *ManagementLimit `json:"limit,omitempty"`
}
type ManagementResult struct {
	Request   string    `json:"requestId"`
	Operation string    `json:"operation"`
	Target    string    `json:"target"`
	Subject   string    `json:"subject,omitempty"`
	At        time.Time `json:"at"`
}
type Management interface {
	ManagementView(context.Context, Credentials, ManagementQuery) (ManagementView, error)
	Manage(context.Context, Credentials, ManagementCommand) (ManagementResult, error)
	ManagementResult(context.Context, Credentials, string) (ManagementResult, error)
	ResolveManagement(context.Context, Credentials, string) (ManagementResult, error)
}
type RecoveryCommand struct {
	Instance      string `json:"instanceId"`
	Request       string `json:"requestId"`
	Operation     string `json:"operation"`
	ExpectedAdmin string `json:"expectedAdmin,omitempty"`
	Login         string `json:"login"`
	Password      string `json:"password,omitempty"`
	Reason        string `json:"reason"`
	Operator      string `json:"operator"`
	MaxProxyDays  *int   `json:"maxProxyDays,omitempty"`
}
