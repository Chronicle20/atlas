package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v79 ida=0x68a756
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v95 ida=0x6987a0
func TestOperationMerchantRemoveItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantRemoveItem{index: 42}
			output := OperationMerchantRemoveItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}

// TestOperationMerchantRemoveItemBytes pins the wire bytes for the entrusted-merchant
// remove-item arm: a single uint16 index (LE). The #Merchant arm shares the base
// CPersonalShopDlg::MoveItemToInventory (entrusted sub-op 0x26 vs personal-shop 0x1B)
// and carries the same body across versions; no MajorVersion() gate.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v83 ida=0x6fdcdf
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v87 ida=0x741271
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=jms_v185 ida=0x762e26
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v84 ida=0x719ffd
func TestOperationMerchantRemoveItemBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationMerchantRemoveItem{index: 42}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 2a00
	want := "2a00"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}

// TestOperationMerchantRemoveItemV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::MoveItemToInventory#Merchant (sub_6662DB, merchant arm mode 0x24 @0x6663b5): shared body Encode2(index). Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v72 ida=0x6662db
func TestOperationMerchantRemoveItemV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0500" {
		t.Errorf("v72 bytes: got %s, want 0500", got)
	}
}
