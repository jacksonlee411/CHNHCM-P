package geoarea

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"testing"
)

func TestCandidateCursorBoundaries(t *testing.T) {
	b := queryCursor{Kind: "geoarea", Instance: foundation.NewID(), Subject: foundation.NewID(), Session: foundation.NewID(), Day: "2026-11-14", Query: strings.Repeat("a", 64), Source: "7"}
	id := foundation.NewID()
	token := b.encode("440100", id)
	if len(token) > 4096 {
		t.Fatal("cursor exceeds its decoder bound")
	}
	key, got, e := b.decode(token)
	if e != nil || key != "440100" || got != id {
		t.Fatal(key, got, e)
	}
	for _, change := range []func(*queryCursor){func(v *queryCursor) { v.Kind = "country" }, func(v *queryCursor) { v.Instance = foundation.NewID() }, func(v *queryCursor) { v.Subject = foundation.NewID() }, func(v *queryCursor) { v.Session = foundation.NewID() }, func(v *queryCursor) { v.Day = "2026-11-15" }, func(v *queryCursor) { v.Query = strings.Repeat("b", 64) }, func(v *queryCursor) { v.Source = "8" }} {
		altered := b
		change(&altered)
		if _, _, e := altered.decode(token); e == nil {
			t.Fatal("changed binding accepted")
		}
	}
	for _, bad := range []string{token + "x", strings.Repeat("x", 4097), "", strings.Replace(token, ".", "_", 1)} {
		if _, _, e := b.decode(bad); e == nil {
			t.Fatal("tampered cursor accepted")
		}
	}
}
