package conversation

import (
	"testing"
)

// TestRPSAction_RESTRoundTrip verifies that an rpsAction state survives a
// TransformState → ExtractState round trip (the REST/seed ingestion path).
// Before this wiring, ExtractState's default arm rejected "rpsAction" with
// "invalid state type: rpsAction", which would hard-fail the NPC 9000019 seed.
func TestRPSAction_RESTRoundTrip(t *testing.T) {
	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}

	state, err := NewStateBuilder().SetId("playRPS").SetRPSAction(rpsAction).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	rest, err := TransformState(state)
	if err != nil {
		t.Fatalf("TransformState: %v", err)
	}
	if rest.StateType != string(RPSActionType) {
		t.Errorf("rest.StateType = %q, want %q", rest.StateType, RPSActionType)
	}
	if rest.RPSAction == nil {
		t.Fatalf("rest.RPSAction = nil after TransformState")
	}
	if rest.RPSAction.NpcId != 9000019 {
		t.Errorf("rest RPSAction.NpcId = %d, want %d", rest.RPSAction.NpcId, 9000019)
	}
	if rest.RPSAction.EntryCostMeso != 1000 {
		t.Errorf("rest RPSAction.EntryCostMeso = %d, want %d", rest.RPSAction.EntryCostMeso, 1000)
	}
	if rest.RPSAction.FailureState != "noMeso" {
		t.Errorf("rest RPSAction.FailureState = %q, want %q", rest.RPSAction.FailureState, "noMeso")
	}

	back, err := ExtractState(rest)
	if err != nil {
		t.Fatalf("ExtractState: %v", err)
	}
	if back.Type() != RPSActionType {
		t.Errorf("extracted Type() = %q, want %q", back.Type(), RPSActionType)
	}
	got := back.RPSAction()
	if got == nil {
		t.Fatalf("extracted RPSAction() = nil")
	}
	if got.NpcId() != 9000019 {
		t.Errorf("extracted NpcId() = %d, want %d", got.NpcId(), 9000019)
	}
	if got.EntryCostMeso() != 1000 {
		t.Errorf("extracted EntryCostMeso() = %d, want %d", got.EntryCostMeso(), 1000)
	}
	if got.FailureState() != "noMeso" {
		t.Errorf("extracted FailureState() = %q, want %q", got.FailureState(), "noMeso")
	}
}
