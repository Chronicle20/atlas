package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// INVENTORY_OPERATION v48 (GMS_v48_1_DEVM.exe, port 13337).
//
// Client read order — CWvsContext::OnInventoryOperation @0x71a4f6:
//   Decode1(exclReq)      /*0x71a50e*/  (clears the exclusive-request latch)
//   Decode1(count)        /*0x71a552*/
//   count x per entry:
//     Decode1(mode)       /*0x71a570*/
//     Decode1(invType)    /*0x71a57b*/
//     Decode2(slot)       /*0x71a583*/
//     mode body:
//       0 = Add            GW_ItemSlotBase::Decode (opaque) /*0x71a6cd*/
//       1 = QuantityUpdate Decode2(newQuantity)            /*0x71a670*/
//       2 = Move           Decode2(newSlot)                /*0x71a5cd*/
//       3 = Remove         (no extra read)
//   post-loop, when any entry set the move-out flag (mode 2/3, invType==EQUIP,
//   negative slot): Decode1(addMov) /*0x71a75a*/.
//
// This is byte-identical to the verified v79 handler @0x96953e (same header,
// same mode enum, same post-loop addMov byte) — the inventory change codec and
// the opaque model.Asset blob carry no version gate below v79 (Asset gates are
// GMS>12 [true], GMS>28 [true], GMS>=84 [false], so v48 shares the v61/v72/v79
// bracket). Each arm is proven by equality with the verified v79 encoding of the
// same input (OPAQUE_LEDGER exception for the Add blob).
//
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v48 ida=0x71a4f6
func TestAddEquipmentBytesV48(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got48, got79) {
		t.Fatalf("v48 = % X, want (v79) % X", got48, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryChangeMove version=gms_v48 ida=0x71a4f6
func TestChangeMoveBytesV48(t *testing.T) {
	input := NewChangeMove(false, 2, 3, 7)
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got48, got79) {
		t.Fatalf("v48 = % X, want (v79) % X", got48, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryQuantityUpdate version=gms_v48 ida=0x71a4f6
func TestQuantityUpdateBytesV48(t *testing.T) {
	input := NewQuantityUpdate(true, 2, 5, 100)
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got48, got79) {
		t.Fatalf("v48 = % X, want (v79) % X", got48, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryRemove version=gms_v48 ida=0x71a4f6
func TestRemoveBytesV48(t *testing.T) {
	input := NewInventoryRemove(false, 2, 3)
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got48, got79) {
		t.Fatalf("v48 = % X, want (v79) % X", got48, got79)
	}
}

// packet-audit:verify packet=inventory/clientbound/InventoryChangeBatch version=gms_v48 ida=0x71a4f6
func TestChangeBatchBytesV48(t *testing.T) {
	input := NewChangeBatch(true, inventory.NewMoveEntry(2, 3, 7))
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if !bytes.Equal(got48, got79) {
		t.Fatalf("v48 = % X, want (v79) % X", got48, got79)
	}
}
