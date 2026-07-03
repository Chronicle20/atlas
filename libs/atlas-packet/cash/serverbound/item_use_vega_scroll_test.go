package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestItemUseVegaScrollRoundTrip(t *testing.T) {
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 305419896)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ItemUseVegaScroll{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestItemUseVegaScrollBytes locks the six-int32 little-endian wire shape:
// equipTab(1) equipSlot(5) scrollTab(2) scrollSlot(7) flag(1)
// updateTime(0x12345678) — 24 bytes, version-independent (no gate in codec).
func TestItemUseVegaScrollBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 0x12345678)
	want := "01000000" + "05000000" + "02000000" + "07000000" + "01000000" + "78563412"
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := hex.EncodeToString(input.Encode(l, ctx)(nil))
			if got != want {
				t.Errorf("bytes: got %s, want %s", got, want)
			}
		})
	}
}
