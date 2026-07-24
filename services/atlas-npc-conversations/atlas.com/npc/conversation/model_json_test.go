package conversation

import (
	"encoding/json"
	"testing"
)

// TestRPSActionState_JSONRoundTrip verifies that a StateModel built with an
// rpsAction state survives a MarshalJSON/UnmarshalJSON round trip, mirroring
// the gachaponAction envelope wiring (npcId, entryCostMeso, failureState).
func TestRPSActionState_JSONRoundTrip(t *testing.T) {
	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}

	state, err := NewStateBuilder().
		SetId("playRPS").
		SetRPSAction(rpsAction).
		Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}

	// Confirm the envelope carries the "rpsAction" key (not silently dropped).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["rpsAction"]; !ok {
		t.Fatalf("marshaled state envelope missing \"rpsAction\" key: %s", data)
	}

	var roundTripped StateModel
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	if roundTripped.Type() != RPSActionType {
		t.Errorf("Type() = %q, want %q", roundTripped.Type(), RPSActionType)
	}
	if roundTripped.Id() != "playRPS" {
		t.Errorf("Id() = %q, want %q", roundTripped.Id(), "playRPS")
	}

	got := roundTripped.RPSAction()
	if got == nil {
		t.Fatalf("RPSAction() = nil after round trip")
	}
	if got.NpcId() != 9000019 {
		t.Errorf("NpcId() = %d, want %d", got.NpcId(), 9000019)
	}
	if got.EntryCostMeso() != 1000 {
		t.Errorf("EntryCostMeso() = %d, want %d", got.EntryCostMeso(), 1000)
	}
	if got.FailureState() != "noMeso" {
		t.Errorf("FailureState() = %q, want %q", got.FailureState(), "noMeso")
	}
}
