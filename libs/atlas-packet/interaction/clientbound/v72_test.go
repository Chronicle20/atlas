package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 PLAYER_INTERACTION family verification —
// CMiniRoomBaseDlg::OnPacketBase @0x60e235 (GMS_v72.1_U_DEVM.exe, port 13339).
// The v72 dispatcher read order is byte-identical to the verified v79
// (@0x62cd21): Decode1(mode); with an active mini-room it switches
//
//	 3 -> OnLeaveBase           @0x60e952 -> InteractionLeave
//	 4 -> OnAvatar              @0x60ebe3
//	 6 -> vtable[80] (OnChat)             -> InteractionChat
//	 9 -> OnEnterBase           @0x60ed33 -> InteractionEnter
//	10 -> OnInviteResultStatic  @0x60ec7c -> InteractionInviteResult
//	default -> vtable[64] sub-dispatch (CPersonalShopDlg::OnPacket; mode 25 =
//	           hired-merchant refresh -> InteractionUpdateMerchant)
//
// and the no-room arms mode 2 -> OnCheckSSN2Static @0x60e77d / mode 5 ->
// OnInviteStatic @0x60e2d1 cover ENTER_RESULT / INVITE. This matches the v79
// switch exactly. None of the interaction clientbound codecs carry a
// MajorVersion gate, so each v72 encode is byte-equal to the IDA-verified v83
// encode (cross-version equality, the door/SpawnDoor discipline).
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionChat version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v72 ida=0x60e235
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v72 ida=0x60e235
func TestInteractionArmsV72(t *testing.T) {
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	visitor := interaction.NewBaseVisitor(1, v79DetAvatar(), "Visitor")
	room := interaction.NewPersonalShopRoom(
		 0, // owner position
		[]interaction.Visitor{interaction.NewBaseVisitor(0, v79DetAvatar(), "ShopOwner")},
		"CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	items := []interaction.RoomShopItem{testShopItem()}
	type arm struct {
		name string
		v72  []byte
		v83  []byte
	}
	arms := []arm{
		{"Invite", NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v72)(nil), NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v83)(nil)},
		{"InviteResult", NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v72)(nil), NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v83)(nil)},
		{"Enter", NewInteractionEnter(9, visitor).Encode(nil, v72)(nil), NewInteractionEnter(9, visitor).Encode(nil, v83)(nil)},
		{"EnterResultSuccess", NewInteractionEnterResultSuccess(5, room).Encode(nil, v72)(nil), NewInteractionEnterResultSuccess(5, room).Encode(nil, v83)(nil)},
		{"EnterResultError", NewInteractionEnterResultError(5, 2).Encode(nil, v72)(nil), NewInteractionEnterResultError(5, 2).Encode(nil, v83)(nil)},
		{"Chat", NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v72)(nil), NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v83)(nil)},
		{"Leave", NewInteractionLeave(3, 2, 0).Encode(nil, v72)(nil), NewInteractionLeave(3, 2, 0).Encode(nil, v83)(nil)},
		{"UpdateMerchant", NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v72)(nil), NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v72, a.v83) {
			t.Errorf("%s v72 != v83\n v72: % x\n v83: % x", a.name, a.v72, a.v83)
		}
	}
}
