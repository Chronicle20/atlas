package scopeguard

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed allowlist.txt
var allowlistRaw string

//go:embed callsite-allowlist.txt
var callsiteAllowlistRaw string

// EntityAllowlist maps an entity allowlist key (see entityAllowlistKey) to
// its written exemption reason (Rule 1). CallsiteAllowlist maps a
// callsiteKey (pkgPath/file:line) to its written exemption reason (Rule 2).
// Both are parsed once at package init from the checked-in .txt files — see
// parseAllowlist for the line format and the "reason is mandatory" rule.
//
// Exported (rather than the more usual unexported package var) solely so
// analyzer_test.go — in this same package, not an external _test package —
// can substitute a fixture allowlist for the "allowlisted" test case without
// touching the real checked-in allowlist.txt.
var (
	EntityAllowlist   map[string]string
	CallsiteAllowlist map[string]string
)

func init() {
	var err error
	EntityAllowlist, err = parseAllowlist(allowlistRaw)
	if err != nil {
		panic("scopeguard: allowlist.txt: " + err.Error())
	}
	CallsiteAllowlist, err = parseAllowlist(callsiteAllowlistRaw)
	if err != nil {
		panic("scopeguard: callsite-allowlist.txt: " + err.Error())
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
