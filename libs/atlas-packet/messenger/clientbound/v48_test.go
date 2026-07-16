package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 MESSENGER family — clientbound dispatcher CUIMessenger::OnPacket sub_61D8B8
// @0x61d8b8 (GMS_v48_1_DEVM.exe, port 13337), routed from CField::OnPacket case 238.
// Decode1(mode) then switch; case 3 (RequestInvite) is special-cased before the
// window-open guard. The mode table is byte-identical to v72/v79/v83 (cases 0..8):
//
//	0 -> sub_61B860 OnEnter           -> MessengerAdd
//	     Decode1(position)+AvatarLook::Decode+DecodeStr(name)+Decode1(channelId)+Decode1(pad)
//	1 -> sub_61BA71 OnSelfEnterResult -> MessengerJoin     Decode1(position)
//	2 -> sub_61BB63 OnLeave           -> MessengerRemove   Decode1(position)
//	3 -> sub_61DB2C OnInvite          -> MessengerRequestInvite
//	     DecodeStr(fromName)+Decode1(pad)+Decode4(messengerId)+Decode1(pad)
//	4 -> sub_61D94F OnInviteResult    -> MessengerInviteSent    DecodeStr(msg)+Decode1(success)
//	5 -> sub_61DA3F OnBlocked         -> MessengerInviteDeclined DecodeStr(msg)+Decode1(declineMode)
//	6 -> sub_61DC8C OnChat            -> MessengerChat     DecodeStr(message)
//	7 -> sub_61DF11 OnAvatar          -> MessengerUpdate   Decode1(position)+AvatarLook::Decode
//	8 -> sub_61DF6C OnMigrated        (no atlas struct)
//
// Each per-mode body read order is byte-identical to the IDA-verified v83 codec
// (@0x8511fc); the avatar arms use cross-version equality (model.Avatar gates only
// on GMS<=28 vs >28, and v48 is >28 == v83), the scalar arms assert exact wire.

// The avatar arms cannot use cross-version equality to v83: v48 is < 61, so
// AvatarLook (model.Avatar) takes the legacy single-4-byte-pet path (IDA sub_49E1E0
// @0x49e2b9) instead of v83's three pet ints, making the avatar block genuinely
// shorter. The messenger-owned FRAME (mode + position + [avatar block] + name +
// channelId + pad for Add; mode + position + [avatar block] for Update) is the
// IDA-verified read order of sub_61B860 / sub_61DF11; a single-equip deterministic
// avatar + round-trip + boundary-byte assertions verify that frame.

// packet-audit:verify packet=messenger/clientbound/MessengerAdd version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v48 ida=0x61d8b8
func TestMessengerAvatarArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	ava := v79DetAvatar()

	// Add: mode + position + avatar + name + channelId + pad.
	addWire := NewMessengerAdd(0, 2, ava, "TestPlayer", 3).Encode(nil, v48)(nil)
	if addWire[0] != 0 || addWire[1] != 2 {
		t.Errorf("v48 Add head: got % x want mode=00 position=02", addWire[:2])
	}
	// Tail: name "TestPlayer" (len 10) then channelId (3) + pad (0).
	wantTail := []byte{0x0a, 0x00, 'T', 'e', 's', 't', 'P', 'l', 'a', 'y', 'e', 'r', 0x03, 0x00}
	if n := len(addWire); n < len(wantTail) || !bytes.Equal(addWire[n-len(wantTail):], wantTail) {
		t.Errorf("v48 Add tail: got % x want % x (name+channelId+pad)", addWire[len(addWire)-len(wantTail):], wantTail)
	}
	addOut := Add{}
	pt.RoundTrip(t, v48, NewMessengerAdd(0, 2, ava, "TestPlayer", 3).Encode, addOut.Decode, nil)
	if addOut.Mode() != 0 || addOut.Position() != 2 || addOut.Name() != "TestPlayer" || addOut.ChannelId() != 3 {
		t.Errorf("v48 Add round-trip: mode=%d position=%d name=%q channelId=%d",
			addOut.Mode(), addOut.Position(), addOut.Name(), addOut.ChannelId())
	}
	if len(addOut.Avatar().Equipment()) != len(ava.Equipment()) {
		t.Errorf("v48 Add avatar equip count: got %d want %d", len(addOut.Avatar().Equipment()), len(ava.Equipment()))
	}

	// Update: mode + position + avatar (no name / channelId).
	updWire := NewMessengerUpdate(7, 1, ava).Encode(nil, v48)(nil)
	if updWire[0] != 7 || updWire[1] != 1 {
		t.Errorf("v48 Update head: got % x want mode=07 position=01", updWire[:2])
	}
	updOut := Update{}
	pt.RoundTrip(t, v48, NewMessengerUpdate(7, 1, ava).Encode, updOut.Decode, nil)
	if updOut.Mode() != 7 || updOut.Position() != 1 {
		t.Errorf("v48 Update round-trip: mode=%d position=%d", updOut.Mode(), updOut.Position())
	}
	if len(updOut.Avatar().Equipment()) != len(ava.Equipment()) {
		t.Errorf("v48 Update avatar equip count: got %d want %d", len(updOut.Avatar().Equipment()), len(ava.Equipment()))
	}
}

// packet-audit:verify packet=messenger/clientbound/MessengerJoin version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerChat version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerInviteSent version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v48 ida=0x61d8b8
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v48 ida=0x61d8b8
func TestMessengerScalarArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	if got := NewMessengerJoin(1, 2).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{1, 2}) {
		t.Errorf("v48 Join: got % x want 01 02", got)
	}
	if got := NewMessengerRemove(2, 4).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{2, 4}) {
		t.Errorf("v48 Remove: got % x want 02 04", got)
	}
	if got := NewMessengerChat(6, "Hi").Encode(nil, v48)(nil); !bytes.Equal(got, []byte{6, 0x02, 0x00, 'H', 'i'}) {
		t.Errorf("v48 Chat: got % x want 06 02 00 48 69", got)
	}
	if got := NewMessengerInviteSent(4, "Bob", true).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{4, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v48 InviteSent: got % x want 04 03 00 42 6f 62 01", got)
	}
	if got := NewMessengerInviteDeclined(5, "Bob", 1).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{5, 0x03, 0x00, 'B', 'o', 'b', 1}) {
		t.Errorf("v48 InviteDeclined: got % x want 05 03 00 42 6f 62 01", got)
	}
	want := []byte{3, 0x03, 0x00, 'B', 'o', 'b', 0x00, 0x05, 0x00, 0x00, 0x00, 0x00}
	if got := NewMessengerRequestInvite(3, "Bob", 5).Encode(nil, v48)(nil); !bytes.Equal(got, want) {
		t.Errorf("v48 RequestInvite: got % x want % x", got, want)
	}
}
