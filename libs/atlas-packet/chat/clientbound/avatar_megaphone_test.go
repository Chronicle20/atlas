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
			if output.Flag() != 1 {
				t.Errorf("flag: got %v, want 1", output.Flag())
			}
		})
	}
}

func TestClearAvatarMegaphoneByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewClearAvatarMegaphone()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 1 {
		t.Fatalf("payload length: got %d, want 1", len(actual))
	}
	if actual[0] != 1 {
		t.Errorf("payload byte: got %d, want 1", actual[0])
	}
}

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
