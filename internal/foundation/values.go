// Package foundation holds small cross-context values, not business policies.
package foundation

import (
	"crypto/rand"
	"fmt"
	"time"
	_ "time/tzdata"
)

// Credentials is transport-neutral authentication material; only Access interprets it.
type Credentials struct {
	Token, CSRF string
	Write       bool
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

var uuidHex = func() (table [256]bool) {
	for c := byte('0'); c <= '9'; c++ {
		table[c] = true
	}
	for c := byte('a'); c <= 'f'; c++ {
		table[c] = true
	}
	return
}()

func validUUIDHex(s string) bool {
	for i := range len(s) {
		if !uuidHex[s[i]] {
			return false
		}
	}
	return true
}

func ValidID(s string) bool {
	if len(s) != 36 || s[14] < '1' || s[14] > '5' || !(s[19] >= '8' && s[19] <= '9' || s[19] >= 'a' && s[19] <= 'b') || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	return validUUIDHex(s[:8]) && validUUIDHex(s[9:13]) && validUUIDHex(s[14:18]) && validUUIDHex(s[19:23]) && validUUIDHex(s[24:])
}

func ValidDate(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10 && t.Year() >= 1 && t.Year() <= 9999 && t.Format("2006-01-02") == s
}

type Fault struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Status  int    `json:"-"`
}

func (e *Fault) Error() string { return e.Code }
func Fail(status int, code, message string) *Fault {
	return &Fault{Status: status, Code: code, Message: message}
}

var Shanghai = func() *time.Location {
	z, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic("embedded Asia/Shanghai timezone unavailable")
	}
	return z
}()
