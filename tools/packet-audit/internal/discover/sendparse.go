package discover

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// coutCtorRe matches a COutPacket constructor call and captures the opcode
// literal (2nd argument).  Only integer literals (decimal or hex, with an
// optional trailing 'u') are captured; a variable or expression 2nd arg does
// NOT match and is silently skipped — it cannot be statically verified.
//
// Pattern breakdown:
//
//	COutPacket::COutPacket   — demangled constructor name
//	\s*\(\s*                 — open paren with optional whitespace
//	[^,()]+                  — first argument (any non-comma, non-paren chars)
//	,\s*                     — comma separator
//	(0x[0-9A-Fa-f]+|\d+)     — capture group: hex or decimal literal
//	u?                       — optional unsigned suffix
//	\s*\)                    — close paren
var coutCtorRe = regexp.MustCompile(
	`COutPacket::COutPacket\s*\(\s*[^,()]+,\s*(0x[0-9A-Fa-f]+|\d+)u?\s*\)`,
)

// ParseSendOpcodes returns the set of opcodes passed as the 2nd argument to
// COutPacket constructor calls in a decompiled send function, sorted ascending,
// deduped.  Only integer-literal opcodes (decimal, hex, optional 'u' suffix)
// are returned; a variable/expression 2nd arg is skipped (cannot be statically
// verified).
func ParseSendOpcodes(text string) []int {
	seen := map[int]struct{}{}
	for _, m := range coutCtorRe.FindAllStringSubmatch(text, -1) {
		lit := m[1]
		var val int64
		var err error
		if strings.HasPrefix(lit, "0x") || strings.HasPrefix(lit, "0X") {
			val, err = strconv.ParseInt(lit[2:], 16, 32)
		} else {
			val, err = strconv.ParseInt(lit, 10, 32)
		}
		if err != nil {
			continue
		}
		seen[int(val)] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}
