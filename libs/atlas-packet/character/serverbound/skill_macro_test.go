package serverbound

import (
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// wireUniform is the same fixture the clientbound test settled on in Task 6:
// count(1) + per entry { name(2+len) + shout(1) + skillId1(4) + skillId2(4) +
// skillId3(4) }. The client's send order (serverbound FlushToSvr) and its
// receive order (clientbound OnMacroSysDataInit) are the same field order per
// layout-derivation.md, so the fixture is reused verbatim.
//
// Shout polarity is INVERTED (wire 0 = shout on) per layout-derivation.md's
// "Shout polarity" verdict: entry0's wire byte 00 decodes to shout=true,
// entry1's wire byte 01 decodes to shout=false.
const wireUniform = "02" +
	"0400" + "42756666" + "00" + "2b460f00" + "2c460f00" + "00000000" +
	"0600" + "41747461636b" + "01" + "2d460f00" + "00000000" + "00000000"

// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v61 ida=0x59746c
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v72 ida=0x5e39e0
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v79 ida=0x6022db
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v83 ida=0x631919
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v84 ida=0x646e7f
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v87 ida=0x66a505
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v92 ida=0x602ed0
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=gms_v95 ida=0x60ea20
// packet-audit:verify packet=character/serverbound/CharacterSkillMacroHandle version=jms_v185 ida=0x6a3466
func TestSkillMacroDecode(t *testing.T) {
	cases := []struct{ variant pt.TenantVariant }{
		{pt.Variants[8]},  // GMS v61
		{pt.Variants[9]},  // GMS v72
		{pt.Variants[10]}, // GMS v79
		{pt.Variants[1]},  // GMS v83
		{pt.Variants[5]},  // GMS v84
		{pt.Variants[2]},  // GMS v87
		{pt.Variants[11]}, // GMS v92
		{pt.Variants[3]},  // GMS v95
		{pt.Variants[4]},  // JMS v185
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			raw, err := hex.DecodeString(wireUniform)
			if err != nil {
				t.Fatalf("fixture hex: %v", err)
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)
			m := SkillMacro{}
			m.Decode(nil, ctx)(&reader, nil)

			got := m.Macros()
			if len(got) != 2 {
				t.Fatalf("count: got %d, want 2", len(got))
			}
			if got[0].Name() != "Buff" || !got[0].Shout() ||
				got[0].SkillId1() != skill.Id(1001003) ||
				got[0].SkillId2() != skill.Id(1001004) ||
				got[0].SkillId3() != skill.Id(0) {
				t.Errorf("entry 0: got %+v", got[0])
			}
			if got[1].Name() != "Attack" || got[1].Shout() ||
				got[1].SkillId1() != skill.Id(1001005) ||
				got[1].SkillId2() != skill.Id(0) ||
				got[1].SkillId3() != skill.Id(0) {
				t.Errorf("entry 1: got %+v", got[1])
			}
		})
	}
}

func TestSkillMacroDecodeClampsCount(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// A count byte of 0xFF with no entry bytes behind it: the decoder must stop
	// at the client's capacity (layout-derivation.md "Capacity"), not allocate
	// and parse 255 entries off an exhausted reader.
	raw := []byte{0xFF}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	m := SkillMacro{}
	m.Decode(nil, ctx)(&reader, nil)
	if len(m.Macros()) > maxMacroCount {
		t.Errorf("decoded %d entries, want at most %d", len(m.Macros()), maxMacroCount)
	}
}

// TestSkillMacroCrossSeamRoundTrip encodes with the SERVERBOUND struct and
// decodes with the SERVERBOUND struct — the intra-file agreement check. The
// cross-DIRECTION check (clientbound Encode → serverbound Decode) is not a
// round trip at all on the wire and is deliberately not asserted here: the two
// directions are proven separately against absolute bytes, which is what
// stopped the double-inversion in the old shared struct from cancelling out
// (design §1.1, bug_matrix_roundtrip_fixture_false_verify).
func TestSkillMacroCrossSeamRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSkillMacro(
				NewSkillMacroEntry("Buff", true, skill.Id(1001003), skill.Id(1001004), skill.Id(0)),
				NewSkillMacroEntry("Attack", false, skill.Id(1001005), skill.Id(0), skill.Id(0)),
			)
			output := SkillMacro{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Macros()) != len(input.Macros()) {
				t.Fatalf("count: got %d, want %d", len(output.Macros()), len(input.Macros()))
			}
			for i, e := range output.Macros() {
				if e.Name() != input.Macros()[i].Name() {
					t.Errorf("macros[%d].Name: got %v, want %v", i, e.Name(), input.Macros()[i].Name())
				}
				if e.Shout() != input.Macros()[i].Shout() {
					t.Errorf("macros[%d].Shout: got %v, want %v", i, e.Shout(), input.Macros()[i].Shout())
				}
				if e.SkillId1() != input.Macros()[i].SkillId1() {
					t.Errorf("macros[%d].SkillId1: got %v, want %v", i, e.SkillId1(), input.Macros()[i].SkillId1())
				}
			}
		})
	}
}
