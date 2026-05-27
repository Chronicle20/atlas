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
	Calls []struct {
		Op      string `json:"op"`
		Comment string `json:"comment"`
		Guard   string `json:"guard,omitempty"`
	} `json:"calls"`
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

func (s *ExportSource) Resolve(_ context.Context, fname string) (Fields, error) {
	raw, ok := s.file.Functions[fname]
	if !ok {
		return Fields{}, fmt.Errorf("idasrc: function %q not in export", fname)
	}
	dir := DirClientbound
	if raw.Direction == "serverbound" {
		dir = DirServerbound
	}
	out := Fields{Function: fname, Address: raw.Address, Direction: dir}
	// Auto-prepend dispatcher prefix when annotated.
	out.Calls = append(out.Calls, dispatcherPrefix(raw.Dispatcher)...)
	for i, c := range raw.Calls {
		op, err := parsePrim(c.Op)
		if err != nil {
			return Fields{}, fmt.Errorf("call %d (%s): %w", i, fname, err)
		}
		out.Calls = append(out.Calls, FieldCall{Op: op, Comment: c.Comment, Guard: c.Guard})
	}
	return out, nil
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
