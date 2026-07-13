package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v83 ida=0xa0a25e
// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v95 ida=0x9e10e0
// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v79 ida=0x9703a3
// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v72 ida=0x91e45b
func TestShopScannerItemUseRoundTrip(t *testing.T) {
	input := ShopScannerItemUse{serial: "SN-1", source: 3, itemId: 2310000, searchItemId: 1302000, descending: true, updateTime: 12345678}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			legacy := itemUseLegacyFrame(tenant.MustFromContext(ctx))
			output := &ShopScannerItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != 3 {
				t.Errorf("source = %d, want 3", output.Source())
			}
			if output.ItemId() != 2310000 {
				t.Errorf("itemId = %d, want 2310000", output.ItemId())
			}
			if legacy {
				// pre-v83 [str serial][short pos][int itemId]: only the serial
				// and source/itemId survive; there is no search target on the wire.
				if output.Serial() != "SN-1" {
					t.Errorf("serial = %q, want SN-1", output.Serial())
				}
				if output.SearchItemId() != 0 || output.Descending() || output.UpdateTime() != 0 {
					t.Errorf("legacy frame leaked search fields: searchItemId=%d descending=%t updateTime=%d",
						output.SearchItemId(), output.Descending(), output.UpdateTime())
				}
				return
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

// TestShopScannerItemUseWireShape pins the two frames:
//   - v83+/JMS: [short nPOS][int nItemID][int searchItemId][byte bDescending][int updateTime]
//     — NO leading updateTime on any version, v95 included (verified 0x9e10e0).
//   - pre-v83 GMS: [str serial][short nPOS][int nItemID] (verified v79 0x9703a3, v72 0x91e45b).
func TestShopScannerItemUseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopScannerItemUse{serial: "SN-1", source: 3, itemId: 2310000, searchItemId: 1302000, descending: false, updateTime: 12345678}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := in.Encode(l, ctx)(nil)
			if itemUseLegacyFrame(tenant.MustFromContext(ctx)) {
				// [short len][4 bytes "SN-1"][short nPOS][int nItemID] = 2+4+2+4 = 12
				if len(b) != 12 {
					t.Fatalf("legacy wire size = %d bytes, want 12: % x", len(b), b)
				}
				if binary.LittleEndian.Uint16(b[0:2]) != 4 {
					t.Errorf("serial len = %d, want 4", binary.LittleEndian.Uint16(b[0:2]))
				}
				if string(b[2:6]) != "SN-1" {
					t.Errorf("serial = %q, want SN-1", string(b[2:6]))
				}
				if binary.LittleEndian.Uint16(b[6:8]) != 3 {
					t.Errorf("nPOS = %d, want 3", binary.LittleEndian.Uint16(b[6:8]))
				}
				if binary.LittleEndian.Uint32(b[8:12]) != 2310000 {
					t.Errorf("nItemID = %d, want 2310000", binary.LittleEndian.Uint32(b[8:12]))
				}
				return
			}
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
