package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationDeclineInviteV61Body pins the gms_v61 messenger DECLINE-invite
// serverbound wire (op 110).
//
// IDA-verified sender — sub_6D3765 @0x6d3765 (GMS_v61.1_U_DEVM.exe, port 13338),
// the incoming-invite handler's auto-decline (blacklist) path. This is the ONLY
// v61 send-site that emits messenger sub-op 5; the CUIFadeYesNo "No" button
// (CUIFadeYesNo::OnButtonClicked @0x4df6a6, case 0) sends nothing:
//
//	COutPacket::COutPacket(v9, 110)           @0x… → opcode 110 (registry op 110,
//	                                                 template MESSENGER handler).
//	Encode1(5u)                               → sub-op 5 = DECLINE.
//	EncodeStr(fromName)                        → the inviter's name (DecodeStr'd
//	                                             from the incoming invite).
//	EncodeStr(CWvsContext::GetCharacterName)   → my own character name.
//	Encode1(1u)                                → trailing flag; v61 always emits 1
//	                                             here (atlas models it as alwaysZero;
//	                                             the field is on the wire either way).
//	CClientSocket::SendPacket
//
// Atlas messenger/serverbound OperationDeclineInvite is the SERVER-side decoder:
// the leading sub-op byte (5) is consumed by the MessengerOperationHandle
// dispatcher before this codec runs, so the codec models [fromName, myName,
// trailing byte]. Version-agnostic; the v61 body order matches the codec.
// WriteAsciiString = uint16-LE length + bytes; WriteByte = one byte.
//
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v61 ida=0x6d3765
func TestOperationDeclineInviteV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := OperationDeclineInvite{
		fromName:   "Bob",
		myName:     "Me",
		alwaysZero: 1, // v61 sub_6D3765 emits Encode1(1u)
	}
	want := []byte{
		0x03, 0x00, 'B', 'o', 'b', // fromName = "Bob" (EncodeStr)
		0x02, 0x00, 'M', 'e', // myName = "Me" (EncodeStr)
		0x01, // trailing flag = 1 (Encode1(1u) @sub_6D3765)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 OperationDeclineInvite golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
