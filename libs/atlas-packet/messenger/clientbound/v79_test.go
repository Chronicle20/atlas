package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79DetAvatar builds an avatar with a SINGLE equipment entry so the
// map[slot.Position]uint32 iteration order is deterministic (Go randomizes map
// range order, which would make the cross-version byte compare flaky). One
// entry still exercises the >28 avatar-look path — the only MajorVersion gate,
// identical for v79 and v83.
func v79DetAvatar() model.Avatar {
	return model.NewAvatar(0, 1, 20000, false, 30000,
		map[slot.Position]uint32{5: 1040002}, map[slot.Position]uint32{}, map[int8]uint32{})
}

// v79 MESSENGER (op 0x123) family verification — CUIMessenger::OnPacket
// @0x7bc0a5 (GMS_v79_1_DEVM.exe, port 13340). The dispatcher does
// Decode1(mode) then switches:
//
//	0 -> OnEnter           @0x7b9ede  (member entered, full avatar)  -> MessengerAdd
//	1 -> OnSelfEnterResult @0x7ba11f  (self enter, slot)             -> MessengerJoin
//	2 -> OnLeave           @0x7ba20c  (slot)                         -> MessengerRemove
//	3 -> OnInvite          @0x7bc321  (fromName + messengerId)       -> MessengerRequestInvite
//	4 -> OnInviteResult    @0x7bc13c  (targetName + result flag)     -> MessengerInviteSent
//	5 -> OnBlocked         @0x7bc230  (targetName + decline mode)    -> MessengerInviteDeclined
//	6 -> OnChat            @0x7bc480  (chat text)                    -> MessengerChat
//	7 -> OnAvatar          @0x7bc7f8  (slot + AvatarLook)            -> MessengerUpdate
//	8 -> OnMigrated        @0x7bc893  (no atlas struct)
//
// The mode table is byte-identical to v83 (the family carries no non-uniform
// shift below v87). Every body codec in this package gates only on
// MajorVersion()<=28 vs >28 (avatar look) — v79 and v83 are both >28, so each
// v79 encode is byte-equal to the IDA-verified v83 encode (cross-version
// equality, the door/SpawnDoor discipline). Mode-only / scalar arms additionally
// assert their exact wire shape.

// packet-audit:verify packet=messenger/clientbound/MessengerAdd version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v79 ida=0x7bc0a5
func TestMessengerAvatarArmsV79(t *testing.T) {
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	ava := v79DetAvatar()
	type arm struct {
		name string
		v79  []byte
		v83  []byte
	}
	arms := []arm{
		{"Add", NewMessengerAdd(0, 1, ava, "TestPlayer", 3).Encode(nil, v79)(nil), NewMessengerAdd(0, 1, ava, "TestPlayer", 3).Encode(nil, v83)(nil)},
		{"Update", NewMessengerUpdate(7, 1, ava).Encode(nil, v79)(nil), NewMessengerUpdate(7, 1, ava).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v79, a.v83) {
			t.Errorf("%s v79 != v83\n v79: % x\n v83: % x", a.name, a.v79, a.v83)
		}
	}
}

// packet-audit:verify packet=messenger/clientbound/MessengerJoin version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerChat version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerInviteSent version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v79 ida=0x7bc0a5
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v79 ida=0x7bc0a5
func TestMessengerScalarArmsV79(t *testing.T) {
	v79 := pt.CreateContext("GMS", 79, 1)
	// Join: mode(1)=1 + slot(1)=2 (OnSelfEnterResult Decode1 slot).
	if got := NewMessengerJoin(1, 2).Encode(nil, v79)(nil); !bytes.Equal(got, []byte{1, 2}) {
		t.Errorf("v79 Join: got % x want 01 02", got)
	}
	// Remove: mode(1)=2 + slot(1)=4 (OnLeave Decode1 slot).
	if got := NewMessengerRemove(2, 4).Encode(nil, v79)(nil); !bytes.Equal(got, []byte{2, 4}) {
		t.Errorf("v79 Remove: got % x want 02 04", got)
	}
	// Chat: mode(1)=6 + AsciiString("Hi") (OnChat DecodeStr).
	if got := NewMessengerChat(6, "Hi").Encode(nil, v79)(nil); !bytes.Equal(got, []byte{6, 0x02, 0x00, 'H', 'i'}) {
		t.Errorf("v79 Chat: got % x want 06 02 00 48 69", got)
	}
	// InviteSent: mode(1)=4 + AsciiString("Bob") + success flag(1)=1.
	if got := NewMessengerInviteSent(4, "Bob", true).Encode(nil, v79)(nil); !bytes.Equal(got, []byte{4, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v79 InviteSent: got % x want 04 03 00 42 6f 62 01", got)
	}
	// InviteDeclined: mode(1)=5 + AsciiString("Bob") + declineMode(1)=1.
	if got := NewMessengerInviteDeclined(5, "Bob", 1).Encode(nil, v79)(nil); !bytes.Equal(got, []byte{5, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v79 InviteDeclined: got % x want 05 03 00 42 6f 62 01", got)
	}
	// RequestInvite: mode(1)=3 + AsciiString("Bob") + WriteByte(0) +
	//   messengerId(4 LE)=5 + WriteByte(0) (request_invite.go Encode order).
	want := []byte{3, 0x03, 0x00, 'B', 'o', 'b', 0x00, 0x05, 0x00, 0x00, 0x00, 0x00}
	if got := NewMessengerRequestInvite(3, "Bob", 5).Encode(nil, v79)(nil); !bytes.Equal(got, want) {
		t.Errorf("v79 RequestInvite: got % x want % x", got, want)
	}
}
