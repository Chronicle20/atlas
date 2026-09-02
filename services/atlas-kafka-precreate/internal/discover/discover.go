// Package discover holds the pure half of the precreate tool: the
// environment scrape that decides which topics exist, and the two
// classification rules the seeding pass keys off. Nothing here performs
// I/O or touches a Kafka type, which is what lets the whole of it be
// table-tested without a broker (design §5.2).
package discover

import (
	"sort"
	"strings"
)

// Topics is the outcome of scraping the process environment for topic-shaped
// variables, split by the cleanup policy their topic must be created with.
type Topics struct {
	Plain   []string // sorted, de-duplicated, never nil
	Compact []string // sorted, de-duplicated, never nil
}

// Union returns the sorted, de-duplicated merge of Plain and Compact.
func (t Topics) Union() []string {
	set := make(map[string]struct{}, len(t.Plain)+len(t.Compact))
	for _, name := range t.Plain {
		set[name] = struct{}{}
	}
	for _, name := range t.Compact {
		set[name] = struct{}{}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Groups parses the newline-delimited consumer group list arrived at from a
// YAML block scalar. Only empty lines are dropped; interior whitespace is
// never trimmed, and group order is preserved as input order.
func Groups(raw string) []string {
	lines := strings.Split(raw, "\n")
	groups := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		groups = append(groups, line)
	}
	return groups
}

// StateIsSeedable reports whether a consumer group in the given state is
// safe to seed offsets for. This is a deliberate allowlist, not a denylist:
// a state a future Kafka version introduces falls into "active" and is
// skipped, and skipping never mutates a committed offset (FR-3.4, NFR-6).
func StateIsSeedable(state string) bool {
	switch state {
	case "Empty", "Dead", "":
		return true
	default:
		return false
	}
}
