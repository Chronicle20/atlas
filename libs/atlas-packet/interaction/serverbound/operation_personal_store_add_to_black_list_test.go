package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v79 ida=0x68ab52
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v95 ida=0x69b1c0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v87 ida=0x741526
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v83 ida=0x6fdf8e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=jms_v185 ida=0x7630d5
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v84 ida=0x71a2ac
func TestOperationPersonalStoreAddToBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreAddToBlackList{slot: 1, name: "TestName"}
			output := OperationPersonalStoreAddToBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}

// TestOperationPersonalStoreAddToBlackListV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::OnClickBanButton (sub_66658A): Encode1(0x1A)=mode @0x666633 then Encode1(slot)@0x66663e, EncodeStr(name)@0x666657. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreAddToBlackList version=gms_v72 ida=0x66658a
func TestOperationPersonalStoreAddToBlackListV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreAddToBlackList{slot: 2, name: "hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "0202006869" {
		t.Errorf("v72 bytes: got %s, want 0202006869", got)
	}
}
