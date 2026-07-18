package clientbound

import (
	"bytes"
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CASH_SHOP_OPEN / SET_CASH_SHOP (CStage::OnSetCashShop) — the scene-transition
// packet that opens the in-client Cash Shop. The wire body is the full migrate-in
// CharacterData block (the SAME block field/clientbound/SetItc reuses), then the
// cash-shop payload the client reads in CCashShop::CCashShop -> CCashShop::LoadData:
//
//	Decode1        bCashShopAuthorized (GMS; absent in JMS, which reads account first)
//	DecodeStr      m_sNexonClubID (account name; JMS reads unconditionally)
//	SetSaleInfo    Decode4 nNotSaleCount (GMS only) + Decode2 special + [Decode2 JMS] + Decode1 discounts
//	DecodeBuffer(1080)  the "Best" block — exactly 9*2*5 int32 triples = 1080 bytes
//	DecodeStock    Decode2 count + entries
//	DecodeLimitGoods  Decode2 count + entries (GMS>12 || JMS)
//	DecodeZeroGoods   Decode2 count + entries (GMS only)
//	Decode1        m_bEventOn
//	Decode4        m_nHighestCharacterLevelInThisAccount (GMS only)
//
// The read-site addresses below are pinned as packet-audit:verify machine markers
// on the byte-tests in this file; each carries a fresh evidence record
// (docs/packets/evidence/<version>/cash.clientbound.CashShopOpen.yaml) keyed to the
// export function CStage::OnSetCashShop, promoting CASH_SHOP_OPEN to ✅ in the
// coverage matrix for every version whose client reads the MODERN body (task-113).
//
//	version   OnSetCashShop  body reader (CCashShop ctor -> CCashShop::LoadData)
//	gms_v72   0x6c16c3       ctor 0x461fc9 -> LoadData 0x46a706 (Stock/Limit/Zero + ctor Decode1+Decode4)
//	gms_v79   0x6f11c8       ctor 0x462e86 -> LoadData 0x46b86c (Stock/Limit/Zero + ctor Decode1+Decode4)
//	gms_v83   0x776a4f       ctor 0x468223 -> LoadData 0x471f37 (DecodeStock/LimitGoods/ZeroGoods)
//	gms_v84   0x7993b6       (v84 body byte-identical to v83)
//	gms_v87   0x7c4d0c       ctor 0x471159 -> LoadData 0x47c848 (ctor Decode1+Decode4)
//	gms_v95   0x71adf0       ctor 0x4938b0 -> LoadData 0x492ea0 (symbolicated m_bEventOn / m_nHighest...)
//	jms_v185  0x7ef5f2       ctor 0x47811b -> LoadData 0x4839a9 (account-first, no ZeroGoods, no nHighest)
//
// NOTE (task-113 IDA finding): gms_v48 (opcode 74, CStage handler 0x5c4d9c) and
// gms_v61 (0x65a973) DO have a Cash Shop scene-entry packet, but their clients read
// a LEGACY body: CCashShop::LoadData has NO DecodeZeroGoods and the ctor reads only
// one Decode1 (NO trailing Decode4 nHighest). This writer's GMS branch unconditionally
// emits DecodeZeroGoods (2B) + nHighest (4B) for every GMS version >12, so it does NOT
// match the v48/v61 wire. Those cells are intentionally NOT promoted (would be a false
// ✅); see docs/tasks/task-113-gms-legacy-versions/promote-cashshopopen.md.

// cashShopTestCharacterData builds a deterministic CharacterData block shared by the
// round-trip and golden byte-tests so the leading envelope is known-good.
func cashShopTestCharacterData() charpkt.CharacterData {
	return charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: charpkt.InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}
}

// TestCashShopOpenRoundTrip proves the full body round-trips byte-exactly across
// every pt.Variant. The markers below link the IMPLEMENTED registry versions whose
// client reads the modern body to this test (v83/v84/v87/v95/jms).
//
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v83 ida=0x776a4f
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v84 ida=0x7993b6
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v87 ida=0x7c4d0c
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v95 ida=0x71adf0
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=jms_v185 ida=0x7ef5f2
func TestCashShopOpenRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cd := cashShopTestCharacterData()
			input := NewCashShopOpen(cd, "TestAccount")
			output := CashShopOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterData().Stats.Id != cd.Stats.Id {
				t.Errorf("stats id: got %v, want %v", output.CharacterData().Stats.Id, cd.Stats.Id)
			}
			if output.AccountName() != "TestAccount" {
				t.Errorf("accountName: got %q, want %q", output.AccountName(), "TestAccount")
			}
		})
	}
}

// TestCashShopOpenLegacyGolden pins the trailing cash-shop config block for the
// legacy GMS versions gms_v79/v72 (which pt.Variants does not enumerate). For every
// GMS major version > 12 the writer takes the identical branch, so the trailing 23
// bytes are byte-identical to v83's. The assertion is tail-anchored (the
// CharacterData envelope + 1080-byte Best block are large/variable-length): the last
// "Decode Best" int32 triple (8, 1, 50000047) is followed by the Stock/Limit/Zero
// zero-count shorts, bEventOn, and the nHighestCharacterLevelInThisAccount (200) int.
// v79/v72 clients read exactly this (IDA-verified: CCashShop::LoadData has
// DecodeStock/LimitGoods/ZeroGoods and the ctor reads Decode1 bEventOn + Decode4
// nHighest — 0x46b86c/0x462e86 for v79, 0x46a706/0x461fc9 for v72).
//
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v79 ida=0x6f11c8
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v72 ida=0x6c16c3
func TestCashShopOpenLegacyGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// Trailing 23 bytes:
	//   last Best triple: WriteInt(8) WriteInt(1) WriteInt(50000047=0x02FAF0AF)
	//   DecodeStock short 0, DecodeLimitGoods short 0, DecodeZeroGoods short 0
	//   bEventOn false, nHighestCharacterLevelInThisAccount 200 (0xC8)
	wantTail := []byte{
		0x08, 0x00, 0x00, 0x00, // Best i=8
		0x01, 0x00, 0x00, 0x00, // Best j=1
		0xAF, 0xF0, 0xFA, 0x02, // Best sn 50000047
		0x00, 0x00, // DecodeStock count 0
		0x00, 0x00, // DecodeLimitGoods count 0
		0x00, 0x00, // DecodeZeroGoods count 0
		0x00,                   // bEventOn false
		0xC8, 0x00, 0x00, 0x00, // nHighestCharacterLevelInThisAccount 200
	}
	// Reference: the GMS>12 body is version-stable, so v79/v72 bytes == v83 bytes.
	ref := NewCashShopOpen(cashShopTestCharacterData(), "TestAccount").
		Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	for _, major := range []uint16{79, 72} {
		t.Run(fmt.Sprintf("gms_v%d", major), func(t *testing.T) {
			ctx := pt.CreateContext("GMS", major, 1)
			input := NewCashShopOpen(cashShopTestCharacterData(), "TestAccount")
			b := input.Encode(l, ctx)(nil)
			if len(b) < len(wantTail) {
				t.Fatalf("buffer too short: %d bytes", len(b))
			}
			tail := b[len(b)-len(wantTail):]
			if !bytes.Equal(tail, wantTail) {
				t.Errorf("v%d trailing config block mismatch:\n got %v\nwant %v", major, tail, wantTail)
			}
			if !bytes.Equal(b, ref) {
				t.Errorf("v%d body diverges from v83 (GMS>12 must be version-stable)", major)
			}
		})
	}
}

// TestCashShopOpenLegacyBodyV48V61 pins the LEGACY Cash Shop body for gms_v48 and
// gms_v61. IDA-verified (task-113): both clients read a body that OMITS the modern
// GMS DecodeZeroGoods (2 bytes) and the trailing Decode4 nHighest (4 bytes):
//
//	v48  CStage handler sub_5C4D9C 0x5c4d9c -> CCashShop ctor sub_447122 0x447122:
//	     LoadData sub_44E1E5 0x44e1e5 reads Decode1 auth, DecodeStr account,
//	     SetSaleInfo sub_71E3E4 (Decode4 nNotSaleCount + Decode2 special + Decode1
//	     discounts), DecodeBuffer(0x438=1080) @0x44e993, then exactly TWO post-buffer
//	     decoders — sub_44F10B DecodeStock @0x44e99d and sub_44F152 DecodeLimitGoods
//	     @0x44e9a7 — and NO DecodeZeroGoods. The ctor then reads a SINGLE Decode1
//	     bEventOn (this[316] @0x447249) — NO Decode4 nHighest.
//	v61  CStage::OnSetCashShop 0x65a973 -> CCashShop::CCashShop 0x453549:
//	     CCashShop::LoadData 0x45b539 reads Decode1 auth, DecodeStr account,
//	     CWvsContext::SetSaleInfo 0x8474e6, DecodeBuffer(1080), then sub_45C497
//	     DecodeStock and sub_45C4DE DecodeLimitGoods — NO DecodeZeroGoods. The ctor
//	     reads a SINGLE Decode1 bEventOn (this[317] @0x4536d4) — NO Decode4 nHighest.
//
// The writer gates DecodeZeroGoods + nHighest on GMS MajorAtLeast(72), so the v48/v61
// (major < 72) encode is the v83 modern encode with those 6 bytes removed. This test
// asserts (a) the exact 17-byte legacy tail (ending at bEventOn — no ZeroGoods, no
// nHighest), (b) the encode is 6 bytes shorter than v83, and (c) every byte through
// DecodeLimitGoods is byte-identical to the modern v83 encode.
//
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v48 ida=0x5c4d9c
// packet-audit:verify packet=cash/clientbound/CashShopOpen version=gms_v61 ida=0x65a973
func TestCashShopOpenLegacyBodyV48V61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// Legacy trailing 17 bytes: last Best triple (8,1,50000047), DecodeStock 0,
	// DecodeLimitGoods 0, bEventOn false. NO DecodeZeroGoods, NO nHighest.
	wantTail := []byte{
		0x08, 0x00, 0x00, 0x00, // Best i=8
		0x01, 0x00, 0x00, 0x00, // Best j=1
		0xAF, 0xF0, 0xFA, 0x02, // Best sn 50000047
		0x00, 0x00, // DecodeStock count 0
		0x00, 0x00, // DecodeLimitGoods count 0
		0x00, // bEventOn false
	}
	// Modern v83 reference (has ZeroGoods 2B + nHighest 4B the legacy body lacks).
	ref := NewCashShopOpen(cashShopTestCharacterData(), "TestAccount").
		Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	for _, major := range []uint16{48, 61} {
		t.Run(fmt.Sprintf("gms_v%d", major), func(t *testing.T) {
			ctx := pt.CreateContext("GMS", major, 1)
			b := NewCashShopOpen(cashShopTestCharacterData(), "TestAccount").Encode(l, ctx)(nil)
			if len(b) < len(wantTail) {
				t.Fatalf("buffer too short: %d bytes", len(b))
			}
			if tail := b[len(b)-len(wantTail):]; !bytes.Equal(tail, wantTail) {
				t.Errorf("v%d legacy tail mismatch:\n got %v\nwant %v", major, tail, wantTail)
			}
			// Legacy body is exactly 6 bytes shorter than the modern v83 body.
			if len(b) != len(ref)-6 {
				t.Errorf("v%d length %d, want %d (v83 %d minus ZeroGoods 2B + nHighest 4B)", major, len(b), len(ref)-6, len(ref))
			}
			// Everything through DecodeLimitGoods must equal the modern v83 encode.
			if !bytes.Equal(b[:len(b)-1], ref[:len(b)-1]) {
				t.Errorf("v%d prefix through DecodeLimitGoods diverges from modern v83 body", major)
			}
		})
	}
}
