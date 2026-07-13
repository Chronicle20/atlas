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
// cover ENTER_RESULT / INVITE. This matches the v61 switch exactly.
//
// The interaction clientbound dispatcher codecs themselves carry no MajorVersion
// gate, so non-avatar arms are byte-equal to the IDA-verified v83 encode. The
// avatar-bearing arms (Enter, EnterResultSuccess) embed AvatarLook, which DOES
// diverge: the shared AvatarLook::Decode (v48 sub_49E1E0 @0x49e2b9) reads a single
// 4-byte pet int where v61/v83 (@0x4b77b1) read DecodeBuffer(12)=3 pet ints. So a
// v48 avatar block is exactly 8 bytes shorter than v83 per embedded avatar. Avatar
// content itself is byte-verified by the model + char-list v48 fixtures.
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
		// avatarCount is the number of embedded AvatarLook blocks; each accounts
		// for an 8-byte v48<->v83 pet-section delta (single int vs 3 ints).
		avatarCount int
	}
	arms := []arm{
		{"Invite", NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v48)(nil), NewInteractionInvite(4, 1, "TestPlayer", 12345).Encode(nil, v83)(nil), 0},
		{"InviteResult", NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v48)(nil), NewInteractionInviteResult(5, 1, "Room is full").Encode(nil, v83)(nil), 0},
		{"Enter", NewInteractionEnter(9, visitor).Encode(nil, v48)(nil), NewInteractionEnter(9, visitor).Encode(nil, v83)(nil), 1},
		{"EnterResultSuccess", NewInteractionEnterResultSuccess(5, room).Encode(nil, v48)(nil), NewInteractionEnterResultSuccess(5, room).Encode(nil, v83)(nil), 1},
		{"EnterResultError", NewInteractionEnterResultError(5, 2).Encode(nil, v48)(nil), NewInteractionEnterResultError(5, 2).Encode(nil, v83)(nil), 0},
		{"Chat", NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v48)(nil), NewInteractionChat(6, 7, 1, "Player : Hello").Encode(nil, v83)(nil), 0},
		{"Leave", NewInteractionLeave(3, 2, 0).Encode(nil, v48)(nil), NewInteractionLeave(3, 2, 0).Encode(nil, v83)(nil), 0},
		{"UpdateMerchant", NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v48)(nil), NewInteractionUpdateMerchant(25, 50000, items).Encode(nil, v83)(nil), 0},
	}
	for _, a := range arms {
		if a.avatarCount == 0 {
			if !bytes.Equal(a.v48, a.v83) {
				t.Errorf("%s v48 != v83\n v48: % x\n v83: % x", a.name, a.v48, a.v83)
			}
			continue
		}
		// Avatar-bearing arm: v48 is exactly 8*avatarCount bytes shorter than v83.
		if len(a.v48) != len(a.v83)-8*a.avatarCount {
			t.Errorf("%s v48 len %d, want v83 len %d - %d (single-int pet)\n v48: % x\n v83: % x",
				a.name, len(a.v48), len(a.v83), 8*a.avatarCount, a.v48, a.v83)
		}
	}
}
