package saga

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUnmarshalRebalanceAPStep(t *testing.T) {
	raw := []byte(`{
		"stepId": "rebalance_ap-42",
		"status": "pending",
		"action": "rebalance_ap",
		"payload": {
			"characterId": 42,
			"worldId": 0,
			"channelId": 1,
			"targets": [
				{"stat": "dexterity", "floor": 20}
			]
		},
		"createdAt": "2026-04-24T00:00:00Z",
		"updatedAt": "2026-04-24T00:00:00Z"
	}`)

	var step Step[any]
	if err := json.Unmarshal(raw, &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != RebalanceAP {
		t.Fatalf("expected action RebalanceAP, got %q", step.Action)
	}
	p, ok := step.Payload.(RebalanceAPPayload)
	if !ok {
		t.Fatalf("expected RebalanceAPPayload, got %T", step.Payload)
	}
	if p.CharacterId != 42 {
		t.Errorf("characterId: expected 42, got %d", p.CharacterId)
	}
	if len(p.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != RebalanceStatDexterity {
		t.Errorf("stat: expected dexterity, got %q", p.Targets[0].Stat)
	}
	if p.Targets[0].Floor != 20 {
		t.Errorf("floor: expected 20, got %d", p.Targets[0].Floor)
	}

	// Silence unused import warning if uuid isn't used elsewhere in the test.
	_ = uuid.Nil
}
