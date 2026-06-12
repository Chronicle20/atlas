// Package opregistry owns the per-version operation universe: which packet
// operations exist in which client version, with provenance (design §5.1).
package opregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Direction string

const (
	DirClientbound Direction = "clientbound"
	DirServerbound Direction = "serverbound"
)

// Applicability of an (op, direction) in a version.
type Applicability int

const (
	Unknown Applicability = iota // no registry file for the version
	Absent                       // file exists, op not listed
	Present                      // op listed
)

type IDARef struct {
	Address uint64 `yaml:"address"`
}

type Entry struct {
	Op         string    `yaml:"op"`
	Direction  Direction `yaml:"direction"`
	Opcode     int       `yaml:"opcode"`
	FName      string    `yaml:"fname"`
	FNameAlts  []string  `yaml:"fname_alts,omitempty"`
	Provenance string    `yaml:"provenance"` // csv-import | ida-discovered | manual
	IDA        *IDARef   `yaml:"ida,omitempty"`
	Note       string    `yaml:"note,omitempty"`
}

type VersionFile struct {
	Entries []Entry
	byKey   map[string]Entry // "op|direction"
}

func key(op string, dir Direction) string { return op + "|" + string(dir) }

func validProvenance(p string) bool {
	return p == "csv-import" || p == "ida-discovered" || p == "manual"
}

// LoadVersion reads one registry YAML and validates schema + uniqueness.
func LoadVersion(path string) (*VersionFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	vf := &VersionFile{Entries: entries, byKey: make(map[string]Entry, len(entries))}
	for i, e := range entries {
		if e.Op == "" || (e.Direction != DirClientbound && e.Direction != DirServerbound) {
			return nil, fmt.Errorf("%s entry %d: missing op or invalid direction %q", path, i, e.Direction)
		}
		if !validProvenance(e.Provenance) {
			return nil, fmt.Errorf("%s entry %d (%s): invalid provenance %q", path, i, e.Op, e.Provenance)
		}
		if e.Provenance == "ida-discovered" && e.IDA == nil {
			return nil, fmt.Errorf("%s entry %d (%s): ida-discovered without ida.address", path, i, e.Op)
		}
		k := key(e.Op, e.Direction)
		if _, dup := vf.byKey[k]; dup {
			return nil, fmt.Errorf("%s: duplicate (op,direction) %s/%s", path, e.Op, e.Direction)
		}
		vf.byKey[k] = e
	}
	return vf, nil
}

func (v *VersionFile) Lookup(op string, dir Direction) (Entry, bool) {
	e, ok := v.byKey[key(op, dir)]
	return e, ok
}

// ByFName returns the entry whose fname (or any fname_alt) matches.
func (v *VersionFile) ByFName(fname string, dir Direction) (Entry, bool) {
	for _, e := range v.Entries {
		if e.Direction != dir {
			continue
		}
		if e.FName == fname {
			return e, true
		}
		for _, a := range e.FNameAlts {
			if a == fname {
				return e, true
			}
		}
	}
	return Entry{}, false
}

// Registry is the loaded set of version files.
type Registry struct {
	Versions map[string]*VersionFile // version key -> file (nil entry means file missing)
}

// LoadDir loads every <version>.yaml present in dir for the given version keys.
// Missing files are recorded as absent from the map (Applicability => Unknown).
func LoadDir(dir string, versionKeys []string) (Registry, error) {
	r := Registry{Versions: make(map[string]*VersionFile)}
	for _, vk := range versionKeys {
		p := filepath.Join(dir, vk+".yaml")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			continue
		}
		vf, err := LoadVersion(p)
		if err != nil {
			return Registry{}, err
		}
		r.Versions[vk] = vf
	}
	return r, nil
}

func (r Registry) Applicability(op string, dir Direction, versionKey string) Applicability {
	vf, ok := r.Versions[versionKey]
	if !ok {
		return Unknown
	}
	if _, present := vf.Lookup(op, dir); present {
		return Present
	}
	return Absent
}

// AllOps returns the union of (op, direction) across all loaded versions,
// sorted for deterministic iteration: clientbound first, then by op name.
func (r Registry) AllOps() []struct {
	Op  string
	Dir Direction
} {
	seen := map[string]struct {
		Op  string
		Dir Direction
	}{}
	for _, vf := range r.Versions {
		for _, e := range vf.Entries {
			seen[key(e.Op, e.Direction)] = struct {
				Op  string
				Dir Direction
			}{e.Op, e.Direction}
		}
	}
	out := make([]struct {
		Op  string
		Dir Direction
	}, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir == DirClientbound
		}
		return out[i].Op < out[j].Op
	})
	return out
}
