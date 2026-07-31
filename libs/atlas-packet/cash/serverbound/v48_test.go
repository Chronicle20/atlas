package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 cash-shop operation byte fixtures. All arms send COutPacket(160)
// (CASHSHOP_OPERATION) followed by a mode byte, which the dispatcher strips
// before the codec body.

// TestIncreaseStorageBytesV48 — CCashShop::OnIncTrunkCount @0x44aad1 encodes
// Encode1(6) @0x44abf4 (mode), Encode1(isPoints) @0x44ac04, Encode1(0) @0x44ac0d.
// No Encode4 between them, so the v48 body after the mode is two bytes. v83
// @0x46c55b and v87 @0x4763e0 read Decode1, Decode1, Decode4, Decode1.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseStorage version=gms_v48 ida=0x44aad1
func TestIncreaseStorageBytesV48(t *testing.T) {
	in := ShopOperationIncreaseStorage{isPoints: true, currency: 4000, item: false}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x01, 0x00} // isPoints, item — no currency int
	if !bytes.Equal(got, want) {
		t.Errorf("v48 increase storage:\n got % x\nwant % x", got, want)
	}
	// v83 keeps the currency int.
	v83 := in.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(v83, []byte{0x01, 0xA0, 0x0F, 0x00, 0x00, 0x00}) {
		t.Errorf("v83 increase storage: got % x", v83)
	}
}

// TestMoveFromCashInventoryBytesV48 — CCashShop::OnMoveCashItemLtoS @0x44ec2c
// encodes Encode1(0x0A) @0x44edb7 (mode), EncodeBuffer(sn, 8) @0x44edc5,
// Encode1(inventoryType) @0x44edd0, Encode2(slot) @0x44eddb. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationMoveFromCashInventory version=gms_v48 ida=0x44ec2c
func TestMoveFromCashInventoryBytesV48(t *testing.T) {
	in := ShopOperationMoveFromCashInventory{serialNumber: 0x0102030405060708, inventoryType: 2, slot: 3}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // sn — EncodeBuffer(8)
		0x02,       // inventoryType — Encode1
		0x03, 0x00, // slot          — Encode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 move from cash inventory:\n got % x\nwant % x", got, want)
	}
}

// TestMoveToCashInventoryBytesV48 — CCashShop::OnMoveCashItemStoL @0x44ee3a
// encodes Encode1(0x0B) @0x44eeb6 (mode), EncodeBuffer(sn, 8) @0x44eec4,
// Encode1(inventoryType) @0x44eecf. No slot short on this arm. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationMoveToCashInventory version=gms_v48 ida=0x44ee3a
func TestMoveToCashInventoryBytesV48(t *testing.T) {
	in := ShopOperationMoveToCashInventory{serialNumber: 0x0102030405060708, inventoryType: 2}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x02}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 move to cash inventory:\n got % x\nwant % x", got, want)
	}
}

// TestBuyWorldTransferBytesV48 — CCashShop::SendBuyTransferWorldItemPacket
// @0x44fbfc encodes Encode1(0x24) @0x44fc24 (mode), Encode4(serialNumber)
// @0x44fc2f, Encode4(targetWorld) @0x44fc3a. Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=gms_v48 ida=0x44fbfc
func TestBuyWorldTransferBytesV48(t *testing.T) {
	in := ShopOperationBuyWorldTransfer{serialNumber: 0x01020304, targetWorld: 2}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x04, 0x03, 0x02, 0x01, 0x02, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 buy world transfer:\n got % x\nwant % x", got, want)
	}
}

// TestBuyNameChangeBytesV48 — CCashShop::SendBuyNameChangeItemPacket @0x44f9a5
// encodes the mode byte, Encode4(serialNumber), EncodeStr(oldName),
// EncodeStr(newName). Shape-stable.
//
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNameChange version=gms_v48 ida=0x44f9a5
func TestBuyNameChangeBytesV48(t *testing.T) {
	in := ShopOperationBuyNameChange{serialNumber: 0x01020304, oldName: "ab", newName: "cd"}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // serialNumber — Encode4
		0x02, 0x00, 'a', 'b', // oldName      — EncodeStr
		0x02, 0x00, 'c', 'd', // newName      — EncodeStr
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 buy name change:\n got % x\nwant % x", got, want)
	}
}
