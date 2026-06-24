package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
