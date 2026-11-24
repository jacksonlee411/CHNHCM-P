package foundation

import "golang.org/x/text/unicode/norm"

// UnicodeNormalizationVersion identifies the Unicode data used by these small
// text primitives. Business name policies belong to their owning context.
const UnicodeNormalizationVersion = norm.Version

func NFC(s string) string                  { return norm.NFC.String(s) }
func CanonicalCombiningClass(r rune) uint8 { return norm.NFC.PropertiesString(string(r)).CCC() }
