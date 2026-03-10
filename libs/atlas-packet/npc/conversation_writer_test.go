package npc

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestNpcConversationSay(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "Hello adventurer!", Next: true, Previous: false}
	detailBytes := detail.Encode(l, context.Background())(nil)

	input := NewNpcConversation(0, 2100, 0, 0, 0, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestNpcConversationSayWithSecondaryNpc(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "Two NPCs talking!", Next: true, Previous: true}
	detailBytes := detail.Encode(l, context.Background())(nil)

	// param = 4 triggers writing secondaryNpcTemplateId
	input := NewNpcConversation(0, 2100, 0, 4, 9999, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestNpcConversationAskMenu(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &AskMenuConversationDetail{Message: "#L0#Option 1#l\r\n#L1#Option 2#l"}
	detailBytes := detail.Encode(l, context.Background())(nil)

	input := NewNpcConversation(0, 2100, 5, 0, 0, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestNpcConversationAccessors(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "test", Next: true, Previous: false}
	detailBytes := detail.Encode(l, context.Background())(nil)

	m := NewNpcConversation(1, 2100, 3, 4, 5000, detailBytes)
	if m.SpeakerTypeId() != 1 {
		t.Errorf("expected SpeakerTypeId 1, got %d", m.SpeakerTypeId())
	}
	if m.SpeakerTemplateId() != 2100 {
		t.Errorf("expected SpeakerTemplateId 2100, got %d", m.SpeakerTemplateId())
	}
	if m.MsgType() != 3 {
		t.Errorf("expected MsgType 3, got %d", m.MsgType())
	}
	if m.Param() != 4 {
		t.Errorf("expected Param 4, got %d", m.Param())
	}
	if m.SecondaryNpcTemplateId() != 5000 {
		t.Errorf("expected SecondaryNpcTemplateId 5000, got %d", m.SecondaryNpcTemplateId())
	}
	if m.Operation() != NpcConversationWriter {
		t.Errorf("expected Operation %s, got %s", NpcConversationWriter, m.Operation())
	}
}
