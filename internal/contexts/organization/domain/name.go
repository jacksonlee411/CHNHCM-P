package domain

import "chnhcm.local/chnhcm/internal/foundation"

const NameMaxLength = foundation.NameMaxLength
const NameGuidance = foundation.NameGuidance
const nameUnicodeVersion = foundation.NameUnicodeVersion

func sameName(a, b string) bool { return foundation.NFC(a) == foundation.NFC(b) }

// NormalizeName preserves M02 errors while sharing only the pure Unicode profile.
func NormalizeName(s string, required bool) (string, error) {
	v, err := foundation.NormalizeName(s, required)
	if err != nil {
		return "", foundation.Fail(400, "M02_INVALID_INPUT", err.Error())
	}
	return v, nil
}
