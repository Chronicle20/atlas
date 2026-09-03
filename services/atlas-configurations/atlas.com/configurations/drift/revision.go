package drift

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// hash is the one hashing primitive: lowercase hex SHA-256 over bytes.
func hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sectionDoc returns the sub-document a section name selects. A Named
// section is the single key of that name (absent -> an empty Doc, so a
// missing section still has a stable hash). Properties is every
// comparable key not in Named.
func sectionDoc(d Doc, section string) Doc {
	out := Doc{}
	if section == Properties {
		for k, v := range d {
			if !isNamed(k) {
				out[k] = v
			}
		}
		return out
	}
	if v, ok := d[section]; ok {
		out[section] = v
	}
	return out
}

func isNamed(k string) bool {
	for _, n := range Named {
		if n == k {
			return true
		}
	}
	return false
}

// All returns every section name in report order: Properties first, then
// Named. The result is always the full set, so a caller never has to
// distinguish an absent key from a false one.
func All() []string {
	out := make([]string, 0, len(Named)+1)
	out = append(out, Properties)
	out = append(out, Named...)
	return out
}

// Sections returns the per-section hex SHA-256 of d. The map is ALWAYS
// fully populated -- every name All() returns is present, even when the
// section is absent from d.
func Sections(d Doc) (map[string]string, error) {
	out := make(map[string]string, len(Named)+1)
	for _, name := range All() {
		b, err := json.Marshal(sectionDoc(d, name))
		if err != nil {
			return nil, err
		}
		out[name] = hash(b)
	}
	return out, nil
}

// Aggregate returns one hex SHA-256 over the whole comparable document.
// NOT comparable with templates.Revision -- see the package doc.
func Aggregate(d Doc) (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return hash(b), nil
}

// Compare reports whether stored diverges from base, in aggregate and per
// section. per is always fully populated.
func Compare(base, stored Doc) (bool, map[string]bool, error) {
	ba, err := Aggregate(base)
	if err != nil {
		return false, nil, err
	}
	sa, err := Aggregate(stored)
	if err != nil {
		return false, nil, err
	}
	bs, err := Sections(base)
	if err != nil {
		return false, nil, err
	}
	ss, err := Sections(stored)
	if err != nil {
		return false, nil, err
	}

	per := make(map[string]bool, len(bs))
	for _, name := range All() {
		per[name] = bs[name] != ss[name]
	}
	return ba != sa, per, nil
}
