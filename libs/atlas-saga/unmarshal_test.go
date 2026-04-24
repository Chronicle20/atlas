package saga

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalRebalanceAPStep(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		charId  uint32
		targets []RebalanceTarget
	}{
		{
			name: "single target strength",
			raw: `{
				"stepId": "rebalance_ap-1",
				"status": "pending",
				"action": "rebalance_ap",
				"payload": {
					"characterId": 1,
					"worldId": 0,
					"channelId": 1,
					"targets": [
						{"stat": "strength", "floor": 35}
					]
				},
				"createdAt": "2026-04-24T00:00:00Z",
				"updatedAt": "2026-04-24T00:00:00Z"
			}`,
			charId: 1,
			targets: []RebalanceTarget{
				{Stat: RebalanceStatStrength, Floor: 35},
			},
		},
		{
			name: "single target dexterity",
			raw: `{
				"stepId": "rebalance_ap-2",
				"status": "pending",
				"action": "rebalance_ap",
				"payload": {
					"characterId": 2,
					"worldId": 0,
					"channelId": 1,
					"targets": [
						{"stat": "dexterity", "floor": 25}
					]
				},
				"createdAt": "2026-04-24T00:00:00Z",
				"updatedAt": "2026-04-24T00:00:00Z"
			}`,
			charId: 2,
			targets: []RebalanceTarget{
				{Stat: RebalanceStatDexterity, Floor: 25},
			},
		},
		{
			name: "single target intelligence",
			raw: `{
				"stepId": "rebalance_ap-3",
				"status": "pending",
				"action": "rebalance_ap",
				"payload": {
					"characterId": 3,
					"worldId": 0,
					"channelId": 1,
					"targets": [
						{"stat": "intelligence", "floor": 20}
					]
				},
				"createdAt": "2026-04-24T00:00:00Z",
				"updatedAt": "2026-04-24T00:00:00Z"
			}`,
			charId: 3,
			targets: []RebalanceTarget{
				{Stat: RebalanceStatIntelligence, Floor: 20},
			},
		},
		{
			name: "single target luck",
			raw: `{
				"stepId": "rebalance_ap-4",
				"status": "pending",
				"action": "rebalance_ap",
				"payload": {
					"characterId": 4,
					"worldId": 0,
					"channelId": 1,
					"targets": [
						{"stat": "luck", "floor": 25}
					]
				},
				"createdAt": "2026-04-24T00:00:00Z",
				"updatedAt": "2026-04-24T00:00:00Z"
			}`,
			charId: 4,
			targets: []RebalanceTarget{
				{Stat: RebalanceStatLuck, Floor: 25},
			},
		},
		{
			name: "multi-target thunder breaker STR+DEX",
			raw: `{
				"stepId": "rebalance_ap-5",
				"status": "pending",
				"action": "rebalance_ap",
				"payload": {
					"characterId": 5,
					"worldId": 0,
					"channelId": 1,
					"targets": [
						{"stat": "strength", "floor": 20},
						{"stat": "dexterity", "floor": 20}
					]
				},
				"createdAt": "2026-04-24T00:00:00Z",
				"updatedAt": "2026-04-24T00:00:00Z"
			}`,
			charId: 5,
			targets: []RebalanceTarget{
				{Stat: RebalanceStatStrength, Floor: 20},
				{Stat: RebalanceStatDexterity, Floor: 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var step Step[any]
			if err := json.Unmarshal([]byte(tt.raw), &step); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if step.Action != RebalanceAP {
				t.Fatalf("expected action RebalanceAP, got %q", step.Action)
			}
			p, ok := step.Payload.(RebalanceAPPayload)
			if !ok {
				t.Fatalf("expected RebalanceAPPayload, got %T", step.Payload)
			}
			if p.CharacterId != tt.charId {
				t.Errorf("characterId: expected %d, got %d", tt.charId, p.CharacterId)
			}
			if len(p.Targets) != len(tt.targets) {
				t.Fatalf("expected %d targets, got %d", len(tt.targets), len(p.Targets))
			}
			for i, want := range tt.targets {
				got := p.Targets[i]
				if got.Stat != want.Stat {
					t.Errorf("target[%d].Stat: expected %q, got %q", i, want.Stat, got.Stat)
				}
				if got.Floor != want.Floor {
					t.Errorf("target[%d].Floor: expected %d, got %d", i, want.Floor, got.Floor)
				}
			}
		})
	}
}
