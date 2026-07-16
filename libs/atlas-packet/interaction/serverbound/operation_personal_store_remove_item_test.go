package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v79 ida=0x68a756
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v95 ida=0x6987a0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v87 ida=0x741271
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v83 ida=0x6fdcdf
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=jms_v185 ida=0x762e26
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v84 ida=0x719ffd
func TestOperationPersonalStoreRemoveItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreRemoveItem{index: 7}
			output := OperationPersonalStoreRemoveItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}

// TestOperationPersonalStoreRemoveItemV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::MoveItemToInventory (sub_6662DB): Encode1(0x19 personal)=mode @0x6663b5 then Encode2(index) @0x6663c2. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreRemoveItem version=gms_v72 ida=0x6662db
func TestOperationPersonalStoreRemoveItemV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreRemoveItem{index: 5}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0500" {
		t.Errorf("v72 bytes: got %s, want 0500", got)
	}
}
