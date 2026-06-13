package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v83 ida=0x746fad
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v87 ida=0x7921a8
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v95 ida=0x6dce00
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=jms_v185 ida=0x7b7c95
func TestContinueConversationSelectionWideRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationSelection{selection: 42, wide: true}
			output := ContinueConversationSelection{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}

func TestContinueConversationSelectionNarrowRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationSelection{selection: 3, wide: false}
			output := ContinueConversationSelection{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Selection() != input.Selection() {
				t.Errorf("selection: got %v, want %v", output.Selection(), input.Selection())
			}
		})
	}
}
