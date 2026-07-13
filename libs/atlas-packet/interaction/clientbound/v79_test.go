package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79DetAvatar builds an avatar with a SINGLE equipment entry so the
// map[slot.Position]uint32 iteration order is deterministic — the cross-version
// byte comparison would otherwise be flaky on multi-entry maps (Go randomizes
// map range order). One entry still fully exercises the >28 avatar-look path,
// which is the only MajorVersion gate (identical for v79 and v83).
func v79DetAvatar() model.Avatar {
	return model.NewAvatar(0, 1, 20000, false, 30000,
		map[slot.Position]uint32{5: 1040002}, map[slot.Position]uint32{}, map[int8]uint32{})
}

// v79 PLAYER_INTERACTION (op 0x124) family verification —
// CMiniRoomBaseDlg::OnPacketBase @0x62cd21 (GMS_v79_1_DEVM.exe, port 13340).
// The dispatcher does Decode1(mode); with an active mini-room it switches:
//
//	 3 -> OnLeaveBase            @0x62d5dd  -> InteractionLeave
//	 4 -> OnAvatar              @0x62da11
//	 6 -> vtable[80] (OnChat)              -> InteractionChat
//	 9 -> OnEnterBase           @0x62db61  -> InteractionEnter
//	10 -> OnInviteResultStatic  @0x62daaa  -> InteractionInviteResult
//	default -> vtable[64] sub-dispatch (CPersonalShopDlg::OnPacket; mode 25 =
//	           hired-merchant refresh -> InteractionUpdateMerchant)
//
// The no-room arms (OnCheckSSN2Static / OnInviteStatic / OnEnterResultStatic /
// OnLeaveBase) cover INVITE / ENTER_RESULT. v79 is GMS so the jms_v185
// personal-shop -3 sub-mode shift does NOT apply — UPDATE_MERCHANT = mode 25
// like all GMS versions. None of the interaction clientbound codecs carry a
// MajorVersion gate, so each v79 encode is byte-equal to the IDA-verified v83
// encode (cross-version equality, the door/SpawnDoor discipline).

// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInvite version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionInviteResult version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnter version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultSuccess version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionEnterResultError version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionChat version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionLeave version=gms_v79 ida=0x62cd21
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionUpdateMerchant version=gms_v79 ida=0x62cd21
func TestInteractionArmsV79(t *testing.T) {
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	visitor := interaction.NewBaseVisitor(1, v79DetAvatar(), "Visitor")
	room := interaction.NewPersonalShopRoom(
		true,
		[]interaction.Visitor{interaction.NewBaseVisitor(0, v79DetAvatar(), "ShopOwner")},
		"CoolShop", 16, []interaction.RoomShopItem{testShopItem()})
	items := []interaction.RoomShopItem{testShopItem()}
	type arm struct {
		name string
		v79  []byte
		v83  []byte
	}
	arms := []arm{
		{"Invite", NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v79)(nil), NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v83)(nil)},
		{"InviteResult", NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v79)(nil), NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v83)(nil)},
		{"Enter", NewInteractionEnter(9, visitor).Encode(nil, v79)(nil), NewInteractionEnter(9, visitor).Encode(nil, v83)(nil)},
		{"EnterResultSuccess", NewInteractionEnterResultSuccess(5, room).Encode(nil, v79)(nil), NewInteractionEnterResultSuccess(5, room).Encode(nil, v83)(nil)},
		{"EnterResultError", NewInteractionEnterResultError(5, 2).Encode(nil, v79)(nil), NewInteractionEnterResultError(5, 2).Encode(nil, v83)(nil)},
		{"Chat", NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v79)(nil), NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v83)(nil)},
		{"Leave", NewInteractionLeave(3, 2, 0).Encode(nil, v79)(nil), NewInteractionLeave(3, 2, 0).Encode(nil, v83)(nil)},
		{"UpdateMerchant", NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v79)(nil), NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v79, a.v83) {
			t.Errorf("%s v79 != v83\n v79: % x\n v83: % x", a.name, a.v79, a.v83)
		}
	}
}
