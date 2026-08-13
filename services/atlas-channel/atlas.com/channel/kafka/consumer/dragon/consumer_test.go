package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"
	"testing"
)

// recipientPolicy is the single fact each handler encodes: CREATED and
// DESTROYED go map-wide including the owner; MOVED excludes the owner because
// their client already rendered the motion locally and re-sending double-applies
// it (the same reasoning as the summon move relay).
func TestRecipientPolicyPerEventType(t *testing.T) {
	cases := []struct {
		eventType     string
		excludesOwner bool
	}{
		{dragonmsg.EventDragonStatusCreated, false},
		{dragonmsg.EventDragonStatusMoved, true},
		{dragonmsg.EventDragonStatusDestroyed, false},
	}
	for _, c := range cases {
		if got := excludesOwner(c.eventType); got != c.excludesOwner {
			t.Errorf("%s: excludesOwner = %v, want %v", c.eventType, got, c.excludesOwner)
		}
	}
}

func TestUnknownEventTypeExcludesNobodyAndIsNotBroadcast(t *testing.T) {
	if handles("SOMETHING_ELSE") {
		t.Fatal("an unrecognised dragon event type must not be broadcast")
	}
}
