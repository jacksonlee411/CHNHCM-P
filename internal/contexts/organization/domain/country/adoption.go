package country

import "chnhcm.local/chnhcm/internal/foundation"

// Adoption is an exact country member captured by a real consumer.
type Adoption struct{ Date, ObjectID, SourceVersion, RecordID, Revision string }

func ValidateAdoption(id string, a Adoption) error {
	if !foundation.ValidDate(a.Date) || a.ObjectID != id || !foundation.ValidID(id) || !foundation.ValidID(a.SourceVersion) || !foundation.ValidID(a.RecordID) || !foundation.ValidID(a.Revision) {
		return foundation.Fail(409, "M02_REFERENCE_UNAVAILABLE", "国家采用依据无效")
	}
	return nil
}
