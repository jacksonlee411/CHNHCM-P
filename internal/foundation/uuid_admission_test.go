package foundation

import (
	"regexp"
	"testing"
)

func TestUUIDAdmissionKeepsCanonicalVersionVariantAndEveryByte(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	base := "12345678-abcd-4123-8123-123456789abc"
	for i := range len(base) {
		for value := 0; value < 256; value++ {
			candidate := []byte(base)
			candidate[i] = byte(value)
			if got, want := ValidID(string(candidate)), pattern.Match(candidate); got != want {
				t.Fatalf("position=%d byte=%d got=%v want=%v", i, value, got, want)
			}
		}
	}
	for _, value := range []string{"", base[:35], base + "0", base + "\n", "中文" + base} {
		if ValidID(value) {
			t.Fatal("noncanonical UUID admitted")
		}
	}
}
