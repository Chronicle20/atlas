package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
