package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestSkillPrepareForeignRoundTrip exercises Encode/Decode symmetry for
// CharacterSkillPrepareForeign across all tenant variants. Wire-spec §3:
// charId u32, skillId u32, level u8, action u16, actionSpeed u8. Field order
// is identical across all five versions.
func TestSkillPrepareForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterSkillPrepareForeign(1001, 3121004, 10, 0x0142, 4)
			output := CharacterSkillPrepareForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
			if output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.Action() != input.Action() {
				t.Errorf("action: got %v, want %v", output.Action(), input.Action())
			}
			if output.ActionSpeed() != input.ActionSpeed() {
				t.Errorf("actionSpeed: got %v, want %v", output.ActionSpeed(), input.ActionSpeed())
			}
		})
	}
}

// TestSkillPrepareForeignOperation verifies Operation() returns the foreign writer
// const (not the bug pattern where foreign structs return the non-foreign const).
func TestSkillPrepareForeignOperation(t *testing.T) {
	m := NewCharacterSkillPrepareForeign(1, 3121004, 1, 0x0001, 1)
	if got := m.Operation(); got != CharacterSkillPrepareForeignWriter {
		t.Errorf("Operation() = %q, want %q", got, CharacterSkillPrepareForeignWriter)
	}
}

// TestSkillPrepareForeignByteFixture asserts the exact encoded bytes match
// wire-spec §3 field order: charId u32 LE, skillId u32 LE, level u8, action u16 LE, actionSpeed u8.
// All five versions encode identically (no version delta for clientbound prepare).
//
// Byte fixture: field order/opcode pinned per docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md (IDB-verified). Coverage-matrix linkage deferred — see task-099 follow-up (prepare/cancel fnames not yet in the packet-audit IDA exports).
func TestSkillPrepareForeignByteFixture(t *testing.T) {
	// charId=1001 (0x000003E9 LE = E9 03 00 00)
	// skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00)
	// level=10 (0x0A)
	// action=0x0142 LE = 42 01
	// actionSpeed=4 (0x04)
	expected := []byte{
		0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
		0x0A,       // level=10
		0x42, 0x01, // action=0x0142 LE
		0x04,       // actionSpeed=4
	}

	versions := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}

	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			input := NewCharacterSkillPrepareForeign(1001, 3121004, 10, 0x0142, 4)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if len(got) != len(expected) {
				t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X",
					len(got), len(expected), got, expected)
			}
			for i := range expected {
				if got[i] != expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X",
						i, got[i], expected[i], got, expected)
					break
				}
			}
		})
	}
}
