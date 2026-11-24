package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"testing"
	"time"
)

func TestProfileInputs(t *testing.T) {
	for _, v := range []string{"", " \n张", "1-25", strings.Repeat("青", 201), "王\u2028青"} {
		if _, e := Name(v); e == nil {
			t.Errorf("accepted invalid name %q", v)
		}
	}
	if got, e := Name("  A\u0301 娜·青 "); e != nil || got != "Á 娜·青" {
		t.Fatal("NFC name")
	}
	if v, e := Number(" r-01 "); e != nil || v != "R-01" {
		t.Fatal("number normalization")
	}
	for _, v := range []string{"-A", "A-", "Ａ1", "A_1", strings.Repeat("a", 33)} {
		if _, e := Number(v); e == nil {
			t.Fatal("invalid number admitted")
		}
	}
	for _, v := range []string{"", "  ", "A\n1", strings.Repeat("a", 201)} {
		if _, e := Material("form", v); e == nil {
			t.Fatal("invalid evidence admitted")
		}
	}
	if ref, e := Material("form", " 00012a "); e != nil || ref != "00012A" {
		t.Fatal("issued form number")
	}
	if number, e := Document("resident_id", " 00000020000101001x "); e != nil || number != "00000020000101001X" {
		t.Fatal("document normalization")
	}
	for _, pair := range [][2]string{{"", "abc"}, {"passport", ""}, {"resident_id", "123"}, {"other", "ab\ncd"}} {
		if _, e := Document(pair[0], pair[1]); e == nil {
			t.Fatal("invalid document accepted")
		}
	}
	for _, v := range []string{"1+2345", "1234", "+12---345", "1234567890123456", " 12345 ", "１２３４５"} {
		if _, e := Contact("phone", v); e == nil {
			t.Fatal("invalid phone admitted")
		}
	}
	if v, e := Contact("email", "A@EXAMPLE.TEST"); e != nil || v != "A@example.test" || Mask("email", v) != "A****" {
		t.Fatal("email normalization")
	}
	if Mask("phone", "+8612345") != "****2345" {
		t.Fatal("phone mask")
	}
	at := time.Date(2027, 3, 31, 23, 59, 59, 0, foundation.Shanghai)
	if e := Decision(at, "correction", "D-1", "2027-04-01", false); e != nil {
		t.Fatal(e)
	}
	if e := Decision(at.Add(time.Second), "migration", "D-1", "2027-04-01", true); e == nil {
		t.Fatal("expired decision")
	}
	if expiry := Cutoff(at, 1); !expiry.Equal(at.Add(time.Second)) {
		t.Fatal("Beijing cutoff", expiry)
	}
}
