package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 MESSENGER family verification — CUIMessenger::OnPacket @0x777b25
// (GMS_v72.1_U_DEVM.exe, port 13339). The dispatcher does Decode1(mode) then
// switches, byte-identical mode table to v79/v83:
//
//	0 -> OnEnter           -> MessengerAdd
//	1 -> OnSelfEnterResult -> MessengerJoin
//	2 -> OnLeave           -> MessengerRemove
//	3 -> OnInvite          -> MessengerRequestInvite
//	4 -> OnInviteResult    -> MessengerInviteSent
//	5 -> OnBlocked         -> MessengerInviteDeclined
//	6 -> OnChat            -> MessengerChat
//	7 -> OnAvatar          -> MessengerUpdate
//	8 -> OnMigrated        (no atlas struct)
//
// Each body codec gates only on MajorVersion()<=28 vs >28 (avatar look); v72 and
// v83 are both >28, so each v72 encode is byte-equal to the IDA-verified v83 encode
// (cross-version equality). Mode-only/scalar arms additionally assert their exact wire shape.

// packet-audit:verify packet=messenger/clientbound/MessengerAdd version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v72 ida=0x777b25
func TestMessengerAvatarArmsV72(t *testing.T) {
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	ava := v79DetAvatar()
	type arm struct {
		name string
		v72  []byte
		v83  []byte
	}
	arms := []arm{
		{"Add", NewMessengerAdd(0, 1, ava, "TestPlayer", 3).Encode(nil, v72)(nil), NewMessengerAdd(0, 1, ava, "TestPlayer", 3).Encode(nil, v83)(nil)},
		{"Update", NewMessengerUpdate(7, 1, ava).Encode(nil, v72)(nil), NewMessengerUpdate(7, 1, ava).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v72, a.v83) {
			t.Errorf("%s v72 != v83\n v72: % x\n v83: % x", a.name, a.v72, a.v83)
		}
	}
}

// packet-audit:verify packet=messenger/clientbound/MessengerJoin version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerChat version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerInviteSent version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v72 ida=0x777b25
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v72 ida=0x777b25
func TestMessengerScalarArmsV72(t *testing.T) {
	v72 := pt.CreateContext("GMS", 72, 1)
	if got := NewMessengerJoin(1, 2).Encode(nil, v72)(nil); !bytes.Equal(got, []byte{1, 2}) {
		t.Errorf("v72 Join: got % x want 01 02", got)
	}
	if got := NewMessengerRemove(2, 4).Encode(nil, v72)(nil); !bytes.Equal(got, []byte{2, 4}) {
		t.Errorf("v72 Remove: got % x want 02 04", got)
	}
	if got := NewMessengerChat(6, "Hi").Encode(nil, v72)(nil); !bytes.Equal(got, []byte{6, 0x02, 0x00, 'H', 'i'}) {
		t.Errorf("v72 Chat: got % x want 06 02 00 48 69", got)
	}
	if got := NewMessengerInviteSent(4, "Bob", true).Encode(nil, v72)(nil); !bytes.Equal(got, []byte{4, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v72 InviteSent: got % x want 04 03 00 42 6f 62 01", got)
	}
	if got := NewMessengerInviteDeclined(5, "Bob", 1).Encode(nil, v72)(nil); !bytes.Equal(got, []byte{5, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v72 InviteDeclined: got % x want 05 03 00 42 6f 62 01", got)
	}
	want := []byte{3, 0x03, 0x00, 'B', 'o', 'b', 0x00, 0x05, 0x00, 0x00, 0x00, 0x00}
	if got := NewMessengerRequestInvite(3, "Bob", 5).Encode(nil, v72)(nil); !bytes.Equal(got, want) {
		t.Errorf("v72 RequestInvite: got % x want % x", got, want)
	}
}
