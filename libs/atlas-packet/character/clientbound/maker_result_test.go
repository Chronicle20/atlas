package clientbound

import (
	"bytes"
	"context"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// makerResultVersions is the eight versions on which MAKER_RESULT exists. The
// wire layout is field-for-field IDENTICAL on all eight — see the §3 per-version
// read-order address tables and the §4 verdict table in
// docs/tasks/task-285-maker-skill-crafting/wire-derivation.md — so every arm is
// asserted against ONE shared expectation ranged over the eight variants. Any
// divergence would show up as a byte mismatch on exactly the diverging column.
//
// idx pins the pt.Variants slot; the guard on v.Name catches index drift.
// ida is that version's CUserLocal::OnMakerResult entry address.
//
// Marker placement: ALL FOUR mode-carrying arms (Create, CreateWithUpgrade,
// MonsterCrystal, Disassemble) carry packet-audit verify markers on every
// applicable version, each backed by its own pinned evidence record and its own
// audit report generated from that version's CUserLocal::OnMakerResult#<Arm>
// export entry. That is the full-arm coverage a mode-prefix dispatcher needs
// before its op row may read ✅ (docs/packets/evidence/families.yaml): the op
// row grades worst-of across every writer sharing the base fname, so an
// unverified arm would hold the whole row down rather than hide behind a
// verified sibling.
//
// MakerResultFailed is deliberately excluded and has NO marker, NO evidence
// record and NO export entry: it writes no mode field at all (the client
// returns at nResult > 1, before the nMode Decode4), so it is not a mode arm.
// See docs/tasks/task-285-maker-skill-crafting/ruling-failed-arm.md. It keeps
// its byte fixture and its Encode/Decode round-trip below.
var makerResultVersions = []struct {
	idx  int
	name string
	ida  string
}{
	{9, "GMS v72", "0x86a152"},
	{10, "GMS v79", "0x8b5af5"},
	{1, "GMS v83", "0x95dad3"},
	{5, "GMS v84", "0x99bdbc"},
	{2, "GMS v87", "0x9e01b2"},
	{11, "GMS v92", "0x8f5d70"},
	{3, "GMS v95", "0x9102f0"},
	{4, "JMS v185", "0xa29527"},
}

func makerResultContext(t *testing.T, i int) context.Context {
	t.Helper()
	c := makerResultVersions[i]
	v := pt.Variants[c.idx]
	if v.Name != c.name {
		t.Fatalf("pt.Variants[%d] = %q, want %q (index drifted)", c.idx, v.Name, c.name)
	}
	return pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
}

// Wire fixtures. Every field is a little-endian i32 (Decode4 in the client)
// except bNoItemGain and bUsedCatalyst, which are single bytes (Decode1).
// Item ids: 1082002 = 0x108292, 4011001 = 0x3D33F9, 4011002 = 0x3D33FA,
// 4021313 = 0x3D5C41, 4130000 = 0x3F04D0, 4000000 = 0x3D0900,
// 4000001 = 0x3D0901.
var (
	// nResult 0, nMode 1, item awarded, two materials, one gem, catalyst used,
	// 1200 meso charged.
	makerResultCreateBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x01, 0x00, 0x00, 0x00, // nMode = 1
		0x00,                   // bNoItemGain = 0 -> the pair FOLLOWS
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x01, 0x00, 0x00, 0x00, // nItemCount = 1
		0x02, 0x00, 0x00, 0x00, // nNumUsedItem = 2
		0xF9, 0x33, 0x3D, 0x00, // used[0].nItemID = 4011001
		0x05, 0x00, 0x00, 0x00, // used[0].nCount  = 5
		0xFA, 0x33, 0x3D, 0x00, // used[1].nItemID = 4011002
		0x03, 0x00, 0x00, 0x00, // used[1].nCount  = 3
		0x01, 0x00, 0x00, 0x00, // nNumUsedGem = 1
		0x41, 0x5C, 0x3D, 0x00, // gem[0].nItemID = 4021313
		0x01,                   // bUsedCatalyst = 1
		0xD0, 0x04, 0x3F, 0x00, // nCatalystItemID = 4130000
		0xB0, 0x04, 0x00, 0x00, // nMesoCost = 1200
	}
	// The inverted-flag regression fixture: bNoItemGain = 1 SUPPRESSES the
	// nTargetItemID/nItemCount pair, so nNumUsedItem follows the flag byte
	// directly (client reads the pair only when the byte is ZERO —
	// gms_v95 Decode1 @0x910717, `if ( !… )` guard).
	makerResultCreateNoItemBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x01, 0x00, 0x00, 0x00, // nMode = 1
		0x01,                   // bNoItemGain = 1 -> the pair is SUPPRESSED
		0x00, 0x00, 0x00, 0x00, // nNumUsedItem = 0
		0x00, 0x00, 0x00, 0x00, // nNumUsedGem = 0
		0x00,                   // bUsedCatalyst = 0
		0xB0, 0x04, 0x00, 0x00, // nMesoCost = 1200
	}
	// bUsedCatalyst = 0 suppresses nCatalystItemID, so nMesoCost follows the
	// flag byte directly (gms_v95 Decode1 @0x9109c6 gating Decode4 @0x9109d8).
	makerResultCreateNoCatalystBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x01, 0x00, 0x00, 0x00, // nMode = 1
		0x00,                   // bNoItemGain = 0
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x01, 0x00, 0x00, 0x00, // nItemCount = 1
		0x00, 0x00, 0x00, 0x00, // nNumUsedItem = 0
		0x00, 0x00, 0x00, 0x00, // nNumUsedGem = 0
		0x00,                   // bUsedCatalyst = 0 -> nCatalystItemID SUPPRESSED
		0xB0, 0x04, 0x00, 0x00, // nMesoCost = 1200
	}
	// Byte-identical to makerResultCreateBytes except nMode = 2.
	makerResultCreateWithUpgradeBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x02, 0x00, 0x00, 0x00, // nMode = 2
		0x00,                   // bNoItemGain = 0
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x01, 0x00, 0x00, 0x00, // nItemCount = 1
		0x02, 0x00, 0x00, 0x00, // nNumUsedItem = 2
		0xF9, 0x33, 0x3D, 0x00, // used[0].nItemID = 4011001
		0x05, 0x00, 0x00, 0x00, // used[0].nCount  = 5
		0xFA, 0x33, 0x3D, 0x00, // used[1].nItemID = 4011002
		0x03, 0x00, 0x00, 0x00, // used[1].nCount  = 3
		0x01, 0x00, 0x00, 0x00, // nNumUsedGem = 1
		0x41, 0x5C, 0x3D, 0x00, // gem[0].nItemID = 4021313
		0x01,                   // bUsedCatalyst = 1
		0xD0, 0x04, 0x3F, 0x00, // nCatalystItemID = 4130000
		0xB0, 0x04, 0x00, 0x00, // nMesoCost = 1200
	}
	// nMode 3: exactly two Decode4 reads and NO meso field.
	makerResultMonsterCrystalBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x03, 0x00, 0x00, 0x00, // nMode = 3
		0x00, 0x09, 0x3D, 0x00, // nTargetItemID = 4000000 (crystal produced)
		0x01, 0x09, 0x3D, 0x00, // nSourceItemID = 4000001 (item consumed)
	}
	// nMode 4: two reward stacks, 500 meso charged.
	makerResultDisassembleBytes = []byte{
		0x00, 0x00, 0x00, 0x00, // nResult = 0
		0x04, 0x00, 0x00, 0x00, // nMode = 4
		0x92, 0x82, 0x10, 0x00, // nDisassembledItemID = 1082002
		0x02, 0x00, 0x00, 0x00, // nNumRewardItem = 2
		0x00, 0x09, 0x3D, 0x00, // reward[0].nItemID = 4000000
		0x03, 0x00, 0x00, 0x00, // reward[0].nCount  = 3
		0x01, 0x09, 0x3D, 0x00, // reward[1].nItemID = 4000001
		0x01, 0x00, 0x00, 0x00, // reward[1].nCount  = 1
		0xF4, 0x01, 0x00, 0x00, // nMesoCost = 500
	}
	// nResult = 2 (> 1): the client stops before the nMode Decode4, so the
	// packet is exactly four bytes. Writing a mode here would desynchronise it.
	makerResultFailedBytes = []byte{
		0x02, 0x00, 0x00, 0x00, // nResult = 2
	}
)

func newMakerResultCreateFixture() MakerResultCreate {
	return NewMakerResultCreate(1, 0, false, 1082002, 1,
		[]MakerMaterial{NewMakerMaterial(4011001, 5), NewMakerMaterial(4011002, 3)},
		[]uint32{4021313}, true, 4130000, 1200)
}

func newMakerResultCreateWithUpgradeFixture() MakerResultCreateWithUpgrade {
	return NewMakerResultCreateWithUpgrade(2, 0, false, 1082002, 1,
		[]MakerMaterial{NewMakerMaterial(4011001, 5), NewMakerMaterial(4011002, 3)},
		[]uint32{4021313}, true, 4130000, 1200)
}

func assertMakerResultBytes(t *testing.T, ctx context.Context, actual []byte, expected []byte) {
	t.Helper()
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch:\n got %v\nwant %v", actual, expected)
	}
}

// decodeInto runs a Decode closure over in and fails if bytes are left over.
func decodeInto(t *testing.T, ctx context.Context, in []byte, decode func(*request.Reader, map[string]interface{})) {
	t.Helper()
	req := request.Request(in)
	r := request.NewRequestReader(&req, 0)
	decode(&r, nil)
	if r.Available() != 0 {
		t.Errorf("decode left %d unconsumed bytes", r.Available())
	}
}

// TestMakerResultCreateByteOutput verifies the wire-exact byte output of the
// mode-1 CREATE arm on every applicable version.
//
// IDA evidence (wire-derivation.md §3, mode 1/2 row per version): after the
// i32 nResult and the i32 nMode, the client reads Decode1 bNoItemGain, the
// conditional (nTargetItemID, nItemCount) pair, the nNumUsedItem loop of
// (id, count) pairs, the nNumUsedGem id-only loop, Decode1 bUsedCatalyst with
// its conditional nCatalystItemID, and finally nMesoCost. gms_v95 reference
// addresses: 0x910717 / 0x910732 / 0x910746 / 0x91080e / 0x91082a / 0x91083d /
// 0x9108ef / 0x910904 / 0x9109c6 / 0x9109d8 / 0x910aa2.
//
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v72 ida=0x86a152
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v79 ida=0x8b5af5
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v83 ida=0x95dad3
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v84 ida=0x99bdbc
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v87 ida=0x9e01b2
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v92 ida=0x8f5d70
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=gms_v95 ida=0x9102f0
// packet-audit:verify packet=character/clientbound/MakerResultCreate version=jms_v185 ida=0xa29527
func TestMakerResultCreateByteOutput(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := newMakerResultCreateFixture()
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, m.Encode, nil), makerResultCreateBytes)

			// Round-trip: Decode is the field-for-field mirror of Encode.
			var d MakerResultCreate
			decodeInto(t, ctx, makerResultCreateBytes, d.Decode(nil, ctx))
			if d.Mode() != 1 || d.Result() != 0 || d.NoItemAwarded() ||
				d.TargetItemId() != 1082002 || d.ItemNum() != 1 ||
				len(d.Materials()) != 2 || d.Materials()[1].ItemId() != 4011002 ||
				d.Materials()[1].Count() != 3 || len(d.GemItemIds()) != 1 ||
				d.GemItemIds()[0] != 4021313 || !d.CatalystUsed() ||
				d.CatalystItemId() != 4130000 || d.MesoCost() != 1200 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultCreateBytes)
		})
	}
}

// TestMakerResultCreateNoItemAwarded is the inverted-flag regression test:
// bNoItemGain is TRUTHY when NO item was awarded, and a truthy byte SUPPRESSES
// the nTargetItemID/nItemCount pair (gms_v95 Decode1 @0x910717 followed by an
// `if ( !… )` guard around the pair at 0x910732 / 0x910746). Encoding the pair
// anyway would shift every following field by eight bytes.
func TestMakerResultCreateNoItemAwarded(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			// The suppressed pair is still populated on the struct — the
			// encoder must drop it on the strength of the flag alone.
			m := NewMakerResultCreate(1, 0, true, 1082002, 1, nil, nil, false, 0, 1200)
			actual := pt.Encode(t, ctx, m.Encode, nil)
			assertMakerResultBytes(t, ctx, actual, makerResultCreateNoItemBytes)
			// The byte after the flag is nNumUsedItem, not nTargetItemID.
			if actual[9] != 0x00 || actual[10] != 0x00 || actual[11] != 0x00 || actual[12] != 0x00 {
				t.Errorf("expected nNumUsedItem directly after bNoItemGain, got % X", actual[9:13])
			}

			var d MakerResultCreate
			decodeInto(t, ctx, makerResultCreateNoItemBytes, d.Decode(nil, ctx))
			if !d.NoItemAwarded() || d.TargetItemId() != 0 || d.ItemNum() != 0 || d.MesoCost() != 1200 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultCreateNoItemBytes)
		})
	}
}

// TestMakerResultCreateNoCatalyst asserts bUsedCatalyst = 0 suppresses
// nCatalystItemID so nMesoCost follows the flag byte directly (gms_v95
// Decode1 @0x9109c6 gating Decode4 @0x9109d8, then Decode4 @0x910aa2).
func TestMakerResultCreateNoCatalyst(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := NewMakerResultCreate(1, 0, false, 1082002, 1, nil, nil, false, 4130000, 1200)
			actual := pt.Encode(t, ctx, m.Encode, nil)
			assertMakerResultBytes(t, ctx, actual, makerResultCreateNoCatalystBytes)

			var d MakerResultCreate
			decodeInto(t, ctx, makerResultCreateNoCatalystBytes, d.Decode(nil, ctx))
			if d.CatalystUsed() || d.CatalystItemId() != 0 || d.MesoCost() != 1200 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultCreateNoCatalystBytes)
		})
	}
}

// TestMakerResultCreateWithUpgradeByteOutput verifies the wire-exact byte
// output of the mode-2 CREATE_WITH_UPGRADE arm on every applicable version.
// The full byte string is asserted (not just the mode) so an edit to one create
// arm that forgets the other is caught here.
//
// IDA evidence: the client's mode dispatch takes modes 1 and 2 through the SAME
// body (gms_v72/79/83/84/92/95 compile it as one switch arm; v87 and jms_v185
// render `if ( v2 == 1 || v2 == 2 )`) — wire-derivation.md §3/§4. The addresses
// are the mode 1/2 row cited on TestMakerResultCreateByteOutput.
//
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v72 ida=0x86a152
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v79 ida=0x8b5af5
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v83 ida=0x95dad3
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v84 ida=0x99bdbc
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v87 ida=0x9e01b2
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v92 ida=0x8f5d70
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=gms_v95 ida=0x9102f0
// packet-audit:verify packet=character/clientbound/MakerResultCreateWithUpgrade version=jms_v185 ida=0xa29527
func TestMakerResultCreateWithUpgradeByteOutput(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := newMakerResultCreateWithUpgradeFixture()
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, m.Encode, nil), makerResultCreateWithUpgradeBytes)

			var d MakerResultCreateWithUpgrade
			decodeInto(t, ctx, makerResultCreateWithUpgradeBytes, d.Decode(nil, ctx))
			if d.Mode() != 2 || d.TargetItemId() != 1082002 || len(d.Materials()) != 2 ||
				len(d.GemItemIds()) != 1 || !d.CatalystUsed() || d.MesoCost() != 1200 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultCreateWithUpgradeBytes)
		})
	}
}

// TestMakerResultCreateArmsAreWireIdenticalExceptMode pins the mode-1/mode-2
// body equivalence: identical inputs must produce identical bytes everywhere
// except the four nMode bytes at offset 4.
func TestMakerResultCreateArmsAreWireIdenticalExceptMode(t *testing.T) {
	ctx := makerResultContext(t, 6) // GMS v95, the reference IDB
	one := pt.Encode(t, ctx, newMakerResultCreateFixture().Encode, nil)
	two := pt.Encode(t, ctx, newMakerResultCreateWithUpgradeFixture().Encode, nil)
	if len(one) != len(two) {
		t.Fatalf("arm lengths differ: %d vs %d", len(one), len(two))
	}
	if !bytes.Equal(one[8:], two[8:]) {
		t.Errorf("create arms diverge after nMode:\n mode1 % X\n mode2 % X", one[8:], two[8:])
	}
	if one[4] != 0x01 || two[4] != 0x02 {
		t.Errorf("nMode bytes = %#x / %#x, want 0x01 / 0x02", one[4], two[4])
	}
}

// TestMakerResultMonsterCrystalByteOutput verifies the wire-exact byte output
// of the mode-3 MONSTER_CRYSTAL arm on every applicable version.
//
// IDA evidence (wire-derivation.md §3, mode-3 columns): exactly two Decode4
// reads after nResult/nMode — nTargetItemID then nSourceItemID. gms_v95
// @0x91037a and @0x91038d; gms_v72 @0x86a1bd / @0x86a1c0; gms_v79 @0x8b5b60 /
// @0x8b5b71; gms_v83 @0x95db3e / @0x95db4f; gms_v84 @0x99be27 / @0x99be38;
// gms_v87 @0x9e021d / @0x9e022e; gms_v92 @0x8f5dfa / @0x8f5e0d; jms_v185
// @0xa29592 / @0xa295a4. Several IDBs assign the second read to a byte-width
// local; the callee is ?Decode4@CInPacket@@QAEKXZ in every case, so four bytes
// are consumed (wire-derivation.md §3 "Note on apparent widths"). There is NO
// meso field on this arm — the 16-byte length assertion below is what proves it.
//
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v72 ida=0x86a152
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v79 ida=0x8b5af5
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v83 ida=0x95dad3
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v84 ida=0x99bdbc
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v87 ida=0x9e01b2
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v92 ida=0x8f5d70
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=gms_v95 ida=0x9102f0
// packet-audit:verify packet=character/clientbound/MakerResultMonsterCrystal version=jms_v185 ida=0xa29527
func TestMakerResultMonsterCrystalByteOutput(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := NewMakerResultMonsterCrystal(3, 0, 4000000, 4000001)
			actual := pt.Encode(t, ctx, m.Encode, nil)
			assertMakerResultBytes(t, ctx, actual, makerResultMonsterCrystalBytes)
			if len(actual) != 16 {
				t.Errorf("encoded length = %d, want 16 (no meso field on mode 3)", len(actual))
			}

			var d MakerResultMonsterCrystal
			decodeInto(t, ctx, makerResultMonsterCrystalBytes, d.Decode(nil, ctx))
			if d.Mode() != 3 || d.Result() != 0 || d.CrystalItemId() != 4000000 || d.LeftoverItemId() != 4000001 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultMonsterCrystalBytes)
		})
	}
}

// TestMakerResultDisassembleByteOutput verifies the wire-exact byte output of
// the mode-4 DISASSEMBLE arm on every applicable version.
//
// IDA evidence (wire-derivation.md §3, mode-4 columns): Decode4
// nDisassembledItemID, Decode4 nNumRewardItem, an (id, count) loop, then
// Decode4 nMesoCost. gms_v95 @0x910516 / @0x91058b / @0x9105a9 / @0x9105bc /
// @0x91068f; gms_v72 @0x86a2f2 / @0x86a35d / @0x86a376 / @0x86a383 / @0x86a409;
// gms_v79 @0x8b5c95 / @0x8b5d00 / @0x8b5d19 / @0x8b5d26 / @0x8b5dac; gms_v83
// @0x95dca1 / @0x95dd0c / @0x95dd25 / @0x95dd32 / @0x95dddb; gms_v84 @0x99bf8a
// / @0x99bff5 / @0x99c00e / @0x99c01b / @0x99c0c4; gms_v87 @0x9e0350 /
// @0x9e03b2 / @0x9e03cb / @0x9e03d8 / @0x9e0478; gms_v92 @0x8f5f96 / @0x8f600b
// / @0x8f6029 / @0x8f603c / @0x8f610f; jms_v185 @0xa296c5 / @0xa29726 /
// @0xa2973f / @0xa2974d / @0xa297ef.
//
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v72 ida=0x86a152
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v79 ida=0x8b5af5
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v83 ida=0x95dad3
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v84 ida=0x99bdbc
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v87 ida=0x9e01b2
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v92 ida=0x8f5d70
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=gms_v95 ida=0x9102f0
// packet-audit:verify packet=character/clientbound/MakerResultDisassemble version=jms_v185 ida=0xa29527
func TestMakerResultDisassembleByteOutput(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := NewMakerResultDisassemble(4, 0, 1082002,
				[]MakerMaterial{NewMakerMaterial(4000000, 3), NewMakerMaterial(4000001, 1)}, 500)
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, m.Encode, nil), makerResultDisassembleBytes)

			var d MakerResultDisassemble
			decodeInto(t, ctx, makerResultDisassembleBytes, d.Decode(nil, ctx))
			if d.Mode() != 4 || d.DisassembledItemId() != 1082002 || len(d.Crystals()) != 2 ||
				d.Crystals()[0].ItemId() != 4000000 || d.Crystals()[0].Count() != 3 ||
				d.Crystals()[1].ItemId() != 4000001 || d.Crystals()[1].Count() != 1 ||
				d.MesoCost() != 500 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultDisassembleBytes)
		})
	}
}

// TestMakerResultFailedByteOutput verifies the wire-exact byte output of the
// bodyless failed arm on every applicable version.
//
// IDA evidence (wire-derivation.md §1): the nResult guard compiles to a pair of
// equality tests, not a magnitude compare — gms_v72 `call Decode4` @0x86a17c,
// `cmp eax, esi` (esi == 0) with `jz` @0x86a185, `cmp eax, 1` with `jnz`
// @0x86a18a, and only then `call Decode4` for nMode @0x86a192. A nResult
// outside {0, 1} therefore skips the mode read entirely and the client falls
// through to CUIItemMaker::OnItemMakeResult(nResult, 0, 0, 0), reading nothing
// further. The four-byte length assertion is what proves no mode leaked out.
func TestMakerResultFailedByteOutput(t *testing.T) {
	for i, c := range makerResultVersions {
		t.Run(c.name, func(t *testing.T) {
			ctx := makerResultContext(t, i)
			m := NewMakerResultFailed(2)
			actual := pt.Encode(t, ctx, m.Encode, nil)
			assertMakerResultBytes(t, ctx, actual, makerResultFailedBytes)
			if len(actual) != 4 {
				t.Errorf("encoded length = %d, want 4 (nResult only, no mode)", len(actual))
			}

			var d MakerResultFailed
			decodeInto(t, ctx, makerResultFailedBytes, d.Decode(nil, ctx))
			if d.Result() != 2 {
				t.Fatalf("round-trip field mismatch: %s", d.String())
			}
			assertMakerResultBytes(t, ctx, pt.Encode(t, ctx, d.Encode, nil), makerResultFailedBytes)
		})
	}
}

// TestMakerResultOperationName pins the shared writer name every arm reports.
func TestMakerResultOperationName(t *testing.T) {
	ops := []string{
		newMakerResultCreateFixture().Operation(),
		newMakerResultCreateWithUpgradeFixture().Operation(),
		NewMakerResultMonsterCrystal(3, 0, 1, 2).Operation(),
		NewMakerResultDisassemble(4, 0, 1, nil, 0).Operation(),
		NewMakerResultFailed(2).Operation(),
	}
	for _, op := range ops {
		if op != MakerResultWriter {
			t.Errorf("Operation() = %q, want %q", op, MakerResultWriter)
		}
	}
}
