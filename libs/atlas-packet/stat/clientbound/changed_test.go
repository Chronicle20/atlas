package clientbound

import (
	"bytes"
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

// packet-audit:verify packet=stat/clientbound/Changed version=gms_v83 ida=0xa1fb52
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v87 ida=0xab6e77
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v95 ida=0x9fd5d0
// packet-audit:verify packet=stat/clientbound/Changed version=jms_v185 ida=0xb06632
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v84 ida=0xa6ae08
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

// TestStatChangedV79 pins the gms_v79 STAT_CHANGED (op 28) clientbound wire.
//
// IDA-verified client decode (GMS_v79_1_DEVM.exe, port 13340) —
// CWvsContext::OnStatChanged @0x96a140:
//
//	Decode1 @0x96a15c → exclRequestSent (bool).
//	GW_CharacterStat::DecodeChangeStat @0x96a1c3 → mask + stat fields. Inside
//	  DecodeChangeStat @0x4d72d1: Decode4(mask) @0x4d72de, then per-bit; LEVEL
//	  (mask 0x10) reads Decode1 @0x4d735d → a single byte.
//	The trailing bSecondaryStatChangedPoint Decode1 @0x96a1d3 is gated on
//	  (mask & 0x180008) — the three pet-SN bits. LEVEL alone does NOT set those
//	  bits, so the v79 client reads NO trailing byte here.
//
// v79 matches the v83/v87 shape exactly: narrow stat widths (LEVEL=1 byte) and
// NO v95 battle-recovery second byte. atlas Changed.Encode writes one
// unconditional trailing byte (the over-written secondary-stat flag, client-
// conditional read — same accepted behavior verified for v83/v87 in
// TestStatChangedV95WireWidths). LEVEL index 4 in testStatOptions → mask 0x10.
//
//	bool(1) + mask(4) + level(1) + trailing(1) = 7 bytes.
//
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v79 ida=0x96a140
func TestStatChangedV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := testStatOptions()
	ctx := pt.CreateContext("GMS", 79, 1)
	in := NewStatChanged([]Update{NewUpdate(constants.TypeLevel, 120)}, true)
	want := []byte{
		0x01,                   // Decode1 exclRequestSent = true
		0x10, 0x00, 0x00, 0x00, // Decode4 mask = 0x10 (LEVEL, index 4)
		0x78,                   // Decode1 LEVEL = 120
		0x00,                   // trailing secondary-stat flag (atlas over-writes; client reads only when mask&0x180008)
	}
	if got := in.Encode(l, ctx)(opts); !bytes.Equal(got, want) {
		t.Errorf("v79 StatChanged golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestStatChangedV72 pins the gms_v72 STAT_CHANGED clientbound wire.
//
// IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port 13339) —
// CWvsContext::OnStatChanged @0x9186db:
//
//	Decode1 @0x9186f8 → exclRequestSent (bool).
//	GW_CharacterStat::DecodeChangeStat @0x91874f → mask + stat fields. Inside
//	  DecodeChangeStat @0x4cf59e: Decode4(mask) @0x4cf5ab, then per-bit; LEVEL
//	  (mask 0x10) reads Decode1 @0x4cf62a → a single byte.
//	The trailing bSecondaryStatChangedPoint Decode1 @0x91875f is gated on
//	  (mask & 0x180008) @0x918752 — the three pet-SN bits. LEVEL alone does NOT
//	  set those bits, so the v72 client reads NO trailing byte here.
//
// Byte-identical to the verified v79 wire (narrow LEVEL=1 byte, no v95
// battle-recovery second byte). atlas Changed.Encode writes one unconditional
// trailing byte (over-written secondary-stat flag, client-conditional read —
// same accepted behavior verified for v79/v83/v87). LEVEL index 4 → mask 0x10.
//
//	bool(1) + mask(4) + level(1) + trailing(1) = 7 bytes.
//
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v72 ida=0x9186db
func TestStatChangedV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := testStatOptions()
	ctx := pt.CreateContext("GMS", 72, 1)
	in := NewStatChanged([]Update{NewUpdate(constants.TypeLevel, 120)}, true)
	want := []byte{
		0x01,                   // Decode1 exclRequestSent = true (@0x9186f8)
		0x10, 0x00, 0x00, 0x00, // Decode4 mask = 0x10 (LEVEL, index 4) (@0x4cf5ab)
		0x78,                   // Decode1 LEVEL = 120 (@0x4cf62a)
		0x00,                   // trailing secondary-stat flag (atlas over-writes; client reads only when mask&0x180008 @0x918752)
	}
	if got := in.Encode(l, ctx)(opts); !bytes.Equal(got, want) {
		t.Errorf("v72 StatChanged golden mismatch\n got: % x\nwant: % x", got, want)
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
