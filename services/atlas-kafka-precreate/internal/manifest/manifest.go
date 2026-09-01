// Package manifest reads the mounted topic manifest (topics.yaml, rendered
// by libs/atlas-kafka/gen from the repo-wide topic scan) and resolves it
// against the process environment into the same discover.Topics shape the
// old environment scrape produced. Nothing here performs Kafka I/O.
package manifest

import (
	"fmt"
	"os"
	"sort"

	"atlas.com/kafka-precreate/internal/discover"
	"gopkg.in/yaml.v3"
)

// Entry is a single topic manifest row: the environment variable token that
// resolves to the concrete topic name, and the cleanup policy the topic
// must be created with.
type Entry struct {
	Token   string `yaml:"token"`
	Cleanup string `yaml:"cleanup"`
}

// Manifest is the parsed shape of topics.yaml.
type Manifest struct {
	Topics []Entry `yaml:"topics"`
}

const (
	cleanupDelete  = "delete"
	cleanupCompact = "compact"
)

// Parse decodes a topic manifest document and validates every entry's
// cleanup policy is one this package understands.
func Parse(data []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parsing topic manifest: %w", err)
	}

	if len(m.Topics) == 0 {
		return Manifest{}, fmt.Errorf("topic manifest is empty")
	}

	for _, entry := range m.Topics {
		switch entry.Cleanup {
		case cleanupDelete, cleanupCompact:
			// ok
		default:
			return Manifest{}, fmt.Errorf("topic manifest entry %q has unknown cleanup policy %q", entry.Token, entry.Cleanup)
		}
	}

	return m, nil
}

// Resolve walks m.Topics in sorted token order, looking each token's
// concrete topic name up via look, and classifies the result into
// discover.Topics' Plain/Compact split. A token that look cannot resolve to
// a non-empty value is fatal: the manifest names an environment variable
// that must be set, and precreate cannot proceed without it.
func Resolve(m Manifest, look func(string) (string, bool)) (discover.Topics, error) {
	entries := make([]Entry, len(m.Topics))
	copy(entries, m.Topics)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Token < entries[j].Token
	})

	plain := make(map[string]struct{})
	compact := make(map[string]struct{})

	for _, entry := range entries {
		value, ok := look(entry.Token)
		if !ok || value == "" {
			return discover.Topics{}, fmt.Errorf("topic manifest token [%s] has no value in the environment", entry.Token)
		}

		if entry.Cleanup == cleanupCompact {
			compact[value] = struct{}{}
			continue
		}
		plain[value] = struct{}{}
	}

	for name := range compact {
		delete(plain, name)
	}

	return discover.Topics{
		Plain:   sortedKeys(plain),
		Compact: sortedKeys(compact),
	}, nil
}

// Load reads and parses the topic manifest at path.
func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading topic manifest %s: %w", path, err)
	}
	return Parse(data)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
