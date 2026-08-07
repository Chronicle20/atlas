package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestShopSellByteV79 pins the gms_v79 NPC_SHOP SELL body (op byte 1, dispatcher
// prefix; body only here).
//
// IDA: CShopDlg::SendSellRequest @0x6d6b1d (renamed from sub_6D6B1D;
// GMS_v79_1_DEVM.exe) builds COutPacket(59):
//
//	Encode1 op=1 (SELL)  @0x6d6cd9  (dispatcher prefix, not in body)
//	Encode2 slot         @0x6d6ce4
//	Encode4 itemId       @0x6d6cef
//	Encode2 quantity     @0x6d6cfa
//
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v79 ida=0x6d6b1d
func TestShopSellByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ShopSell{slot: 5, itemId: 1000000, quantity: 10}.Encode(l, ctx)(nil)
	want := []byte{
		0x05, 0x00, // slot=5           @0x6d6ce4
		0x40, 0x42, 0x0F, 0x00, // itemId=1000000  @0x6d6cef
		0x0A, 0x00, // quantity=10      @0x6d6cfa
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ShopSell: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v83 ida=0x756a04
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v87 ida=0x7a256b
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v95 ida=0x6e7260
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=jms_v185 ida=0x7cacab
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v84 ida=0x778cb8
func TestShopSellRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopSell{slot: 5, itemId: 1000000, quantity: 10}
			output := ShopSell{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
		})
	}
}

// TestShopSellByteV72 pins the gms_v72 NPC_SHOP SELL body (op byte 1, dispatcher
// prefix; body only here).
//
// IDA: the v72 sell handler sub_6A8D8F (GMS_v72.1_U_DEVM.exe) builds COutPacket(60):
//
//	Encode1 op=1 (SELL)  @0x6a8f4b  (dispatcher prefix, not in body)
//	Encode2 slot         @0x6a8f56
//	Encode4 itemId       @0x6a8f61
//	Encode2 quantity     @0x6a8f6c
//
// Body byte-identical to v79.
//
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v72 ida=0x6a8d8f
func TestShopSellByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ShopSell{slot: 5, itemId: 1000000, quantity: 10}.Encode(l, ctx)(nil)
	want := []byte{
		0x05, 0x00, // slot=5           @0x6a8f56
		0x40, 0x42, 0x0F, 0x00, // itemId=1000000  @0x6a8f61
		0x0A, 0x00, // quantity=10      @0x6a8f6c
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ShopSell: got % x, want % x", got, want)
	}
}

// TestShopSellByteV61 pins the gms_v61 NPC_SHOP SELL body. The v61 shop dialog
// sell-button handler sub_646EAE@0x646eae (GMS_v61.1_U_DEVM.exe) builds
// COutPacket(57):
//
//	Encode1 op=1 (SELL)  @0x64705d  (dispatcher prefix, not in body)
//	Encode2 slot         @0x647068
//	Encode4 itemId       @0x647073
//	Encode2 quantity     @0x64707e
//
// Body byte-identical to v72.
//
// packet-audit:verify packet=npc/serverbound/NpcShopSell version=gms_v61 ida=0x646eae
func TestShopSellByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ShopSell{slot: 5, itemId: 1000000, quantity: 10}.Encode(l, ctx)(nil)
	want := []byte{
		0x05, 0x00, // slot=5           @0x647068
		0x40, 0x42, 0x0F, 0x00, // itemId=1000000  @0x647073
		0x0A, 0x00, // quantity=10      @0x64707e
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ShopSell: got % x, want % x", got, want)
	}
}
