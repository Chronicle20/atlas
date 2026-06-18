package party

import (
	party2 "atlas-channel/kafka/message/party"
	"encoding/json"
	"testing"
)

// TestLeaveCommandProviderExpelCarriesTarget guards the party-expel bug: the
// expel command must carry the TARGET character id in its body, distinct from
// the actor (expeller). Previously the target was dropped, so the parties
// service expelled the actor (the party leader) instead.
func TestLeaveCommandProviderExpelCarriesTarget(t *testing.T) {
	const expeller = uint32(100)
	const target = uint32(200)
	const partyId = uint32(5)

	msgs, err := LeaveCommandProvider(expeller, partyId, target, true)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd party2.Command[party2.LeaveCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unable to unmarshal command: %v", err)
	}

	if cmd.ActorId != expeller {
		t.Errorf("ActorId = %d, want %d (the expeller)", cmd.ActorId, expeller)
	}
	if !cmd.Body.Force {
		t.Errorf("Body.Force = false, want true for an expel")
	}
	if cmd.Body.PartyId != partyId {
		t.Errorf("Body.PartyId = %d, want %d", cmd.Body.PartyId, partyId)
	}
	if cmd.Body.CharacterId != target {
		t.Errorf("Body.CharacterId = %d, want %d (the expel target)", cmd.Body.CharacterId, target)
	}
}
