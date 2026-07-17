package clientbound

import (
	"bytes"
	"fmt"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// SET_ITC (CStage::OnSetITC) — the MTS/ITC scene-transition packet. The wire
// body is the full migrate-in CharacterData block (the SAME block CashShopOpen /
// CStage::OnSetCashShop encodes), then the account name (ZXString), then the
// five ITC config int32s, then an 8-byte server-now FILETIME (the ITC clock
// sync, not a date — see the SetItcWriter doc).
//
// IDA read order is byte-identical in every version (the body reader is
// CITC::LoadData / sub_59EF9D / sub_5AF339; the per-version client-side account
// display formatting around the single DecodeStr does not change the wire). The
// CStage::OnSetITC read-site addresses below are pinned as packet-audit:verify
// machine markers on the golden byte-tests in this file; each carries a fresh
// evidence record (docs/packets/evidence/<version>/field.clientbound.SetItc.yaml)
// keyed to the export function CStage::OnSetITC, which promotes SET_ITC to ✅ in
// the coverage matrix for every version it is implemented in (task-113):
//
//	version   OnSetITC     CITC::LoadData / body reader
//	gms_v61   0x65b3b4     (CStage::OnPacket case ']'(93))
//	gms_v72   0x6c2145     (CStage::OnPacket case 's'(115))
//	gms_v79   0x6f1c4a     (CStage::OnPacket @0x6f079f case 'w'(119))
//	gms_v83   0x7774d1     sub_59EF9D  0x59ef9d
//	gms_v84   0x799e7a     sub_5AF339  0x5af339   (via CITC ctor sub_5AE011)
//	gms_v87   0x7c57d0     0x5ced61               (via CITC::CITC 0x5cd970)
//	gms_v95   0x71af60     0x574a60               (via CITC::CITC 0x574d00; named fields)
//	jms_v185  0x7ef6fa     0x60448e               (via CITC::CITC 0x60311a)

// mtsTestCharacterData builds a deterministic CharacterData block matching the
// CashShopOpen test fixture so the byte layout of the leading block is shared
// and known-good.
func mtsTestCharacterData() charpkt.CharacterData {
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

// TestSetItcDefaultsGolden asserts the trailing config block — the part this
// writer owns beyond the reused CharacterData envelope. The CharacterData block
// is variable-length, so we anchor the assertion at the END of the buffer:
// the last 8 bytes are the server-now FILETIME, preceded by the five LE int32s,
// preceded by the account-name ZXString. The Cosmic-faithful defaults are the
// IDA-confirmed five Decode4 values (5000/7/500/24/168); the 8-byte server-time
// value is passed explicitly by the test.
func TestSetItcDefaultsGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 0)
	serverTime := [8]byte{0x70, 0xAA, 0xA7, 0xC5, 0x4E, 0xC1, 0xCA, 0x01}
	input := NewSetItc(mtsTestCharacterData(), "AC", serverTime)
	b := input.Encode(l, ctx)(nil)

	// Trailing 28 bytes = accountName ZXString (2 len + "AC") + 5×int32 + 8-byte server-now FILETIME.
	//   "AC": 02 00 41 43
	//   listingFee 5000   = 0x1388 -> 88 13 00 00
	//   commissionRate 7         -> 07 00 00 00
	//   commissionBase 500 = 0x1F4 -> F4 01 00 00
	//   auctionMin 24      = 0x18  -> 18 00 00 00
	//   auctionMax 168     = 0xA8  -> A8 00 00 00
	//   serverTime               -> 70 AA A7 C5 4E C1 CA 01 (the explicit arg above)
	wantTail := []byte{
		0x02, 0x00, 0x41, 0x43, // "AC"
		0x88, 0x13, 0x00, 0x00, // listingFee 5000
		0x07, 0x00, 0x00, 0x00, // commissionRate 7
		0xF4, 0x01, 0x00, 0x00, // commissionBase 500
		0x18, 0x00, 0x00, 0x00, // auctionMin 24
		0xA8, 0x00, 0x00, 0x00, // auctionMax 168
		0x70, 0xAA, 0xA7, 0xC5, 0x4E, 0xC1, 0xCA, 0x01, // server-now FILETIME
	}
	if len(b) < len(wantTail) {
		t.Fatalf("buffer too short: %d bytes", len(b))
	}
	tail := b[len(b)-len(wantTail):]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("trailing config block mismatch:\n got %v\nwant %v", tail, wantTail)
	}
}

// TestSetItcExplicitConfigGolden asserts the trailing block with explicit,
// distinct ITC config values so each of the five int32 positions and the date
// buffer are independently pinned (no value aliasing with the defaults).
func TestSetItcExplicitConfigGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 0)
	date := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	input := NewSetItcWithConfig(mtsTestCharacterData(), "AC",
		0x11223344, 0x55667788, 0x99AABBCC, 0x0D0E0F10, 0x21222324, date)
	b := input.Encode(l, ctx)(nil)

	wantTail := []byte{
		0x02, 0x00, 0x41, 0x43, // "AC"
		0x44, 0x33, 0x22, 0x11, // listingFee 0x11223344
		0x88, 0x77, 0x66, 0x55, // commissionRate 0x55667788
		0xCC, 0xBB, 0xAA, 0x99, // commissionBase 0x99AABBCC
		0x10, 0x0F, 0x0E, 0x0D, // auctionMin 0x0D0E0F10
		0x24, 0x23, 0x22, 0x21, // auctionMax 0x21222324
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // server-now FILETIME
	}
	tail := b[len(b)-len(wantTail):]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("trailing config block mismatch:\n got %v\nwant %v", tail, wantTail)
	}
}

// TestSetItcRoundTrip proves the full body (CharacterData envelope + account
// name + 5×int32 + 8-byte date) round-trips byte-exactly across every variant
// (gms_v83/v84/v87/v95/jms). The body is version-stable per the IDA addresses
// cited at the top of this file.
//
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v83 ida=0x7774d1
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v84 ida=0x799e7a
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v87 ida=0x7c57d0
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v95 ida=0x71af60
// packet-audit:verify packet=field/clientbound/SetItc version=jms_v185 ida=0x7ef6fa
func TestSetItcRoundTrip(t *testing.T) {
	date := [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x11, 0x22, 0x33, 0x44}
	input := NewSetItcWithConfig(mtsTestCharacterData(), "TestAccount",
		5000, 7, 500, 24, 168, date)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := SetItc{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterData().Stats.Id != input.CharacterData().Stats.Id {
				t.Errorf("stats id: got %v, want %v", output.CharacterData().Stats.Id, input.CharacterData().Stats.Id)
			}
			if output.AccountName() != input.AccountName() {
				t.Errorf("accountName: got %q, want %q", output.AccountName(), input.AccountName())
			}
			if output.ListingFee() != input.ListingFee() {
				t.Errorf("listingFee: got %v, want %v", output.ListingFee(), input.ListingFee())
			}
			if output.CommissionRate() != input.CommissionRate() {
				t.Errorf("commissionRate: got %v, want %v", output.CommissionRate(), input.CommissionRate())
			}
			if output.CommissionBase() != input.CommissionBase() {
				t.Errorf("commissionBase: got %v, want %v", output.CommissionBase(), input.CommissionBase())
			}
			if output.AuctionMinHours() != input.AuctionMinHours() {
				t.Errorf("auctionMinHours: got %v, want %v", output.AuctionMinHours(), input.AuctionMinHours())
			}
			if output.AuctionMaxHours() != input.AuctionMaxHours() {
				t.Errorf("auctionMaxHours: got %v, want %v", output.AuctionMaxHours(), input.AuctionMaxHours())
			}
			if output.ServerTime() != input.ServerTime() {
				t.Errorf("serverTime: got %v, want %v", output.ServerTime(), input.ServerTime())
			}
		})
	}
}

// TestSetItcLegacyGolden pins the trailing SET_ITC config block for the legacy
// GMS versions gms_v79/v72/v61 (which pt.Variants does not enumerate). The body
// reader after the migrate-in CharacterData block is version-agnostic for
// clients ≥ v61 — a single DecodeStr (account name) + five Decode4 ITC config
// ints + an 8-byte DecodeBuffer server-now FILETIME — so the trailing 28 bytes
// are byte-identical to v83's. The assertion is tail-anchored (the CharacterData
// envelope is variable length) exactly like TestSetItcDefaultsGolden.
//
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v79 ida=0x6f1c4a
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v72 ida=0x6c2145
// packet-audit:verify packet=field/clientbound/SetItc version=gms_v61 ida=0x65b3b4
func TestSetItcLegacyGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	serverTime := [8]byte{0x70, 0xAA, 0xA7, 0xC5, 0x4E, 0xC1, 0xCA, 0x01}
	wantTail := []byte{
		0x02, 0x00, 0x41, 0x43, // "AC"
		0x88, 0x13, 0x00, 0x00, // listingFee 5000
		0x07, 0x00, 0x00, 0x00, // commissionRate 7
		0xF4, 0x01, 0x00, 0x00, // commissionBase 500
		0x18, 0x00, 0x00, 0x00, // auctionMin 24
		0xA8, 0x00, 0x00, 0x00, // auctionMax 168
		0x70, 0xAA, 0xA7, 0xC5, 0x4E, 0xC1, 0xCA, 0x01, // server-now FILETIME
	}
	for _, major := range []uint16{79, 72, 61} {
		t.Run(fmt.Sprintf("gms_v%d", major), func(t *testing.T) {
			ctx := pt.CreateContext("GMS", major, 1)
			input := NewSetItc(mtsTestCharacterData(), "AC", serverTime)
			b := input.Encode(l, ctx)(nil)
			if len(b) < len(wantTail) {
				t.Fatalf("buffer too short: %d bytes", len(b))
			}
			tail := b[len(b)-len(wantTail):]
			if !bytes.Equal(tail, wantTail) {
				t.Errorf("v%d trailing config block mismatch:\n got %v\nwant %v", major, tail, wantTail)
			}
		})
	}
}
