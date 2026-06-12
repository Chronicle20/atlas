package idasrc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

type allowEntry struct {
	FName  string `json:"fname"`
	Case   int64  `json:"case"`
	Reason string `json:"reason"`
}

// Allowlist records intentionally-unimplemented client dispatch cases so the
// bijection check counts but does not flag them as missing-mode.
type Allowlist struct {
	set map[string]map[int64]bool
}

// LoadAllowlist reads a per-version _unimplemented.json. A missing file yields an
// empty (suppress-nothing) allowlist with no error, so the path can be passed
// unconditionally.
func LoadAllowlist(path string) (*Allowlist, error) {
	al := &Allowlist{set: map[string]map[int64]bool{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return al, nil
		}
		return nil, err
	}
	var doc struct {
		Entries []allowEntry `json:"entries"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	for _, e := range doc.Entries {
		if al.set[e.FName] == nil {
			al.set[e.FName] = map[int64]bool{}
		}
		al.set[e.FName][e.Case] = true
	}
	return al, nil
}

// Suppressed reports whether (fname, case) is an allowlisted unimplemented case.
func (a *Allowlist) Suppressed(fname string, c int64) bool {
	return a.set[fname] != nil && a.set[fname][c]
}
