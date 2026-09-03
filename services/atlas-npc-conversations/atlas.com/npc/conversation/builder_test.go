package conversation

import (
	"strings"
	"testing"
)

// mustSiblingChoice builds a minimal valid ChoiceModel for use in sibling setters
// that require at least one choice.
func mustSiblingChoice(t *testing.T) ChoiceModel {
	t.Helper()
	c, err := NewChoiceBuilder().SetText("choice").Build()
	if err != nil {
		t.Fatalf("failed to build choice: %v", err)
	}
	return c
}

// mustAskText builds a minimal valid AskTextModel.
func mustAskText(t *testing.T) *AskTextModel {
	t.Helper()
	m, err := NewAskTextBuilder().
		SetText("Password!").
		SetMaxLength(32).
		SetContextKey("answer").
		SetNextState("wrong").
		Build()
	if err != nil {
		t.Fatalf("failed to build askText: %v", err)
	}
	return m
}

// siblingCase describes one of the 12 other StateBuilder setters that must
// clear askText (and be cleared by it).
type siblingCase struct {
	name  string
	apply func(t *testing.T, b *StateBuilder) *StateBuilder
	isNil func(s StateModel) bool
}

func siblingCases(t *testing.T) []siblingCase {
	t.Helper()

	dialogue, err := NewDialogueBuilder().
		SetDialogueType(SendOk).
		SetText("hello").
		AddChoice(mustSiblingChoice(t)).
		AddChoice(mustSiblingChoice(t)).
		Build()
	if err != nil {
		t.Fatalf("failed to build dialogue: %v", err)
	}

	operation, err := NewOperationBuilder().SetType("op").Build()
	if err != nil {
		t.Fatalf("failed to build operation: %v", err)
	}
	genericAction, err := NewGenericActionBuilder().AddOperation(operation).Build()
	if err != nil {
		t.Fatalf("failed to build genericAction: %v", err)
	}

	craftAction, err := NewCraftActionBuilder().
		SetItemId("item").
		AddMaterial(1).
		AddQuantity(1).
		SetSuccessState("success").
		SetFailureState("failure").
		SetMissingMaterialsState("missing").
		Build()
	if err != nil {
		t.Fatalf("failed to build craftAction: %v", err)
	}

	transportAction, err := NewTransportActionBuilder().
		SetRouteName("route").
		SetFailureState("failure").
		Build()
	if err != nil {
		t.Fatalf("failed to build transportAction: %v", err)
	}

	gachaponAction, err := NewGachaponActionBuilder().
		SetGachaponId("gachapon").
		SetTicketItemId(1).
		SetFailureState("failure").
		Build()
	if err != nil {
		t.Fatalf("failed to build gachaponAction: %v", err)
	}

	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(1).
		SetEntryCostMeso(1).
		SetFailureState("failure").
		Build()
	if err != nil {
		t.Fatalf("failed to build rpsAction: %v", err)
	}

	partyQuestAction, err := NewPartyQuestActionBuilder().
		SetQuestId("quest").
		SetFailureState("failure").
		Build()
	if err != nil {
		t.Fatalf("failed to build partyQuestAction: %v", err)
	}

	partyQuestBonusAction, err := NewPartyQuestBonusActionBuilder().
		SetFailureState("failure").
		Build()
	if err != nil {
		t.Fatalf("failed to build partyQuestBonusAction: %v", err)
	}

	listSelection, err := NewListSelectionBuilder().
		SetTitle("title").
		AddChoice(mustSiblingChoice(t)).
		Build()
	if err != nil {
		t.Fatalf("failed to build listSelection: %v", err)
	}

	askNumber, err := NewAskNumberBuilder().
		SetText("how many?").
		SetMaxValue(10).
		Build()
	if err != nil {
		t.Fatalf("failed to build askNumber: %v", err)
	}

	askStyle, err := NewAskStyleBuilder().
		SetText("pick a style").
		AddStyle(1).
		SetNextState("next").
		Build()
	if err != nil {
		t.Fatalf("failed to build askStyle: %v", err)
	}

	askSlideMenu, err := NewAskSlideMenuBuilder().
		SetTitle("title").
		AddChoice(mustSiblingChoice(t)).
		Build()
	if err != nil {
		t.Fatalf("failed to build askSlideMenu: %v", err)
	}

	return []siblingCase{
		{"SetDialogue", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetDialogue(dialogue) }, func(s StateModel) bool { return s.Dialogue() == nil }},
		{"SetGenericAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetGenericAction(genericAction) }, func(s StateModel) bool { return s.GenericAction() == nil }},
		{"SetCraftAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetCraftAction(craftAction) }, func(s StateModel) bool { return s.CraftAction() == nil }},
		{"SetTransportAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetTransportAction(transportAction) }, func(s StateModel) bool { return s.TransportAction() == nil }},
		{"SetGachaponAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetGachaponAction(gachaponAction) }, func(s StateModel) bool { return s.GachaponAction() == nil }},
		{"SetRPSAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetRPSAction(rpsAction) }, func(s StateModel) bool { return s.RPSAction() == nil }},
		{"SetPartyQuestAction", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetPartyQuestAction(partyQuestAction) }, func(s StateModel) bool { return s.PartyQuestAction() == nil }},
		{"SetPartyQuestBonusAction", func(t *testing.T, b *StateBuilder) *StateBuilder {
			return b.SetPartyQuestBonusAction(partyQuestBonusAction)
		}, func(s StateModel) bool { return s.PartyQuestBonusAction() == nil }},
		{"SetListSelection", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetListSelection(listSelection) }, func(s StateModel) bool { return s.ListSelection() == nil }},
		{"SetAskNumber", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetAskNumber(askNumber) }, func(s StateModel) bool { return s.AskNumber() == nil }},
		{"SetAskStyle", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetAskStyle(askStyle) }, func(s StateModel) bool { return s.AskStyle() == nil }},
		{"SetAskSlideMenu", func(t *testing.T, b *StateBuilder) *StateBuilder { return b.SetAskSlideMenu(askSlideMenu) }, func(s StateModel) bool { return s.AskSlideMenu() == nil }},
	}
}

// TestStateBuilderSetAskTextClearsSiblings asserts that calling a sibling
// setter followed by SetAskText leaves the sibling's model pointer nil and
// AskText() populated.
func TestStateBuilderSetAskTextClearsSiblings(t *testing.T) {
	for _, c := range siblingCases(t) {
		t.Run(c.name, func(t *testing.T) {
			b := NewStateBuilder().SetId("state-1")
			c.apply(t, b)
			b.SetAskText(mustAskText(t))

			s, err := b.Build()
			if err != nil {
				t.Fatalf("Build() returned error: %v", err)
			}
			if s.AskText() == nil {
				t.Fatal("expected AskText() to be non-nil")
			}
			if !c.isNil(s) {
				t.Fatalf("expected %s's model to be nil after SetAskText", c.name)
			}
		})
	}
}

// TestStateBuilderSetAskTextIsClearedBySiblings asserts the reverse
// direction: calling SetAskText then a sibling setter clears askText. This
// is the assertion that catches a missed `b.askText = nil` clear site in one
// of the 12 sibling setters.
func TestStateBuilderSetAskTextIsClearedBySiblings(t *testing.T) {
	for _, c := range siblingCases(t) {
		t.Run(c.name, func(t *testing.T) {
			b := NewStateBuilder().SetId("state-1")
			b.SetAskText(mustAskText(t))
			c.apply(t, b)

			s, err := b.Build()
			if err != nil {
				t.Fatalf("Build() returned error: %v", err)
			}
			if s.AskText() != nil {
				t.Fatalf("expected AskText() to be nil after %s", c.name)
			}
		})
	}
}

// TestStateBuilderBuildRejectsAskTextWithNilModel asserts Build() rejects an
// askText-typed state whose askText model was never set.
func TestStateBuilderBuildRejectsAskTextWithNilModel(t *testing.T) {
	b := NewStateBuilder().SetId("state-1")
	b.stateType = AskTextType

	_, err := b.Build()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestAskTextBuilderBuild(t *testing.T) {
	tests := []struct {
		name    string
		build   func() *AskTextBuilder
		wantErr string
	}{
		{
			name: "minimal valid",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetText("Password!").
					SetMaxLength(32).
					SetContextKey("answer").
					SetNextState("wrong")
			},
		},
		{
			name: "full",
			build: func() *AskTextBuilder {
				match1, err := NewAskTextMatchBuilder().SetValue("Open Sesame").SetNextState("open").Build()
				if err != nil {
					t.Fatalf("failed to build match1: %v", err)
				}
				match2, err := NewAskTextMatchBuilder().SetValueFromContext("{context.pw}").SetNextState("open2").Build()
				if err != nil {
					t.Fatalf("failed to build match2: %v", err)
				}
				return NewAskTextBuilder().
					SetText("Password!").
					SetDefaultText("hint").
					SetMinLength(1).
					SetMaxLength(32).
					SetContextKey("answer").
					SetNextState("wrong").
					AddMatch(*match1).
					AddMatch(*match2)
			},
		},
		{
			name: "missing text",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetMaxLength(32).
					SetContextKey("a").
					SetNextState("n")
			},
			wantErr: "text",
		},
		{
			name: "zero maxLength",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetText("t").
					SetMaxLength(0).
					SetContextKey("a").
					SetNextState("n")
			},
			wantErr: "maxLength",
		},
		{
			name: "minLength > maxLength",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetText("t").
					SetMinLength(10).
					SetMaxLength(5).
					SetContextKey("a").
					SetNextState("n")
			},
			wantErr: "minLength",
		},
		{
			name: "missing contextKey",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetText("t").
					SetMaxLength(32).
					SetContextKey("").
					SetNextState("n")
			},
			wantErr: "contextKey",
		},
		{
			name: "missing nextState",
			build: func() *AskTextBuilder {
				return NewAskTextBuilder().
					SetText("t").
					SetMaxLength(32).
					SetContextKey("a")
			},
			wantErr: "nextState",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.build().Build()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.name {
			case "minimal valid":
				if m.MinLength() != 0 {
					t.Errorf("expected MinLength() == 0, got %d", m.MinLength())
				}
				if m.DefaultText() != "" {
					t.Errorf("expected DefaultText() == \"\", got %q", m.DefaultText())
				}
				if len(m.Matches()) != 0 {
					t.Errorf("expected Matches() to be empty, got %d", len(m.Matches()))
				}
			case "full":
				if len(m.Matches()) != 2 {
					t.Fatalf("expected 2 matches, got %d", len(m.Matches()))
				}
				if m.Matches()[0].Value() != "Open Sesame" {
					t.Errorf("expected first match value to be preserved in declaration order, got %q", m.Matches()[0].Value())
				}
				if m.Matches()[1].ValueFromContext() != "{context.pw}" {
					t.Errorf("expected second match to be preserved in declaration order, got %q", m.Matches()[1].ValueFromContext())
				}
			}
		})
	}
}

func TestAskTextMatchBuilderBuild(t *testing.T) {
	tests := []struct {
		name    string
		build   func() *AskTextMatchBuilder
		wantErr string
	}{
		{
			name: "literal",
			build: func() *AskTextMatchBuilder {
				return NewAskTextMatchBuilder().SetValue("Open Sesame").SetNextState("open")
			},
		},
		{
			name: "from context",
			build: func() *AskTextMatchBuilder {
				return NewAskTextMatchBuilder().SetValueFromContext("{context.pw}").SetNextState("open")
			},
		},
		{
			name: "neither",
			build: func() *AskTextMatchBuilder {
				return NewAskTextMatchBuilder().SetNextState("open")
			},
			wantErr: "value",
		},
		{
			name: "both",
			build: func() *AskTextMatchBuilder {
				return NewAskTextMatchBuilder().SetValue("x").SetValueFromContext("{context.pw}").SetNextState("open")
			},
			wantErr: "value",
		},
		{
			name: "missing nextState",
			build: func() *AskTextMatchBuilder {
				return NewAskTextMatchBuilder().SetValue("x")
			},
			wantErr: "nextState",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.build().Build()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
