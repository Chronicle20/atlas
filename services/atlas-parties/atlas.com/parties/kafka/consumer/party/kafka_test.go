package party

import (
	"encoding/json"
	"testing"
)

// TestLeaveCommandBodyDecodesExpelTarget guards the consumer half of the
// party-expel bug: the LEAVE command body must decode the target character id
// so the force (expel) branch removes the target, not the actor.
func TestLeaveCommandBodyDecodesExpelTarget(t *testing.T) {
	raw := []byte(`{"actorId":100,"type":"LEAVE","body":{"partyId":5,"force":true,"characterId":200}}`)

	var cmd commandEvent[leaveCommandBody]
	if err := json.Unmarshal(raw, &cmd); err != nil {
		t.Fatalf("unable to unmarshal command: %v", err)
	}

	if cmd.ActorId != 100 {
		t.Errorf("ActorId = %d, want 100 (the expeller)", cmd.ActorId)
	}
	if !cmd.Body.Force {
		t.Errorf("Body.Force = false, want true for an expel")
	}
	if cmd.Body.CharacterId != 200 {
		t.Errorf("Body.CharacterId = %d, want 200 (the expel target)", cmd.Body.CharacterId)
	}
}
