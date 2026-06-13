package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v83 ida=0x7467ab
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v87 ida=0x791828
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v95 ida=0x6dc110
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=jms_v185 ida=0x7b7315
func TestContinueConversationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ContinueConversation{lastMessageType: 3, action: 1}
			output := ContinueConversation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.LastMessageType() != input.LastMessageType() {
				t.Errorf("lastMessageType: got %v, want %v", output.LastMessageType(), input.LastMessageType())
			}
			if output.Action() != input.Action() {
				t.Errorf("action: got %v, want %v", output.Action(), input.Action())
			}
		})
	}
}
