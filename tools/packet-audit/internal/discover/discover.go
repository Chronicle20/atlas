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
