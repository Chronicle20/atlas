package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestSkillPrepareForeignRoundTrip exercises Encode/Decode symmetry for
// SkillPrepareForeign across all tenant variants. Wire-spec §3:
// charId u32, skillId u32, level u8, action u16, actionSpeed u8. Field order
// is identical across all five versions.
func TestSkillPrepareForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// action 0x42 fits both the legacy 1-byte field (GMS<79) and the
			// 2-byte short (v79+); the high-byte 2-byte case is pinned by the
			// multi-version TestSkillPrepareForeignByteFixture (action 0x0142).
			input := NewSkillPrepareForeign(1001, 3121004, 10, 0x42, 4)
			output := SkillPrepareForeign{}
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

// TestSkillPrepareForeignV72ByteFixture pins the legacy GMS v72 wire, which reads
// the action/direction field as a SINGLE byte (bit7=bLeft, bits0-6=nAction) instead
// of the 2-byte short used at v79+. IDA-verified: CUserRemote::OnSkillPrepare
// @0x889e3d (GMS_v72.1_U_DEVM.exe, port 13339) reads Decode4 skillId @0x889e86,
// Decode1 level @0x889e99, Decode1 action @0x889ec7 (>>7 / &0x7F — ONE byte),
// Decode1 actionSpeed @0x889ee8. charId(4) leads (consumed by the pool dispatcher).
func TestSkillPrepareForeignV72ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	// action=0x42 fits the legacy 1-byte field (bit7=0, action=0x42).
	input := NewSkillPrepareForeign(1001, 3121004, 10, 0x42, 4)
	expected := []byte{
		0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
		0x0A, // level=10                          @0x889e99
		0x42, // action=0x42 (1 BYTE on v72)       @0x889ec7
		0x04, // actionSpeed=4                      @0x889ee8
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}

// TestSkillPrepareForeignV61ByteFixture pins the very-legacy GMS v61 wire, which reads
// the action/direction field as a SINGLE byte (bit7=bLeft, bits0-6=nAction) — the same
// as v72 (both < 79). IDA-verified: the real per-op handler CUserRemote::OnSkillPrepare
// @0x7c9963 (GMS_v61.1_U_DEVM.exe, port 13338 — registry's dispatcher note-address
// 0x7bd75a is the pool switch, not the handler) reads Decode4 skillId @0x7c99ac, Decode1
// level @0x7c99bf, Decode1 action @0x7c99ed (>>7 / &0x7F — ONE byte), Decode1 actionSpeed
// @0x7c9a0e. charId(4) leads (consumed by the pool dispatcher). Byte-identical to v72.
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v61 ida=0x7c9963
func TestSkillPrepareForeignV61ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	// action=0x42 fits the legacy 1-byte field (bit7=0, action=0x42).
	input := NewSkillPrepareForeign(1001, 3121004, 10, 0x42, 4)
	expected := []byte{
		0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
		0x0A, // level=10                          @0x7c99bf
		0x42, // action=0x42 (1 BYTE on v61)       @0x7c99ed
		0x04, // actionSpeed=4                      @0x7c9a0e
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}

// TestSkillPrepareForeignOperation verifies Operation() returns the foreign writer
// const (not the bug pattern where foreign structs return the non-foreign const).
func TestSkillPrepareForeignOperation(t *testing.T) {
	m := NewSkillPrepareForeign(1, 3121004, 1, 0x0001, 1)
	if got := m.Operation(); got != CharacterSkillPrepareForeignWriter {
		t.Errorf("Operation() = %q, want %q", got, CharacterSkillPrepareForeignWriter)
	}
}

// TestSkillPrepareForeignByteFixture asserts the exact encoded bytes match
// wire-spec §3 field order: charId u32 LE, skillId u32 LE, level u8, action u16 LE, actionSpeed u8.
// All five versions encode identically (no version delta for clientbound prepare).
//
// Byte fixture: field order/opcode pinned per docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md (IDB-verified).
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v72 ida=0x889e3d
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v79 ida=0x8d6cd6
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v83 ida=0x980a81
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v84 ida=0x9c0c5f
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v87 ida=0xa06135
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=gms_v95 ida=0x953a30
// packet-audit:verify packet=character/clientbound/CharacterSkillPrepareForeign version=jms_v185 ida=0xa53f49
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
		0x04, // actionSpeed=4
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
			input := NewSkillPrepareForeign(1001, 3121004, 10, 0x0142, 4)
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
