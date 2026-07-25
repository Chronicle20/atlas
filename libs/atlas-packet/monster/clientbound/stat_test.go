package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMonsterStatSetByteOutputV83 verifies the exact wire bytes for a
// single-stat (Speed) MonsterStatSet against the client read order from the
// checked-in IDA export docs/packets/ida-exports/gms_v83.json, entry
// CMob::OnStatSet @ 0x66c301 (ordered calls):
//
//	Decode4   — "dwMobId — read by CMobPool::OnMobPacket before dispatch"
//	DecodeBuf — "uFlagSet: 16-byte UINT128 stat mask"
//	DecodeBuf — "per-stat body via CMob::ProcessStatSet"
//
// The mob temporary-stat blob is a register-boundary opaque type
// (docs/packets/audits/OPAQUE_LEDGER.md, "mob temporary-stat blob" row,
// VERIFIED-EXCEPTION): the export cannot decompose the mask-driven body, so
// per the ledger discipline this byte test beside the struct is the oracle.
// The audit report (docs/packets/audits/gms_v83/MonsterStatSet.md, verdict ✅)
// records that Atlas's trailing tDelay/calcDamageStatIndex/bStat writes are
// absorbed by the trailing opaque buffer the client consumes.
//
// Expected wire, byte-for-byte (fixture: mobId=5001, one Speed stat from mob
// skill 126/SkillTypeSlow level 1, amount -40):
//
//	89 13 00 00             mobId 5001 LE                    (export Decode4)
//	00 ×12, 40 00 00 00     UINT128 mask, Speed bit (shift 6 — 7th stat in
//	                        the encode order; mask quarters H.hi/H.lo/L.hi/
//	                        L.lo each LE)                    (export DecodeBuf #1, 16 bytes)
//	D8 FF                   value -40 int16 LE               (export DecodeBuf #2 from here on)
//	7E 00                   sourceId 126 int16 LE (mob-skill source: id < 200)
//	01 00                   sourceLevel 1 int16 LE
//	FF FF                   expiry sentinel -1 int16 LE
//	00 00                   tDelay int16
//	00                      m_nCalcDamageStatIndex byte
//	00                      bStat byte (Speed is a movement-affecting stat)
//
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v83 ida=0x66c301
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v87 ida=0x6a71cc
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v95 ida=0x652660
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=jms_v185 ida=0x6e9a8e
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v83 ida=0x66c424
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v87 ida=0x6a72ef
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v95 ida=0x652780
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=jms_v185 ida=0x6e9bb1
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v84 ida=0x682603
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v84 ida=0x682726
func TestMonsterStatSetByteOutputV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	v := test.Variants[1] // GMS v83
	ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatSet(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001
		0x00, 0x00, 0x00, 0x00, // mask H.hi
		0x00, 0x00, 0x00, 0x00, // mask H.lo
		0x00, 0x00, 0x00, 0x00, // mask L.hi
		0x40, 0x00, 0x00, 0x00, // mask L.lo — Speed bit (shift 6)
		0xD8, 0xFF, // value -40
		0x7E, 0x00, // sourceId 126 (mob skill → int16)
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay
		0x00, // m_nCalcDamageStatIndex
		0x00, // bStat (movement-affecting)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatSetByteOutputV79 verifies the same single-stat (Speed) wire
// against the v79 client read order. uniqueId is consumed by
// CMobPool::OnMobPacket @0x646d46 (Decode4 @0x646d50) before switching on op
// 220 -> CMob::OnStatSet @0x63ae2b (GMS_v79_1_DEVM.exe, port 13340):
//
//	DecodeBuffer(16) @0x63ae55 — UINT128 stat mask
//	MobStat::DecodeTemporary @0x63ae7c — per-stat body (value/source/expiry),
//	                                      a register-boundary opaque blob
//	Decode2 @0x63afa1 — tDelay (v13)
//	Decode1 @0x63afa4 — m_nCalcDamageStatIndex (stored at +301)
//
// Per docs/packets/audits/OPAQUE_LEDGER.md ("mob temporary-stat blob",
// VERIFIED-EXCEPTION) the mask-driven body cannot be statically decomposed; the
// trailing bStat byte for movement-affecting stats (Speed) is absorbed within
// the opaque body the client consumes. The MonsterTemporaryStat encoder is
// version-independent, so the v79 wire is byte-identical to v83.
//
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v79 ida=0x63ae2b
func TestMonsterStatSetByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatSet(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4 @0x646d50
		0x00, 0x00, 0x00, 0x00, // mask H.hi — DecodeBuffer(16) @0x63ae55
		0x00, 0x00, 0x00, 0x00, // mask H.lo
		0x00, 0x00, 0x00, 0x00, // mask L.hi
		0x40, 0x00, 0x00, 0x00, // mask L.lo — Speed bit (shift 6)
		0xD8, 0xFF, // value -40 (opaque body @0x63ae7c)
		0x7E, 0x00, // sourceId 126 (mob skill -> int16)
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay — Decode2 @0x63afa1
		0x00, // m_nCalcDamageStatIndex — Decode1 @0x63afa4
		0x00, // bStat (movement-affecting; absorbed in opaque body)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 statset bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatSetByteOutputV72 verifies the single-stat (Speed) wire against
// the v72 client read order. uniqueId is consumed by CMobPool::OnMobPacket
// @0x62560d (Decode4 @0x625617) before switching on op 214 -> sub_61B59E
// (OnStatSet) @0x61b59e (GMS_v72.1_U_DEVM.exe, port 13339):
//
//	Decode4 @0x61b5c3 — LEGACY single 32-bit stat mask (v79 was DecodeBuffer(16))
//	MobStat::DecodeTemporary @0x61b5c6 — per-stat body (value/source/expiry)
//	Decode2 @0x61b679 — tDelay
//	Decode1 @0x61b687 — m_nCalcDamageStatIndex
//
// v72 mob temp-stat mask is a bare 4-byte word (model.go legacyMobStatMask); the
// per-stat body and trailer are byte-identical to v79. bStat (movement-affecting
// Speed) is absorbed within the opaque body per OPAQUE_LEDGER.
//
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v72 ida=0x61b59e
func TestMonsterStatSetByteOutputV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatSet(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4 @0x625617
		0x40, 0x00, 0x00, 0x00, // LEGACY 4-byte mask — Speed bit (shift 6) — Decode4 @0x61b5c3
		0xD8, 0xFF, // value -40 (opaque body @0x61b5c6)
		0x7E, 0x00, // sourceId 126 (mob skill -> int16)
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay — Decode2 @0x61b679
		0x00, // m_nCalcDamageStatIndex — Decode1 @0x61b687
		0x00, // bStat (movement-affecting; absorbed in opaque body)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 statset bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatResetByteOutputV79 verifies the StatReset wire against the v79
// client read order. uniqueId via CMobPool::OnMobPacket @0x646d50, op 221 ->
// CMob::OnStatReset @0x63b3b2 (GMS_v79_1_DEVM.exe, port 13340):
//
//	DecodeBuffer(16) @0x63b3ce — UINT128 stat mask
//	sub_704E92 @0x63b3f1 — MobStat reset body (opaque; absorbs tDelay)
//	Decode1 @0x63b4b0 — m_nCalcDamageStatIndex (v7, stored at +301)
//
// The StatSet/StatReset codecs share the symmetric trailer
// (tDelay+calcIndex+bStat); OnStatReset reads only the final Decode1 at the
// function level, so tDelay/bStat fall inside the opaque reset body per the
// OPAQUE_LEDGER discipline. Version-independent encoder -> byte-identical to v83.
//
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v79 ida=0x63b3b2
func TestMonsterStatResetByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatReset(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4 @0x646d50
		0x00, 0x00, 0x00, 0x00, // mask H.hi — DecodeBuffer(16) @0x63b3ce
		0x00, 0x00, 0x00, 0x00, // mask H.lo
		0x00, 0x00, 0x00, 0x00, // mask L.hi
		0x40, 0x00, 0x00, 0x00, // mask L.lo — Speed bit (shift 6)
		0xD8, 0xFF, // value -40 (opaque reset body @0x63b3f1)
		0x7E, 0x00, // sourceId 126
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay (absorbed in opaque reset body)
		0x00, // m_nCalcDamageStatIndex — Decode1 @0x63b4b0
		0x00, // bStat (movement-affecting; absorbed)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 statreset bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatResetByteOutputV72 verifies the StatReset wire against the v72
// client read order. uniqueId via CMobPool::OnMobPacket @0x625617, op 215 ->
// CMob::OnStatReset @0x61b6a6 (GMS_v72.1_U_DEVM.exe, port 13339):
//
//	Decode4 @0x61b6b8 — LEGACY single 32-bit stat mask (v79 was DecodeBuffer(16))
//	sub_6D2109 @0x61b6c3 — MobStat reset body (opaque; absorbs tDelay)
//	Decode1 @0x61b754 — m_nCalcDamageStatIndex
//
// v72 mask is a bare 4-byte word (model.go legacyMobStatMask); reset body/trailer
// byte-identical to v79 apart from mask width.
//
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v72 ida=0x61b6a6
func TestMonsterStatResetByteOutputV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatReset(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4 @0x625617
		0x40, 0x00, 0x00, 0x00, // LEGACY 4-byte mask — Speed bit (shift 6) — Decode4 @0x61b6b8
		0xD8, 0xFF, // value -40 (opaque reset body @0x61b6c3)
		0x7E, 0x00, // sourceId 126
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay (absorbed in opaque reset body)
		0x00, // m_nCalcDamageStatIndex — Decode1 @0x61b754
		0x00, // bStat (movement-affecting; absorbed)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 statreset bytes:\n got % x\nwant % x", got, want)
	}
}

func TestMonsterStatSet(t *testing.T) {
	stat := model.NewMonsterTemporaryStat()
	input := NewMonsterStatSet(5001, stat)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterStatReset(t *testing.T) {
	stat := model.NewMonsterTemporaryStat()
	input := NewMonsterStatReset(5001, stat)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
