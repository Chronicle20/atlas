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

// TestClaimRequestByteOutputV83 verifies the wire-exact byte output of
// ClaimRequest for GMS v83.
// IDA evidence (session 41f13e0d, v83_Me MapleStory_dump.exe.i64):
//
//	CWvsContext::SendClaimRequest@0xa2719c (independently re-decompiled this
//	pass, not just re-cited from the registry's task-23/task-23c notes):
//	COutPacket::COutPacket(&v45, 0x6A) @0xa27631 -- opcode 106/0x6A, matches
//	registry gms_v83.yaml op CLAIM_REQUEST. COutPacket::Encode1(bChatClaim)
//	@0xa27640 (bChatClaim set to 1 in the DoModal==1000 branch @0xa274d2, 0 in
//	the DoModal==1001 branch @0xa272bd of the preceding dialog-result switch).
//	COutPacket::EncodeStr(targetName) @0xa27659. COutPacket::Encode1(reasonType)
//	@0xa27664. COutPacket::EncodeStr(description) @0xa2767d. Guarded
//	COutPacket::EncodeStr(chatLog) @0xa276f5, gated on the same bChatClaim byte
//	(*a2) that fed the first Encode1. CClientSocket::SendPacket @0xa27710.
//	Byte-identical to the v72/v79/v95 shape already coded in the
//	(version-ungated) Encode below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v83 ida=0xa2719c
func TestClaimRequestByteOutputV83(t *testing.T) {
	v := pt.Variants[1] // GMS v83
	if v.Name != "GMS v83" {
		t.Fatalf("pt.Variants[1] = %q, want %q (index drifted)", v.Name, "GMS v83")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0xa27640
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0xa27659
		0x02,                   // Encode1 reasonType @0xa27664
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0xa2767d
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0xa276f5
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimRequestByteOutputV84 verifies the wire-exact byte output of
// ClaimRequest for GMS v84.
// IDA evidence (session 5881cf84, GMS_v84.1_U_DEVM.i64):
//
//	CWvsContext::SendClaimRequest@0xa72957 (named this pass; was
//	sub_A72957, size 0x6bf -- unnamed in the shipped v84 IDB, located by
//	byte-scanning for the "6A 6A" push-0x6A-then-push-0x6A opcode pattern
//	near the OnClaimResult/OnSetClaimSvrAvailableTime cluster and confirming
//	the COutPacket ctor site): COutPacket::COutPacket((COutPacket*)&v42, 106)
//	@0xa72dec -- opcode 106/0x6A, matches registry gms_v84.yaml op
//	CLAIM_REQUEST. COutPacket::Encode1(bChatClaim=v60) @0xa72dfb (v60 set to
//	1 in the DoModal==1000 branch @0xa72c8d, 0 in the DoModal==1001 branch
//	@0xa72a78 of the preceding dialog-result switch). COutPacket::EncodeStr
//	(targetName) @0xa72e14. COutPacket::Encode1(reasonType=v52) @0xa72e1f.
//	COutPacket::EncodeStr(description) @0xa72e38. Guarded
//	COutPacket::EncodeStr(chatLog) @0xa72eb0, gated on the same v60
//	(bChatClaim) that fed the first Encode1. CClientSocket::SendPacket
//	@0xa72ecb. Byte-identical to the v72/v79/v83/v95 shape already coded in
//	the (version-ungated) Encode below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v84 ida=0xa72957
func TestClaimRequestByteOutputV84(t *testing.T) {
	v := pt.Variants[5] // GMS v84
	if v.Name != "GMS v84" {
		t.Fatalf("pt.Variants[5] = %q, want %q (index drifted)", v.Name, "GMS v84")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0xa72dfb
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0xa72e14
		0x02,                   // Encode1 reasonType @0xa72e1f
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0xa72e38
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0xa72eb0
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimRequestByteOutputV87 verifies the wire-exact byte output of
// ClaimRequest for GMS v87.
// IDA evidence (session d51ecbd3, GMSv87_4GB.exe.i64):
//
//	CWvsContext::SendClaimRequest@0xabee09 (independently decompiled this
//	pass): COutPacket::COutPacket(&a3, 0x6D) @0xabf29e -- opcode 109/0x6D,
//	matches registry gms_v87.yaml op CLAIM_REQUEST (also matches
//	STATUS.md's pre-filled v87 column value of 0x06D -- unlike the four
//	clientbound cells above, this serverbound opcode is NOT stale).
//	COutPacket::Encode1(a2[0]=bChatClaim) @0xabf2ad (a2[0] set to 1 in the
//	DoModal==1000 branch @0xabf13f, 0 in the DoModal==1001 branch @0xabef2a
//	of the preceding dialog-result switch). COutPacket::EncodeStr
//	(targetName=v60) @0xabf2c6. COutPacket::Encode1(reasonType=v53[0])
//	@0xabf2d1. COutPacket::EncodeStr(description=v57) @0xabf2ea. Guarded
//	COutPacket::EncodeStr(chatLog) @0xabf362, gated on the same a2[0]
//	(bChatClaim, checked via `if (*a2)` @0xabf2f2) that fed the first
//	Encode1. CClientSocket::SendPacket @0xabf37d. Byte-identical to the
//	v72/v79/v83/v84/v95 shape already coded in the (version-ungated) Encode
//	below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v87 ida=0xabee09
func TestClaimRequestByteOutputV87(t *testing.T) {
	v := pt.Variants[2] // GMS v87
	if v.Name != "GMS v87" {
		t.Fatalf("pt.Variants[2] = %q, want %q (index drifted)", v.Name, "GMS v87")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0xabf2ad
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0xabf2c6
		0x02,                   // Encode1 reasonType @0xabf2d1
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0xabf2ea
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0xabf362
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimRequestByteOutputV92 verifies the wire-exact byte output of
// ClaimRequest for GMS v92.
// IDA evidence (session acdfccff, GMS_v92_1_DEVM.exe.i64):
//
//	CWvsContext::SendClaimRequest@0x9d9c30 (independently decompiled this
//	pass): COutPacket::COutPacket(&v66, 0x75u) @0x9da25c -- opcode 117/0x75,
//	matches registry gms_v92.yaml op CLAIM_REQUEST (also matches
//	STATUS.md's pre-filled v92 column value of 0x075). COutPacket::Encode1
//	(&v66, v26=bChatClaim) @0x9da26e (v26 = v57, set to 1 on the DoModal==
//	1000/"chat claim" path @0x9da0b6, 0 on the DoModal==1001/"regular
//	claim" path @0x9d9df8). COutPacket::EncodeStr(&v66, v48=targetName)
//	@0x9da28d. COutPacket::Encode1(&v66, v61=reasonType) @0x9da29b.
//	COutPacket::EncodeStr(&v66, v48=description) @0x9da2ba. Guarded
//	COutPacket::EncodeStr(&v66, v48=chatLog) @0x9da375, gated on the same
//	v26 (bChatClaim, checked via `if (v26)` @0x9da2c1) that fed the first
//	Encode1. CClientSocket::SendPacket @0x9da3a4. Byte-identical to the
//	v72/v79/v83/v84/v87 shape already coded in the (version-ungated) Encode
//	below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v92 ida=0x9d9c30
func TestClaimRequestByteOutputV92(t *testing.T) {
	v := pt.Variants[11] // GMS v92
	if v.Name != "GMS v92" {
		t.Fatalf("pt.Variants[11] = %q, want %q (index drifted)", v.Name, "GMS v92")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0x9da26e
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0x9da28d
		0x02,                   // Encode1 reasonType @0x9da29b
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0x9da2ba
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0x9da375
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("byte output mismatch: got %v want %v", actual, expected)
	}
}

// TestClaimRequestByteOutputV95 verifies the wire-exact byte output of
// ClaimRequest for GMS v95.
// IDA evidence (session 79906a1e, GMS_v95.0_U_DEVM.exe.i64):
//
//	CWvsContext::SendClaimRequest@0xa05fb0 (independently decompiled this
//	pass): COutPacket::COutPacket(&oPacket, 118) @0xa065dc -- opcode
//	118/0x76, matches registry gms_v95.yaml op CLAIM_REQUEST (also matches
//	STATUS.md's pre-filled v95 column value of 0x076). COutPacket::Encode1
//	(&oPacket, v33=bChatClaim) @0xa065ee (v33 set to 1 on the DoModal==1000
//	"chat claim" path @0xa06436, 0 on the DoModal==1001 "regular claim"
//	path @0xa06178). COutPacket::EncodeStr(&oPacket, v49[0]=targetName)
//	@0xa0660d. COutPacket::Encode1(&oPacket, nType=reasonType) @0xa0661b.
//	COutPacket::EncodeStr(&oPacket, v49[0]=description) @0xa0663a. Guarded
//	COutPacket::EncodeStr(&oPacket, v49[0]=chatLog) @0xa066f5, gated on the
//	same v33 (bChatClaim, checked via `if ( v33 )` @0xa06641) that fed the
//	first Encode1. CClientSocket::SendPacket @0xa06724. Byte-identical to
//	the v72/v79/v83/v84/v87/v92 shape already coded in the (version-ungated)
//	Encode below.
//
// packet-audit:verify packet=report/serverbound/ClaimRequest version=gms_v95 ida=0xa05fb0
func TestClaimRequestByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95
	if v.Name != "GMS v95" {
		t.Fatalf("pt.Variants[3] = %q, want %q (index drifted)", v.Name, "GMS v95")
	}
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	expected := []byte{
		0x01,                         // Encode1 bChatClaim @0xa065ee
		0x03, 0x00, 0x62, 0x6F, 0x62, // EncodeStr "bob" @0xa0660d
		0x02,                   // Encode1 reasonType @0xa0661b
		0x02, 0x00, 0x68, 0x69, // EncodeStr "hi" @0xa0663a
		0x02, 0x00, 0x79, 0x6F, // EncodeStr "yo" (guarded on bChatClaim) @0xa066f5
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
