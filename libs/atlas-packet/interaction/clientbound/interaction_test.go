package clientbound

import (
	"bytes"
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
// in jms is mode 22 (NOT 25): jms shifts the personal-shop sub-modes down by 3 vs gms
// (gms 24/25/26/27 buy/refresh/sold/move = jms 21/22/23/24). The mode-22 default case
// virtual-dispatches into CPersonalShopDlg::OnPacket sub_761650 case 22 -> vtable[28]
// OnRefresh override = CEntrustedShopDlg::OnRefresh @0x54adb9 (Decode4 meso -> chains
// CPersonalShopDlg::OnRefresh sub_761dba count+item loop)):
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=jms_v185 ida=0x6da7b4
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=jms_v185 ida=0x6daa56
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=jms_v185 ida=0x6dace2
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=jms_v185 ida=0x6da234
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=jms_v185 ida=0x6da234
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=jms_v185 ida=0x6dad8a
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=jms_v185 ida=0x54adb9
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
	room := interaction.NewPersonalShopRoom(true, visitors, "CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	input := NewInteractionEnterResultSuccess(5, room)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionEnterResultSuccess{}).Decode, nil)
		})
	}
}

// TestInteractionEnterResultSuccessBytes is the byte-exact fixture that pins the
// EnterResultSuccess (mode 5) header — including the second header byte the encoder
// previously omitted, which shifted the visitor list and caused the live v83
// "error 38" over-read on personal-store setup.
//
// Read order (all versions read it identically — CMiniRoomBaseDlg::OnEnterResultBase
// is version-stable: v83 @0x65ec3d, v87 sub_698F32 @0x698f32, v95 @0x638e30,
// jms sub_6DABDB @0x6dabdb — each: Decode1 capacity, Decode1 header, visitor loop):
//
//	mode                 = 5     (ENTER_RESULT; passed to NewInteractionEnterResultSuccess)
//	roomType             = 4     (PersonalShop; Room.Encode, OnEnterResultStatic Decode1 @0x65e02b)
//	capacity             = 4     (OnEnterResultBase Decode1 -> *(this+0xCC) @0x65ec5d)
//	ownerView byte       = 1/0   (OnEnterResultBase Decode1 -> *(this+0xC8) @0x65ec6b; the fix)
//	0xFF                 =       (empty visitor list; slot<0 breaks loop @0x65ec7d)
//	title "AB"           = 02 00 41 42   (CPersonalShopDlg::OnEnterResult DecodeStr @0x6fc62a)
//	maxItemCount = 16    = 10    (Decode1 -> *(this+109) @0x6fc683)
//	itemCount = 0        = 00    (CPersonalShopDlg::OnRefresh Decode1 @0x6fcc64; vtable+112 @0xAFD498)
//
// The client branches on the ownerView byte in CPersonalShopDlg::OnEnterResult
// @0x6fc528 (`if(*(this+50))`): 1 = owner add-item management UI, 0 = visitor buy UI.
// (The EnterResultSuccess packet-audit:verify markers for every version live in the
// block above TestInteractionInviteRoundTrip; this fixture backs the gms_v83 one.)
func TestInteractionEnterResultSuccessBytes(t *testing.T) {
	cases := []struct {
		name      string
		ownerView bool
		want      []byte
	}{
		{"owner", true, []byte{0x05, 0x04, 0x04, 0x01, 0xFF, 0x02, 0x00, 0x41, 0x42, 0x10, 0x00}},
		{"visitor", false, []byte{0x05, 0x04, 0x04, 0x00, 0xFF, 0x02, 0x00, 0x41, 0x42, 0x10, 0x00}},
	}
	for _, tc := range cases {
		room := interaction.NewPersonalShopRoom(tc.ownerView, nil, "AB", 16, nil)
		input := NewInteractionEnterResultSuccess(5, room)
		for _, v := range test.Variants {
			t.Run(tc.name+"/"+v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				b := test.Encode(t, ctx, input.Encode, nil)
				if !bytes.Equal(b, tc.want) {
					t.Fatalf("bytes: got % x, want % x", b, tc.want)
				}
			})
		}
	}
}

// TestInteractionEnterResultSuccessMerchantHeaderBytes pins ONLY the version-stable
// header region of the hired-merchant (roomType 5) enter-result: the same
// OnEnterResultBase capacity + ownerView-byte + visitor-terminator that the fix
// restores. The tail (messages/ownerName/meso/items) is NOT asserted here because
// Atlas's current merchant tail diverges from CEntrustedShopDlg::OnEnterResult
// @0x518873 (owner vs customer read different fields — meso/sold-items/total via
// DecodeSoldItemList @0x518efd only in the customer branch, plus a trailing title +
// maxItem byte). That tail rework is tracked separately; see the task report.
func TestInteractionEnterResultSuccessMerchantHeaderBytes(t *testing.T) {
	// visitor (ownerView=false): roomType 5, capacity 4, header 0, then 0xFF.
	room := interaction.NewMerchantShopRoom(false, nil, nil, "", 16, 0, nil)
	input := NewInteractionEnterResultSuccess(5, room)
	wantHeader := []byte{0x05, 0x05, 0x04, 0x00, 0xFF}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := test.Encode(t, ctx, input.Encode, nil)
			if len(b) < len(wantHeader) || !bytes.Equal(b[:len(wantHeader)], wantHeader) {
				t.Fatalf("header bytes: got % x, want prefix % x", b, wantHeader)
			}
		})
	}
}

// TestInteractionUpdateMerchantBytes is an encode-only byte fixture for the
// UPDATE_MERCHANT mode. The hired-merchant refresh leaf CEntrustedShopDlg::OnRefresh
// reads Decode4(meso) then chains CPersonalShopDlg::OnRefresh: Decode1(count) +
// count x {Decode2 perBundle, Decode2 quantity, Decode4 price, GW_ItemSlotBase}.
// Each asserted byte traces to that read order. The leading mode byte is
// version-dependent: 25 in gms_v83/v84/v87/v95 (IDA: v83 0x6fc42d / v95 0x69c820
// switch case 25 -> OnRefresh), 22 in jms_v185 (IDA: CPersonalShopDlg::OnPacket
// sub_761650 case 22 -> CEntrustedShopDlg::OnRefresh sub_54adb9), because jms
// shifts the personal-shop sub-modes down by 3.
func TestInteractionUpdateMerchantBytes(t *testing.T) {
	items := []interaction.RoomShopItem{testShopItem()}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			// JMS personal-shop refresh = mode 22; all GMS versions = mode 25.
			wantMode := byte(25)
			if v.Region == "JMS" {
				wantMode = 22
			}
			input := NewInteractionUpdateMerchant(wantMode, 50000, items)
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := test.Encode(t, ctx, input.Encode, nil)
			// mode + meso(50000 LE) + count(1) + item{perBundle,quantity,price} + asset(...)
			// mode
			if b[0] != wantMode {
				t.Fatalf("mode: got %d, want %d", b[0], wantMode)
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
