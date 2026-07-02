package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestShopBuyByteV79 pins the gms_v79 NPC_SHOP BUY body (op byte 0, written by
// the dispatcher; body only here).
//
// IDA: CShopDlg::SendBuyRequest @0x6d68a3 (renamed from sub_6D68A3;
// GMS_v79_1_DEVM.exe, port 13340) builds COutPacket(59):
//
//	Encode1 op=0 (BUY)          @0x6d6a58  (dispatcher prefix, not in body)
//	Encode2 slot                @0x6d6a76
//	Encode4 itemId              @0x6d6a86
//	Encode2 quantity            @0x6d6a91
//	Encode4 discountPrice        @0x6d6a9c
//
// v79 is GMS so the trailing discountPrice int is present (region-gated).
//
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v79 ida=0x6d68a3
func TestShopBuyByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}.Encode(l, ctx)(nil)
	want := []byte{
		0x03, 0x00, // slot=3           @0x6d6a76
		0x80, 0x84, 0x1E, 0x00, // itemId=2000000  @0x6d6a86
		0x05, 0x00, // quantity=5       @0x6d6a91
		0xE8, 0x03, 0x00, 0x00, // discountPrice=1000 @0x6d6a9c
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ShopBuy: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v83 ida=0x7561c1
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v87 ida=0x7a1d49
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v95 ida=0x6e9bb0
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v84 ida=0x778475
func TestShopBuyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}
			output := ShopBuy{}
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
			// The trailing discountPrice int is GMS-only AND post-legacy; JMS185
			// omits it (CShopDlg::SendBuyRequest@0x7ca2c9 ends after the quantity
			// short) and the legacy GMS range (<72, e.g. v61 sub_646C41) also
			// omits it.
			if v.Region == "GMS" && v.MajorVersion >= 72 {
				if output.DiscountPrice() != input.DiscountPrice() {
					t.Errorf("discountPrice: got %v, want %v", output.DiscountPrice(), input.DiscountPrice())
				}
			}
		})
	}
}

// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=jms_v185 ida=0x7ca2c9
func TestShopBuyDiscountPriceGate(t *testing.T) {
	input := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}

	gmsCtx := pt.CreateContext("GMS", 95, 1)
	gms := input.Encode(nil, gmsCtx)(nil)
	// slot(2) + itemId(4) + quantity(2) + discountPrice(4) = 12
	if len(gms) != 12 {
		t.Errorf("GMS: expected 12 bytes (with discountPrice), got %d", len(gms))
	}

	jmsCtx := pt.CreateContext("JMS", 185, 1)
	jms := input.Encode(nil, jmsCtx)(nil)
	// slot(2) + itemId(4) + quantity(2) = 8 (no discountPrice)
	if len(jms) != 8 {
		t.Errorf("JMS: expected 8 bytes (no discountPrice), got %d", len(jms))
	}
}

// TestShopBuyByteV72 pins the gms_v72 NPC_SHOP BUY body (op byte 0, dispatcher
// prefix; body only here).
//
// IDA: the v72 buy handler sub_6A8B15 (GMS_v72.1_U_DEVM.exe, port 13339) builds
// COutPacket(60) — the prior agent found only the CLOSE arm @0x6a5f39; the buy
// send is this shop-dialog OK-button handler:
//
//	Encode1 op=0 (BUY)     @0x6a8cca  (dispatcher prefix, not in body)
//	Encode2 slot           @0x6a8ce8
//	Encode4 itemId         @0x6a8cf8
//	Encode2 quantity       @0x6a8d03
//	Encode4 discountPrice  @0x6a8d0e
//
// v72 is GMS so the trailing discountPrice int is present. Body byte-identical to v79.
//
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v72 ida=0x6a8b15
func TestShopBuyByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}.Encode(l, ctx)(nil)
	want := []byte{
		0x03, 0x00, // slot=3           @0x6a8ce8
		0x80, 0x84, 0x1E, 0x00, // itemId=2000000  @0x6a8cf8
		0x05, 0x00, // quantity=5       @0x6a8d03
		0xE8, 0x03, 0x00, 0x00, // discountPrice=1000 @0x6a8d0e
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ShopBuy: got % x, want % x", got, want)
	}
}

// TestShopBuyByteV61 pins the gms_v61 NPC_SHOP BUY body. The v61 shop dialog
// buy-button handler sub_646C41@0x646c41 (GMS_v61.1_U_DEVM.exe, port 13338)
// builds COutPacket(57):
//
//	Encode1 op=0 (BUY)  @0x646df6  (dispatcher prefix, not in body)
//	Encode2 slot        @0x646e14
//	Encode4 itemId      @0x646e24
//	Encode2 quantity    @0x646e2f
//
// UNLIKE v72+ there is NO trailing discountPrice int — the legacy GMS shop send
// ends after the quantity short (region+version gate: present only for GMS>=72).
//
// packet-audit:verify packet=npc/serverbound/NpcShopBuy version=gms_v61 ida=0x646c41
func TestShopBuyByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ShopBuy{slot: 3, itemId: 2000000, quantity: 5, discountPrice: 1000}.Encode(l, ctx)(nil)
	want := []byte{
		0x03, 0x00, // slot=3           @0x646e14
		0x80, 0x84, 0x1E, 0x00, // itemId=2000000  @0x646e24
		0x05, 0x00, // quantity=5       @0x646e2f
		// NO discountPrice — legacy GMS omits it.
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ShopBuy: got % x, want % x", got, want)
	}
}
