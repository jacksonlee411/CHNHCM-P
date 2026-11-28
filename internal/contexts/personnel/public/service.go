package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// Service exposes people independently of work relationships and assignments.
type Service interface {
	AddressTypes(context.Context, foundation.Credentials) (AddressTypeList, error)
	AddressTypeDetail(context.Context, foundation.Credentials, string) (AddressTypeDetail, error)
	AddressTypeSource(context.Context, foundation.Credentials, string, string) (AddressTypeSource, error)
	AddressTypeCandidates(context.Context, foundation.Credentials, string, string) ([]AddressTypeOption, error)
	SaveAddressType(context.Context, foundation.Credentials, AddressTypeCommand) (AddressType, error)
	AddressTypeReceipt(context.Context, foundation.Credentials, string) (AddressTypeReceipt, error)
	CheckHirePerson(context.Context, foundation.Credentials, string) error
	CheckIdentity(context.Context, foundation.Credentials, HireIdentityRequest) (HireIdentityResult, error)
	Entry(context.Context, foundation.Credentials) (Entry, error)
	CreateQuery(context.Context, foundation.Credentials, string) (string, error)
	List(context.Context, foundation.Credentials, Query) (Directory, error)
	Navigation(context.Context, foundation.Credentials, string, Query) (Nav, error)
	Detail(context.Context, foundation.Credentials, string, string, string) (Detail, error)
	Differences(context.Context, foundation.Credentials, string, string) ([]Difference, error)
	Changes(context.Context, foundation.Credentials, string, ChangeQuery) (ChangePage, error)
	Register(context.Context, foundation.Credentials, Register) (Result, error)
	Correct(context.Context, foundation.Credentials, string, Correct) (Result, error)
	Renumber(context.Context, foundation.Credentials, string, Renumber) (Result, error)
	Resolve(context.Context, foundation.Credentials, string) (Result, error)
	ResolveNumber(context.Context, foundation.Credentials, string) (Summary, error)
	GetRecordedNumber(context.Context, foundation.Credentials, string, string) (string, error)
}
