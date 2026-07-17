package saga

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

func TestUnmarshalTransferAPStep(t *testing.T) {
	raw := `{
		"stepId": "transfer_ap-1",
		"status": "pending",
		"action": "transfer_ap",
		"payload": {
			"characterId": 100,
			"worldId": 0,
			"channelId": 1,
			"from": "STRENGTH",
			"to": "DEXTERITY"
		},
		"createdAt": "2026-07-02T00:00:00Z",
		"updatedAt": "2026-07-02T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != TransferAP {
		t.Fatalf("expected action TransferAP, got %q", step.Action)
	}
	p, ok := step.Payload.(TransferAPPayload)
	if !ok {
		t.Fatalf("expected TransferAPPayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.From != "STRENGTH" {
		t.Errorf("from: expected STRENGTH, got %q", p.From)
	}
	if p.To != "DEXTERITY" {
		t.Errorf("to: expected DEXTERITY, got %q", p.To)
	}
}

func TestUnmarshalTransferSPStep(t *testing.T) {
	raw := `{
		"stepId": "transfer_sp-1",
		"status": "pending",
		"action": "transfer_sp",
		"payload": {
			"characterId": 100,
			"worldId": 0,
			"channelId": 1,
			"jobId": 200,
			"fromSkillId": 2001002,
			"toSkillId": 2001003,
			"itemTier": 1,
			"targetMaxLevel": 20
		},
		"createdAt": "2026-07-02T00:00:00Z",
		"updatedAt": "2026-07-02T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != TransferSP {
		t.Fatalf("expected action TransferSP, got %q", step.Action)
	}
	p, ok := step.Payload.(TransferSPPayload)
	if !ok {
		t.Fatalf("expected TransferSPPayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.JobId != 200 {
		t.Errorf("jobId: expected 200, got %d", p.JobId)
	}
	if p.FromSkillId != 2001002 {
		t.Errorf("fromSkillId: expected 2001002, got %d", p.FromSkillId)
	}
	if p.ToSkillId != 2001003 {
		t.Errorf("toSkillId: expected 2001003, got %d", p.ToSkillId)
	}
	if p.ItemTier != 1 {
		t.Errorf("itemTier: expected 1, got %d", p.ItemTier)
	}
	if p.TargetMaxLevel != 20 {
		t.Errorf("targetMaxLevel: expected 20, got %d", p.TargetMaxLevel)
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
			"listingId": "22222222-2222-2222-2222-222222222222",
			"sellerName": "Seller",
			"saleType": "buy_now",
			"listValue": 1000,
			"buyNowPrice": 1500,
			"commissionRate": 0.1,
			"category": "equip",
			"subCategory": "onehanded",
			"endsAt": "2026-06-20T00:00:00Z",
			"minIncrement": 50
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
	if p.SellerName != "Seller" {
		t.Errorf("sellerName: expected Seller, got %q", p.SellerName)
	}
	if p.SaleType != "buy_now" {
		t.Errorf("saleType: expected buy_now, got %q", p.SaleType)
	}
	if p.ListValue != 1000 {
		t.Errorf("listValue: expected 1000, got %d", p.ListValue)
	}
	if p.BuyNowPrice == nil || *p.BuyNowPrice != 1500 {
		t.Errorf("buyNowPrice: expected 1500, got %v", p.BuyNowPrice)
	}
	if p.CommissionRate != 0.1 {
		t.Errorf("commissionRate: expected 0.1, got %v", p.CommissionRate)
	}
	if p.Category != "equip" {
		t.Errorf("category: expected equip, got %q", p.Category)
	}
	if p.SubCategory != "onehanded" {
		t.Errorf("subCategory: expected onehanded, got %q", p.SubCategory)
	}
	if p.EndsAt == nil || !p.EndsAt.Equal(time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("endsAt: expected 2026-06-20T00:00:00Z, got %v", p.EndsAt)
	}
	if p.MinIncrement != 50 {
		t.Errorf("minIncrement: expected 50, got %d", p.MinIncrement)
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
			"listingId": "22222222-2222-2222-2222-222222222222",
			"worldId": 0,
			"sellerId": 200,
			"sellerName": "Seller",
			"saleType": "buy_now",
			"templateId": 1302000,
			"quantity": 1,
			"strength": 5,
			"dexterity": 6,
			"intelligence": 7,
			"luck": 8,
			"hp": 100,
			"mp": 50,
			"weaponAttack": 30,
			"magicAttack": 20,
			"weaponDefense": 10,
			"magicDefense": 12,
			"accuracy": 14,
			"avoidability": 16,
			"hands": 1,
			"speed": 4,
			"jump": 3,
			"slots": 7,
			"level": 2,
			"itemLevel": 9,
			"itemExp": 12345,
			"ringId": 999,
			"viciousCount": 2,
			"flags": 64,
			"listValue": 1000,
			"buyNowPrice": 1500,
			"commissionRate": 0.1,
			"category": "equip",
			"subCategory": "onehanded",
			"endsAt": "2026-06-20T00:00:00Z",
			"minIncrement": 50
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
	if p.SellerId != 200 {
		t.Errorf("sellerId: expected 200, got %d", p.SellerId)
	}
	if p.SellerName != "Seller" {
		t.Errorf("sellerName: expected Seller, got %q", p.SellerName)
	}
	if p.SaleType != "buy_now" {
		t.Errorf("saleType: expected buy_now, got %q", p.SaleType)
	}
	if p.TemplateId != 1302000 {
		t.Errorf("templateId: expected 1302000, got %d", p.TemplateId)
	}
	if p.Quantity != 1 {
		t.Errorf("quantity: expected 1, got %d", p.Quantity)
	}
	if p.Strength != 5 || p.Dexterity != 6 || p.Intelligence != 7 || p.Luck != 8 {
		t.Errorf("stat block STR/DEX/INT/LUK mismatch: %d/%d/%d/%d", p.Strength, p.Dexterity, p.Intelligence, p.Luck)
	}
	if p.HP != 100 || p.MP != 50 {
		t.Errorf("HP/MP mismatch: %d/%d", p.HP, p.MP)
	}
	if p.WeaponAttack != 30 || p.MagicAttack != 20 || p.WeaponDefense != 10 || p.MagicDefense != 12 {
		t.Errorf("atk/def block mismatch: %d/%d/%d/%d", p.WeaponAttack, p.MagicAttack, p.WeaponDefense, p.MagicDefense)
	}
	if p.Accuracy != 14 || p.Avoidability != 16 || p.Hands != 1 || p.Speed != 4 || p.Jump != 3 || p.Slots != 7 {
		t.Errorf("acc/avoid/hands/speed/jump/slots mismatch: %d/%d/%d/%d/%d/%d", p.Accuracy, p.Avoidability, p.Hands, p.Speed, p.Jump, p.Slots)
	}
	if p.Level != 2 || p.ItemLevel != 9 {
		t.Errorf("level/itemLevel mismatch: %d/%d", p.Level, p.ItemLevel)
	}
	if p.ItemExp != 12345 || p.RingId != 999 || p.ViciousCount != 2 || p.Flags != 64 {
		t.Errorf("itemExp/ringId/viciousCount/flags mismatch: %d/%d/%d/%d", p.ItemExp, p.RingId, p.ViciousCount, p.Flags)
	}
	if p.ListValue != 1000 {
		t.Errorf("listValue: expected 1000, got %d", p.ListValue)
	}
	if p.BuyNowPrice == nil || *p.BuyNowPrice != 1500 {
		t.Errorf("buyNowPrice: expected 1500, got %v", p.BuyNowPrice)
	}
	if p.CommissionRate != 0.1 {
		t.Errorf("commissionRate: expected 0.1, got %v", p.CommissionRate)
	}
	if p.Category != "equip" || p.SubCategory != "onehanded" {
		t.Errorf("category/subCategory mismatch: %q/%q", p.Category, p.SubCategory)
	}
	if p.EndsAt == nil || !p.EndsAt.Equal(time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("endsAt: expected 2026-06-20T00:00:00Z, got %v", p.EndsAt)
	}
	if p.MinIncrement != 50 {
		t.Errorf("minIncrement: expected 50, got %d", p.MinIncrement)
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
			"worldId": 0,
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
			"listingId": "22222222-2222-2222-2222-222222222222",
			"buyerId": 100,
			"worldId": 0
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
	if p.BuyerId != 100 {
		t.Errorf("buyerId: expected 100, got %d", p.BuyerId)
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

func TestUnmarshalStartRPSGameStep(t *testing.T) {
	raw := `{
		"stepId": "start_rps_game-1",
		"status": "pending",
		"action": "start_rps_game",
		"payload": { "characterId": 100, "worldId": 0, "channelId": 1, "npcId": 9000019 },
		"createdAt": "2026-07-04T00:00:00Z",
		"updatedAt": "2026-07-04T00:00:00Z"
	}`
	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != StartRPSGame {
		t.Fatalf("expected action StartRPSGame, got %s", step.Action)
	}
	p, ok := step.Payload.(StartRPSGamePayload)
	if !ok {
		t.Fatalf("expected StartRPSGamePayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 || p.NpcId != 9000019 {
		t.Errorf("payload mismatch: %+v", p)
	}
}

func TestUnmarshalSetAssetOwnerStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"set_asset_owner","payload":{"characterId":7,"inventoryType":1,"slot":-5,"owner":"Tumi"},"createdAt":"2026-07-02T00:00:00Z","updatedAt":"2026-07-02T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(SetAssetOwnerPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.Owner != "Tumi" || p.Slot != -5 || p.InventoryType != 1 || p.CharacterId != 7 {
		t.Fatalf("payload = %+v", p)
	}
}

func TestUnmarshalApplyAssetLockStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"apply_asset_lock","payload":{"characterId":7,"inventoryType":1,"slot":3,"expiration":"2026-08-01T12:00:00Z"},"createdAt":"2026-07-02T00:00:00Z","updatedAt":"2026-07-02T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(ApplyAssetLockPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	wantExpiration, err := time.Parse(time.RFC3339, "2026-08-01T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if p.CharacterId != 7 || p.InventoryType != 1 || p.Slot != 3 || !p.Expiration.Equal(wantExpiration) {
		t.Fatalf("payload = %+v", p)
	}
}

func TestUnmarshalIncubatorResultStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"incubator_result","payload":{"characterId":7,"worldId":0,"channelId":1,"itemId":4001126,"count":3},"createdAt":"2026-07-02T00:00:00Z","updatedAt":"2026-07-02T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(IncubatorResultPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.CharacterId != 7 || p.WorldId != 0 || p.ChannelId != 1 || p.ItemId != 4001126 || p.Count != 3 {
		t.Fatalf("payload = %+v", p)
	}
}

func TestUnmarshalDestroyAssetFromSlotTemplateId(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"destroy_asset_from_slot","payload":{"characterId":7,"inventoryType":4,"slot":2,"quantity":1,"templateId":4001126},"createdAt":"2026-07-02T00:00:00Z","updatedAt":"2026-07-02T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(DestroyAssetFromSlotPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.TemplateId != 4001126 {
		t.Fatalf("payload = %+v", p)
	}
}
