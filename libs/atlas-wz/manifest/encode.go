package manifest

import (
	"bytes"
	"encoding/json"
	"sort"
)

// Marshal serializes m with deterministic key ordering inside any nested map.
// Go's json package iterates struct fields in declaration order (deterministic)
// but iterates maps in randomized order; this wrapper canonicalizes both.
func Marshal(m Manifest) ([]byte, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var canon any
	if err := json.Unmarshal(raw, &canon); err != nil {
		return nil, err
	}
	return marshalSorted(canon)
}

func Unmarshal(b []byte) (Manifest, error) {
	var m Manifest
	err := json.Unmarshal(b, &m)
	return m, err
}

func marshalSorted(v any) ([]byte, error) {
	switch tv := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(tv))
		for k := range tv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			vb, err := marshalSorted(tv[k])
			if err != nil {
				return nil, err
			}
			buf.Write(vb)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, e := range tv {
			if i > 0 {
				buf.WriteByte(',')
			}
			eb, err := marshalSorted(e)
			if err != nil {
				return nil, err
			}
			buf.Write(eb)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	default:
		return json.Marshal(v)
	}
}
