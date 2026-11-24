package foundation

import (
	"regexp"
	"testing"
)

var uuidReference = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestCanonicalUUIDAdmission(t *testing.T) {
	base := "12345678-abcd-4123-8123-123456789abc"
	for _, s := range []string{"", base, base + "\n", "12345678-ABCD-4123-8123-123456789ABC", "00000000-0000-0000-0000-000000000000", "12345678-abcd-6123-8123-123456789abc"} {
		if ValidID(s) != uuidReference.MatchString(s) {
			t.Fatalf("UUID admission differs for %q", s)
		}
	}
	// Every position, every byte: includes separators, version, variant, non-ASCII
	// and control characters. The oracle is the previous canonical contract.
	for i := range len(base) {
		for c := 0; c < 256; c++ {
			b := []byte(base)
			b[i] = byte(c)
			s := string(b)
			if ValidID(s) != uuidReference.MatchString(s) {
				t.Fatalf("UUID admission differs at %d byte %d", i, c)
			}
		}
	}
}

var uuidBenchmarkResult bool

func BenchmarkCanonicalUUID(b *testing.B) {
	s := "12345678-abcd-4123-8123-123456789abc"
	for b.Loop() {
		uuidBenchmarkResult = ValidID(s)
	}
}
