package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func testAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

func TestMessengerAddRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			ava := testAvatar()
			input := NewMessengerAdd(1, 2, ava, "TestPlayer", 3)
			output := Add{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Position() != input.Position() {
				t.Errorf("position: got %v, want %v", output.Position(), input.Position())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			// Avatar face/hair not written for GMS v28 — check equipment which is always present
			if len(output.Avatar().Equipment()) != len(ava.Equipment()) {
				t.Errorf("avatar equipment count: got %v, want %v", len(output.Avatar().Equipment()), len(ava.Equipment()))
			}
		})
	}
}

func TestMessengerUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			ava := testAvatar()
			input := NewMessengerUpdate(5, 1, ava, "OtherPlayer", 2)
			output := Update{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Position() != input.Position() {
				t.Errorf("position: got %v, want %v", output.Position(), input.Position())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if len(output.Avatar().Equipment()) != len(ava.Equipment()) {
				t.Errorf("avatar equipment count: got %v, want %v", len(output.Avatar().Equipment()), len(ava.Equipment()))
			}
		})
	}
}
