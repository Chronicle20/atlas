package conversation

import (
	conversation2 "atlas-channel/kafka/message/npc/conversation"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
)

// TestAnnounceTextConversation asserts the announced packet model — the
// NpcConversation message type and the AskTextConversationDetail fields it
// carries — rather than encoded bytes. Byte-level encoding is Task 13's job;
// msgType is also resolved to an opaque runtime opcode at encode time
// (atlas_packet.ResolveCode) and so cannot be round-tripped back to
// NpcConversationMessageTypeAskText from bytes alone.
func TestAnnounceTextConversation(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		def            string
		min            uint16
		max            uint16
		secondaryNpcId uint32
	}{
		{
			name:    "full",
			message: "Password!",
			def:     "prefill",
			min:     1,
			max:     32,
		},
		{
			name:    "empty default",
			message: "Enter:",
			def:     "",
			min:     0,
			max:     8,
		},
		{
			name:           "secondary npc",
			message:        "Password!",
			def:            "prefill",
			min:            1,
			max:            32,
			secondaryNpcId: 9010000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ncm := newTextConversation(1012000, tc.message, tc.def, tc.min, tc.max, "NPC", true, tc.secondaryNpcId)

			if ncm.MsgType != npcpkt.NpcConversationMessageTypeAskText {
				t.Errorf("MsgType = %v, want %v", ncm.MsgType, npcpkt.NpcConversationMessageTypeAskText)
			}
			if ncm.SecondaryNpcTemplateId != tc.secondaryNpcId {
				t.Errorf("SecondaryNpcTemplateId = %d, want %d", ncm.SecondaryNpcTemplateId, tc.secondaryNpcId)
			}

			detail, ok := ncm.ConversationDetail.(*npcpkt.AskTextConversationDetail)
			if !ok {
				t.Fatalf("ConversationDetail is %T, want *npcpkt.AskTextConversationDetail", ncm.ConversationDetail)
			}
			if detail.Message != tc.message {
				t.Errorf("Message = %q, want %q", detail.Message, tc.message)
			}
			if detail.Def != tc.def {
				t.Errorf("Def = %q, want %q", detail.Def, tc.def)
			}
			if detail.Min != tc.min {
				t.Errorf("Min = %d, want %d", detail.Min, tc.min)
			}
			if detail.Max != tc.max {
				t.Errorf("Max = %d, want %d", detail.Max, tc.max)
			}
		})
	}
}

// TestHandleTextConversationCommandIgnoresOtherTypes matches the guard at
// consumer.go:86 (handleNumberConversationCommand) — a command whose Type
// isn't CommandTypeText must be ignored before touching sc or wp, so a
// zero-value server.Model and a nil writer.Producer are safe to pass.
func TestHandleTextConversationCommandIgnoresOtherTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	var wp writer.Producer

	c := conversation2.CommandEvent[conversation2.CommandTextBody]{
		Type:    conversation2.CommandTypeNumber,
		Message: "should be ignored",
		Body:    conversation2.CommandTextBody{DefaultValue: "x", MinLength: 1, MaxLength: 8},
	}

	// If the guard at the top of handleTextConversationCommand did not
	// return early, sc.Is would be called against a zero-value server.Model
	// and would deterministically panic or fail — either way this call
	// completing without incident proves the guard fired.
	handleTextConversationCommand(server.Model{}, wp)(l, context.Background(), c)
}

// TestGetNPCTalkTypeText proves the "TEXT" case landed in getNPCTalkType,
// which panics on any unmapped string (consumer.go:234).
func TestGetNPCTalkTypeText(t *testing.T) {
	got := getNPCTalkType("TEXT")
	if got != npcpkt.NpcConversationMessageTypeAskText {
		t.Errorf("getNPCTalkType(\"TEXT\") = %v, want %v", got, npcpkt.NpcConversationMessageTypeAskText)
	}
}
