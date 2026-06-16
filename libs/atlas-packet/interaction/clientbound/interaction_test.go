package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func testAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, map[slot.Position]uint32{}, map[int8]uint32{})
}

func testShopItem() interaction.RoomShopItem {
	return interaction.RoomShopItem{
		PerBundle: 1,
		Quantity:  100,
		Price:     5000,
		Asset:     model.NewAsset(true, 1, 2000000, time.Time{}),
	}
}

// packet-audit:verify packet=interaction/clientbound/InteractionInteractionChat version=gms_v95 ida=0x639ad0
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v95 ida=0x637510
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v95 ida=0x639500
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v95 ida=0x638f80
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v95 ida=0x637d70
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v95 ida=0x637a40
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v95 ida=0x639500
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v95 ida=0x51cc30
//
// gms_v83 (dispatcher 0x65df4c; base modes 2/3/4/5/10 version-stable, hired-merchant
// refresh chains CEntrustedShopDlg::OnRefresh -> CPersonalShopDlg::OnRefresh):
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v83 ida=0x65e53b
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v83 ida=0x65e848
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v83 ida=0x65ed1c
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v83 ida=0x65dff3
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v83 ida=0x65dff3
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v83 ida=0x65edb5
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v83 ida=0x518852
//
// gms_v84 (dispatcher 0x673db5; structurally identical to v83):
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v84 ida=0x6743a4
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v84 ida=0x6746b1
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v84 ida=0x674bad
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v84 ida=0x673e5c
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v84 ida=0x673e5c
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v84 ida=0x674c55
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v84 ida=0x5218ca
//
// gms_v87 (dispatcher 0x698251; structurally identical to v83):
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v87 ida=0x69884b
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v87 ida=0x698b61
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v87 ida=0x699039
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v87 ida=0x6982f8
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v87 ida=0x6982f8
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v87 ida=0x6990ea
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v87 ida=0x53b2fc
//
// jms_v185 (dispatcher 0x6da198; base modes 2/3/4/5/10 version-stable. UPDATE_MERCHANT
// is a genuine blocker: the mode-25 default case virtual-dispatches into a hired-merchant
// OnRefresh that is UNNAMED in this IDB, so the leaf address is unresolved — see blockers):
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=jms_v185 ida=0x6da7b4
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=jms_v185 ida=0x6daa56
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=jms_v185 ida=0x6dace2
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=jms_v185 ida=0x6da234
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=jms_v185 ida=0x6da234
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=jms_v185 ida=0x6dad8a
func TestInteractionInviteRoundTrip(t *testing.T) {
	input := NewInteractionInvite(4, 1, "TestPlayer", 12345)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionInvite{}).Decode, nil)
		})
	}
}

func TestInteractionInviteResultRoundTrip(t *testing.T) {
	input := NewInteractionInviteResult(5, 1, "Room is full")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionInviteResult{}).Decode, nil)
		})
	}
}

func TestInteractionChatRoundTrip(t *testing.T) {
	input := NewInteractionChat(6, 7, 1, "TestPlayer : Hello world")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionChat{}).Decode, nil)
		})
	}
}

func TestInteractionLeaveRoundTrip(t *testing.T) {
	input := NewInteractionLeave(10, 2, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionLeave{}).Decode, nil)
		})
	}
}

func TestInteractionEnterResultErrorRoundTrip(t *testing.T) {
	input := NewInteractionEnterResultError(5, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionEnterResultError{}).Decode, nil)
		})
	}
}

// TestInteractionEnterRoundTrip exercises mode 4 (ENTER): mode byte + the
// interaction.Visitor substruct (OnEnterBase Decode1 slot + DecodeAvatar +
// DecodeStr userID, version-stable across v83/v84/v87/v95/jms).
func TestInteractionEnterRoundTrip(t *testing.T) {
	input := NewInteractionEnter(4, interaction.NewBaseVisitor(1, testAvatar(), "Visitor"))
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionEnter{}).Decode, nil)
		})
	}
}

// TestInteractionEnterResultSuccessRoundTrip exercises mode 5 success path:
// mode byte + the interaction.Room substruct (OnEnterResultBase roomType +
// maxUsers + per-slot avatar loop terminated by slot 0xFF).
func TestInteractionEnterResultSuccessRoundTrip(t *testing.T) {
	visitors := []interaction.Visitor{interaction.NewBaseVisitor(0, testAvatar(), "ShopOwner")}
	room := interaction.NewPersonalShopRoom(visitors, "CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	input := NewInteractionEnterResultSuccess(5, room)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionEnterResultSuccess{}).Decode, nil)
		})
	}
}

// TestInteractionUpdateMerchantBytes is an encode-only byte fixture for mode 25
// (UPDATE_MERCHANT). The hired-merchant refresh leaf CEntrustedShopDlg::OnRefresh
// reads Decode4(meso) then chains CPersonalShopDlg::OnRefresh: Decode1(count) +
// count x {Decode2 perBundle, Decode2 quantity, Decode4 price, GW_ItemSlotBase}.
// Each asserted byte traces to that read order.
func TestInteractionUpdateMerchantBytes(t *testing.T) {
	items := []interaction.RoomShopItem{testShopItem()}
	input := NewInteractionUpdateMerchant(25, 50000, items)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := test.Encode(t, ctx, input.Encode, nil)
			// mode(25) + meso(50000 LE) + count(1) + item{perBundle,quantity,price} + asset(...)
			// mode
			if b[0] != 25 {
				t.Fatalf("mode: got %d, want 25", b[0])
			}
			// meso little-endian uint32 = 50000 = 0x0000C350
			if b[1] != 0x50 || b[2] != 0xC3 || b[3] != 0x00 || b[4] != 0x00 {
				t.Fatalf("meso bytes: got % x, want 50 c3 00 00", b[1:5])
			}
			// count
			if b[5] != 1 {
				t.Fatalf("count: got %d, want 1", b[5])
			}
			// perBundle (short LE) = 1
			if b[6] != 0x01 || b[7] != 0x00 {
				t.Fatalf("perBundle bytes: got % x, want 01 00", b[6:8])
			}
			// quantity (short LE) = 100 = 0x0064
			if b[8] != 0x64 || b[9] != 0x00 {
				t.Fatalf("quantity bytes: got % x, want 64 00", b[8:10])
			}
			// price (int LE) = 5000 = 0x00001388
			if b[10] != 0x88 || b[11] != 0x13 || b[12] != 0x00 || b[13] != 0x00 {
				t.Fatalf("price bytes: got % x, want 88 13 00 00", b[10:14])
			}
			// asset bytes follow (GW_ItemSlotBase) — presence asserts the loop body ran.
			if len(b) <= 14 {
				t.Fatalf("missing asset bytes after price; total len %d", len(b))
			}
		})
	}
}
