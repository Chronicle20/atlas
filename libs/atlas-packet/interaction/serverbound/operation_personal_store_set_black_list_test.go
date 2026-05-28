package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
