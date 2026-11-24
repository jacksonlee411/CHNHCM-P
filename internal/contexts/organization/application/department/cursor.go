package department

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"chnhcm.local/chnhcm/internal/foundation"
)

// A process key deliberately invalidates external cursors on API restart.
// The token authenticates query position; it never contains a permission.
var cursorKey = func() []byte { key := make([]byte, 32); _, _ = rand.Read(key); return key }()

type queryCursor struct {
	Version                                                          int `json:"v"`
	Kind, Instance, Enterprise, Subject, Session, Day, Query, Source string
	Key, ID                                                          string
}

func (binding queryCursor) encode(key, id string) string {
	binding.Version = 1
	binding.Key = key
	binding.ID = id
	data, _ := json.Marshal(binding)
	mac := hmac.New(sha256.New, cursorKey)
	_, _ = mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (binding queryCursor) decode(token string) (key, id string, err error) {
	invalid := func() (string, string, error) {
		return "", "", foundation.Fail(409, "M02_CURSOR_CHANGED", "查询位置已失效，请重新查询")
	}
	if len(token) > 4096 {
		return invalid()
	}
	dataText, signatureText, ok := strings.Cut(token, ".")
	if !ok {
		return invalid()
	}
	data, e := base64.RawURLEncoding.DecodeString(dataText)
	if e != nil {
		return invalid()
	}
	signature, e := base64.RawURLEncoding.DecodeString(signatureText)
	if e != nil {
		return invalid()
	}
	mac := hmac.New(sha256.New, cursorKey)
	_, _ = mac.Write(data)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return invalid()
	}
	var value queryCursor
	if json.Unmarshal(data, &value) != nil || value.Version != 1 || !foundation.ValidID(value.ID) {
		return invalid()
	}
	key, id = value.Key, value.ID
	value.Key = ""
	value.ID = ""
	binding.Version = 1
	if value != binding {
		return invalid()
	}
	return key, id, nil
}
