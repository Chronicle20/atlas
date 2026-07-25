package serverbound

import (
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// The 523 arm tail of CWvsContext::SendConsumeCashItemUseRequest case 29
// (v83 jumptable case 0xa0cd0b): CUIShopScanner::SendScanPacket appends
// [int searchItemId][byte bDescending][int updateTime] to the stashed use
// packet unconditionally in both v83 and v95 — the GMS>=95 leading-updateTime
// gate applies only to the ItemUse prefix, not this tail.
func TestItemUseStoreSearchRoundTrip(t *testing.T) {
	input := NewItemUseStoreSearch(1302000, true, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ItemUseStoreSearch{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SearchItemId() != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", output.SearchItemId())
			}
			if !output.Descending() {
				t.Errorf("descending = false, want true")
			}
			if output.UpdateTime() != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", output.UpdateTime())
			}
		})
	}
}

func TestItemUseStoreSearchWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewItemUseStoreSearch(1302000, false, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 9 {
				t.Fatalf("wire size = %d bytes, want 9: % x", len(b), b)
			}
			if binary.LittleEndian.Uint32(b[0:4]) != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", binary.LittleEndian.Uint32(b[0:4]))
			}
			if b[4] != 0x00 {
				t.Errorf("bDescending = 0x%02x, want 0x00", b[4])
			}
			if binary.LittleEndian.Uint32(b[5:9]) != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", binary.LittleEndian.Uint32(b[5:9]))
			}
		})
	}
}
