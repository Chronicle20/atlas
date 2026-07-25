package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAddEquipmentBytesV72 pins the v72 InventoryAdd equip wire. The InventoryAdd
// framing (handler CWvsContext::OnInventoryOperation@0x917ad0, dispatched from
// OnPacket case 26 @0x9025e4) is version-agnostic vs v79, but the embedded
// model.Asset equip blob is NOT: the v72 equip decode (GW_ItemSlotEquip::RawDecode
// @0x4d0172) reads a single Decode4 (experience) where v79 (@0x4d7ee8) reads two
// (experience + hammersApplied). So the v72 equip is exactly 4 bytes shorter than
// v79. Opaque-family (OPAQUE_LEDGER exception): the equip blob is derived from the
// encoder; here we assert its length delta against the verified v79 encoding.
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v72 ida=0x917ad0
func TestAddEquipmentBytesV72(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	got72 := test.Encode(t, test.CreateContext("GMS", 72, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if len(got72) != len(got79)-4 {
		t.Fatalf("v72 equip len %d, want v79 len %d - 4 (no hammersApplied)\n v72=% X\n v79=% X", len(got72), len(got79), got72, got79)
	}
}

// TestAddEquipmentBytesV61 pins the v61 InventoryAdd equip wire. IDA
// GMS_v61.1_U_DEVM.exe: CWvsContext::OnInventoryOperation@0x8422fc frames the
// packet identically to v79, but the v61 equip decode (GW_ItemSlotEquip::RawDecode
// @0x4b4e7d) has NO levelType/level/experience/hammers trailer at all — after the
// owner+flag it reads only a single 8-byte buffer (non-cash). So the v61 equip is
// 22 bytes shorter than v79 (missing levelType+level+exp+2nd buf+int = 18, and
// hammersApplied = 4). Opaque-family (OPAQUE_LEDGER exception): length delta vs the
// verified v79 encoding.
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v61 ida=0x8422fc
func TestAddEquipmentBytesV61(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	got61 := test.Encode(t, test.CreateContext("GMS", 61, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if len(got61) != len(got79)-22 {
		t.Fatalf("v61 equip len %d, want v79 len %d - 22 (no v72/v79 equip trailer)\n v61=% X\n v79=% X", len(got61), len(got79), got61, got79)
	}
}

// The remaining INVENTORY_OPERATION dispatcher modes are byte-identical between
// GMS v72 and v79 (no version gate on the codec; same handler
// CWvsContext::OnInventoryOperation@0x917ad0). Each v72 cell is proven by
// equality with the verified v79 encoding of the same input.
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v72 ida=0x917ad0
func TestChangeMoveBytesV72(t *testing.T) {
	input := NewChangeMove(false, 2, 3, 7)
	got72 := test.Encode(t, test.CreateContext("GMS", 72, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got72, got79) {
		t.Fatalf("v72 = % X, want (v79) % X", got72, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v72 ida=0x917ad0
func TestQuantityUpdateBytesV72(t *testing.T) {
	input := NewQuantityUpdate(true, 2, 5, 100)
	got72 := test.Encode(t, test.CreateContext("GMS", 72, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got72, got79) {
		t.Fatalf("v72 = % X, want (v79) % X", got72, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v72 ida=0x917ad0
func TestRemoveBytesV72(t *testing.T) {
	input := NewInventoryRemove(false, 2, 3)
	got72 := test.Encode(t, test.CreateContext("GMS", 72, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got72, got79) {
		t.Fatalf("v72 = % X, want (v79) % X", got72, got79)
	}
}

// The v61 INVENTORY_OPERATION dispatcher modes (ChangeMove/QuantityUpdate/Remove)
// are byte-identical to the verified v72/v79 encodings — same handler
// CWvsContext::OnInventoryOperation@0x8422fc (v61), no version gate on the codec.
// Each v61 cell is proven by equality with the verified v79 encoding.
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v61 ida=0x8422fc
func TestChangeMoveBytesV61(t *testing.T) {
	input := NewChangeMove(false, 2, 3, 7)
	got61 := test.Encode(t, test.CreateContext("GMS", 61, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got61, got79) {
		t.Fatalf("v61 = % X, want (v79) % X", got61, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v61 ida=0x8422fc
func TestQuantityUpdateBytesV61(t *testing.T) {
	input := NewQuantityUpdate(true, 2, 5, 100)
	got61 := test.Encode(t, test.CreateContext("GMS", 61, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got61, got79) {
		t.Fatalf("v61 = % X, want (v79) % X", got61, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v61 ida=0x8422fc
func TestRemoveBytesV61(t *testing.T) {
	input := NewInventoryRemove(false, 2, 3)
	got61 := test.Encode(t, test.CreateContext("GMS", 61, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got61, got79) {
		t.Fatalf("v61 = % X, want (v79) % X", got61, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v83 ida=0xa1ead9
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v83 ida=0xa1ead9
// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v83 ida=0xa1ead9
// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v83 ida=0xa1ead9
//
// v79: CWvsContext::OnInventoryOperation @0x96953e (GMS_v79_1_DEVM.exe port
// 13340) reads Decode1(exclReq)+Decode1(count) then per entry
// Decode1(mode)+Decode1(invType)+Decode2(slot)+[mode body], with a single
// post-loop addMov byte when any entry set nCurItemPos (equip move/remove with
// a negative slot). For Atlas's count=1 packets the post-loop addMov coincides
// with the per-entry inline byte. Modes: 0=Add(GW_ItemSlot opaque), 1=QuantityUpdate
// (Decode2), 2=Move(Decode2 newSlot), 3=Remove. Version-agnostic vs v83 (mode
// enum + header identical); the round-trip variants cover the wire (GMS v28 == v79).
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v79 ida=0x96953e
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v79 ida=0x96953e
// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v79 ida=0x96953e
// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v79 ida=0x96953e
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v95 ida=0xa08a70
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v95 ida=0xa08a70
// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v95 ida=0xa08a70
// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v95 ida=0xa08a70
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v84 ida=0xa69d8f
// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v84 ida=0xa69d8f
// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v84 ida=0xa69d8f
// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v84 ida=0xa69d8f
func TestQuantityUpdateRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewQuantityUpdate(true, 2, 5, 100)
			output := QuantityUpdate{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
		})
	}
}

func TestChangeMoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// non-equip type, no addMov byte
			input := NewChangeMove(false, 2, 3, 7)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestChangeMoveEquipToSlotRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, newSlot < 0 => addMov = 2
			input := NewChangeMove(true, 1, 5, -1)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestChangeMoveUnequipRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, oldSlot < 0 => addMov = 1
			input := NewChangeMove(true, 1, -1, 5)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestRemoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// non-equip type, no addMov byte
			input := NewInventoryRemove(false, 2, 3)
			output := Remove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}

func TestRemoveEquipRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, slot < 0 => addMov byte written
			input := NewInventoryRemove(true, 1, -1)
			output := Remove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}

func TestAddStackableRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(5, 0, 0)
	input := NewInventoryAdd(false, 2, 5, asset)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Add{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
			if output.Asset().Quantity() != input.Asset().Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Asset().Quantity(), input.Asset().Quantity())
			}
		})
	}
}

func TestAddEquipmentRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Add{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
			if output.Asset().Strength() != input.Asset().Strength() {
				t.Errorf("strength: got %v, want %v", output.Asset().Strength(), input.Asset().Strength())
			}
			if output.Asset().Slots() != input.Asset().Slots() {
				t.Errorf("slots: got %v, want %v", output.Asset().Slots(), input.Asset().Slots())
			}
		})
	}
}
