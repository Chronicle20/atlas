package idasrc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type exportFn struct {
	Address string `json:"address"`
	// Direction is "clientbound" or "serverbound".
	Direction string `json:"direction"`
	// Dispatcher, when set, names a known dispatcher chain whose prefix
	// bytes are auto-prepended to the Calls list during Resolve. Supported
	// values:
	//
	//   "per-mob"         → Decode4 dwMobId (read by CMobPool::OnMobPacket
	//                       before dispatch to CMob::On*).
	//   "per-pet"         → Decode4 characterId + Decode1 slot (read by
	//                       CUserPool::OnUserRemotePacket and
	//                       CUser::OnPetPacket before dispatch to CPet::On*).
	//   "per-pet-remote"  → Decode4 characterId (read by
	//                       CUserPool::OnUserRemotePacket before dispatch to
	//                       CUserRemote::OnPetActivated). No slot, since
	//                       OnPetActivated does not go through OnPetPacket.
	//
	// When set, the JSON entry's own "calls" list MUST omit these prefix
	// bytes — they are added by the resolver. Unrecognized values are
	// ignored (no prefix added) — this is a forward-compat hook for new
	// dispatcher chains.
	Dispatcher string `json:"dispatcher,omitempty"`
	// Notes is free-form documentation that does not affect resolution.
	Notes string `json:"notes,omitempty"`
	Calls []rawCall `json:"calls"`
}

type rawCall struct {
	Op      string `json:"op"`
	Comment string `json:"comment"`
	Guard   string `json:"guard,omitempty"`
	// Ref names a sibling FName to inline at this position. Only consulted
	// when Op == "Delegate" (task-065 item 8 — sub-function descent). The
	// referenced FName's resolved Calls list (including its own
	// dispatcher prefix and recursive Delegates) is spliced into the
	// parent's Calls at this position, with the Delegate's Guard ANDed
	// onto each inlined Call.
	Ref string `json:"ref,omitempty"`
}

type exportFile struct {
	Binary      string              `json:"binary"`
	MD5         string              `json:"md5"`
	GeneratedAt string              `json:"generated_at"`
	Functions   map[string]exportFn `json:"functions"`
}

type ExportSource struct {
	file exportFile
}

func NewExportSource(path string) (*ExportSource, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f exportFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &ExportSource{file: f}, nil
}

// Functions returns all FNames in the export.
func (s *ExportSource) Functions() []string {
	out := make([]string, 0, len(s.file.Functions))
	for k := range s.file.Functions {
		out = append(out, k)
	}
	return out
}

func (s *ExportSource) Resolve(ctx context.Context, fname string) (Fields, error) {
	return s.resolveWithVisited(ctx, fname, map[string]bool{})
}

// resolveWithVisited is the workhorse that handles recursive Delegate descent.
// The shared `visited` set tracks FNames currently on the resolve stack so a
// cycle (A → B → A) terminates with an error rather than infinite recursion.
//
// We do NOT remove fname from visited on return — a diamond pattern (A → B,
// A → C, B → C, C unreachable from itself) is fine; the cycle detector only
// trips when the SAME fname appears twice on the active descent path.
func (s *ExportSource) resolveWithVisited(ctx context.Context, fname string, visited map[string]bool) (Fields, error) {
	if visited[fname] {
		return Fields{}, fmt.Errorf("idasrc: Delegate cycle through %q", fname)
	}
	raw, ok := s.file.Functions[fname]
	if !ok {
		return Fields{}, fmt.Errorf("idasrc: function %q not in export", fname)
	}
	visited[fname] = true
	defer delete(visited, fname)

	dir := DirClientbound
	if raw.Direction == "serverbound" {
		dir = DirServerbound
	}
	out := Fields{Function: fname, Address: raw.Address, Direction: dir}
	// Auto-prepend dispatcher prefix when annotated.
	out.Calls = append(out.Calls, dispatcherPrefix(raw.Dispatcher)...)
	for i, c := range raw.Calls {
		if c.Op == "Delegate" {
			if c.Ref == "" {
				return Fields{}, fmt.Errorf("call %d (%s): Delegate op requires ref", i, fname)
			}
			sub, err := s.resolveWithVisited(ctx, c.Ref, visited)
			if err != nil {
				return Fields{}, fmt.Errorf("call %d (%s): delegate to %q: %w", i, fname, c.Ref, err)
			}
			// Splice the sub's calls in at this position, AND-ing the
			// Delegate's own guard into each inlined call's guard.
			for _, sc := range sub.Calls {
				inlined := sc
				inlined.Guard = combineGuards(c.Guard, sc.Guard)
				out.Calls = append(out.Calls, inlined)
			}
			continue
		}
		op, err := parsePrim(c.Op)
		if err != nil {
			return Fields{}, fmt.Errorf("call %d (%s): %w", i, fname, err)
		}
		out.Calls = append(out.Calls, FieldCall{Op: op, Comment: c.Comment, Guard: c.Guard})
	}
	return out, nil
}

// combineGuards AND-s two free-form guard expressions, omitting empties so we
// don't generate "() && (x)" textual noise.
func combineGuards(outer, inner string) string {
	switch {
	case outer == "" && inner == "":
		return ""
	case outer == "":
		return inner
	case inner == "":
		return outer
	default:
		return "(" + outer + ") && (" + inner + ")"
	}
}

// dispatcherPrefix returns the FieldCalls that should be auto-prepended to a
// leaf op's wire layout for the named dispatcher chain. Returns nil for the
// empty kind (most entries) and for unrecognized kinds (forward-compat).
//
// The prefixes mirror the bytes that the in-game dispatcher reads before
// forwarding the remaining payload to the leaf handler:
//
//   per-mob         → CMobPool::OnMobPacket reads Decode4 mobId, then routes
//                     to CMob::On*.
//   per-pet         → CUserPool::OnUserRemotePacket reads Decode4 characterId,
//                     then CUser::OnPetPacket reads Decode1 slot, then routes
//                     to CPet::On*.
//   per-pet-remote  → CUserPool::OnUserRemotePacket reads Decode4 characterId,
//                     then routes to CUserRemote::OnPetActivated. The slot
//                     byte is part of the leaf payload here, not the prefix.
//
// Keep this list narrow and well-tested — adding a new dispatcher requires a
// matching test in export_test.go.
func dispatcherPrefix(kind string) []FieldCall {
	switch kind {
	case "":
		return nil
	case "per-mob":
		return []FieldCall{
			{Op: Decode4, Comment: "dwMobId — auto-prepended via dispatcher: per-mob (CMobPool::OnMobPacket)"},
		}
	case "per-pet":
		return []FieldCall{
			{Op: Decode4, Comment: "characterId — auto-prepended via dispatcher: per-pet (CUserPool::OnUserRemotePacket)"},
			{Op: Decode1, Comment: "slot — auto-prepended via dispatcher: per-pet (CUser::OnPetPacket)"},
		}
	case "per-pet-remote":
		return []FieldCall{
			{Op: Decode4, Comment: "characterId — auto-prepended via dispatcher: per-pet-remote (CUserPool::OnUserRemotePacket)"},
		}
	}
	return nil
}

func parsePrim(s string) (Primitive, error) {
	switch s {
	case "Decode1", "Encode1":
		return Decode1, nil
	case "Decode2", "Encode2":
		return Decode2, nil
	case "Decode4", "Encode4":
		return Decode4, nil
	case "Decode8", "Encode8":
		return Decode8, nil
	case "DecodeStr", "EncodeStr":
		return DecodeStr, nil
	case "DecodeBuffer", "EncodeBuffer", "DecodeBuf", "EncodeBuf":
		return DecodeBuf, nil
	}
	return 0, fmt.Errorf("unknown primitive %q", s)
}
