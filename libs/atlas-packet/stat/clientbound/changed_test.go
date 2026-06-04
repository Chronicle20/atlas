package clientbound

import (
	"testing"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func testStatOptions() map[string]interface{} {
	return map[string]interface{}{
		"statistics": []interface{}{
			"SKIN", "FACE", "HAIR", "PET_SN_1", "LEVEL", "JOB",
			"STRENGTH", "DEXTERITY", "INTELLIGENCE", "LUCK",
			"HP", "MAX_HP", "MP", "MAX_MP",
			"AVAILABLE_AP", "AVAILABLE_SP", "EXPERIENCE", "FAME",
			"MESO", "PET_SN_2", "PET_SN_3", "GACHAPON_EXPERIENCE",
		},
	}
}

func TestStatChangedSingleRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged([]Update{
				NewUpdate(constants.TypeLevel, 120),
			}, true)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if !output.ExclRequestSent() {
				t.Error("expected exclRequestSent to be true")
			}
			if len(output.Updates()) != 1 {
				t.Fatalf("updates count: got %v, want 1", len(output.Updates()))
			}
			if output.Updates()[0].Stat() != constants.TypeLevel {
				t.Errorf("stat type: got %v, want %v", output.Updates()[0].Stat(), constants.TypeLevel)
			}
			if output.Updates()[0].Value() != 120 {
				t.Errorf("value: got %v, want 120", output.Updates()[0].Value())
			}
		})
	}
}

func TestStatChangedMultipleRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged([]Update{
				NewUpdate(constants.TypeHp, 5000),
				NewUpdate(constants.TypeMp, 3000),
				NewUpdate(constants.TypeExperience, 100000),
				NewUpdate(constants.TypeMeso, 999999),
			}, false)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if output.ExclRequestSent() {
				t.Error("expected exclRequestSent to be false")
			}
			if len(output.Updates()) != 4 {
				t.Fatalf("updates count: got %v, want 4", len(output.Updates()))
			}
		})
	}
}

// TestStatChangedV95WireWidths proves the two v95 wire fixes at the byte level
// (round-trip tests can't: the pre-fix encoder/decoder were wrong-but-symmetric):
//   - HP/MaxHP/MP/MaxMP widened from int16 (2 bytes) to int32 (4 bytes) in v95
//     (GW_CharacterStat::DecodeChangeStat, mask bits 0x400/0x800/0x1000/0x2000).
//   - a second trailing flag byte (battle-recovery-info, CWvsContext::OnStatChanged)
//     present in v95.
//
// A single-HP packet is bool(1) + mask(4) + HP + trailing:
//
//	v95: 1 + 4 + 4 + 2 = 11 bytes
//	v83: 1 + 4 + 2 + 1 =  8 bytes
//	v87: 1 + 4 + 2 + 1 =  8 bytes (mirrors v83 — task-080 B4.1)
//
// v87 evidence (GMSv87_4GB.exe, md5 2e692f3a…):
//   - GW_CharacterStat::DecodeChangeStat @ 0x502252 reads HP/MaxHP/MP/MaxMP
//     (masks 0x400/0x800/0x1000/0x2000) via CInPacket::Decode2 → NARROW int16,
//     same as v83; v95 widened to Decode4. The v95Plus gate writes int16 for
//     v87 → CORRECT.
//   - CWvsContext::OnStatChanged @ 0xab6e77 reads ONE trailing Decode1 byte
//     (bSecondaryStatChangedPoint, gated on mask & 0x180008) and NO second
//     battle-recovery-info byte; the v95Plus second trailing byte is correctly
//     omitted for v87. v87 mirrors v83 exactly for this packet.
func TestStatChangedV95WireWidths(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := testStatOptions()
	in := NewStatChanged([]Update{NewUpdate(constants.TypeHp, 1000)}, false)

	if v95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(opts); len(v95) != 11 {
		t.Errorf("v95 single-HP packet = %d bytes, want 11 (4-byte HP + 2 trailing): % x", len(v95), v95)
	}
	if v87 := in.Encode(l, pt.CreateContext("GMS", 87, 1))(opts); len(v87) != 8 {
		t.Errorf("v87 single-HP packet = %d bytes, want 8 (2-byte HP + 1 trailing): % x", len(v87), v87)
	}
	if v83 := in.Encode(l, pt.CreateContext("GMS", 83, 1))(opts); len(v83) != 8 {
		t.Errorf("v83 single-HP packet = %d bytes, want 8 (2-byte HP + 1 trailing): % x", len(v83), v83)
	}
}

func TestStatChangedEmptyRoundTrip(t *testing.T) {
	opts := testStatOptions()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewStatChanged(nil, false)
			output := Changed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, opts)
			if len(output.Updates()) != 0 {
				t.Errorf("updates count: got %v, want 0", len(output.Updates()))
			}
		})
	}
}
