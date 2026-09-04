package quest

import (
	"atlas-npc-conversations/conversation"
	"reflect"
	"testing"
)

// TestTransformRoundTrip_AskTextQuestLayer verifies that the quest package's
// own TransformAskText/ExtractAskText pair round-trips a domain value through
// its REST wire form without loss. This is a separate copy of the NPC REST
// layer with no compiler link between them, so it needs its own coverage.
func TestTransformRoundTrip_AskTextQuestLayer(t *testing.T) {
	t.Run("AskTextQuestLayer", func(t *testing.T) {
		match1, err := conversation.NewAskTextMatchBuilder().
			SetValue("Open Sesame").
			SetNextState("open").
			Build()
		if err != nil {
			t.Fatalf("build match1: %v", err)
		}
		match2, err := conversation.NewAskTextMatchBuilder().
			SetValueFromContext("{context.magatiaPassword}").
			SetNextState("open-magatia").
			Build()
		if err != nil {
			t.Fatalf("build match2: %v", err)
		}

		at, err := conversation.NewAskTextBuilder().
			SetText("The door reacts to the entry pass inserted. #bPassword#k!").
			SetDefaultText("").
			SetMinLength(1).
			SetMaxLength(32).
			SetContextKey("answer").
			SetNextState("wrong-password").
			AddMatch(*match1).
			AddMatch(*match2).
			Build()
		if err != nil {
			t.Fatalf("build askText: %v", err)
		}

		rest := TransformAskText(*at)
		got, err := ExtractAskText(rest)
		if err != nil {
			t.Fatalf("ExtractAskText: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskText returned nil")
		}
		if !reflect.DeepEqual(*got, *at) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *at)
		}
		if len(got.Matches()) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(got.Matches()))
		}
		if got.Matches()[0].Value() != "Open Sesame" || got.Matches()[0].NextState() != "open" {
			t.Errorf("match[0] mismatch: %#v", got.Matches()[0])
		}
		if got.Matches()[1].ValueFromContext() != "{context.magatiaPassword}" || got.Matches()[1].NextState() != "open-magatia" {
			t.Errorf("match[1] mismatch: %#v", got.Matches()[1])
		}
	})
}
