package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 MESSENGER serverbound family — client sends via COutPacket(92) (op 0x5C),
// Encode1(mode) then a per-mode body. IDA-verified send-sites (GMS_v48_1_DEVM.exe,
// port 13337):
//
//	mode 0 AnswerInvite (OnCreate)  sub_61A701 @0x61a701  Encode1(0)+Encode4(messengerId)
//	mode 2 Operation/leave (OnDestroy) sub_61AC75 @0x61ac75  Encode1(2)  (mode only)
//	mode 3 Invite (SendInviteMsg)   dispatcher sub_61D8B8 @0x61d8b8  Encode1(3)+EncodeStr(target)
//	mode 5 DeclineInvite            sub_4BCE54 @0x4bce54  Encode1(5)+EncodeStr(from)+EncodeStr(me)+Encode1(0)
//	mode 6 Chat (ProcessChat)       sub_61B27C @0x61b27c  Encode1(6)+EncodeStr(charName+text)
//
// The atlas codecs model the body AFTER the leading mode byte is consumed by the
// MessengerOperationHandle dispatcher (except Operation, which itself carries the
// mode byte). Bodies are version-agnostic; v48 send bodies match the codecs.

// packet-audit:verify packet=messenger/serverbound/MessengerOperation version=gms_v48 ida=0x61ac75
// packet-audit:verify packet=messenger/serverbound/MessengerOperationAnswerInvite version=gms_v48 ida=0x61a701
// packet-audit:verify packet=messenger/serverbound/MessengerOperationChat version=gms_v48 ida=0x61b27c
// packet-audit:verify packet=messenger/serverbound/MessengerOperationInvite version=gms_v48 ida=0x61d8b8
func TestMessengerServerboundArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)

	// mode 2 leave: Operation encodes the mode byte itself.
	if got := (Operation{mode: 2}).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{2}) {
		t.Errorf("v48 Operation(leave): got % x want 02", got)
	}
	// mode 0 answer-invite body: Encode4(messengerId).
	if got := (OperationAnswerInvite{messengerId: 5}).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{5, 0, 0, 0}) {
		t.Errorf("v48 OperationAnswerInvite: got % x want 05 00 00 00", got)
	}
	// mode 6 chat body: EncodeStr(text).
	if got := (OperationChat{msg: "Hi"}).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{0x02, 0x00, 'H', 'i'}) {
		t.Errorf("v48 OperationChat: got % x want 02 00 48 69", got)
	}
	// mode 3 invite body: EncodeStr(target).
	if got := (OperationInvite{targetCharacter: "Bob"}).Encode(nil, v48)(nil); !bytes.Equal(got, []byte{0x03, 0x00, 'B', 'o', 'b'}) {
		t.Errorf("v48 OperationInvite: got % x want 03 00 42 6f 62", got)
	}

	// Round-trips confirm the decoders mirror the encoders.
	ao := Operation{}
	pt.RoundTrip(t, v48, (Operation{mode: 2}).Encode, ao.Decode, nil)
	if ao.Mode() != 2 {
		t.Errorf("v48 Operation round-trip mode: got %d want 2", ao.Mode())
	}
	ai := OperationAnswerInvite{}
	pt.RoundTrip(t, v48, (OperationAnswerInvite{messengerId: 5}).Encode, ai.Decode, nil)
	if ai.MessengerId() != 5 {
		t.Errorf("v48 OperationAnswerInvite round-trip: got %d want 5", ai.MessengerId())
	}
	ac := OperationChat{}
	pt.RoundTrip(t, v48, (OperationChat{msg: "Hi"}).Encode, ac.Decode, nil)
	if ac.Msg() != "Hi" {
		t.Errorf("v48 OperationChat round-trip: got %q want Hi", ac.Msg())
	}
	iv := OperationInvite{}
	pt.RoundTrip(t, v48, (OperationInvite{targetCharacter: "Bob"}).Encode, iv.Decode, nil)
	if iv.TargetCharacter() != "Bob" {
		t.Errorf("v48 OperationInvite round-trip: got %q want Bob", iv.TargetCharacter())
	}
}

// TestMessengerDeclineInviteV48 pins the mode-5 DECLINE body (op 92). IDA-verified
// send sub_4BCE54 @0x4bce54 (CFadeWnd::SendCloseMessage role) arm this[47]==0:
// COutPacket(92)+Encode1(5)+EncodeStr(fromName)+EncodeStr(myName)+Encode1(0). The
// atlas codec models the body after the dispatcher-consumed mode byte.
//
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v48 ida=0x4bce54
func TestMessengerDeclineInviteV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	input := OperationDeclineInvite{fromName: "Bob", myName: "Me", alwaysZero: 0}
	want := []byte{0x03, 0x00, 'B', 'o', 'b', 0x02, 0x00, 'M', 'e', 0x00}
	if got := pt.Encode(t, v48, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 OperationDeclineInvite golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
