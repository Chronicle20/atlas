package pet

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The pet Kafka contract is duplicated across atlas-pets (owner), atlas-channel,
// and atlas-saga-orchestrator, in three separate Go modules with no mirror guard
// (only trade has one). A field name or json tag changed in one and not the
// others fails NO build — it decodes into a zero-valued body at runtime,
// silently. These fixtures are byte-for-byte what atlas-pets' structs marshal;
// if this side drifts, the assertions below go red.
func TestNameChangedStatusEventBodyDecodesOwnerWire(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	wire := []byte(`{"petId":7,"ownerId":42,"type":"NAME_CHANGED","body":{"slot":3,"name":"Renamed","previousName":"Original","transactionId":"11111111-2222-3333-4444-555555555555"}}`)

	var e StatusEvent[NameChangedStatusEventBody]
	if err := json.Unmarshal(wire, &e); err != nil {
		t.Fatalf("Unmarshal = %v", err)
	}

	if e.PetId != 7 || e.OwnerId != 42 || e.Type != StatusEventTypeNameChanged {
		t.Fatalf("envelope drifted: %+v", e)
	}
	if e.Body.Slot != 3 || e.Body.Name != "Renamed" || e.Body.PreviousName != "Original" || e.Body.TransactionId != txn {
		t.Fatalf("body drifted: %+v", e.Body)
	}
}

func TestRenameCommandBodyDecodesOwnerWire(t *testing.T) {
	wire := []byte(`{"transactionId":"11111111-2222-3333-4444-555555555555","actorId":42,"petId":7,"type":"RENAME","body":{"name":"Renamed"}}`)

	var c Command[RenameCommandBody]
	if err := json.Unmarshal(wire, &c); err != nil {
		t.Fatalf("Unmarshal = %v", err)
	}
	if c.ActorId != 42 || c.PetId != 7 || c.Type != CommandPetRename || c.Body.Name != "Renamed" {
		t.Fatalf("drifted: %+v", c)
	}
}
