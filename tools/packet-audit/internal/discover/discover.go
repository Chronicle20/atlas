// Package discover enumerates a version's operation universe from its IDA
// database (task-085 design §5.2): clientbound via the client packet
// dispatcher's switch, serverbound via send-op constant sites.
package discover

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	callRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*(?:::[~A-Za-z_][A-Za-z0-9_]*)?|sub_[0-9A-Fa-f]+)\s*\(`)
)

// ParseDispatch walks Hex-Rays text of the packet dispatcher and yields one
// DispatchCase per case label, binding pending (fallthrough) labels to the
// first call in the case body.
func ParseDispatch(text string) ([]DispatchCase, error) {
	var out []DispatchCase
	var pending []int
	for _, line := range strings.Split(text, "\n") {
		if m := caseRe.FindStringSubmatch(line); m != nil {
			op, err := parseIntLabel(m[1])
			if err != nil {
				return nil, fmt.Errorf("bad case label %q: %w", m[1], err)
			}
			pending = append(pending, op)
			// a call on the same line as the label binds immediately (fall through
			// to the call-check below)
		}
		if len(pending) == 0 {
			continue
		}
		if m := callRe.FindStringSubmatch(line); m != nil && !isNoise(m[1]) {
			for _, op := range pending {
				out = append(out, DispatchCase{Opcode: op, Handler: m[1]})
			}
			pending = nil
		}
		if strings.Contains(line, "break;") {
			pending = nil // empty case body: no handler discovered
		}
	}
	return out, nil
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
func isNoise(name string) bool {
	switch name {
	case "memset", "memcpy", "operator", "if", "while", "switch":
		return true
	}
	return strings.HasPrefix(name, "CInPacket::")
}
