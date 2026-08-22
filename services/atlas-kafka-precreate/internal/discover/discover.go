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

const (
	commandPrefix = "COMMAND_TOPIC_"
	eventPrefix   = "EVENT_TOPIC_"
)

// compactVars names the three config-projection variables whose topics must
// carry cleanup.policy=compact. Their consumers replay from first-offset at
// every boot to rebuild tenant/service config state and the outbox never
// re-emits a (topic, key) it already delivered, so under the default DELETE
// cleanup retention empties the topic ~7 days after the last config change
// and every later projection boot has nothing to replay. Events are keyed,
// so compaction retains the latest snapshot per key forever.
var compactVars = map[string]struct{}{
	"EVENT_TOPIC_CONFIGURATION_TENANT_STATUS":      {},
	"EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS":     {},
	"EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS": {},
}

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

// FromEnviron scrapes environ (entries shaped "KEY=VALUE", as returned by
// os.Environ) for COMMAND_TOPIC_* and EVENT_TOPIC_* variables and classifies
// each resulting topic name as plain or compacted.
func FromEnviron(environ []string) Topics {
	plain := make(map[string]struct{})
	compact := make(map[string]struct{})

	for _, entry := range environ {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if value == "" {
			continue
		}

		hasCommandPrefix := strings.HasPrefix(name, commandPrefix)
		hasEventPrefix := strings.HasPrefix(name, eventPrefix)
		if !hasCommandPrefix && !hasEventPrefix {
			continue
		}

		if _, ok := compactVars[name]; ok {
			compact[value] = struct{}{}
			continue
		}
		plain[value] = struct{}{}
	}

	for name := range compact {
		delete(plain, name)
	}

	return Topics{
		Plain:   sortedKeys(plain),
		Compact: sortedKeys(compact),
	}
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
