package idasrc

import (
	"strconv"
	"strings"
)

// DefaultGuardToken is the guard text the parser stamps on a switch `default:` /
// trailing `else` arm's reads. A Selector{Default:true} matches exactly these,
// and a normal case Selector never matches them.
const DefaultGuardToken = "<default>"

// Selector picks a single dispatch path through a switch handler by matching a
// case label against the parser-emitted guard. Discriminator "" matches any
// discriminator (only the case value must match). Default selects the
// default/else arm (reads carrying DefaultGuardToken) instead of a case value.
type Selector struct {
	Discriminator string `json:"discriminator,omitempty"` // "" matches any discriminator
	Case          int64  `json:"case"`
	Default       bool   `json:"default,omitempty"` // matches the default/else arm
	// Guard, when set, matches a read whose composed guard contains this exact
	// branch-condition clause (a non-equality dispatch arm, e.g. "v5 < 5").
	Guard string `json:"guard,omitempty"`
}

// ExtractShape returns the per-dispatch-path wire-shape reads from a resolved
// function: the pre-branch reads (empty guard, occurring before the first matched
// read — the discriminator + common header) followed by the reads whose composed
// guard satisfies every selector, in source order. Empty dispatch returns all calls.
func ExtractShape(f Fields, dispatch []Selector) []FieldCall {
	if len(dispatch) == 0 {
		return append([]FieldCall(nil), f.Calls...)
	}
	firstMatch := -1
	for i := range f.Calls {
		if guardSatisfies(f.Calls[i].Guard, dispatch) {
			firstMatch = i
			break
		}
	}
	var out []FieldCall
	if firstMatch >= 0 {
		for i := 0; i < firstMatch; i++ {
			if f.Calls[i].Guard == "" {
				out = append(out, f.Calls[i])
			}
		}
	}
	for i := range f.Calls {
		if guardSatisfies(f.Calls[i].Guard, dispatch) {
			out = append(out, f.Calls[i])
		}
	}
	return out
}

// guardSatisfies reports whether the composed guard satisfies every selector:
// for each selector there must exist a clause "X == V" in the guard where the
// left side equals the selector's discriminator (or the discriminator is "")
// and V parses to the selector's case value.
func guardSatisfies(guard string, dispatch []Selector) bool {
	for _, sel := range dispatch {
		if !clauseMatches(guard, sel) {
			return false
		}
	}
	return true
}

func clauseMatches(guard string, sel Selector) bool {
	if sel.Default {
		// A default selector matches iff the guard is exactly the default token.
		return strings.TrimSpace(guard) == DefaultGuardToken
	}
	if strings.TrimSpace(guard) == DefaultGuardToken {
		// A normal case selector never matches a default-arm read.
		return false
	}
	if sel.Guard != "" {
		// Verbatim clause match: some &&-clause equals sel.Guard exactly.
		for _, clause := range strings.Split(guard, "&&") {
			clause = strings.TrimSpace(clause)
			clause = strings.TrimPrefix(clause, "(")
			clause = strings.TrimSuffix(clause, ")")
			if strings.TrimSpace(clause) == sel.Guard {
				return true
			}
		}
		return false
	}
	for _, clause := range strings.Split(guard, "&&") {
		clause = strings.TrimSpace(clause)
		clause = strings.TrimPrefix(clause, "(")
		clause = strings.TrimSuffix(clause, ")")
		clause = strings.TrimSpace(clause)
		parts := strings.SplitN(clause, "==", 2)
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		val, ok := parseIntLit(strings.TrimSpace(parts[1]))
		if !ok {
			continue
		}
		if sel.Discriminator != "" && left != sel.Discriminator {
			continue
		}
		if val == sel.Case {
			return true
		}
	}
	return false
}

// parseIntLit parses a decimal or 0x/0X hex integer literal. Non-integer values
// (e.g. "loop n") return ok=false so they never match a case selector.
// C integer suffixes (u, U, l, L, and combinations such as ul, UL, ll, LL)
// are stripped before parsing so that guard values emitted by the upgraded IDA
// decompiler (e.g. "9u", "0xAu") are handled correctly. This is defensive:
// parse.go's reCase already strips the suffix from emitted guards, but a
// guard stored with a suffix (e.g. from an older export or a test fixture that
// embeds the suffix directly) will still parse correctly here.
func parseIntLit(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	// Strip a trailing run of C integer suffix characters [uUlL].
	i := len(s)
	for i > 0 && (s[i-1] == 'u' || s[i-1] == 'U' || s[i-1] == 'l' || s[i-1] == 'L') {
		i--
	}
	s = s[:i]
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
