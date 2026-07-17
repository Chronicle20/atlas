package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func testMegaphoneAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342):
//
//	CWvsContext::OnSetAvatarMegaphone@0xa2a486:
//	  Decode4(itemId) -> DecodeStr(name) -> DecodeStr(line1) -> DecodeStr(line2)
//	  -> DecodeStr(line3) -> DecodeStr(line4) -> Decode4(channelId) ->
//	  Decode1(whispersOn) -> AvatarLook::Decode(senderLook).
//	Read order matches SetAvatarMegaphone.Encode exactly: itemId, name, 4
//	lines, channelId, whispersOn, look.
//
// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v83 ida=0xa2a486
func TestSetAvatarMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			look := testMegaphoneAvatar()
			lines := [4]string{"line one", "line two", "line three", "line four"}
			input := NewSetAvatarMegaphone(5390000, "TestPlayer", lines, 3, true, look)
			output := SetAvatarMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Lines() != input.Lines() {
				t.Errorf("lines: got %v, want %v", output.Lines(), input.Lines())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.WhispersOn() != input.WhispersOn() {
				t.Errorf("whispersOn: got %v, want %v", output.WhispersOn(), input.WhispersOn())
			}
			// Avatar face/hair not written for GMS v28 — check equipment which is always present.
			if len(output.Look().Equipment()) != len(look.Equipment()) {
				t.Errorf("look equipment count: got %v, want %v", len(output.Look().Equipment()), len(look.Equipment()))
			}
		})
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::OnSetAvatarMegaphone@0xa017e0:
//
//	v5=Decode4(itemId) -> DecodeStr(sName) -> DecodeStr(s1) -> DecodeStr(s2)
//	-> DecodeStr(s3) -> DecodeStr(s4) -> v6=Decode4(channelId) ->
//	bWhisper=Decode1(whispersOn) -> AvatarLook::Decode(v34, senderLook).
//	Read order matches SetAvatarMegaphone.Encode exactly (itemId, name, 4
//	lines, channelId, whispersOn, look) — byte-identical to gms_v83
//	(confirms the existing "IDA v83≡v95" comment).
//
// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v95 ida=0xa017e0
func TestSetAvatarMegaphoneRoundTripV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	look := testMegaphoneAvatar()
	lines := [4]string{"line one", "line two", "line three", "line four"}
	input := NewSetAvatarMegaphone(5390000, "TestPlayer", lines, 3, true, look)
	output := SetAvatarMegaphone{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.ItemId() != input.ItemId() {
		t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
	}
	if output.Lines() != input.Lines() {
		t.Errorf("lines: got %v, want %v", output.Lines(), input.Lines())
	}
	if output.ChannelId() != input.ChannelId() {
		t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
	}
	if output.WhispersOn() != input.WhispersOn() {
		t.Errorf("whispersOn: got %v, want %v", output.WhispersOn(), input.WhispersOn())
	}
}

func TestClearAvatarMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClearAvatarMegaphone()
			output := ClearAvatarMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342):
//
//	CWvsContext::OnClearAvatarMegaphone@0xa2a65b never reads from its
//	CInPacket argument — the whole body is:
//	  if (this->m_tAM_LastUpdate) CAvatarMegaphone::ByeAvatarMegaphone();
//	  this->m_tAM_LastUpdate = 0;
//	No Decode* call of any kind. The wire body is therefore EMPTY (opcode
//	only). The prior Cosmic-derived single guard byte does not match.
//
// packet-audit:verify packet=chat/clientbound/ChatClearAvatarMegaphone version=gms_v83 ida=0xa2a65b
func TestClearAvatarMegaphoneByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewClearAvatarMegaphone()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Fatalf("payload length: got %d, want 0 (empty body — client reads nothing)", len(actual))
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::OnClearAvatarMegaphone@0x9f0c90:
//
//	void __thiscall CWvsContext::OnClearAvatarMegaphone(this, iPacket) {
//	  if (this->m_bAvatarMegaphone) CAvatarMegaphone::ByeAvatarMegaphone(...);
//	  this->m_bAvatarMegaphone = 0;
//	}
//	The iPacket argument is never touched — zero Decode* calls of any kind
//	(same finding as gms_v83's OnClearAvatarMegaphone@0xa2a65b, which used a
//	different member name m_tAM_LastUpdate but the identical zero-read
//	shape). Wire body is EMPTY.
//
// packet-audit:verify packet=chat/clientbound/ChatClearAvatarMegaphone version=gms_v95 ida=0x9f0c90
func TestClearAvatarMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewClearAvatarMegaphone()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Fatalf("payload length: got %d, want 0 (empty body — client reads nothing)", len(actual))
	}
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342):
//
//	CWvsContext::OnAvatarMegaphoneRes@0xa2a3bc:
//	  v3 = Decode1(code) - 83.
//	  if (v3 == 0)   -> code==83: Notice(SP_3972 "waiting line ... 15 seconds")
//	                    NO further DecodeStr call.
//	  else if (v3==1)-> code==84: Notice(SP_3745 "over level 10")
//	                    NO further DecodeStr call.
//	  else           -> DecodeStr(message) IS called, message shown verbatim.
//	Confirms: code byte first; trailing message string present ONLY when
//	code is neither 83 (WAITING_LINE) nor 84 (LEVEL_GATE) — exactly the
//	hasMessage gating AvatarMegaphoneResult already implements, and confirms
//	the seed values WAITING_LINE=83 / LEVEL_GATE=84.
//
// packet-audit:verify packet=chat/clientbound/ChatAvatarMegaphoneResult version=gms_v83 ida=0xa2a3bc
func TestAvatarMegaphoneResultRoundTripNoMessage(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewAvatarMegaphoneResult(83, "")
			output := AvatarMegaphoneResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.HasMessage() {
				t.Errorf("hasMessage: got true, want false")
			}
			if output.Message() != "" {
				t.Errorf("message: got %q, want empty", output.Message())
			}
		})
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::OnAvatarMegaphoneRes@0xa016c0:
//
//	v4 = Decode1(code) - 96.
//	if (!v4)   -> code==96: Notice(SP 4013) — NO further DecodeStr.
//	if (v4==1) -> code==97: Notice(SP 3785) — NO further DecodeStr.
//	else       -> DecodeStr(message) IS called, message shown verbatim.
//	The base offset SHIFTED from 83 (gms_v83) to 96 (gms_v95) — a real,
//	IDA-confirmed divergence, NOT the same literal codes as v83. The
//	branch STRUCTURE is byte-identical to gms_v83's OnAvatarMegaphoneRes
//	(first branch = no-message reason #1, second branch = no-message
//	reason #2, else = DecodeStr), so the first/second branch → WAITING_LINE
//	/LEVEL_GATE order-correspondence established for v83 carries over:
//	code 96 = WAITING_LINE, code 97 = LEVEL_GATE. template_gms_95_1.json's
//	AvatarMegaphoneResult.errorCodes (previously WAITING_LINE=83/LEVEL_GATE=84,
//	copied uncritically from the v83 seed) is corrected in this commit to
//	WAITING_LINE=96/LEVEL_GATE=97. This is a genuine client-crash-class bug:
//	a v95 tenant sending code 84 (old table) hits neither 96 nor 97, so the
//	client falls into the "else" branch and tries to DecodeStr a trailing
//	string the codec never wrote.
//	The hasMessage gating (AvatarMegaphoneResult) itself needs NO struct
//	change — it never branches on the resolved byte, only on caller intent.
//
// packet-audit:verify packet=chat/clientbound/ChatAvatarMegaphoneResult version=gms_v95 ida=0xa016c0
func TestAvatarMegaphoneResultRoundTripNoMessageV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewAvatarMegaphoneResult(96, "")
	output := AvatarMegaphoneResult{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Code() != input.Code() {
		t.Errorf("code: got %v, want %v", output.Code(), input.Code())
	}
	if output.HasMessage() {
		t.Errorf("hasMessage: got true, want false")
	}
}

func TestAvatarMegaphoneResultRoundTripWithMessage(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewAvatarMegaphoneResult(1, "some notice text")
			output := AvatarMegaphoneResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if !output.HasMessage() {
				t.Errorf("hasMessage: got false, want true")
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %q, want %q", output.Message(), input.Message())
			}
		})
	}
}
