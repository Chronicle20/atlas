package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v95 ida=0x5de435
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v95 ida=0x5de17f
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v95 ida=0x5de284
func TestCharacterViewAllCountRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllCount{code: 3, worldCount: 5, unk: 0}
			output := CharacterViewAllCount{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldCount() != input.WorldCount() {
				t.Errorf("worldCount: got %v, want %v", output.WorldCount(), input.WorldCount())
			}
		})
	}
}

func TestCharacterViewAllCharactersRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			stats := model.NewCharacterStatistics(
				99, "ViewAllChar", 0, 2, 20000, 30000,
				[3]uint64{10, 20, 30},
				40, 100,
				30, 25, 20, 15,
				1000, 1000, 500, 500,
				3, false, 2,
				50000, 50, 1000,
				100000000, 0,
			)
			avatar := model.NewAvatar(0, 2, 20000, false, 30000, nil, nil, nil)
			// viewAll=true: no family byte; gm=false: rank fields are written
			entry := model.NewCharacterListEntry(stats, avatar, true, false, 5, 1, 3, 2)
			input := NewCharacterViewAllCharacters(0, world.Id(0), []model.CharacterListEntry{entry})
			output := CharacterViewAllCharacters{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if len(output.Characters()) != len(input.Characters()) {
				t.Errorf("characters len: got %v, want %v", len(output.Characters()), len(input.Characters()))
			}
		})
	}
}

func TestCharacterViewAllSearchFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllSearchFailed{code: 4}
			output := CharacterViewAllSearchFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}

// TestCharacterViewAllErrorByteOutput pins the CharacterViewAllError wire body.
// CLogin::OnViewAllCharResult is a server→client dispatcher keyed on a leading
// Decode1(mode/code). The error/notice modes resolve as:
//
//	v83 @0x5facca, v87 @0x6328eb, v95 @0x5de284 (case 2/3/6/7 block).
//	case 2 (RemoveNoticeConnecting+ResetVAC, StringPool 0xFBE): NO further reads
//	       — body is the single code byte. This is the path the code-only
//	       CharacterViewAllError struct models, identical in shape to its
//	       already-verified sibling CharacterViewAllSearchFailed.
//
// NOTE (coverage boundary, derived from the decompile, not papered over): the
// case 3/6/7 path at the report address additionally reads Decode1(hasMsg) and,
// if set, DecodeStr(msg) (v83 @0x5fadd5/0x5fade4; v95 @0x5de292/0x5de2a2).
// Atlas's code-only struct does NOT model that flag+string variant and Atlas
// never emits it (the struct is unused by services; the code byte is the
// dispatcher selector). The single-byte fixture below is the exact, faithful
// wire for the mode-2 error path the struct represents.
//
// NO packet-audit:verify marker is attached: the IDA exports
// (docs/packets/ida-exports/*.json) harvested a `#CharacterViewAllSearchFailed`
// slice at this address but NOT a distinct `#CharacterViewAllError` slice, so
// `evidence pin --ida CLogin::OnViewAllCharResult#CharacterViewAllError` is
// unresolvable (exit 3). A tier-1 cell cannot verify without fresh evidence;
// adding a marker-only would regress this sibling from partial→incomplete.
// This test stands as regression protection until the export is re-harvested.
func TestCharacterViewAllErrorByteOutput(t *testing.T) {
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	}{
		{"GMS v83", "GMS", 83, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.Major, v.Minor)
			got := pt.Encode(t, ctx, NewCharacterViewAllError(2).Encode, nil)
			// body = code byte (the dispatcher mode selector). No further reads
			// on the mode-2 path.
			if len(got) != 1 || got[0] != 2 {
				t.Errorf("%s: got % x, want [02]", v.Name, got)
			}
		})
	}
}

func TestCharacterViewAllErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllError{code: 5}
			output := CharacterViewAllError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
