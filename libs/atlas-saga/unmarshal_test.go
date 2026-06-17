package saga

import (
	"encoding/json"
	"strings"
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

func TestCreateAndEquipAssetPayload_UseAverageStats_RoundTrip(t *testing.T) {
	in := CreateAndEquipAssetPayload{
		CharacterId:     42,
		Item:            ItemPayload{TemplateId: 1002357, Quantity: 1},
		UseAverageStats: true,
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(bs), `"useAverageStats":true`) {
		t.Fatalf("expected useAverageStats:true in payload, got %s", string(bs))
	}
	var out CreateAndEquipAssetPayload
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.UseAverageStats {
		t.Fatalf("expected UseAverageStats=true after round-trip, got false")
	}

	// Backwards-compat: missing field decodes to false.
	var legacy CreateAndEquipAssetPayload
	if err := json.Unmarshal([]byte(`{"characterId":7,"item":{"templateId":1,"quantity":1}}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.UseAverageStats {
		t.Fatalf("expected legacy payload to default UseAverageStats=false")
	}
}

func TestCharacterCreatePayload_GmAndMeso_RoundTrip(t *testing.T) {
	in := CharacterCreatePayload{
		AccountId: 1,
		Name:      "AdminHero",
		Gm:        2,
		Meso:      100_000_000,
	}
	bs, _ := json.Marshal(in)
	if !strings.Contains(string(bs), `"gm":2`) || !strings.Contains(string(bs), `"meso":100000000`) {
		t.Fatalf("expected gm/meso in payload, got %s", string(bs))
	}
	var out CharacterCreatePayload
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Gm != 2 || out.Meso != 100_000_000 {
		t.Fatalf("expected gm=2 meso=1e8, got gm=%d meso=%d", out.Gm, out.Meso)
	}

	// Backwards-compat: legacy payload defaults both to zero.
	var legacy CharacterCreatePayload
	if err := json.Unmarshal([]byte(`{"accountId":1,"name":"Foo"}`), &legacy); err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if legacy.Gm != 0 || legacy.Meso != 0 {
		t.Fatalf("expected gm=0 meso=0 from legacy payload")
	}
}

func TestUnmarshalAwaitInventoryCreatedStep(t *testing.T) {
	raw := `{
		"stepId": "await_inventory_created-1",
		"status": "pending",
		"action": "await_inventory_created",
		"payload": {
			"characterId": 12345
		},
		"createdAt": "2026-05-15T00:00:00Z",
		"updatedAt": "2026-05-15T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AwaitInventoryCreated {
		t.Fatalf("expected action AwaitInventoryCreated, got %q", step.Action)
	}
	p, ok := step.Payload.(AwaitInventoryCreatedPayload)
	if !ok {
		t.Fatalf("expected AwaitInventoryCreatedPayload, got %T", step.Payload)
	}
	if p.CharacterId != 12345 {
		t.Errorf("characterId: expected 12345, got %d", p.CharacterId)
	}
}

func TestUnmarshalEvolvePetStep(t *testing.T) {
	raw := `{
		"stepId": "evolve_pet-1",
		"status": "pending",
		"action": "evolve_pet",
		"payload": {
			"characterId": 100,
			"petId": 200
		},
		"createdAt": "2026-06-12T00:00:00Z",
		"updatedAt": "2026-06-12T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != EvolvePet {
		t.Fatalf("expected action EvolvePet, got %q", step.Action)
	}
	p, ok := step.Payload.(EvolvePetPayload)
	if !ok {
		t.Fatalf("expected EvolvePetPayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.PetId != 200 {
		t.Errorf("petId: expected 200, got %d", p.PetId)
	}
}

func TestUnmarshalTransferToMtsStep(t *testing.T) {
	raw := `{
		"stepId": "transfer_to_mts-1",
		"status": "pending",
		"action": "transfer_to_mts",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"characterId": 100,
			"worldId": 0,
			"sourceInventoryType": 1,
			"assetId": 555,
			"quantity": 1,
			"listingId": "22222222-2222-2222-2222-222222222222"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != TransferToMts {
		t.Fatalf("expected action TransferToMts, got %q", step.Action)
	}
	p, ok := step.Payload.(TransferToMtsPayload)
	if !ok {
		t.Fatalf("expected TransferToMtsPayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.AssetId != 555 {
		t.Errorf("assetId: expected 555, got %d", p.AssetId)
	}
}

func TestUnmarshalWithdrawFromMtsStep(t *testing.T) {
	raw := `{
		"stepId": "withdraw_from_mts-1",
		"status": "pending",
		"action": "withdraw_from_mts",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"characterId": 101,
			"worldId": 0,
			"holdingId": "33333333-3333-3333-3333-333333333333",
			"inventoryType": 2
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != WithdrawFromMts {
		t.Fatalf("expected action WithdrawFromMts, got %q", step.Action)
	}
	p, ok := step.Payload.(WithdrawFromMtsPayload)
	if !ok {
		t.Fatalf("expected WithdrawFromMtsPayload, got %T", step.Payload)
	}
	if p.CharacterId != 101 {
		t.Errorf("characterId: expected 101, got %d", p.CharacterId)
	}
	if p.InventoryType != 2 {
		t.Errorf("inventoryType: expected 2, got %d", p.InventoryType)
	}
}

func TestUnmarshalAcceptToMtsListingStep(t *testing.T) {
	raw := `{
		"stepId": "accept_to_mts_listing-1",
		"status": "pending",
		"action": "accept_to_mts_listing",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"listingId": "22222222-2222-2222-2222-222222222222"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AcceptToMtsListing {
		t.Fatalf("expected action AcceptToMtsListing, got %q", step.Action)
	}
	p, ok := step.Payload.(AcceptToMtsListingPayload)
	if !ok {
		t.Fatalf("expected AcceptToMtsListingPayload, got %T", step.Payload)
	}
	if p.ListingId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("listingId mismatch, got %s", p.ListingId)
	}
}

func TestUnmarshalReleaseFromMtsHoldingStep(t *testing.T) {
	raw := `{
		"stepId": "release_from_mts_holding-1",
		"status": "pending",
		"action": "release_from_mts_holding",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"holdingId": "33333333-3333-3333-3333-333333333333"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != ReleaseFromMtsHolding {
		t.Fatalf("expected action ReleaseFromMtsHolding, got %q", step.Action)
	}
	p, ok := step.Payload.(ReleaseFromMtsHoldingPayload)
	if !ok {
		t.Fatalf("expected ReleaseFromMtsHoldingPayload, got %T", step.Payload)
	}
	if p.HoldingId.String() != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("holdingId mismatch, got %s", p.HoldingId)
	}
}

func TestUnmarshalMtsSettlePurchaseStep(t *testing.T) {
	raw := `{
		"stepId": "mts_settle_purchase-1",
		"status": "pending",
		"action": "mts_settle_purchase",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"listingId": "22222222-2222-2222-2222-222222222222",
			"buyerId": 100,
			"buyerAccountId": 10,
			"sellerId": 200,
			"sellerAccountId": 20,
			"markedUpPrice": 1100,
			"listValue": 1000
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != MtsSettlePurchase {
		t.Fatalf("expected action MtsSettlePurchase, got %q", step.Action)
	}
	p, ok := step.Payload.(MtsSettlePurchasePayload)
	if !ok {
		t.Fatalf("expected MtsSettlePurchasePayload, got %T", step.Payload)
	}
	if p.BuyerId != 100 {
		t.Errorf("buyerId: expected 100, got %d", p.BuyerId)
	}
	if p.MarkedUpPrice != 1100 {
		t.Errorf("markedUpPrice: expected 1100, got %d", p.MarkedUpPrice)
	}
	if p.ListValue != 1000 {
		t.Errorf("listValue: expected 1000, got %d", p.ListValue)
	}
}

func TestUnmarshalMtsMoveListingToHoldingStep(t *testing.T) {
	raw := `{
		"stepId": "mts_move_listing_to_holding-1",
		"status": "pending",
		"action": "mts_move_listing_to_holding",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"listingId": "22222222-2222-2222-2222-222222222222"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != MtsMoveListingToHolding {
		t.Fatalf("expected action MtsMoveListingToHolding, got %q", step.Action)
	}
	p, ok := step.Payload.(MtsMoveListingToHoldingPayload)
	if !ok {
		t.Fatalf("expected MtsMoveListingToHoldingPayload, got %T", step.Payload)
	}
	if p.ListingId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("listingId mismatch, got %s", p.ListingId)
	}
}

func TestUnmarshalMtsBidEscrowStep(t *testing.T) {
	raw := `{
		"stepId": "mts_bid_escrow-1",
		"status": "pending",
		"action": "mts_bid_escrow",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"listingId": "22222222-2222-2222-2222-222222222222",
			"bidderId": 100,
			"bidderAccountId": 10,
			"amount": -500
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != MtsBidEscrow {
		t.Fatalf("expected action MtsBidEscrow, got %q", step.Action)
	}
	p, ok := step.Payload.(MtsBidEscrowPayload)
	if !ok {
		t.Fatalf("expected MtsBidEscrowPayload, got %T", step.Payload)
	}
	if p.BidderId != 100 {
		t.Errorf("bidderId: expected 100, got %d", p.BidderId)
	}
	if p.Amount != -500 {
		t.Errorf("amount: expected -500, got %d", p.Amount)
	}
}

func TestUnmarshalAwaitInventoryCreatedStep_ZeroCharacterId(t *testing.T) {
	// Mirrors the sentinel-payload shape that character-factory emits before
	// orchestrator result-forwarding substitutes the real characterId.
	raw := `{
		"stepId": "await_inventory_created-1",
		"status": "pending",
		"action": "await_inventory_created",
		"payload": {"characterId": 0},
		"createdAt": "2026-05-15T00:00:00Z",
		"updatedAt": "2026-05-15T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	p, ok := step.Payload.(AwaitInventoryCreatedPayload)
	if !ok {
		t.Fatalf("expected AwaitInventoryCreatedPayload, got %T", step.Payload)
	}
	if p.CharacterId != 0 {
		t.Errorf("expected sentinel characterId=0, got %d", p.CharacterId)
	}
}
