package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Legacy GMS SetAvatarMegaphone fixtures (task-123 legacy phase 1). All four
// anchors (v48/v61/v72/v79) decompile to the IDENTICAL read order already
// verified for v83/v84/v87/v95:
//
//	Decode4(itemId) -> DecodeStr(name) -> DecodeStr(line1) -> DecodeStr(line2)
//	-> DecodeStr(line3) -> DecodeStr(line4) -> Decode4(channelId) ->
//	Decode1(whispersOn) -> AvatarLook::Decode(look).
//
// No codec change was needed — SetAvatarMegaphone's non-JMS branch already
// matches this shape exactly. These are corroborating fixtures only.

// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v48 ida=0x721295
func TestSetAvatarMegaphoneRoundTripV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
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

// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v61 ida=0x84aaf8
func TestSetAvatarMegaphoneRoundTripV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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

// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v72 ida=0x9221a8
func TestSetAvatarMegaphoneRoundTripV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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

// packet-audit:verify packet=chat/clientbound/ChatSetAvatarMegaphone version=gms_v79 ida=0x9742dd
func TestSetAvatarMegaphoneRoundTripV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
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
