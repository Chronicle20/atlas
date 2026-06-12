// Package discover enumerates a version's operation universe from its IDA
// database (task-085 design §5.2): clientbound via the client packet
// dispatcher's switch, serverbound via send-op constant sites.
package discover

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// DispatchCase is one (opcode, handler) pair extracted from the dispatcher.
type DispatchCase struct {
	Opcode  int
	Handler string // demangled name or sub_XXXX
}

var (
	// caseRe matches a Hex-Rays case label; the capture group is the numeric
	// literal (hex or decimal); a trailing 'u' suffix is consumed by the `u?`.
	caseRe = regexp.MustCompile(`^\s*case\s+(0x[0-9a-fA-F]+|\d+)u?\s*:`)

	// callRe matches a C-style function call. It captures either a demangled
	// C++ name (including optional leading ~ for destructors) or a sub_XXXX
	// synthetic IDA name. The open-paren is required to avoid matching keywords.
	// Note: the regex intentionally stops at '<' (template bracket) to avoid
	// capturing TSingleton<...> as a truncated name — isNoise drops those.
	callRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*(?:::[~A-Za-z_][A-Za-z0-9_]*)?|sub_[0-9A-Fa-f]+)\s*\(`)
)

// ParseDispatch walks Hex-Rays text of the packet dispatcher and yields one
// DispatchCase per case label, binding pending (fallthrough) labels to the
// first call in the case body.
//
// Scoping rules (robustness against real Hex-Rays output):
//   - Brace depth is tracked per line. The depth at which the FIRST case label
//     appears is recorded as the dispatch depth; only case labels at exactly
//     that depth are treated as dispatch arms. Deeper case labels (nested
//     switch bodies) are silently ignored.
//   - goto/return on a line clears pending labels (like break) so that a tail
//     goto/return case does not leak its labels into the next real case.
//   - All call matches on a line are scanned (not just the first); the first
//     non-noise candidate with a plausible handler shape is chosen.
func ParseDispatch(text string) ([]DispatchCase, error) {
	var out []DispatchCase
	var pending []int
	depth := 0
	dispatchDepth := -1 // depth of the enclosing switch's case labels; -1 = unseen

	for _, line := range strings.Split(text, "\n") {
		// Count brace changes on this line BEFORE acting on case/call so that
		// a closing '}' correctly reduces depth before we check for case labels.
		openCount := strings.Count(line, "{")
		closeCount := strings.Count(line, "}")
		depth += openCount - closeCount

		if m := caseRe.FindStringSubmatch(line); m != nil {
			// Record the dispatch depth on the very first case label seen.
			if dispatchDepth == -1 {
				dispatchDepth = depth
			}
			// Only record labels at the dispatch depth; ignore nested ones.
			if depth == dispatchDepth {
				op, err := parseIntLabel(m[1])
				if err != nil {
					return nil, fmt.Errorf("bad case label %q: %w", m[1], err)
				}
				pending = append(pending, op)
				// a call on the same line as the label falls through to the
				// call-check below
			}
		}

		if len(pending) == 0 {
			continue
		}

		// Strip trailing C++ // comment before scanning calls and terminators so
		// that keywords like "goto" or "return" appearing only in comment text do
		// not falsely clear pending labels.
		codePart := stripLineComment(line)

		// Only bind calls when we are at the dispatch depth. Deeper depths mean
		// we are inside a nested block (e.g., a nested switch inside a case arm)
		// and calls there must not be bound to the enclosing pending labels.
		if depth == dispatchDepth {
			if matches := callRe.FindAllStringSubmatch(codePart, -1); matches != nil {
				for _, m := range matches {
					name := m[1]
					if isNoise(name) {
						continue
					}
					if !isPlausibleHandler(name) {
						continue
					}
					for _, op := range pending {
						out = append(out, DispatchCase{Opcode: op, Handler: name})
					}
					pending = nil
					break
				}
			}
		}

		// break / goto / return all end a case body at the dispatch depth; only
		// check the code portion of the line to avoid false triggers from comment
		// text, and only at the dispatch depth so that break/goto/return inside
		// nested switch arms do not clear the enclosing pending labels.
		if depth == dispatchDepth &&
			(strings.Contains(codePart, "break;") ||
				strings.Contains(codePart, "goto ") ||
				strings.Contains(codePart, "return")) {
			pending = nil
		}
	}
	return out, nil
}

// stripLineComment returns the portion of a Hex-Rays source line before any
// C++ // comment, trimming trailing whitespace. This prevents keywords like
// "goto" or "return" that appear only in comment text from falsely triggering
// end-of-case logic.
func stripLineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return strings.TrimRight(line[:idx], " \t")
	}
	return line
}

// isPlausibleHandler returns true if name looks like a real handler:
// either a class method (contains "::") or an unnamed IDA stub ("sub_" prefix).
// Bare single-word names (alloca, void, new, etc.) are rejected here even if
// isNoise did not catch them explicitly.
func isPlausibleHandler(name string) bool {
	if strings.HasPrefix(name, "sub_") {
		return true
	}
	return strings.Contains(name, "::")
}

func parseIntLabel(s string) (int, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}

// isNoise filters non-handler calls that appear inside dispatch arms.
// It is called for every candidate match; isPlausibleHandler provides a
// second, shape-based gate so isNoise only needs to cover name-based cases.
func isNoise(name string) bool {
	switch name {
	case "memset", "memcpy", "operator", "if", "while", "switch",
		"alloca", "new", "void":
		return true
	}
	// CInPacket:: / COutPacket:: are packet I/O helpers, never dispatch targets.
	if strings.HasPrefix(name, "CInPacket::") || strings.HasPrefix(name, "COutPacket::") {
		return true
	}
	// Singleton accessors (e.g. TSingleton truncated before '<' to just the class name).
	if strings.HasSuffix(name, "::GetInstance") {
		return true
	}
	return false
}

// Discovered is one (opcode, handler, address) result from IDA dispatch-walk.
type Discovered struct {
	Opcode  int
	Handler string
	Address string // hex string, e.g. "0x5e1230"
}

// Collision records that the registry and IDA disagree on the handler for the
// same opcode.
type Collision struct {
	Entry      opregistry.Entry
	Discovered Discovered
}

// ReconcileResult is the output of Reconcile.
type ReconcileResult struct {
	Append             []opregistry.Entry // new ops, provenance ida-discovered
	MissingAtDiscovery []opregistry.Entry // in registry, not found in IDB — review worklist
	Collisions         []Collision        // same opcode, different handler — review worklist
}

// Reconcile compares a seeded VersionFile against discovery output for one
// direction:
//   - discovered op not in registry → append with provenance ida-discovered,
//   - registry entry not found by discovery → flag for review (never auto-deleted),
//   - discovered opcode colliding with a different existing entry → Collisions.
func Reconcile(vf *opregistry.VersionFile, discovered []Discovered, dir opregistry.Direction) ReconcileResult {
	var res ReconcileResult
	byOpcode := map[int]opregistry.Entry{}
	for _, e := range vf.Entries {
		if e.Direction == dir {
			byOpcode[e.Opcode] = e
		}
	}
	seenOpcode := map[int]bool{}
	for _, d := range discovered {
		seenOpcode[d.Opcode] = true
		if e, ok := byOpcode[d.Opcode]; ok {
			if e.FName != d.Handler && !hasAlt(e, d.Handler) && d.Handler != "" {
				res.Collisions = append(res.Collisions, Collision{Entry: e, Discovered: d})
			}
			continue
		}
		addr := parseAddr(d.Address)
		res.Append = append(res.Append, opregistry.Entry{
			Op:         opNameFor(d),
			Direction:  dir,
			Opcode:     d.Opcode,
			FName:      d.Handler,
			Provenance: "ida-discovered",
			IDA:        &opregistry.IDARef{Address: addr},
		})
	}
	for _, e := range vf.Entries {
		if e.Direction == dir && !seenOpcode[e.Opcode] {
			res.MissingAtDiscovery = append(res.MissingAtDiscovery, e)
		}
	}
	return res
}

// hasAlt returns true if the entry's FNameAlts list includes name.
func hasAlt(e opregistry.Entry, name string) bool {
	for _, a := range e.FNameAlts {
		if a == name {
			return true
		}
	}
	return false
}

// parseAddr converts a hex-string address ("0x5e1230") to uint64.
// Returns 0 for empty or unparseable input.
func parseAddr(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "0x") {
		v, err := strconv.ParseUint(low[2:], 16, 64)
		if err != nil {
			return 0
		}
		return v
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// opNameFor derives the placeholder op name for a newly discovered opcode.
// The format is IDA_0X<HEX> (e.g. IDA_0X002 for opcode 2).
// Canonical renames are human edits with provenance: manual (design D9).
func opNameFor(d Discovered) string {
	return fmt.Sprintf("IDA_0X%03X", d.Opcode)
}
