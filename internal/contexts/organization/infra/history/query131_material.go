//go:build integration && query131

package history

import (
	"encoding/hex"
	"encoding/json"

	"chnhcm.local/chnhcm/internal/platform/compact"
)

type diagnosticMaterial131 struct {
	capturedMaterial
	DictionaryDigest, ShapeDigest string
}

func query131Material(kind Kind, instance, enterprise, object, source, start, finish, envelope string, ids, material any) (compact.Values, []byte, error) {
	var m diagnosticMaterial131
	var directory []string
	raw, err := json.Marshal(material)
	if err != nil {
		return compact.Values{}, nil, err
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return compact.Values{}, nil, err
	}
	raw, err = json.Marshal(ids)
	if err != nil {
		return compact.Values{}, nil, err
	}
	if err = json.Unmarshal(raw, &directory); err != nil {
		return compact.Values{}, nil, err
	}
	return decodeQueryMaterial(kind, instance, enterprise, object, source, start, finish, envelope, directory, m)
}

func decodeQueryMaterial(kind Kind, instance, enterprise, object, source, start, finish, envelope string, ids []string, m diagnosticMaterial131) (compact.Values, []byte, error) {
	v := compact.Values{Strings: m.Words, IDs: ids}
	bad := func() (compact.Values, []byte, error) { return compact.Values{}, nil, compact.ErrCorrupt }
	for _, s := range m.Binary {
		b, e := hex.DecodeString(s)
		if e != nil {
			return bad()
		}
		v.Binary = append(v.Binary, b)
	}
	if len(m.Starts) != len(m.Ends) {
		return bad()
	}
	for i, s := range m.Starts {
		v.Ranges = append(v.Ranges, compact.Range{Start: s, End: m.Ends[i]})
	}
	parts := [][]byte{}
	data := []byte{}
	for _, s := range m.Parts {
		b, e := hex.DecodeString(s)
		if e != nil {
			return bad()
		}
		parts = append(parts, b)
		data = append(data, b...)
	}
	dh, sh := hashParts(dictionaryParts(v)), hashParts(parts)
	if hex.EncodeToString(dh) != m.DictionaryDigest || hex.EncodeToString(sh) != m.ShapeDigest {
		return bad()
	}
	seal, err := ReadSegmentDigest(kind, instance, enterprise, object, source, start, finish, v.IDs, dh, sh)
	if err != nil || hex.EncodeToString(seal) != envelope {
		return bad()
	}
	return v, data, nil
}
