package clientbound

import (
	"encoding/binary"
	"testing"
	"time"

	pktmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/clientbound/ShopScannerResult version=gms_v83 ida=0xa28c29
// packet-audit:verify packet=merchant/clientbound/ShopScannerResult version=gms_v95 ida=0xa076c0
// packet-audit:verify packet=merchant/clientbound/ShopScannerHotList version=gms_v83 ida=0xa28c29
// packet-audit:verify packet=merchant/clientbound/ShopScannerHotList version=gms_v95 ida=0xa076c0
func TestShopScannerResultRoundTrip(t *testing.T) {
	records := []ShopScannerRecord{
		NewShopScannerRecord("OwnerA", 910000004, "cheap stuff", 3, 100, 5000, 30001, 0, 2, nil),
		NewShopScannerRecord("OwnerB", 910000010, "arrows", 1, 1000, 9000, 30002, 1, 2, nil),
	}
	input := NewShopScannerResult(6, 2060000, records)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 6 {
				t.Errorf("mode = %d, want 6", output.Mode())
			}
			if output.ItemId() != 2060000 {
				t.Errorf("itemId = %d, want 2060000", output.ItemId())
			}
			if len(output.Records()) != 2 {
				t.Fatalf("record count = %d, want 2", len(output.Records()))
			}
			r0 := output.Records()[0]
			if r0.OwnerName() != "OwnerA" || r0.MapId() != 910000004 || r0.Title() != "cheap stuff" ||
				r0.Bundles() != 3 || r0.BundleSize() != 100 || r0.Price() != 5000 ||
				r0.OwnerId() != 30001 || r0.ChannelId() != 0 || r0.InventoryType() != 2 {
				t.Errorf("record 0 mismatch: %+v", r0)
			}
		})
	}
}

// TestShopScannerResultEmpty pins the faithful no-results shape:
// nCount==0 && nNpcShopPrice==0 makes the client show SP_3637
// ("Unable to find the item you have entered").
func TestShopScannerResultEmpty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerResult(6, 2060000, nil)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// [byte mode][int npcShopPrice][int itemId][int count] = 13 bytes
			if len(b) != 13 {
				t.Fatalf("wire size = %d bytes, want 13: % x", len(b), b)
			}
			if b[0] != 0x06 {
				t.Errorf("mode = 0x%02x, want 0x06", b[0])
			}
			if binary.LittleEndian.Uint32(b[1:5]) != 0 {
				t.Errorf("nNpcShopPrice = %d, want 0", binary.LittleEndian.Uint32(b[1:5]))
			}
			if binary.LittleEndian.Uint32(b[5:9]) != 2060000 {
				t.Errorf("nItemID = %d, want 2060000", binary.LittleEndian.Uint32(b[5:9]))
			}
			if binary.LittleEndian.Uint32(b[9:13]) != 0 {
				t.Errorf("nCount = %d, want 0", binary.LittleEndian.Uint32(b[9:13]))
			}
		})
	}
}

// TestShopScannerResultEquipRow exercises the nTI==1 branch: a full
// GW_ItemSlotBase (slotless, zeroPosition=true) follows the record header.
func TestShopScannerResultEquipRow(t *testing.T) {
	asset := pktmodel.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 0, 0, 0, 0, 17, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	records := []ShopScannerRecord{
		NewShopScannerRecord("OwnerA", 910000004, "swords", 1, 1, 150000, 30001, 0, 1, &asset),
	}
	input := NewShopScannerResult(6, 1302000, records)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Records()) != 1 {
				t.Fatalf("record count = %d, want 1", len(output.Records()))
			}
			r0 := output.Records()[0]
			if r0.InventoryType() != 1 {
				t.Fatalf("inventoryType = %d, want 1", r0.InventoryType())
			}
			if r0.Asset() == nil {
				t.Fatalf("asset = nil, want decoded GW_ItemSlotBase")
			}
		})
	}
}

func TestShopScannerHotListRoundTrip(t *testing.T) {
	input := NewShopScannerHotList(7, []uint32{2060000, 1302000, 4000000})
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerHotList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 7 {
				t.Errorf("mode = %d, want 7", output.Mode())
			}
			if len(output.ItemIds()) != 3 || output.ItemIds()[0] != 2060000 || output.ItemIds()[2] != 4000000 {
				t.Errorf("itemIds = %v, want [2060000 1302000 4000000]", output.ItemIds())
			}
		})
	}
}

// TestShopScannerHotListShortCount: fewer than 10 ever-searched items sends a
// short list — count byte reflects the actual length, no filler.
func TestShopScannerHotListShortCount(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerHotList(7, []uint32{2060000})
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// [byte mode][byte count][1 × int] = 6 bytes
			if len(b) != 6 {
				t.Fatalf("wire size = %d bytes, want 6: % x", len(b), b)
			}
			if b[0] != 0x07 {
				t.Errorf("mode = 0x%02x, want 0x07", b[0])
			}
			if b[1] != 0x01 {
				t.Errorf("count = 0x%02x, want 0x01", b[1])
			}
			if binary.LittleEndian.Uint32(b[2:6]) != 2060000 {
				t.Errorf("itemId = %d, want 2060000", binary.LittleEndian.Uint32(b[2:6]))
			}
		})
	}
}
