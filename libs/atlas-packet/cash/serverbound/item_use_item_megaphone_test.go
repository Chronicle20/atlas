package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseItemMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		whisper bool
		hasItem bool
		invType int32
		slot    int32
	}{
		{"hasItem_true", false, true, 2, 5},
		{"hasItem_false", true, false, 0, 0},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseItemMegaphone(updateTimeFirst)
				input.message = "Item hello!"
				input.whisper = tc.whisper
				input.hasItem = tc.hasItem
				input.invType = tc.invType
				input.slot = tc.slot
				if !updateTimeFirst {
					input.updateTime = 99999
				}
				output := NewItemUseItemMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Message() != input.Message() {
					t.Errorf("message: got %q, want %q", output.Message(), input.Message())
				}
				if output.Whisper() != input.Whisper() {
					t.Errorf("whisper: got %v, want %v", output.Whisper(), input.Whisper())
				}
				if output.HasItem() != input.HasItem() {
					t.Errorf("hasItem: got %v, want %v", output.HasItem(), input.HasItem())
				}
				if tc.hasItem {
					if output.InvType() != input.InvType() {
						t.Errorf("invType: got %v, want %v", output.InvType(), input.InvType())
					}
					if output.Slot() != input.Slot() {
						t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
					}
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
