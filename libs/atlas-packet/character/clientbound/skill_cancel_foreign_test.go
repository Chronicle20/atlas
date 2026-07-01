package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestSkillCancelForeignRoundTrip exercises Encode/Decode symmetry for
// SkillCancelForeign across all tenant variants. Wire-spec §4:
// charId u32, skillId u32. Identical across all five versions.
func TestSkillCancelForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSkillCancelForeign(1001, 3121004)
			output := SkillCancelForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}

// TestSkillCancelForeignOperation verifies Operation() returns the foreign writer
// const (not the bug pattern where foreign structs return the non-foreign const).
func TestSkillCancelForeignOperation(t *testing.T) {
	m := NewSkillCancelForeign(1, 3121004)
	if got := m.Operation(); got != CharacterSkillCancelForeignWriter {
		t.Errorf("Operation() = %q, want %q", got, CharacterSkillCancelForeignWriter)
	}
}

// TestSkillCancelForeignByteFixture asserts the exact encoded bytes match
// wire-spec §4 field order: charId u32 LE, skillId u32 LE.
// All five versions encode identically (no version delta for clientbound cancel).
//
// Byte fixture: field order/opcode pinned per docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md (IDB-verified).
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=gms_v79 ida=0x8d6e4a
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=gms_v83 ida=0x980bf5
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=gms_v84 ida=0x9c0dd3
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=gms_v87 ida=0xa062b1
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=gms_v95 ida=0x954600
// packet-audit:verify packet=character/clientbound/CharacterSkillCancelForeign version=jms_v185 ida=0xa540c4
func TestSkillCancelForeignByteFixture(t *testing.T) {
	// charId=1001 (0x000003E9 LE = E9 03 00 00)
	// skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00)
	expected := []byte{
		0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
	}

	versions := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v79", "GMS", 79},
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}

	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			input := NewSkillCancelForeign(1001, 3121004)
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
