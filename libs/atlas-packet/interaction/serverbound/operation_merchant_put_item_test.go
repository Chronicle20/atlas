package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
