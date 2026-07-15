package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v48 serverbound NPC_SHOP (op 48) mode-arm body fixtures (GMS_v48_1_DEVM.exe,
// port 13337). The CShopDlg buttons each build COutPacket(48) + Encode1(mode) +
// body; the Atlas structs model only the body (after the mode byte):
//   - BUY      sub_5B7422@0x5b7422: Encode1(0) + Encode2 slot + Encode4 itemId +
//              Encode2 count. NO trailing discountPrice int (that field was
//              added at v72; v48 is below the >=72 gate).
//   - SELL     sub_5B7693@0x5b7693: Encode1(1) + Encode2 slot + Encode4 itemId +
//              Encode2 count.
//   - RECHARGE sub_5B78C0@0x5b78c0: Encode1(2) + Encode2 slot.
//
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v48 ida=0x5b7422
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v48 ida=0x5b7693
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v48 ida=0x5b78c0

// ShopBuy: Encode2 slot, Encode4 itemId, Encode2 quantity (no discountPrice).
func TestShopBuyByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ShopBuy{slot: 3, itemId: 2000000, quantity: 100}.Encode(l, ctx)(nil)
	want := []byte{
		0x03, 0x00, // slot (Encode2 @0x5b75fb)
		0x80, 0x84, 0x1E, 0x00, // itemId 2000000 (Encode4 @0x5b760b)
		0x64, 0x00, // quantity 100 (Encode2 @0x5b7616)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ShopBuy: got % x, want % x", got, want)
	}
}

// ShopSell: Encode2 slot, Encode4 itemId, Encode2 quantity.
func TestShopSellByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ShopSell{slot: 5, itemId: 2000000, quantity: 100}.Encode(l, ctx)(nil)
	want := []byte{
		0x05, 0x00, // slot (Encode2 @0x5b7864)
		0x80, 0x84, 0x1E, 0x00, // itemId 2000000 (Encode4 @0x5b786f)
		0x64, 0x00, // quantity 100 (Encode2 @0x5b787a)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ShopSell: got % x, want % x", got, want)
	}
}

// ShopRecharge: Encode2 slot.
func TestShopRechargeByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ShopRecharge{slot: 7}.Encode(l, ctx)(nil)
	want := []byte{0x07, 0x00} // slot (Encode2 @0x5b79eb)
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ShopRecharge: got % x, want % x", got, want)
	}
}
