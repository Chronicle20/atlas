package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v83 ida=0x746a8b
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v87 ida=0x791cd0
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v95 ida=0x6dc790
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=jms_v185 ida=0x7b77bd
func TestContinueConversationTextRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversationText{text: "Hello NPC"}
			output := ContinueConversationText{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Text() != input.Text() {
				t.Errorf("text: got %v, want %v", output.Text(), input.Text())
			}
		})
	}
}
