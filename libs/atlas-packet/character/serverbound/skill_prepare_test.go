package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// buildSampleSkillPrepare mirrors model.sampleSkillPrepareInfo: a standard keydown
// skill (Hurricane 3121004) so the swallowMobId branch (skillId == 33101005) stays
// quiet and the wire structure is driven purely by tenant version.
func buildSampleSkillPrepare() model.SkillPrepareInfo {
	m := model.NewSkillPrepareInfo()
	m.SetSkillId(3121004)
	m.SetLevel(10)
	m.SetAction(0x0142)
	m.SetActionSpeed(4)
	return *m
}

// TestSkillPrepareRoundTrip pins that the serverbound SkillPrepare wrapper delegates
// symmetrically to the shared model.SkillPrepareInfo codec across all tenant variants.
// The model itself (incl. the swallowMobId branch) is production-tested in
// model/skill_prepare_info_test.go.
func TestSkillPrepareRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := SkillPrepare{info: buildSampleSkillPrepare()}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// TestSkillPrepareByteFixture asserts the exact encoded bytes of the serverbound
// SkillPrepare wrapper match wire-spec §1 field order:
// skillId u32 LE, level u8, action u16 LE, actionSpeed u8. The sample skill
// (Hurricane 3121004) is NOT the swallow skill (33101005), so the trailing
// conditional swallowMobId u32 is absent on every version — all five encode
// identically. Bytes trace to CUserLocal::DoActiveSkill_Prepare's COutPacket
// writes (skillId/level/action/actionSpeed) per wire-spec.md (IDB-verified).
//
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v72 ida=0x874c8f
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v79 ida=0x8c17f2
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v83 ida=0x96a86e
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v84 ida=0x9a9761
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v87 ida=0x9ee1e6
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v95 ida=0x941710
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=jms_v185 ida=0xa39cfd
// TestSkillPrepareByteFixtureV72 pins the legacy GMS v72 wire, which encodes the
// action/direction field as a SINGLE byte (bit7=bLeft, bits0-6=nAction) instead of
// the 2-byte short at v79+. IDA-verified: CUserLocal::DoActiveSkill_Prepare @0x874c8f
// (GMS_v72.1_U_DEVM.exe, port 13339) builds COutPacket(92) = Encode4 skillId
// @0x87550b, Encode1 level @0x875516, Encode1 action @0x875535
// (`(bLeft<<7)|(nAction&0x7F)` — ONE byte), Encode1 actionSpeed @0x87553e.
func TestSkillPrepareByteFixtureV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	m := SkillPrepare{info: *model.NewSkillPrepareInfo()}
	m.info.SetSkillId(3121004)
	m.info.SetLevel(10)
	m.info.SetAction(0x42) // fits the legacy 1-byte field
	m.info.SetActionSpeed(4)
	expected := []byte{
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE  @0x87550b
		0x0A, // level=10                              @0x875516
		0x42, // action=0x42 (1 BYTE on v72)           @0x875535
		0x04, // actionSpeed=4                          @0x87553e
	}
	got := pt.Encode(t, ctx, m.Encode, nil)
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

// TestSkillPrepareByteFixtureV61 pins the very-legacy GMS v61 wire, which encodes the
// action/direction field as a SINGLE byte (bit7=bLeft, bits0-6=nAction) — the same as
// v72 (both < 79). IDA-verified: CUserLocal::DoActiveSkill_Prepare @0x7b8001
// (GMS_v61.1_U_DEVM.exe, port 13338 — named from sub_7B8001; registry opcode 85 matches
// the send-site COutPacket(85) @0x7b8711) builds Encode4 skillId @0x7b8723, Encode1 level
// @0x7b872e, Encode1 action @0x7b874b (`(nAction&0x7F)|(bLeft<<7)` — ONE byte), Encode1
// actionSpeed @0x7b8754. Byte-identical to v72 (Δ-7 opcode).
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v61 ida=0x7b8001
func TestSkillPrepareByteFixtureV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := SkillPrepare{info: *model.NewSkillPrepareInfo()}
	m.info.SetSkillId(3121004)
	m.info.SetLevel(10)
	m.info.SetAction(0x42) // fits the legacy 1-byte field
	m.info.SetActionSpeed(4)
	expected := []byte{
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE  @0x7b8723
		0x0A, // level=10                              @0x7b872e
		0x42, // action=0x42 (1 BYTE on v61)           @0x7b874b
		0x04, // actionSpeed=4                          @0x7b8754
	}
	got := pt.Encode(t, ctx, m.Encode, nil)
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

// TestSkillPrepareByteFixtureV48 pins the very-legacy GMS v48 SKILL_EFFECT (op 72) wire,
// byte-identical to v61: the action/direction field is a SINGLE byte (v48 < 79). Body-
// verified op->body binding (distrust symbols): sub_6ADD4C (GMS_v48_1_DEVM.exe, port
// 13337) — the DoActiveSkill_Prepare sender — builds COutPacket(72) @0x6ae20e, Encode4
// skillId @0x6ae220, Encode1 level @0x6ae22b, Encode1 action @0x6ae248
// (`this[896]&0x7F | (this[223]<<7)` — ONE byte), Encode1 actionSpeed @0x6ae251. Opcode
// 72 confirmed at the send-site (not the symbol). swallowMobId is GMS v95+/JMS only.
// packet-audit:verify packet=character/serverbound/CharacterSkillPrepare version=gms_v48 ida=0x6add4c
func TestSkillPrepareByteFixtureV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := SkillPrepare{info: *model.NewSkillPrepareInfo()}
	m.info.SetSkillId(3121004)
	m.info.SetLevel(10)
	m.info.SetAction(0x42) // fits the legacy 1-byte field
	m.info.SetActionSpeed(4)
	expected := []byte{
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE  @0x6ae220
		0x0A, // level=10                              @0x6ae22b
		0x42, // action=0x42 (1 BYTE on v48)           @0x6ae248
		0x04, // actionSpeed=4                          @0x6ae251
	}
	got := pt.Encode(t, ctx, m.Encode, nil)
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

func TestSkillPrepareByteFixture(t *testing.T) {
	// skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00)
	// level=10 (0x0A)
	// action=0x0142 LE = 42 01
	// actionSpeed=4 (0x04)
	expected := []byte{
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
			m := SkillPrepare{info: buildSampleSkillPrepare()}
			got := pt.Encode(t, ctx, m.Encode, nil)
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
