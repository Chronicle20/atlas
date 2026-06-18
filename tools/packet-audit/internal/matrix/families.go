package matrix

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Families holds the mode-prefix dispatcher membership from families.yaml.
// Each entry is a base IDA function name (no #case suffix) whose body is a
// `switch(Decode<mode>)` demultiplexer: one opcode, a leading mode byte that
// selects among many sub-handlers with distinct bodies. An op whose registry
// FName is one of these can never reach ✅ on a single sub-handler's fixture —
// it is capped at StateFamily (see grade.go) until every mode arm is covered.
type Families struct {
	Dispatchers []string `yaml:"dispatchers"`
}

// LoadFamilies reads and parses families.yaml at path. A missing file is not an
// error — it returns empty Families (no op is capped), so the matrix still runs
// in trees that predate the file.
func LoadFamilies(path string) (Families, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Families{}, nil
	}
	if err != nil {
		return Families{}, err
	}
	var f Families
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return Families{}, err
	}
	return f, nil
}

// Set returns the dispatcher membership as a lookup map keyed by base FName,
// suitable for matrix.Inputs.Families.
func (f Families) Set() map[string]bool {
	m := make(map[string]bool, len(f.Dispatchers))
	for _, d := range f.Dispatchers {
		if d != "" {
			m[d] = true
		}
	}
	return m
}
