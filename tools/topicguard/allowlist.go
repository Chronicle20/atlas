package topicguard

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed allowlist.txt
var allowlistRaw string

// RawEnvAllowlist maps an environment variable name that lexically matches
// rawEnvTopicPattern to its written exemption reason — diagnostic 2's
// checkRawEnvTopicRead consults this before reporting. Parsed once at
// package init from the checked-in allowlist.txt; see parseAllowlist for the
// line format and the "reason is mandatory" rule.
//
// Exported (rather than the more usual unexported package var) solely so
// analyzer_test.go can assert against it — mirrors
// tools/scopeguard/allowlist.go's EntityAllowlist shape.
var RawEnvAllowlist map[string]string

func init() {
	var err error
	RawEnvAllowlist, err = parseAllowlist(allowlistRaw)
	if err != nil {
		panic("topicguard: allowlist.txt: " + err.Error())
	}
}

// parseAllowlist parses lines shaped "<key> # <reason>". Blank lines and
// lines whose first non-space character is `#` are comments, skipped
// entirely. Every other line must carry a `#`-delimited reason with
// non-whitespace content — an entry with no reason is a lint failure of the
// allowlist file itself (asserted by TestAllowlistEntriesHaveReasons).
func parseAllowlist(raw string) (map[string]string, error) {
	out := map[string]string{}
	for i, line := range strings.Split(raw, "\n") {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(line, "#")
		if idx < 0 {
			return nil, fmt.Errorf("line %d: %q has no reason (expected \"<key> # <reason>\")", lineNo, trimmed)
		}
		key := strings.TrimSpace(line[:idx])
		reason := strings.TrimSpace(line[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key before '#'", lineNo)
		}
		if reason == "" {
			return nil, fmt.Errorf("line %d: entry %q has no reason after '#'", lineNo, key)
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("line %d: duplicate key %q", lineNo, key)
		}
		out[key] = reason
	}
	return out, nil
}
