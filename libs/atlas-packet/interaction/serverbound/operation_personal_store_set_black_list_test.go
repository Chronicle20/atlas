package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v79 ida=0x68a951
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v87 ida=0x74146f
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=jms_v185 ida=0x763021
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v84 ida=0x71a1f8
func TestOperationPersonalStoreSetBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreSetBlackList{entries: []string{"Alice", "Bob", "Carol"}}
			output := OperationPersonalStoreSetBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i := range input.Entries() {
				if output.Entries()[i] != input.Entries()[i] {
					t.Errorf("entries[%d]: got %v, want %v", i, output.Entries()[i], input.Entries()[i])
				}
			}
		})
	}
}

// TestOperationPersonalStoreSetBlackListBytes pins the wire bytes: short count
// (LE) then count length-prefixed ASCII strings (EncodeStr = short len + bytes).
// Per-entry strings are present in both v83 (IDA
// CPersonalShopDlg::DeliverBlackList@0x6fdeda Encode2 count + loop EncodeStr) and
// v95 (@0x69b0d0), so the string[] shape is unconditional.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v83 ida=0x6fdeda
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v95 ida=0x69b0d0
func TestOperationPersonalStoreSetBlackListBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationPersonalStoreSetBlackList{entries: []string{"Al", "Bo"}}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// count=2 (0200) | "Al" = len2 'A''l' (0200 416c) | "Bo" = len2 'B''o' (0200 426f)
	want := "0200" + "0200" + "416c" + "0200" + "426f"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}

// TestOperationPersonalStoreSetBlackListV72Bytes pins the GMS v72 legacy body (mode byte is
// dispatcher-framed, not part of this sub-struct). IDA v72 CPersonalShopDlg::DeliverBlackList (sub_6664D6): Encode1(0x1C)=mode @0x666505 then Encode2(count)@0x666524, loop EncodeStr(name)@0x666557. Body == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreSetBlackList version=gms_v72 ida=0x6664d6
func TestOperationPersonalStoreSetBlackListV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreSetBlackList{entries: []string{"ab"}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "010002006162" {
		t.Errorf("v72 bytes: got %s, want 010002006162", got)
	}
}
