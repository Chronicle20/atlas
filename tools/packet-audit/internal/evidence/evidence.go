// Package evidence is the structured per-(packet,version) evidence ledger
// replacing prose acceptance (task-085 design §6).
package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var validCategories = map[string]bool{
	"OPAQUE": true, "TRUNCATION": true, "REPRESENTATION": true,
	"OP-MODE-PREFIX": true, "LOOP-EXCLUSIVE-BRANCH": true,
	"VERSION-ABSENT": true, "TIER1-FIXTURE": true,
}

// IDACitation records the IDA function that was inspected when this evidence
// record was pinned.
type IDACitation struct {
	Function        string `yaml:"function"`
	Address         string `yaml:"address"`
	DecompileSHA256 string `yaml:"decompile_sha256"`
}

// Record is one evidence record file (task-085 design §6.1).
type Record struct {
	Packet    string      `yaml:"packet"`
	Direction string      `yaml:"direction"`
	Version   string      `yaml:"version"`
	Category  string      `yaml:"category"`
	IDA       IDACitation `yaml:"ida"`
	Verifies  []string    `yaml:"verifies,omitempty"`
	Notes     string      `yaml:"notes,omitempty"`
}

// LoadRecord reads and validates a Record from the given YAML file.
func LoadRecord(path string) (Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	return loadRecordBytes(raw, path)
}

// ParseRecord validates a marshaled YAML record (e.g. one just produced by
// yaml.Marshal) using the same rules as LoadRecord. name is used in error
// messages only (typically the intended file path).
func ParseRecord(raw []byte, name string) (Record, error) {
	return loadRecordBytes(raw, name)
}

func loadRecordBytes(raw []byte, path string) (Record, error) {
	var r Record
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return Record{}, fmt.Errorf("%s: %w", path, err)
	}
	if r.Packet == "" || r.Version == "" || r.IDA.Function == "" || r.IDA.Address == "" {
		return Record{}, fmt.Errorf("%s: packet/version/ida.function/ida.address are required", path)
	}
	if !validCategories[r.Category] {
		return Record{}, fmt.Errorf("%s: invalid category %q", path, r.Category)
	}
	if r.Direction != "clientbound" && r.Direction != "serverbound" {
		return Record{}, fmt.Errorf("%s: invalid direction %q", path, r.Direction)
	}
	return r, nil
}

// RecordPath is the canonical file location for a record:
// <dir>/<version>/<packet with / -> .>.yaml
func RecordPath(dir, version, packet string) string {
	return filepath.Join(dir, version, strings.ReplaceAll(packet, "/", ".")+".yaml")
}

// LoadDir loads every record under dir (one subdir per version).
func LoadDir(dir string) ([]Record, error) {
	var out []Record
	versions, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if !v.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(dir, v.Name()))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			r, err := LoadRecord(filepath.Join(dir, v.Name(), f.Name()))
			if err != nil {
				return nil, err
			}
			if r.Version != v.Name() {
				return nil, fmt.Errorf("%s: version %q does not match directory %q",
					f.Name(), r.Version, v.Name())
			}
			out = append(out, r)
		}
	}
	return out, nil
}
