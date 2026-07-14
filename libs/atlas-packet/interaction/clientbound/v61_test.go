package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 PLAYER_INTERACTION family verification —
// CMiniRoomBaseDlg::OnPacketBase = sub_5BEC69 @0x5bec69 (GMS_v61.1_U_DEVM.exe,
// port 13338). The v61 dispatcher read order is byte-identical to the verified
// v72 (@0x60e235) / v79 (@0x62cd21): Decode1(mode); with an active mini-room it
// switches
//
//	 3 -> OnLeaveBase   (sub_5BF352) -> InteractionLeave
//	 4 -> OnAvatar      (sub_5BF5AE)
//	 6 -> vtable[80] (OnChat)         -> InteractionChat
//	 9 -> OnEnterBase   (sub_5BF6FE) -> InteractionEnter
//	10 -> OnInviteResult(sub_5BF647) -> InteractionInviteResult
//	default -> vtable[64] sub-dispatch (CPersonalShopDlg::OnPacket; mode 25 =
//	           hired-merchant refresh -> OnRefresh -> InteractionUpdateMerchant)
//
// and the no-room arms mode 2 -> OnCheckSSN2 (sub_5BF17B) / mode 5 -> OnInvite
// (sub_5BED05) cover ENTER_RESULT / INVITE. This matches the v72/v79 switch
// exactly. (The gms_v61 registry mislabels opcode 243->sub_6D34F1: that function
// is the MESSENGER dispatcher — its case 3 calls the messenger sub_6D3765
// auto-decline handler — while sub_5BEC69 is the real PlayerInteraction
// dispatcher; a registry fname swap, out of scope for these cells.) None of the
// interaction clientbound codecs carry a MajorVersion gate, so each v61 encode
// is byte-equal to the IDA-verified v83 encode (cross-version equality).
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionChat version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v61 ida=0x5bec69
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v61 ida=0x5bec69
func TestInteractionArmsV61(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	visitor := interaction.NewBaseVisitor(1, v79DetAvatar(), "Visitor")
	room := interaction.NewPersonalShopRoom(
		 0, // owner position
		[]interaction.Visitor{interaction.NewBaseVisitor(0, v79DetAvatar(), "ShopOwner")},
		"CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	items := []interaction.RoomShopItem{testShopItem()}
	type arm struct {
		name string
		v61  []byte
		v83  []byte
	}
	arms := []arm{
		{"Invite", NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v61)(nil), NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v83)(nil)},
		{"InviteResult", NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v61)(nil), NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v83)(nil)},
		{"Enter", NewInteractionEnter(9, visitor).Encode(nil, v61)(nil), NewInteractionEnter(9, visitor).Encode(nil, v83)(nil)},
		{"EnterResultSuccess", NewInteractionEnterResultSuccess(5, room).Encode(nil, v61)(nil), NewInteractionEnterResultSuccess(5, room).Encode(nil, v83)(nil)},
		{"EnterResultError", NewInteractionEnterResultError(5, 2).Encode(nil, v61)(nil), NewInteractionEnterResultError(5, 2).Encode(nil, v83)(nil)},
		{"Chat", NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v61)(nil), NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v83)(nil)},
		{"Leave", NewInteractionLeave(3, 2, 0).Encode(nil, v61)(nil), NewInteractionLeave(3, 2, 0).Encode(nil, v83)(nil)},
		{"UpdateMerchant", NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v61)(nil), NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v61, a.v83) {
			t.Errorf("%s v61 != v83\n v61: % x\n v83: % x", a.name, a.v61, a.v83)
		}
	}
}
