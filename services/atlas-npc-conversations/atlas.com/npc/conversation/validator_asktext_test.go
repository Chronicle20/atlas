package conversation

import "testing"

// buildAskTextConversation builds a two-state conversation whose askText
// state exercises the given model. The other state has id "wrong-password",
// so "nextState: wrong-password" resolves and any other target does not.
//
// askText is nil for the "nil model" case; StateBuilder rejects a nil
// askText at Build() time, so that state is assembled via struct literal
// instead (same package, unexported fields are accessible).
func buildAskTextConversation(t *testing.T, askText *AskTextModel) testNpcConversation {
	t.Helper()

	var askTextState StateModel
	if askText == nil {
		askTextState = StateModel{id: "ask", stateType: AskTextType, askText: nil}
	} else {
		var err error
		askTextState, err = NewStateBuilder().SetId("ask").SetAskText(askText).Build()
		if err != nil {
			t.Fatalf("build ask state: %v", err)
		}
	}

	other, err := NewDialogueBuilder().
		SetDialogueType(SendOk).
		SetText("Wrong password.").
		AddChoice(mustChoice(t, "Ok", "")).
		AddChoice(mustChoice(t, "Exit", "")).
		Build()
	if err != nil {
		t.Fatalf("build wrong-password dialogue: %v", err)
	}
	otherState, err := NewStateBuilder().SetId("wrong-password").SetDialogue(other).Build()
	if err != nil {
		t.Fatalf("build wrong-password state: %v", err)
	}

	return testNpcConversation{
		npcId:  9000019,
		start:  "ask",
		states: []StateModel{askTextState, otherState},
	}
}

// validAskText returns a valid baseline AskTextModel for mutation by test cases.
func validAskText() *AskTextModel {
	return &AskTextModel{
		text:       "Password!",
		minLength:  1,
		maxLength:  32,
		contextKey: "answer",
		nextState:  "wrong-password",
	}
}

func TestValidateAskText(t *testing.T) {
	tests := []struct {
		name      string
		askText   *AskTextModel
		wantValid bool
		wantField string
		wantType  string
	}{
		{
			name:      "valid",
			askText:   validAskText(),
			wantValid: true,
		},
		{
			name:      "nil model",
			askText:   nil,
			wantValid: false,
			wantField: "askText",
			wantType:  "required",
		},
		{
			name: "empty text",
			askText: func() *AskTextModel {
				m := validAskText()
				m.text = ""
				return m
			}(),
			wantValid: false,
			wantField: "askText.text",
			wantType:  "required",
		},
		{
			name: "empty contextKey",
			askText: func() *AskTextModel {
				m := validAskText()
				m.contextKey = ""
				return m
			}(),
			wantValid: false,
			wantField: "askText.contextKey",
			wantType:  "required",
		},
		{
			name: "empty nextState",
			askText: func() *AskTextModel {
				m := validAskText()
				m.nextState = ""
				return m
			}(),
			wantValid: false,
			wantField: "askText.nextState",
			wantType:  "required",
		},
		{
			name: "unknown nextState",
			askText: func() *AskTextModel {
				m := validAskText()
				m.nextState = "nope"
				return m
			}(),
			wantValid: false,
			wantField: "askText.nextState",
			wantType:  "invalid_reference",
		},
		{
			name: "zero maxLength",
			askText: func() *AskTextModel {
				m := validAskText()
				m.maxLength = 0
				return m
			}(),
			wantValid: false,
			wantField: "askText.maxLength",
		},
		{
			name: "min exceeds max",
			askText: func() *AskTextModel {
				m := validAskText()
				m.minLength = 33
				m.maxLength = 32
				return m
			}(),
			wantValid: false,
			wantField: "askText.minLength",
		},
		{
			name: "min equals max",
			askText: func() *AskTextModel {
				m := validAskText()
				m.minLength = 32
				m.maxLength = 32
				return m
			}(),
			wantValid: true,
		},
		{
			name: "match with neither",
			askText: func() *AskTextModel {
				m := validAskText()
				m.matches = []AskTextMatchModel{{value: "", valueFromContext: "", nextState: "wrong-password"}}
				return m
			}(),
			wantValid: false,
			wantField: "askText.matches[0]",
		},
		{
			name: "match with both",
			askText: func() *AskTextModel {
				m := validAskText()
				m.matches = []AskTextMatchModel{{value: "x", valueFromContext: "{context.pw}", nextState: "wrong-password"}}
				return m
			}(),
			wantValid: false,
			wantField: "askText.matches[0]",
		},
		{
			name: "match unknown nextState",
			askText: func() *AskTextModel {
				m := validAskText()
				m.matches = []AskTextMatchModel{{value: "x", nextState: "nope"}}
				return m
			}(),
			wantValid: false,
			wantField: "askText.matches[0].nextState",
			wantType:  "invalid_reference",
		},
		{
			name: "match valueFromContext malformed",
			askText: func() *AskTextModel {
				m := validAskText()
				m.matches = []AskTextMatchModel{{valueFromContext: "magatiaPassword", nextState: "wrong-password"}}
				return m
			}(),
			wantValid: false,
			wantField: "askText.matches[0].valueFromContext",
			wantType:  "invalid",
		},
		{
			name: "match valueFromContext valid",
			askText: func() *AskTextModel {
				m := validAskText()
				m.matches = []AskTextMatchModel{{valueFromContext: "{context.magatiaPassword}", nextState: "wrong-password"}}
				return m
			}(),
			wantValid: true,
		},
		{
			name: "defaultText shorter than minLength",
			askText: func() *AskTextModel {
				m := validAskText()
				m.defaultText = ""
				m.minLength = 5
				m.maxLength = 32
				return m
			}(),
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := buildAskTextConversation(t, tt.askText)
			result := NewValidator().ValidateNpc(m)

			if result.Valid != tt.wantValid {
				t.Fatalf("ValidateNpc().Valid = %v, want %v; errors=%+v", result.Valid, tt.wantValid, result.Errors)
			}

			if !tt.wantValid && tt.wantField != "" {
				found := false
				for _, e := range result.Errors {
					if e.Field != tt.wantField {
						continue
					}
					if tt.wantType != "" && e.ErrorType != tt.wantType {
						continue
					}
					found = true
				}
				if !found {
					t.Errorf("expected error field=%q type=%q; got %+v", tt.wantField, tt.wantType, result.Errors)
				}
			}
		})
	}
}

// TestValidateAskTextCircularReference verifies the visitor arm walks
// matches[].nextState, not only the fallback nextState: a → b → c → a with
// no state offering an exit. cycleHasNoExit (validator.go) only reports a
// cycle "infinite" when every outgoing edge of every state in the cycle —
// fallback nextState *and* every matches[].nextState — stays inside the
// cycle; giving the fallback an escape (e.g. to a dead-end "exit" state)
// makes cycleHasNoExit report an exit and the cycle goes unreported
// regardless of the match edges, so this fixture keeps both the fallback
// and the sole match on each state inside {a, b, c}. A visitor that omitted
// the matches[] loop would still discover this exact cycle via the fallback
// chain alone here, since both chains coincide; the test therefore proves
// the visitor detects a fully match-driven askText cycle at all, though it
// cannot isolate the matches[] loop from the fallback line-for-line — see
// the task report for why cycleHasNoExit's semantics make full isolation
// with this algorithm infeasible.
func TestValidateAskTextCircularReference(t *testing.T) {
	aAskText := &AskTextModel{
		text:       "A?",
		minLength:  1,
		maxLength:  32,
		contextKey: "answer",
		matches: []AskTextMatchModel{
			{value: "go", nextState: "b"},
		},
		nextState: "b",
	}
	aState, err := NewStateBuilder().SetId("a").SetAskText(aAskText).Build()
	if err != nil {
		t.Fatalf("build state a: %v", err)
	}

	bAskText := &AskTextModel{
		text:       "B?",
		minLength:  1,
		maxLength:  32,
		contextKey: "answer",
		matches: []AskTextMatchModel{
			{value: "go", nextState: "c"},
		},
		nextState: "c",
	}
	bState, err := NewStateBuilder().SetId("b").SetAskText(bAskText).Build()
	if err != nil {
		t.Fatalf("build state b: %v", err)
	}

	cAskText := &AskTextModel{
		text:       "C?",
		minLength:  1,
		maxLength:  32,
		contextKey: "answer",
		matches: []AskTextMatchModel{
			{value: "go", nextState: "a"},
		},
		nextState: "a",
	}
	cState, err := NewStateBuilder().SetId("c").SetAskText(cAskText).Build()
	if err != nil {
		t.Fatalf("build state c: %v", err)
	}

	m := testNpcConversation{
		npcId:  9000019,
		start:  "a",
		states: []StateModel{aState, bState, cState},
	}

	result := NewValidator().ValidateNpc(m)

	if result.Valid {
		t.Fatalf("ValidateNpc returned Valid=true, want false (circular reference via match nextState)")
	}
	found := false
	for _, e := range result.Errors {
		if e.Field == "circular_reference" && e.ErrorType == "infinite_loop" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a circular_reference/infinite_loop error; got %+v", result.Errors)
	}
}
