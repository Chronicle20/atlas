package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v79 ida=0x68a3e3
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v95 ida=0x69c880
func TestOperationMerchantPutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantPutItem{inventoryType: 2, slot: 7, quantity: 15, set: 4, price: 2000}
			output := OperationMerchantPutItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
			if output.Set() != input.Set() {
				t.Errorf("set: got %v, want %v", output.Set(), input.Set())
			}
			if output.Price() != input.Price() {
				t.Errorf("price: got %v, want %v", output.Price(), input.Price())
			}
		})
	}
}

// TestOperationMerchantPutItemBytes pins the wire bytes for the entrusted-merchant
// put-item arm: byte inventoryType, int16 slot (LE), uint16 quantity (LE),
// uint16 set (LE), uint32 price (LE). The #Merchant arm shares the base
// CPersonalShopDlg::PutItem (entrusted sub-op 0x21 vs personal-shop 0x16) and carries
// the same body across versions; the codec has no MajorVersion() gate.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v83 ida=0x6fd96c
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v87 ida=0x740ee6
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=jms_v185 ida=0x762a9e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v84 ida=0x719c8a
func TestOperationMerchantPutItemBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationMerchantPutItem{inventoryType: 2, slot: 7, quantity: 15, set: 4, price: 2000}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 02 | 0700 | 0f00 | 0400 | d0070000
	want := "0207000f000400d0070000"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}

// TestOperationMerchantPutItemV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::PutItem#Merchant (sub_665F5F, merchant arm mode 0x1F @0x6661e8): shared body Encode1(invType),Encode2(slot),Encode2(qty),Encode2(set),Encode4(price). Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v72 ida=0x665f5f
func TestOperationMerchantPutItemV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantPutItem{inventoryType: 2, slot: 5, quantity: 100, set: 7, price: 1000000}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0205006400070040420f00" {
		t.Errorf("v72 bytes: got %s, want 0205006400070040420f00", got)
	}
}
