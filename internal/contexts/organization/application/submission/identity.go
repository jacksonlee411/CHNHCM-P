package submission

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
)

type Kind string

const (
	BusinessUnit Kind = "business_unit"
	Department   Kind = "department"
	Country      Kind = "country"
	GeoArea      Kind = "geoarea"
	Enterprise   Kind = "enterprise"
	Location     Kind = "location"
	LegalEntity  Kind = "legalentity"
)

// Identity comes from a trusted object adapter and authenticated principal.
// Kind selects a fixed lock namespace, never a table or arbitrary SQL identifier.
type Identity struct {
	Kind                                Kind
	InstanceID, SubjectID, SubmissionID string
}

func (i Identity) LockKey() (int64, error) {
	parts := []string{i.InstanceID, i.SubjectID, i.SubmissionID}
	switch i.Kind {
	case BusinessUnit:
		// Preserve the exact existing BU lock encoding.
	case Department:
		parts = append([]string{"organization.department"}, parts...)
	case LegalEntity:
		parts = append([]string{"organization.legalentity"}, parts...)
	case Location:
		parts = append([]string{"organization.location"}, parts...)
	case GeoArea:
		parts = append([]string{"organization.geoarea"}, parts...)
	case Enterprise:
		parts = append([]string{"organization.enterprise"}, parts...)
	case Country:
		parts = append([]string{"organization.country"}, parts...)
	default:
		return 0, errors.New("unknown submission object kind")
	}
	encoded, err := json.Marshal(parts)
	if err != nil {
		return 0, err
	}
	digest := sha256.Sum256(encoded)
	return int64(binary.BigEndian.Uint64(digest[:8])), nil
}

// BindDigest preserves the existing byte format. The canonical request digest
// already binds method and the object's route; this adds its authenticated owner.
func BindDigest(instance, subject string, request []byte) []byte {
	h := sha256.New()
	h.Write([]byte(instance + ":" + subject + ":"))
	h.Write(request)
	return h.Sum(nil)
}
