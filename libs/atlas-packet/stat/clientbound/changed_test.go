package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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
		0x78, // Decode1 LEVEL = 120
		0x00, // trailing secondary-stat flag (atlas over-writes; client reads only when mask&0x180008)
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
		0x78, // Decode1 LEVEL = 120 (@0x4cf62a)
		0x00, // trailing secondary-stat flag (atlas over-writes; client reads only when mask&0x180008 @0x918752)
	}
	if got := in.Encode(l, ctx)(opts); !bytes.Equal(got, want) {
		t.Errorf("v72 StatChanged golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestStatChangedV61 pins the gms_v61 STAT_CHANGED (op 28) clientbound wire with
// a representative multi-bit mask exercising all four field widths (Decode1,
// Decode2, Decode4, DecodeBuffer(8)) plus the gated trailing byte.
//
// IDA-verified client decode (GMS_v61.1_U_DEVM.exe, port 13338) —
// CWvsContext::OnStatChanged @0x842d04:
//
//	Decode1 @0x842d1c → exclRequestSent (bool).
//	GW_CharacterStat::DecodeChangeStat @0x842d77 → mask + stat fields.
//	The trailing bSecondaryStatChangedPoint Decode1 @0x842d87 is gated on
//	  (v40 & 0x180008) @0x842d7a — the three pet-SN bits (0x8/0x80000/0x100000).
//	  This fixture includes PET_SN_1 (bit 0x8) so the client DOES read the
//	  trailing byte; atlas over-writes it (0) unconditionally.
//
// GW_CharacterStat::DecodeChangeStat @0x4b44e8:
//
//	Decode4(mask) @0x4b44f5 → 4-byte mask, then per-set-bit in ASCENDING bit
//	  order. Per-bit field widths (bit → read):
//	  0x1 SKIN Decode1 @0x4b450b (1); 0x2 FACE Decode4 @0x4b451a (4);
//	  0x4 HAIR Decode4 @0x4b4529 (4); 0x8 PET_SN_1 DecodeBuffer 8 @0x4b453b;
//	  0x80000 PET_SN_2 DecodeBuffer 8 @0x4b454d; 0x100000 PET_SN_3
//	  DecodeBuffer 8 @0x4b455f; 0x10 LEVEL Decode1 @0x4b4574 (1);
//	  0x20 JOB Decode2 @0x4b4586 (2); 0x40 STR..0x2000 MAX_MP Decode2 (2);
//	  0x8000 AVAILABLE_SP Decode2 @0x4b4690 (2); 0x10000 EXPERIENCE Decode4
//	  @0x4b46b0 (4); 0x20000 FAME Decode2 @0x4b46d0 (2); 0x40000 MESO Decode4
//	  @0x4b46f0 (4). Note HP/MaxHP/MP/MaxMP (0x400/0x800/0x1000/0x2000) read
//	  Decode2 — NARROW int16, NOT the v95 Decode4 widening (v61 < 95).
//
// Every per-bit width equals the atlas non-v95 encoder width for the same stat
// TYPE, and the mask is Decode4 — BYTE-IDENTICAL to the IDA-verified v72 wire
// (TestStatChangedV72). No codec gate needed: v61 takes the same !v95Plus path.
//
// Fixture (testStatOptions indices → mask bits): SKIN(0)=0x1, PET_SN_1(3)=0x8,
// LEVEL(4)=0x10, JOB(5)=0x20, HP(10)=0x400, EXPERIENCE(16)=0x10000,
// MESO(18)=0x40000 → mask 0x00050439. atlas sorts by index → write order
// SKIN, PET_SN_1, LEVEL, JOB, HP, EXP, MESO — which equals the client's
// ascending-bit read order for exactly this set (no PET_SN_2/3 present to
// reorder ahead of LEVEL).
//
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v61 ida=0x842d04
func TestStatChangedV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := testStatOptions()
	ctx := pt.CreateContext("GMS", 61, 1)
	in := NewStatChanged([]Update{
		NewUpdate(constants.TypeSkin, 7),
		NewUpdate(constants.TypePetSn1, 0x1122334455667788),
		NewUpdate(constants.TypeLevel, 120),
		NewUpdate(constants.TypeJob, 100),
		NewUpdate(constants.TypeHp, 5000),
		NewUpdate(constants.TypeExperience, 100000),
		NewUpdate(constants.TypeMeso, 999999),
	}, true)
	want := []byte{
		0x01,                   // Decode1 exclRequestSent = true (@0x842d1c)
		0x39, 0x04, 0x05, 0x00, // Decode4 mask = 0x00050439 (@0x4b44f5)
		0x07,                                           // SKIN=7 Decode1 (bit 0x1 @0x4b450b)
		0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, // PET_SN_1=0x1122334455667788 DecodeBuffer(8) (bit 0x8 @0x4b453b)
		0x78,       // LEVEL=120 Decode1 (bit 0x10 @0x4b4574)
		0x64, 0x00, // JOB=100 Decode2 (bit 0x20 @0x4b4586)
		0x88, 0x13, // HP=5000 Decode2 (bit 0x400 @0x4b4608)
		0xA0, 0x86, 0x01, 0x00, // EXPERIENCE=100000 Decode4 (bit 0x10000 @0x4b46b0)
		0x3F, 0x42, 0x0F, 0x00, // MESO=999999 Decode4 (bit 0x40000 @0x4b46f0)
		0x00, // trailing bSecondaryStatChangedPoint (mask&0x180008 set via PET_SN_1 → client reads; atlas over-writes 0) @0x842d87
	}
	if got := in.Encode(l, ctx)(opts); !bytes.Equal(got, want) {
		t.Errorf("v61 StatChanged golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestStatChangedV48 pins the gms_v48 STAT_CHANGED (op 27 / 0x1B) clientbound wire
// with the same representative multi-bit mask as the verified v61 twin.
//
// IDA-verified client decode (GMS_v48_1_DEVM.exe, port 13337) —
// CWvsContext::OnStatChanged @0x71aa68:
//
//	Decode1 @0x71aa80 → exclRequestSent (bool).
//	GW_CharacterStat::DecodeChangeStat @0x71aaea → mask + stat fields.
//	The trailing bSecondaryStatChangedPoint Decode1 @0x71aafa is gated on
//	  (v53 & 8) @0x71aaed — the PET_SN_1 bit (0x8). This fixture includes
//	  PET_SN_1 so the client DOES read the trailing byte; atlas over-writes it
//	  (0) unconditionally. (v48 gates on 0x8 only, vs v61's 0x180008; bit 0x8 is
//	  in both, so the byte is read for this fixture either way.)
//
// GW_CharacterStat::DecodeChangeStat @0x49ba4a — Decode4(mask) @0x49ba5a then
// per-set-bit in STRICT ascending bit order. Per-bit field widths:
//
//	0x1 SKIN Decode1 @0x49ba68 (1); 0x2 FACE Decode4 @0x49ba77 (4);
//	0x4 HAIR Decode4 @0x49ba86 (4); 0x8 PET_SN_1 DecodeBuffer 8 @0x49ba96;
//	0x10 LEVEL Decode1 @0x49baaa (1); 0x20 JOB Decode2 @0x49babb (2);
//	0x40..0x2000 STR..MAX_MP Decode2 (2) incl. HP 0x400 Decode2 @0x49bb38
//	(NARROW int16, NOT the v95 Decode4 widening — v48 < 95); 0x8000 AVAILABLE_SP
//	Decode2 @0x49bbb5 (2); 0x10000 EXPERIENCE Decode4 @0x49bbd1 (4); 0x20000 FAME
//	Decode2 @0x49bbf3 (2); 0x40000 MESO Decode4 @0x49bc15 (4). v48 has NO
//	0x80000/0x100000 PET_SN_2/3 bits (highest handled bit is MESO 0x40000).
//
// Every per-bit width equals the atlas non-v95 encoder width for the same stat
// TYPE, and the mask is Decode4 — BYTE-IDENTICAL to the IDA-verified v61 wire
// (TestStatChangedV61). No codec gate needed: v48 takes the same !v95Plus path.
// This fixture uses no PET_SN_2/3 (indices 19/20), so v48's narrower bit set
// does not affect the byte sequence. Fixture mask = 0x00050439 (as in v61).
//
// packet-audit:verify packet=stat/clientbound/Changed version=gms_v48 ida=0x71aa68
func TestStatChangedV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := testStatOptions()
	ctx := pt.CreateContext("GMS", 48, 1)
	in := NewStatChanged([]Update{
		NewUpdate(constants.TypeSkin, 7),
		NewUpdate(constants.TypePetSn1, 0x1122334455667788),
		NewUpdate(constants.TypeLevel, 120),
		NewUpdate(constants.TypeJob, 100),
		NewUpdate(constants.TypeHp, 5000),
		NewUpdate(constants.TypeExperience, 100000),
		NewUpdate(constants.TypeMeso, 999999),
	}, true)
	want := []byte{
		0x01,                   // Decode1 exclRequestSent = true (@0x71aa80)
		0x39, 0x04, 0x05, 0x00, // Decode4 mask = 0x00050439 (@0x49ba5a)
		0x07,                                           // SKIN=7 Decode1 (bit 0x1 @0x49ba68)
		0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, // PET_SN_1 DecodeBuffer(8) (bit 0x8 @0x49ba96)
		0x78,       // LEVEL=120 Decode1 (bit 0x10 @0x49baaa)
		0x64, 0x00, // JOB=100 Decode2 (bit 0x20 @0x49babb)
		0x88, 0x13, // HP=5000 Decode2 narrow (bit 0x400 @0x49bb38)
		0xA0, 0x86, 0x01, 0x00, // EXPERIENCE=100000 Decode4 (bit 0x10000 @0x49bbd1)
		0x3F, 0x42, 0x0F, 0x00, // MESO=999999 Decode4 (bit 0x40000 @0x49bc15)
		0x00, // trailing bSecondaryStatChangedPoint (mask&8 set via PET_SN_1 → client reads; atlas over-writes 0) @0x71aafa
	}
	if got := in.Encode(l, ctx)(opts); !bytes.Equal(got, want) {
		t.Errorf("v48 StatChanged golden mismatch\n got: % x\nwant: % x", got, want)
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
