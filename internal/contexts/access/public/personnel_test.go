package public

import (
	"testing"
	"time"
)

func TestPersonGrantScope(t *testing.T) {
	at := time.Now()
	a := PersonAccess{At: at, Grants: []PersonGrant{
		{Action: "read", Purpose: "directory", Fields: []string{"name", "number"}, IDs: []string{"a", "b"}},
		{Action: "read", Purpose: "directory", Fields: []string{"phone"}, IDs: []string{"b"}},
		{Action: "read", Purpose: "maintenance", Fields: []string{"name", "number", "phone"}},
		{Action: "read", Purpose: "directory", Fields: []string{"name", "number", "phone"}, Expires: at},
		{Action: "read", Purpose: "directory", Fields: []string{"name", "number", "phone"}, IDs: []string{"c", "d"}},
		{Action: "read", Purpose: "directory", Fields: []string{"phone"}, IDs: []string{"d"}, Denied: true},
		{Action: "hold.read", Purpose: "preservation", Fields: []string{"evidence"}, IDs: []string{"a"}, Source: "original"},
	}}
	for _, id := range []string{"a", "b", "d"} {
		if a.Scope("read", "name", "number", "phone").Allows(id) {
			t.Fatal("spliced, expired or denied grant", id)
		}
	}
	if !a.Scope("read", "name", "number", "phone").Allows("c") || !a.Scope("read", "name", "number").Allows("b") {
		t.Fatal("complete scope denied")
	}
	if !a.EvidenceAllowed("a", "original", "hold.read", "evidence") || a.EvidenceAllowed("a", "other", "hold.read", "evidence") || a.Scope("hold.read", "evidence").Allows("a") {
		t.Fatal("source scope mismatch")
	}
}
