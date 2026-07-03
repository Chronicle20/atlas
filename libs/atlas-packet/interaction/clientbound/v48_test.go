package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 PLAYER_INTERACTION clientbound dispatcher —
// CMiniRoomBaseDlg::OnPacketBase = sub_5459C4 @0x5459c4 (GMS_v48_1_DEVM.exe,
// port 13337). The v48 dispatcher read order is byte-identical to the verified
// v61 (sub_5BEC69): Decode1(mode); with an active mini-room it switches
//
//	 3 -> OnLeaveBase    (sub_54607D) -> InteractionLeave
//	 4 -> OnAvatar       (sub_5462E3)
//	 6 -> vtable[+76] (OnChat)         -> InteractionChat
//	 9 -> OnEnterBase    (sub_546433) -> InteractionEnter
//	10 -> OnInviteResult (sub_54637C) -> InteractionInviteResult
//	default -> vtable[+60] sub-dispatch (CPersonalShopDlg::OnPacket; mode 0x18 =
//	           hired-merchant refresh -> OnRefresh -> InteractionUpdateMerchant)
//
// and the no-room arms mode 2 -> OnInvite (sub_545EA6) / mode 5 -> sub_545A60
// cover ENTER_RESULT / INVITE. This matches the v61 switch exactly. None of the
// interaction clientbound codecs carry a MajorVersion gate, so each v48 encode
// is byte-equal to the IDA-verified v83 encode (cross-version equality).
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v48 ida=0x5459c4
func TestInteractionArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	visitor := interaction.NewBaseVisitor(1, v79DetAvatar(), "Visitor")
	room := interaction.NewPersonalShopRoom(
		[]interaction.Visitor{interaction.NewBaseVisitor(0, v79DetAvatar(), "ShopOwner")},
		"CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	items := []interaction.RoomShopItem{testShopItem()}
	type arm struct {
		name string
		v48  []byte
		v83  []byte
	}
	arms := []arm{
		{"Invite", NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v48)(nil), NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v83)(nil)},
		{"InviteResult", NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v48)(nil), NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v83)(nil)},
		{"Enter", NewInteractionEnter(9, visitor).Encode(nil, v48)(nil), NewInteractionEnter(9, visitor).Encode(nil, v83)(nil)},
		{"EnterResultSuccess", NewInteractionEnterResultSuccess(5, room).Encode(nil, v48)(nil), NewInteractionEnterResultSuccess(5, room).Encode(nil, v83)(nil)},
		{"EnterResultError", NewInteractionEnterResultError(5, 2).Encode(nil, v48)(nil), NewInteractionEnterResultError(5, 2).Encode(nil, v83)(nil)},
		{"Chat", NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v48)(nil), NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v83)(nil)},
		{"Leave", NewInteractionLeave(3, 2, 0).Encode(nil, v48)(nil), NewInteractionLeave(3, 2, 0).Encode(nil, v83)(nil)},
		{"UpdateMerchant", NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v48)(nil), NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v48, a.v83) {
			t.Errorf("%s v48 != v83\n v48: % x\n v83: % x", a.name, a.v48, a.v83)
		}
	}
}
