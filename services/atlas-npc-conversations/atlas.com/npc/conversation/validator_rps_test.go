package conversation

import (
	"errors"
	"testing"
)

// testNpcConversation is a minimal NpcConversation implementation for validator tests.
type testNpcConversation struct {
	npcId  uint32
	start  string
	states []StateModel
}

func (t testNpcConversation) NpcId() uint32      { return t.npcId }
func (t testNpcConversation) StartState() string { return t.start }
func (t testNpcConversation) States() []StateModel {
	return t.states
}
func (t testNpcConversation) FindState(id string) (StateModel, error) {
	for _, s := range t.states {
		if s.Id() == id {
			return s, nil
		}
	}
	return StateModel{}, errors.New("state not found")
}

// buildRPSConversation builds the canonical NPC 9000019 shape:
// offer (sendYesNo) → playRPS (rpsAction, failureState=noMeso) → noMeso (sendOk).
func buildRPSConversation(t *testing.T) testNpcConversation {
	t.Helper()

	offer, err := NewDialogueBuilder().
		SetDialogueType(SendYesNo).
		SetText("Care for a game of Rock Paper Scissors?").
		AddChoice(mustChoice(t, "Yes", "playRPS")).
		AddChoice(mustChoice(t, "No", "")).
		AddChoice(mustChoice(t, "Exit", "")).
		Build()
	if err != nil {
		t.Fatalf("build offer dialogue: %v", err)
	}
	offerState, err := NewStateBuilder().SetId("offer").SetDialogue(offer).Build()
	if err != nil {
		t.Fatalf("build offer state: %v", err)
	}

	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}
	rpsState, err := NewStateBuilder().SetId("playRPS").SetRPSAction(rpsAction).Build()
	if err != nil {
		t.Fatalf("build rps state: %v", err)
	}

	noMeso, err := NewDialogueBuilder().
		SetDialogueType(SendOk).
		SetText("You don't have enough meso to play.").
		AddChoice(mustChoice(t, "Ok", "")).
		AddChoice(mustChoice(t, "Exit", "")).
		Build()
	if err != nil {
		t.Fatalf("build noMeso dialogue: %v", err)
	}
	noMesoState, err := NewStateBuilder().SetId("noMeso").SetDialogue(noMeso).Build()
	if err != nil {
		t.Fatalf("build noMeso state: %v", err)
	}

	return testNpcConversation{
		npcId:  9000019,
		start:  "offer",
		states: []StateModel{offerState, rpsState, noMesoState},
	}
}

func mustChoice(t *testing.T, text, nextState string) ChoiceModel {
	t.Helper()
	c, err := NewChoiceBuilder().SetText(text).SetNextState(nextState).Build()
	if err != nil {
		t.Fatalf("build choice %q: %v", text, err)
	}
	return c
}

// TestValidateNpc_RPSActionValid verifies a conversation containing an
// rpsAction state (with a valid failureState) passes validation: no
// "invalid state type", and the failureState target is treated as reachable.
func TestValidateNpc_RPSActionValid(t *testing.T) {
	m := buildRPSConversation(t)

	result := NewValidator().ValidateNpc(m)

	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("unexpected validation error: state=%q field=%q type=%q msg=%q", e.StateId, e.Field, e.ErrorType, e.Message)
		}
		t.Fatalf("ValidateNpc returned Valid=false, want true")
	}
}

// TestValidateNpc_RPSActionMissingFailureStateReference verifies that a
// dangling failureState is reported (mirrors gachaponAction's invalid_reference
// check) rather than silently passing.
func TestValidateNpc_RPSActionMissingFailureStateReference(t *testing.T) {
	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("doesNotExist").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}
	rpsState, err := NewStateBuilder().SetId("playRPS").SetRPSAction(rpsAction).Build()
	if err != nil {
		t.Fatalf("build rps state: %v", err)
	}

	m := testNpcConversation{
		npcId:  9000019,
		start:  "playRPS",
		states: []StateModel{rpsState},
	}

	result := NewValidator().ValidateNpc(m)

	if result.Valid {
		t.Fatalf("ValidateNpc returned Valid=true, want false (dangling failureState)")
	}
	found := false
	for _, e := range result.Errors {
		if e.Field == "rpsAction.failureState" && e.ErrorType == "invalid_reference" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected rpsAction.failureState invalid_reference error; got %+v", result.Errors)
	}
}
