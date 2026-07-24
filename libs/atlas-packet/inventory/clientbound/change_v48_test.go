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
//
//	Decode1(exclReq)      /*0x71a50e*/  (clears the exclusive-request latch)
//	Decode1(count)        /*0x71a552*/
//	count x per entry:
//	  Decode1(mode)       /*0x71a570*/
//	  Decode1(invType)    /*0x71a57b*/
//	  Decode2(slot)       /*0x71a583*/
//	  mode body:
//	    0 = Add            GW_ItemSlotBase::Decode (opaque) /*0x71a6cd*/
//	    1 = QuantityUpdate Decode2(newQuantity)            /*0x71a670*/
//	    2 = Move           Decode2(newSlot)                /*0x71a5cd*/
//	    3 = Remove         (no extra read)
//	post-loop, when any entry set the move-out flag (mode 2/3, invType==EQUIP,
//	negative slot): Decode1(addMov) /*0x71a75a*/.
//
// The v48 InventoryAdd framing is byte-identical to the verified v79 handler
// @0x96953e (same header, same mode enum, same post-loop addMov byte), but the
// embedded model.Asset equip blob is NOT: the v48 equip decode
// (GW_ItemSlotEquip::RawDecode @0x49c332) has no levelType/level/experience/hammers
// trailer — after owner+flag it reads only a single 8-byte buffer (non-cash),
// exactly like v61. So the v48 equip is 22 bytes shorter than v79. Opaque-family
// (OPAQUE_LEDGER exception): length delta vs the verified v79 encoding.
//
// packet-audit:verify packet=inventory/clientbound/InventoryAdd version=gms_v48 ida=0x71a4f6
func TestAddEquipmentBytesV48(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	got48 := test.Encode(t, test.CreateContext("GMS", 48, 1), input.Encode, nil)
	got79 := test.Encode(t, test.CreateContext("GMS", 79, 1), input.Encode, nil)
	if len(got48) != len(got79)-22 {
		t.Fatalf("v48 equip len %d, want v79 len %d - 22 (no v72/v79 equip trailer)\n v48=% X\n v79=% X", len(got48), len(got79), got48, got79)
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
