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
