package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestClaimRequestChatClaimGolden(t *testing.T) {
	// bChatClaim=1 appends the client-supplied chat log string.
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{
		0x01,                         // bChatClaim
		0x03, 0x00, 0x62, 0x6F, 0x62, // "bob"
		0x02,                   // nType
		0x02, 0x00, 0x68, 0x69, // "hi"
		0x02, 0x00, 0x79, 0x6F, // "yo"
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimRequestRegularGolden(t *testing.T) {
	// bChatClaim=0: no chat log trailer.
	input := NewClaimRequest(0, "bob", 0x05, "hi", "")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x00,
		0x03, 0x00, 0x62, 0x6F, 0x62,
		0x05,
		0x02, 0x00, 0x68, 0x69,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}

	// Verify decode direction: bChatClaim=0 branch must NOT read a trailing chatLog.
	l, _ := testlog.NewNullLogger()
	output := ClaimRequest{}
	req := request.Request(expected)
	reader := request.NewRequestReader(&req, 0)
	output.Decode(l, ctx)(&reader, nil)

	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
	if output.IsChatClaim() != input.IsChatClaim() || output.TargetName() != input.TargetName() ||
		output.ReasonType() != input.ReasonType() || output.Description() != input.Description() ||
		output.ChatLog() != input.ChatLog() {
		t.Errorf("decode mismatch: got %+v want %+v", output, input)
	}
}

// TestClaimRequestByteOutputV72 verifies the wire-exact byte output of
// ClaimRequest for GMS v72.
// IDA evidence (session c8acae95, GMS_v72.1_U_DEVM.exe.i64):
//
//	CWvsContext::SendClaimRequest@0x91f2b4 (independently re-decompiled this
//	pass, not just re-cited from the registry's task-23b note):
//	COutPacket::COutPacket(&pkt, 105) @0x91f749 -- opcode 105/0x69, matches
//	registry gms_v72.yaml op CLAIM_REQUEST. Encode1(v66=bChatClaim) @0x91f758
//	(v66 set to 1 in the mode==1000 branch, 0 in the mode==1001 branch of the
//	preceding dialog-result switch). EncodeStr(targetName) @0x91f771.
//	Encode1(v58=reasonType) @0x91f77c. EncodeStr(description) @0x91f795.
//	Guarded EncodeStr(chatLog) @0x91f80d, gated on the same v66 that fed the
//	first Encode1. CClientSocket::SendPacket @0x91f828. Byte-identical to the
//	v83/v95 shape already coded in the (version-ungated) Encode below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v72 ida=0x91f2b4
func TestClaimRequestByteOutputV72(t *testing.T) {
	v := pt.Variants[9] // GMS v72
	if v.Name != "GMS v72" {
		t.Fatalf("pt.Variants[9] = %q, want %q (index drifted)", v.Name, "GMS v72")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0x91f758
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0x91f771
		0x02,                   // Encode1 reasonType @0x91f77c
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0x91f795
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0x91f80d
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimRequestByteOutputV79 verifies the wire-exact byte output of
// ClaimRequest for GMS v79.
// IDA evidence (session 1438cecd, GMS_v79_1_DEVM.exe.i64):
//
//	CWvsContext::SendClaimRequest@0x9711ff (independently re-decompiled this
//	pass, not just re-cited from the registry's task-23b note):
//	COutPacket::COutPacket(&pkt, 104) @0x971694 -- opcode 104/0x68, matches
//	registry gms_v79.yaml op CLAIM_REQUEST. Encode1(v66=bChatClaim) @0x9716a3
//	(v66 set to 1 in the mode==1000 branch, 0 in the mode==1001 branch of the
//	preceding dialog-result switch). EncodeStr(targetName) @0x9716bc.
//	Encode1(v58=reasonType) @0x9716c7. EncodeStr(description) @0x9716e0.
//	Guarded EncodeStr(chatLog) @0x971758, gated on the same v66 that fed the
//	first Encode1. CClientSocket::SendPacket @0x971773. Byte-identical to the
//	v72/v83/v95 shape already coded in the (version-ungated) Encode below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v79 ida=0x9711ff
func TestClaimRequestByteOutputV79(t *testing.T) {
	v := pt.Variants[10] // GMS v79
	if v.Name != "GMS v79" {
		t.Fatalf("pt.Variants[10] = %q, want %q (index drifted)", v.Name, "GMS v79")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0x9716a3
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0x9716bc
		0x02,                   // Encode1 reasonType @0x9716c7
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0x9716e0
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0x971758
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimRequest(1, "alice", 0x03, "harassment in fm1", "alice: mean words")
			output := ClaimRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsChatClaim() != input.IsChatClaim() || output.TargetName() != input.TargetName() ||
				output.ReasonType() != input.ReasonType() || output.Description() != input.Description() ||
				output.ChatLog() != input.ChatLog() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
