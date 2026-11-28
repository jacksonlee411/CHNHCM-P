package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Evidence struct {
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
}
type Value struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
type PrimaryDocument struct {
	Kind   string `json:"kind"`
	Number string `json:"number"`
}
type Register struct {
	Name            string           `json:"name"`
	Evidence        *Evidence        `json:"evidence,omitempty"`
	SubmissionID    string           `json:"submissionId"`
	Contacts        []Value          `json:"contacts,omitempty"`
	PrimaryDocument *PrimaryDocument `json:"primaryDocument,omitempty"`
}
type CountryChoice struct {
	ID            string `json:"id"`
	SourceVersion string `json:"sourceVersion"`
}
type CountryAdoption struct {
	Parents       []CountryChoice `json:"parents,omitempty"`
	ID            string          `json:"id"`
	SourceVersion string          `json:"sourceVersion"`
	RecordID      string          `json:"recordId"`
	Revision      string          `json:"revision"`
	Date          string          `json:"date"`
}
type DocumentRow struct {
	Locked         []string          `json:"locked,omitempty"`
	IssuingCountry *CountryAdoption  `json:"issuingCountryAdoption,omitempty"`
	ItemID         string            `json:"itemId"`
	Revision       string            `json:"revision"`
	Fields         map[string]string `json:"fields"`
	LongTerm       bool              `json:"longTerm,omitempty"`
	Primary        bool              `json:"primary,omitempty"`
}
type DocumentEdit struct {
	IssuingCountry   *CountryChoice    `json:"issuingCountryChoice,omitempty"`
	ItemID           string            `json:"itemId,omitempty"`
	ExpectedRevision string            `json:"expectedRevision,omitempty"`
	Fields           map[string]string `json:"fields,omitempty"`
	LongTerm         *bool             `json:"longTerm,omitempty"`
	Primary          *bool             `json:"primary,omitempty"`
	Remove           bool              `json:"remove,omitempty"`
}

// ProfileItem carries only the currently authorized fields of one personal profile item.
type ProfileItem struct {
	AddressType *AddressTypeAdoption       `json:"addressTypeAdoption,omitempty"`
	Masked      []string                   `json:"masked,omitempty"`
	Hidden      []string                   `json:"hidden,omitempty"`
	ItemID      string                     `json:"itemId"`
	Revision    string                     `json:"revision"`
	Fields      map[string]string          `json:"fields"`
	Adoptions   map[string]CountryAdoption `json:"adoptions,omitempty"`
	Locked      []string                   `json:"locked,omitempty"`
	CanRemove   bool                       `json:"canRemove,omitempty"`
}
type PlaceChoice struct {
	ID            string          `json:"id"`
	SourceVersion string          `json:"sourceVersion"`
	Ancestors     []CountryChoice `json:"ancestors,omitempty"`
}
type ProfileItemEdit struct {
	AddressType      *AddressTypeChoice     `json:"addressTypeChoice,omitempty"`
	ItemID           string                 `json:"itemId,omitempty"`
	ExpectedRevision string                 `json:"expectedRevision,omitempty"`
	Fields           map[string]string      `json:"fields,omitempty"`
	Choices          map[string]PlaceChoice `json:"choices,omitempty"`
	Remove           bool                   `json:"remove,omitempty"`
}
type EditItem struct {
	Purpose          *string `json:"purpose,omitempty"`
	Primary          *bool   `json:"primary,omitempty"`
	ItemID           string  `json:"itemId,omitempty"`
	ExpectedRevision string  `json:"expectedRevision,omitempty"`
	Remove           bool    `json:"remove,omitempty"`
	Kind             string  `json:"kind,omitempty"`
	Text             string  `json:"value,omitempty"`
}
type Correct struct {
	Addresses            []ProfileItemEdit `json:"addresses,omitempty"`
	Family               []ProfileItemEdit `json:"family,omitempty"`
	Education            []ProfileItemEdit `json:"education,omitempty"`
	Credentials          []ProfileItemEdit `json:"credentials,omitempty"`
	BankAccounts         []ProfileItemEdit `json:"bankAccounts,omitempty"`
	HealthSummary        *string           `json:"healthSummary,omitempty"`
	Nationality          *CountryChoice    `json:"nationalityChoice,omitempty"`
	Basic                map[string]string `json:"basic,omitempty"`
	Documents            []DocumentEdit    `json:"documents,omitempty"`
	SubmissionID         string            `json:"submissionId"`
	Expected             string            `json:"expectedSourceVersion"`
	Name                 *string           `json:"name,omitempty"`
	Evidence             *Evidence         `json:"evidence,omitempty"`
	Contacts             []EditItem        `json:"contacts,omitempty"`
	PrimaryDocument      *PrimaryDocument  `json:"primaryDocument,omitempty"`
	ClearPrimaryDocument bool              `json:"-"`
}

// A missing field preserves the original. Only an explicit null removes it.
func (v *Correct) UnmarshalJSON(data []byte) error {
	type plain Correct
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if string(fields["healthSummary"]) == "null" {
		return fmt.Errorf("healthSummary requires a string")
	}
	if err := json.Unmarshal(data, (*plain)(v)); err != nil {
		return err
	}
	v.ClearPrimaryDocument = string(fields["primaryDocument"]) == "null"
	return nil
}
func (v Correct) MarshalJSON() ([]byte, error) {
	type plain Correct
	if !v.ClearPrimaryDocument {
		return json.Marshal(plain(v))
	}
	return json.Marshal(struct {
		plain
		PrimaryDocument *PrimaryDocument `json:"primaryDocument"`
	}{plain(v), nil})
}

type Renumber struct {
	SubmissionID string `json:"submissionId"`
	Expected     string `json:"expectedSourceVersion"`
	Old          string `json:"oldNumber"`
	New          string `json:"newNumber"`
	Reason       string `json:"reason"`
	Ref          string `json:"decisionRef"`
	Until        string `json:"validUntil,omitempty"`
}
type NumberDecision struct {
	Old    string `json:"oldNumber"`
	New    string `json:"newNumber"`
	Reason string `json:"reason"`
	Ref    string `json:"decisionRef"`
	Until  string `json:"validUntil,omitempty"`
}
type Summary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Number     string `json:"number"`
	Historical bool   `json:"historical,omitempty"`
}
type Entry struct {
	CanReadAddressTypes bool     `json:"canReadAddressTypes,omitempty"`
	CanRegisterDocument bool     `json:"canRegisterDocument"`
	CanRegisterEvidence bool     `json:"canRegisterEvidence"`
	CanRead             bool     `json:"canRead"`
	CanRegister         bool     `json:"canRegister"`
	ContactKinds        []string `json:"contactKinds"`
}
type Directory struct {
	Rows    []Summary `json:"rows"`
	Total   int       `json:"total"`
	Keyword string    `json:"keyword"`
}
type Nav struct {
	Position int    `json:"position"`
	Total    int    `json:"total"`
	Prev     string `json:"prev,omitempty"`
	Next     string `json:"next,omitempty"`
}
type Query struct {
	Ref, Sort  string
	Page, Size int
}

// DirectoryKey expands one visible person into caller-authorized rows. The
// consumer owns those row identities and their order within that person.
type DirectoryKey struct {
	Person, ID string
	Order      int
}

// DirectoryLookup holds a single authorized query in the caller's database
// transaction. Both projections use the same retention and grant cutoff.
type DirectoryLookup interface {
	Renew(context.Context) error
	Candidates(ids []string) ([]Summary, string, error)
	Page(keys []DirectoryKey, anchor string) (IndexedDirectory, error)
}

type IndexedPerson struct {
	Summary
	Key string
}
type IndexedDirectory struct {
	Rows    []IndexedPerson
	Total   int
	Keyword string
	Nav     Nav
}

type Source struct {
	Version  string    `json:"sourceVersion"`
	At       time.Time `json:"recordedAt"`
	Operator string    `json:"operator"`
	Action   string    `json:"action"`
}
type ChangeQuery struct {
	Page           *int   `json:"page,omitempty"`
	PageSize       *int   `json:"pageSize,omitempty"`
	Through        string `json:"throughSourceVersion,omitempty"`
	FilterField    string `json:"filterField,omitempty"`
	FilterOperator string `json:"filterOperator,omitempty"`
	FilterValue    string `json:"filterValue,omitempty"`
}

// Omitted keys select defaults. Explicit nulls and an empty source cannot reset
// a previously fixed upper bound or silently change the requested page.
func (q *ChangeQuery) UnmarshalJSON(raw []byte) error {
	type plain ChangeQuery
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return err
	}
	for _, key := range []string{"page", "pageSize", "throughSourceVersion"} {
		if v, ok := keys[key]; ok && (string(v) == "null" || key == "throughSourceVersion" && string(v) == `""`) {
			return fmt.Errorf("invalid %s", key)
		}
	}
	return json.Unmarshal(raw, (*plain)(q))
}

type ChangeGroup struct {
	Source
	Previous string       `json:"previousSourceVersion,omitempty"`
	Diffs    []Difference `json:"diffs"`
}
type ChangePage struct {
	Items      []ChangeGroup `json:"items"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	Total      int           `json:"total"`
	Through    string        `json:"throughSourceVersion"`
	ReadFields []string      `json:"readFields"`
}
type Contact struct {
	Locked  []string `json:"locked,omitempty"`
	Purpose string   `json:"purpose,omitempty"`
	Primary bool     `json:"primary,omitempty"`
	Value
	ItemID   string `json:"itemId"`
	Revision string `json:"revision"`
	Masked   bool   `json:"masked,omitempty"`
}
type Detail struct {
	Addresses       []ProfileItem     `json:"addresses,omitempty"`
	Family          []ProfileItem     `json:"family,omitempty"`
	Education       []ProfileItem     `json:"education,omitempty"`
	Credentials     []ProfileItem     `json:"credentials,omitempty"`
	BankAccounts    []ProfileItem     `json:"bankAccounts,omitempty"`
	HealthSummary   *string           `json:"healthSummary,omitempty"`
	Nationality     *CountryAdoption  `json:"nationalityAdoption,omitempty"`
	Basic           map[string]string `json:"basic,omitempty"`
	Documents       []DocumentRow     `json:"documents,omitempty"`
	Readable        []string          `json:"readable,omitempty"`
	PrimaryDocument *PrimaryDocument  `json:"primaryDocument,omitempty"`
	CanReadDocument bool              `json:"canReadDocument,omitempty"`
	ID              string            `json:"id"`
	Name            string            `json:"name,omitempty"`
	Number          string            `json:"number,omitempty"`
	Source
	Evidence *Evidence       `json:"evidence,omitempty"`
	Contacts []Contact       `json:"contacts,omitempty"`
	Editable []string        `json:"editable,omitempty"`
	History  bool            `json:"history,omitempty"`
	Renumber bool            `json:"renumber,omitempty"`
	Decision *NumberDecision `json:"decision,omitempty"`
	Disposed bool            `json:"disposed,omitempty"`
}
type Result struct {
	OriginalNumber   string `json:"originalNumber,omitempty"`
	ExistingPersonID string `json:"existingPersonId,omitempty"`
	SubmissionID     string `json:"submissionId"`
	Outcome          string `json:"outcome"`
	ID               string `json:"id,omitempty"`
	Version          string `json:"sourceVersion,omitempty"`
}

type Difference struct {
	FieldKey  string `json:"fieldKey"`
	ItemID    string `json:"itemId,omitempty"`
	CanReveal bool   `json:"canReveal,omitempty"`
	Field     string `json:"field"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
}

// The optional link is projected for the current viewer and never stored in fault JSON.
type DocumentConflict struct {
	*foundation.Fault
	ExistingPersonID string `json:"existingPersonId,omitempty"`
}

func (e *DocumentConflict) Unwrap() error { return e.Fault }
