package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v83 ida=0xa0a25e
// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v95 ida=0x9e10e0
func TestShopScannerItemUseRoundTrip(t *testing.T) {
	input := NewShopScannerItemUse(3, 2310000, 1302000, true, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != 3 {
				t.Errorf("source = %d, want 3", output.Source())
			}
			if output.ItemId() != 2310000 {
				t.Errorf("itemId = %d, want 2310000", output.ItemId())
			}
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

// TestShopScannerItemUseWireShape pins
// [short nPOS][int nItemID][int searchItemId][byte bDescending][int updateTime]
// — NO leading updateTime on any version, v95 included (verified 0x9e10e0).
func TestShopScannerItemUseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerItemUse(3, 2310000, 1302000, false, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 15 {
				t.Fatalf("wire size = %d bytes, want 15: % x", len(b), b)
			}
			if binary.LittleEndian.Uint16(b[0:2]) != 3 {
				t.Errorf("nPOS = %d, want 3", binary.LittleEndian.Uint16(b[0:2]))
			}
			if binary.LittleEndian.Uint32(b[2:6]) != 2310000 {
				t.Errorf("nItemID = %d, want 2310000", binary.LittleEndian.Uint32(b[2:6]))
			}
			if binary.LittleEndian.Uint32(b[6:10]) != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", binary.LittleEndian.Uint32(b[6:10]))
			}
			if b[10] != 0x00 {
				t.Errorf("bDescending = 0x%02x, want 0x00", b[10])
			}
			if binary.LittleEndian.Uint32(b[11:15]) != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", binary.LittleEndian.Uint32(b[11:15]))
			}
		})
	}
}
