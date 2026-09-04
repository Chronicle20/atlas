package serverbound

import (
	"bytes"
	"context"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Wire fixtures. Every multi-byte field is a little-endian i32 (Encode4 in the
// client); only bCatalystMounted is a single byte (Encode1). Derived in
// docs/tasks/task-285-maker-skill-crafting/wire-derivation.md, IDENTICAL on all
// eight applicable versions.
var (
	// mode 1, targetItemId 1082002 (0x108292), catalyst mounted, 2 gems
	// (4021313 = 0x3D5C41, 4021314 = 0x3D5C42).
	makerSkillCreateBytes = []byte{
		0x01, 0x00, 0x00, 0x00, // nRecipeClass = 1
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x01,                   // bCatalystMounted = true
		0x02, 0x00, 0x00, 0x00, // nNumGemMounted = 2
		0x41, 0x5C, 0x3D, 0x00, // nGemItemID[0] = 4021313
		0x42, 0x5C, 0x3D, 0x00, // nGemItemID[1] = 4021314
	}
	// mode 2, no catalyst, zero gems.
	makerSkillCreateWithUpgradeBytes = []byte{
		0x02, 0x00, 0x00, 0x00, // nRecipeClass = 2
		0x92, 0x82, 0x10, 0x00, // nTargetItemID = 1082002
		0x00,                   // bCatalystMounted = false
		0x00, 0x00, 0x00, 0x00, // nNumGemMounted = 0
	}
	// mode 3, leftover/recipe item 4000000 (0x3D0900).
	makerSkillMonsterCrystalBytes = []byte{
		0x03, 0x00, 0x00, 0x00, // nRecipeClass = 3
		0x00, 0x09, 0x3D, 0x00, // nRecipeItemID = 4000000
	}
	// mode 4, item 1082002 in inventory type 1, slot 5.
	makerSkillDisassembleBytes = []byte{
		0x04, 0x00, 0x00, 0x00, // nRecipeClass = 4
		0x92, 0x82, 0x10, 0x00, // nRecipeItemID = 1082002
		0x01, 0x00, 0x00, 0x00, // nTI_DisassembleItem = 1
		0x05, 0x00, 0x00, 0x00, // nSlotPosition_DisassembleItem = 5
	}
)

func decodeMakerSkill(t *testing.T, ctx context.Context, in []byte) MakerSkill {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(in)
	r := request.NewRequestReader(&req, 0)
	var m MakerSkill
	m.Decode(l, ctx)(&r, nil)
	if r.Available() != 0 {
		t.Errorf("decode left %d unconsumed bytes", r.Available())
	}
	return m
}

func TestMakerSkillDecodeCreate(t *testing.T) {
	m := decodeMakerSkill(t, pt.CreateContext("GMS", 95, 1), makerSkillCreateBytes)
	if m.Mode() != 1 {
		t.Errorf("Mode() = %d, want 1", m.Mode())
	}
	if m.TargetItemId() != 1082002 {
		t.Errorf("TargetItemId() = %d, want 1082002", m.TargetItemId())
	}
	if !m.UseCatalyst() {
		t.Error("UseCatalyst() = false, want true")
	}
	gems := m.GemItemIds()
	if len(gems) != 2 {
		t.Fatalf("len(GemItemIds()) = %d, want 2", len(gems))
	}
	if gems[0] != 4021313 {
		t.Errorf("GemItemIds()[0] = %d, want 4021313", gems[0])
	}
	if gems[1] != 4021314 {
		t.Errorf("GemItemIds()[1] = %d, want 4021314", gems[1])
	}
}

func TestMakerSkillDecodeCreateWithUpgrade(t *testing.T) {
	m := decodeMakerSkill(t, pt.CreateContext("GMS", 95, 1), makerSkillCreateWithUpgradeBytes)
	if m.Mode() != 2 {
		t.Errorf("Mode() = %d, want 2", m.Mode())
	}
	if m.TargetItemId() != 1082002 {
		t.Errorf("TargetItemId() = %d, want 1082002", m.TargetItemId())
	}
	if m.UseCatalyst() {
		t.Error("UseCatalyst() = true, want false")
	}
	if len(m.GemItemIds()) != 0 {
		t.Errorf("len(GemItemIds()) = %d, want 0", len(m.GemItemIds()))
	}
}

func TestMakerSkillDecodeMonsterCrystal(t *testing.T) {
	m := decodeMakerSkill(t, pt.CreateContext("GMS", 95, 1), makerSkillMonsterCrystalBytes)
	if m.Mode() != 3 {
		t.Errorf("Mode() = %d, want 3", m.Mode())
	}
	if m.LeftoverItemId() != 4000000 {
		t.Errorf("LeftoverItemId() = %d, want 4000000", m.LeftoverItemId())
	}
	// No mode-1 field may leak into a mode-3 request.
	if len(m.GemItemIds()) != 0 {
		t.Errorf("len(GemItemIds()) = %d, want 0", len(m.GemItemIds()))
	}
	if m.TargetItemId() != 0 {
		t.Errorf("TargetItemId() = %d, want 0", m.TargetItemId())
	}
}

func TestMakerSkillDecodeDisassemble(t *testing.T) {
	m := decodeMakerSkill(t, pt.CreateContext("GMS", 95, 1), makerSkillDisassembleBytes)
	if m.Mode() != 4 {
		t.Errorf("Mode() = %d, want 4", m.Mode())
	}
	if m.ItemId() != 1082002 {
		t.Errorf("ItemId() = %d, want 1082002", m.ItemId())
	}
	if m.InventoryType() != 1 {
		t.Errorf("InventoryType() = %d, want 1", m.InventoryType())
	}
	if m.SlotPos() != 5 {
		t.Errorf("SlotPos() = %d, want 5", m.SlotPos())
	}
}

// TestMakerSkillDecodeOutOfRangeModeReadsNoBody covers the client's
// `if (v4 > 0)` / `switch default: break` guard: a mode outside 1..4 carries no
// body at all, so no arm field may be populated.
func TestMakerSkillDecodeOutOfRangeModeReadsNoBody(t *testing.T) {
	m := decodeMakerSkill(t, pt.CreateContext("GMS", 95, 1), []byte{0x05, 0x00, 0x00, 0x00})
	if m.Mode() != 5 {
		t.Errorf("Mode() = %d, want 5", m.Mode())
	}
	if m.TargetItemId() != 0 || m.LeftoverItemId() != 0 || m.ItemId() != 0 || m.InventoryType() != 0 || m.SlotPos() != 0 || len(m.GemItemIds()) != 0 {
		t.Errorf("out-of-range mode populated an arm field: %s", m.String())
	}
}

func makerSkillFixture(mode uint32) (MakerSkill, []byte) {
	switch mode {
	case 1:
		return NewMakerSkill(1, 1082002, true, []uint32{4021313, 4021314}, 0, 0, 0, 0), makerSkillCreateBytes
	case 2:
		return NewMakerSkill(2, 1082002, false, nil, 0, 0, 0, 0), makerSkillCreateWithUpgradeBytes
	case 3:
		return NewMakerSkill(3, 0, false, nil, 4000000, 0, 0, 0), makerSkillMonsterCrystalBytes
	default:
		return NewMakerSkill(4, 0, false, nil, 0, 1082002, 1, 5), makerSkillDisassembleBytes
	}
}

func assertMakerSkillEqual(t *testing.T, got MakerSkill, want MakerSkill) {
	t.Helper()
	if got.Mode() != want.Mode() {
		t.Errorf("Mode() = %d, want %d", got.Mode(), want.Mode())
	}
	if got.TargetItemId() != want.TargetItemId() {
		t.Errorf("TargetItemId() = %d, want %d", got.TargetItemId(), want.TargetItemId())
	}
	if got.UseCatalyst() != want.UseCatalyst() {
		t.Errorf("UseCatalyst() = %t, want %t", got.UseCatalyst(), want.UseCatalyst())
	}
	gg, wg := got.GemItemIds(), want.GemItemIds()
	if len(gg) != len(wg) {
		t.Fatalf("len(GemItemIds()) = %d, want %d", len(gg), len(wg))
	}
	for i := range wg {
		if gg[i] != wg[i] {
			t.Errorf("GemItemIds()[%d] = %d, want %d", i, gg[i], wg[i])
		}
	}
	if got.LeftoverItemId() != want.LeftoverItemId() {
		t.Errorf("LeftoverItemId() = %d, want %d", got.LeftoverItemId(), want.LeftoverItemId())
	}
	if got.ItemId() != want.ItemId() {
		t.Errorf("ItemId() = %d, want %d", got.ItemId(), want.ItemId())
	}
	if got.InventoryType() != want.InventoryType() {
		t.Errorf("InventoryType() = %d, want %d", got.InventoryType(), want.InventoryType())
	}
	if got.SlotPos() != want.SlotPos() {
		t.Errorf("SlotPos() = %d, want %d", got.SlotPos(), want.SlotPos())
	}
}

// TestMakerSkillEncodeDecodeRoundTripPerMode proves Encode and Decode are
// field-for-field mirrors in every arm.
func TestMakerSkillEncodeDecodeRoundTripPerMode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	for _, mode := range []uint32{1, 2, 3, 4} {
		in, _ := makerSkillFixture(mode)
		encoded := pt.Encode(t, ctx, in.Encode, nil)
		out := decodeMakerSkill(t, ctx, encoded)
		assertMakerSkillEqual(t, out, in)
	}
}

// makerSkillVariants are the eight versions on which MAKER_SKILL exists.
// gms_v48 and gms_v61 predate the maker skill and carry no opcode.
var makerSkillVariants = []struct {
	index int
	name  string
}{
	{9, "GMS v72"},
	{10, "GMS v79"},
	{1, "GMS v83"},
	{5, "GMS v84"},
	{2, "GMS v87"},
	{11, "GMS v92"},
	{3, "GMS v95"},
	{4, "JMS v185"},
}

// TestMakerSkillDecodeIsVersionInvariant is the executable form of the C-2
// verdict in wire-derivation.md: the layout is identical on all eight versions,
// so no version gate exists in the codec.
func TestMakerSkillDecodeIsVersionInvariant(t *testing.T) {
	for _, mv := range makerSkillVariants {
		v := pt.Variants[mv.index]
		if v.Name != mv.name {
			t.Fatalf("pt.Variants[%d] = %q, want %q (index drifted)", mv.index, v.Name, mv.name)
		}
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		m := decodeMakerSkill(t, ctx, makerSkillCreateBytes)
		if m.Mode() != 1 || m.TargetItemId() != 1082002 || !m.UseCatalyst() || len(m.GemItemIds()) != 2 || m.GemItemIds()[0] != 4021313 || m.GemItemIds()[1] != 4021314 {
			t.Errorf("%s: mode-1 decode diverged: %s", v.Name, m.String())
		}
	}
}

// assertMakerSkillBytes encodes all four arms under the given variant and
// asserts the exact wire bytes.
func assertMakerSkillBytes(t *testing.T, index int, name string) {
	t.Helper()
	v := pt.Variants[index]
	if v.Name != name {
		t.Fatalf("pt.Variants[%d] = %q, want %q (index drifted)", index, v.Name, name)
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	for _, mode := range []uint32{1, 2, 3, 4} {
		in, expected := makerSkillFixture(mode)
		actual := pt.Encode(t, ctx, in.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("%s mode %d byte output mismatch: got %v want %v", name, mode, actual, expected)
		}
	}
}

// IDA evidence (session 99e435d8, GMS_v72.1_U_DEVM.exe.i64):
// CUIItemMaker::RequestItemMake@0x760cc3 - COutPacket(0x70), then a single
// in-arm Encode4(mode) at 0x760de7 (mode 1/2), 0x760dcd (mode 3), 0x760d8c
// (mode 4); mode-1/2 body Encode4(nTargetItemID)@0x760df5,
// Encode1(bCatalystMounted)@0x760e06, Encode4(nNumGemMounted)@0x760e14 and the
// gem loop Encode4@0x760e39.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v72 ida=0x760cc3
func TestMakerSkillByteOutputV72(t *testing.T) { assertMakerSkillBytes(t, 9, "GMS v72") }

// IDA evidence (session 5a1cd4f3, GMS_v79_1_DEVM.exe.i64):
// CUIItemMaker::RequestItemMake@0x795dc3, one in-arm mode Encode4 at 0x795ee6 /
// 0x795ecc / 0x795e8b; arm bodies identical to v72 (wire-derivation.md §3).
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v79 ida=0x795dc3
func TestMakerSkillByteOutputV79(t *testing.T) { assertMakerSkillBytes(t, 10, "GMS v79") }

// IDA evidence (session 754107bf, MapleStory_dump.exe.i64 v83_Me):
// CUIItemMaker::RequestItemMake@0x827096, one in-arm mode Encode4 at 0x8271b9 /
// 0x82719f / 0x82715e; arm bodies identical to v72.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v83 ida=0x827096
func TestMakerSkillByteOutputV83(t *testing.T) { assertMakerSkillBytes(t, 1, "GMS v83") }

// IDA evidence (session 46c2a2eb, GMS_v84.1_U_DEVM.i64): the v84 IDB carries no
// CUIItemMaker symbols; sub_8524B7 was identified structurally (it ends exactly
// at OnItemMakeResult sub_852685, the same adjacency as v83) and confirmed by
// decompilation - same four guards, COutPacket(113) matching the registry
// opcode, one in-arm mode Encode4 at 0x8525da / 0x8525c0 / 0x85257f.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v84 ida=0x8524b7
func TestMakerSkillByteOutputV84(t *testing.T) { assertMakerSkillBytes(t, 5, "GMS v84") }

// IDA evidence (session c0829805, GMSv87_4GB.exe.i64):
// CUIItemMaker::RequestItemMake@0x88afd1, one in-arm mode Encode4 at 0x88b0f4 /
// 0x88b0da / 0x88b099; the mode-3 arm encodes literal 3 then one item id, the
// mode-4 arm literal 4 then item id, inventory type, slot position.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v87 ida=0x88afd1
func TestMakerSkillByteOutputV87(t *testing.T) { assertMakerSkillBytes(t, 2, "GMS v87") }

// IDA evidence (session 019cd393, GMS_v92_1_DEVM.exe.i64): no CUIItemMaker
// symbols; sub_7AFDC0 identified structurally from OnItemMakeResult sub_7AF6E0
// plus the v95 delta 0x6e0, size 0x20a matching v95, and confirmed by
// decompilation with COutPacket(0x7C) = registry opcode 124. Renders as a
// switch with `default: break` - wire-identical to the v72 if-chain.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v92 ida=0x7afdc0
func TestMakerSkillByteOutputV92(t *testing.T) { assertMakerSkillBytes(t, 11, "GMS v92") }

// IDA evidence (session ecc757f4, GMS_v95.0_U_DEVM.exe.i64): the reference
// version. CUIItemMaker::RequestItemMake@0x7d58d0 is fully typed, naming
// m_nRecipeClass, m_nTargetItem, m_CatalystSlot.bMounted, m_nNumGem_Mounted,
// m_nTI_DisassbleItem and m_nSlotPosition_DisassbleItem.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=gms_v95 ida=0x7d58d0
func TestMakerSkillByteOutputV95(t *testing.T) { assertMakerSkillBytes(t, 3, "GMS v95") }

// IDA evidence (session a977912e, MapleStory_dump_SCY.exe.i64):
// CUIItemMaker::RequestItemMake@0x8b1040, COutPacket(0x6C)@0x8b10db = registry
// opcode 108, one in-arm mode Encode4 at 0x8b1163 / 0x8b1149 / 0x8b1108.
//
// packet-audit:verify packet=character/serverbound/MakerSkill version=jms_v185 ida=0x8b1040
func TestMakerSkillByteOutputJMS185(t *testing.T) { assertMakerSkillBytes(t, 4, "JMS v185") }
