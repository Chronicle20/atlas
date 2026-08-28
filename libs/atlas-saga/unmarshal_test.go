package saga

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

func TestUnmarshalTransferToParcelStep(t *testing.T) {
	raw := `{
		"stepId": "transfer_to_parcel-1",
		"status": "pending",
		"action": "transfer_to_parcel",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"parcelId": "22222222-2222-2222-2222-222222222222",
			"characterId": 100,
			"worldId": 0,
			"sourceInventoryType": 1,
			"assetId": 555,
			"quantity": 1,
			"senderAccountId": 10,
			"senderName": "Sender",
			"recipientId": 200,
			"recipientAccountId": 20,
			"mesoAmount": 1000,
			"feePaid": 100,
			"quick": true,
			"message": "hello",
			"receivableAt": "2026-06-17T00:00:00Z",
			"expiresAt": "2026-06-24T00:00:00Z"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != TransferToParcel {
		t.Fatalf("expected action TransferToParcel, got %q", step.Action)
	}
	p, ok := step.Payload.(TransferToParcelPayload)
	if !ok {
		t.Fatalf("expected TransferToParcelPayload, got %T", step.Payload)
	}
	if p.ParcelId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("parcelId mismatch, got %s", p.ParcelId)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.AssetId != 555 {
		t.Errorf("assetId: expected 555, got %d", p.AssetId)
	}
	if p.MesoAmount != 1000 {
		t.Errorf("mesoAmount: expected 1000, got %d", p.MesoAmount)
	}
	if p.FeePaid != 100 {
		t.Errorf("feePaid: expected 100, got %d", p.FeePaid)
	}
	if !p.Quick {
		t.Errorf("quick: expected true, got %v", p.Quick)
	}
	if !p.ReceivableAt.Equal(time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("receivableAt: expected 2026-06-17T00:00:00Z, got %v", p.ReceivableAt)
	}
}

func TestUnmarshalAcceptToParcelStep(t *testing.T) {
	raw := `{
		"stepId": "accept_to_parcel-1",
		"status": "pending",
		"action": "accept_to_parcel",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"parcelId": "22222222-2222-2222-2222-222222222222",
			"characterId": 100,
			"worldId": 0,
			"senderAccountId": 10,
			"senderName": "Sender",
			"recipientId": 200,
			"recipientAccountId": 20,
			"mesoAmount": 1000,
			"feePaid": 100,
			"quick": true,
			"message": "hello",
			"receivableAt": "2026-06-17T00:00:00Z",
			"expiresAt": "2026-06-24T00:00:00Z",
			"hasItem": true,
			"templateId": 1302000,
			"quantity": 1,
			"owner": "Sender"
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AcceptToParcel {
		t.Fatalf("expected action AcceptToParcel, got %q", step.Action)
	}
	p, ok := step.Payload.(AcceptToParcelPayload)
	if !ok {
		t.Fatalf("expected AcceptToParcelPayload, got %T", step.Payload)
	}
	if p.ParcelId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("parcelId mismatch, got %s", p.ParcelId)
	}
	if p.RecipientId != 200 {
		t.Errorf("recipientId: expected 200, got %d", p.RecipientId)
	}
	if p.TemplateId != 1302000 {
		t.Errorf("templateId: expected 1302000, got %d", p.TemplateId)
	}
	if !p.HasItem {
		t.Errorf("hasItem: expected true, got %v", p.HasItem)
	}
	if p.Owner != "Sender" {
		t.Errorf("owner: expected Sender, got %q", p.Owner)
	}
}

func TestUnmarshalReleaseFromParcelStep(t *testing.T) {
	raw := `{
		"stepId": "release_from_parcel-1",
		"status": "pending",
		"action": "release_from_parcel",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"parcelId": "22222222-2222-2222-2222-222222222222",
			"recipientId": 200
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != ReleaseFromParcel {
		t.Fatalf("expected action ReleaseFromParcel, got %q", step.Action)
	}
	p, ok := step.Payload.(ReleaseFromParcelPayload)
	if !ok {
		t.Fatalf("expected ReleaseFromParcelPayload, got %T", step.Payload)
	}
	if p.ParcelId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("parcelId mismatch, got %s", p.ParcelId)
	}
	if p.RecipientId != 200 {
		t.Errorf("recipientId: expected 200, got %d", p.RecipientId)
	}
}

func TestUnmarshalWithdrawFromParcelStep(t *testing.T) {
	raw := `{
		"stepId": "withdraw_from_parcel-1",
		"status": "pending",
		"action": "withdraw_from_parcel",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"parcelId": "22222222-2222-2222-2222-222222222222",
			"characterId": 100,
			"worldId": 0,
			"inventoryType": 2
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != WithdrawFromParcel {
		t.Fatalf("expected action WithdrawFromParcel, got %q", step.Action)
	}
	p, ok := step.Payload.(WithdrawFromParcelPayload)
	if !ok {
		t.Fatalf("expected WithdrawFromParcelPayload, got %T", step.Payload)
	}
	if p.ParcelId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("parcelId mismatch, got %s", p.ParcelId)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.InventoryType != 2 {
		t.Errorf("inventoryType: expected 2, got %d", p.InventoryType)
	}
}

func TestUnmarshalShowParcelStep(t *testing.T) {
	raw := `{
		"stepId": "show_parcel-1",
		"status": "pending",
		"action": "show_parcel",
		"payload": {
			"characterId": 100,
			"npcId": 2030,
			"worldId": 0,
			"channelId": 1,
			"quick": false
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != ShowParcel {
		t.Fatalf("expected action ShowParcel, got %q", step.Action)
	}
	p, ok := step.Payload.(ShowParcelPayload)
	if !ok {
		t.Fatalf("expected ShowParcelPayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if p.NpcId != 2030 {
		t.Errorf("npcId: expected 2030, got %d", p.NpcId)
	}
	if p.WorldId != 0 {
		t.Errorf("worldId: expected 0, got %d", p.WorldId)
	}
	if p.ChannelId != 1 {
		t.Errorf("channelId: expected 1, got %d", p.ChannelId)
	}
	if p.Quick != false {
		t.Errorf("quick: expected false, got %v", p.Quick)
	}
}

func TestUnmarshalShowParcelStep_Quick(t *testing.T) {
	raw := `{
		"stepId": "show_parcel-1",
		"status": "pending",
		"action": "show_parcel",
		"payload": {
			"characterId": 100,
			"npcId": 2030,
			"worldId": 0,
			"channelId": 1,
			"quick": true
		},
		"createdAt": "2026-06-17T00:00:00Z",
		"updatedAt": "2026-06-17T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	p, ok := step.Payload.(ShowParcelPayload)
	if !ok {
		t.Fatalf("expected ShowParcelPayload, got %T", step.Payload)
	}
	if p.Quick != true {
		t.Errorf("quick: expected true, got %v", p.Quick)
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

func TestUnmarshalEmitMegaphoneStep(t *testing.T) {
	created := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	original := Step[any]{
		StepId: "emit_megaphone-1",
		Status: Pending,
		Action: EmitMegaphone,
		Payload: EmitMegaphonePayload{
			Tier:        "ITEM",
			Scope:       "WORLD",
			WorldId:     0,
			ChannelId:   1,
			CharacterId: 100,
			SenderName:  "Bob",
			SenderMedal: "<Legend>",
			Messages:    []string{"hello", "world"},
			WhispersOn:  true,
			Item: &AssetSnapshot{
				Slot:       1,
				TemplateId: 5100000,
				CashId:     999,
				Quantity:   1,
			},
		},
		CreatedAt: created,
		UpdatedAt: created,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var step Step[any]
	if err := json.Unmarshal(data, &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != EmitMegaphone {
		t.Fatalf("expected action EmitMegaphone, got %q", step.Action)
	}
	p, ok := step.Payload.(EmitMegaphonePayload)
	if !ok {
		t.Fatalf("expected EmitMegaphonePayload, got %T", step.Payload)
	}
	if p.Tier != "ITEM" {
		t.Errorf("tier: expected ITEM, got %q", p.Tier)
	}
	if p.Scope != "WORLD" {
		t.Errorf("scope: expected WORLD, got %q", p.Scope)
	}
	if p.CharacterId != 100 {
		t.Errorf("characterId: expected 100, got %d", p.CharacterId)
	}
	if len(p.Messages) != 2 || p.Messages[0] != "hello" || p.Messages[1] != "world" {
		t.Errorf("messages: expected [hello world], got %v", p.Messages)
	}
	if !p.WhispersOn {
		t.Errorf("whispersOn: expected true, got false")
	}
	if p.Item == nil {
		t.Fatalf("expected non-nil Item")
	}
	if p.Item.TemplateId != 5100000 {
		t.Errorf("item.templateId: expected 5100000, got %d", p.Item.TemplateId)
	}
	if p.Item.CashId != 999 {
		t.Errorf("item.cashId: expected 999, got %d", p.Item.CashId)
	}
}

func TestUnmarshalEnqueueWorldBroadcastStep(t *testing.T) {
	created := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	original := Step[any]{
		StepId: "enqueue_world_broadcast-1",
		Status: Pending,
		Action: EnqueueWorldBroadcast,
		Payload: EnqueueWorldBroadcastPayload{
			Family:          "AVATAR",
			WorldId:         0,
			ChannelId:       1,
			CharacterId:     100,
			SenderName:      "Alice",
			SenderMedal:     "",
			Messages:        []string{"a", "b", "c", "d"},
			WhispersOn:      false,
			ItemId:          5390000,
			TvMessageType:   "HEART",
			DurationSeconds: 30,
			SenderLook: AvatarSnapshot{
				Gender:    0,
				SkinColor: 0,
				Face:      20000,
				Hair:      30000,
				Equips:    map[int16]uint32{-1: 1002140, -5: 1040002},
				Pets:      map[int8]uint32{},
			},
			ReceiverName: "Carol",
			ReceiverLook: &AvatarSnapshot{
				Gender:       1,
				SkinColor:    2,
				Face:         21000,
				Hair:         31000,
				Equips:       map[int16]uint32{-1: 1002141},
				MaskedEquips: map[int16]uint32{-101: 1002999},
				Pets:         map[int8]uint32{1: 5000000},
			},
		},
		CreatedAt: created,
		UpdatedAt: created,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var step Step[any]
	if err := json.Unmarshal(data, &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != EnqueueWorldBroadcast {
		t.Fatalf("expected action EnqueueWorldBroadcast, got %q", step.Action)
	}
	p, ok := step.Payload.(EnqueueWorldBroadcastPayload)
	if !ok {
		t.Fatalf("expected EnqueueWorldBroadcastPayload, got %T", step.Payload)
	}
	if p.Family != "AVATAR" {
		t.Errorf("family: expected AVATAR, got %q", p.Family)
	}
	if p.TvMessageType != "HEART" {
		t.Errorf("tvMessageType: expected HEART, got %q", p.TvMessageType)
	}
	if p.ItemId != 5390000 {
		t.Errorf("itemId: expected 5390000, got %d", p.ItemId)
	}
	if p.ReceiverName != "Carol" {
		t.Errorf("receiverName: expected Carol, got %q", p.ReceiverName)
	}
	if p.ReceiverLook == nil {
		t.Fatalf("expected non-nil ReceiverLook")
	}
	if p.ReceiverLook.Gender != 1 {
		t.Errorf("receiverLook.gender: expected 1, got %d", p.ReceiverLook.Gender)
	}
	if p.ReceiverLook.Equips[-1] != 1002141 {
		t.Errorf("receiverLook.equips[-1]: expected 1002141, got %d", p.ReceiverLook.Equips[-1])
	}
	if p.ReceiverLook.MaskedEquips[-101] != 1002999 {
		t.Errorf("receiverLook.maskedEquips[-101]: expected 1002999, got %d", p.ReceiverLook.MaskedEquips[-101])
	}
	if p.ReceiverLook.Pets[1] != 5000000 {
		t.Errorf("receiverLook.pets[1]: expected 5000000, got %d", p.ReceiverLook.Pets[1])
	}
	if p.SenderLook.Equips[-5] != 1040002 {
		t.Errorf("senderLook.equips[-5]: expected 1040002, got %d", p.SenderLook.Equips[-5])
	}
}

func TestCreateNoteStepUnmarshal(t *testing.T) {
	in := Step[any]{
		StepId: "create_note",
		Status: Pending,
		Action: CreateNote,
		Payload: CreateNotePayload{
			SenderId:   100,
			ReceiverId: 200,
			Message:    "hello",
			Flag:       1,
			GiftNote:   true,
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Step[any]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	p, ok := out.Payload.(CreateNotePayload)
	if !ok {
		t.Fatalf("payload type: got %T, want CreateNotePayload", out.Payload)
	}
	if p.SenderId != 100 || p.ReceiverId != 200 || p.Message != "hello" || p.Flag != 1 || !p.GiftNote {
		t.Errorf("payload round-trip mismatch: %+v", p)
	}
}

// TestUnmarshalTradeSettlement pins that a trade_settlement step round-trips
// through the shared lib's payload unmarshaller into the CONCRETE payload type.
// Forgetting the switch case in Step[T].UnmarshalJSON is silent, not loud: the
// action falls through to the generic `default:` arm (unmarshal.go:600-608),
// which decodes into map[string]any and assigns it via any(payload).(T) — an
// assertion that always succeeds because Saga.Steps is []Step[any]. So the step
// still unmarshals without error and still carries the field values, just
// untyped. This test catches that only because it asserts the concrete
// TradeSettlementPayload type: a map[string]any fails that assertion.
func TestUnmarshalTradeSettlement(t *testing.T) {
	raw := []byte(`{
	  "transactionId": "11111111-1111-1111-1111-111111111111",
	  "sagaType": "trade_transaction",
	  "initiatedBy": "atlas-trades",
	  "steps": [{
	    "stepId": "trade_settlement",
	    "status": "pending",
	    "action": "trade_settlement",
	    "payload": {
	      "transactionId": "11111111-1111-1111-1111-111111111111",
	      "worldId": 1,
	      "channelId": 1,
	      "roomType": 3,
	      "sides": [
	        {"characterId": 100, "mesoStaged": 10000000, "mesoTax": 400000, "mesoDelivered": 9600000,
	         "items": [{"inventoryType": 2, "sourceSlot": 1, "assetId": 55, "templateId": 2000000, "quantity": 5}]},
	        {"characterId": 200, "mesoStaged": 0, "mesoTax": 0, "mesoDelivered": 0, "items": []}
	      ]
	    }
	  }]
	}`)

	var s Saga
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	p, ok := s.Steps[0].Payload.(TradeSettlementPayload)
	if !ok {
		t.Fatalf("payload type: got %T, want TradeSettlementPayload", s.Steps[0].Payload)
	}
	if p.Sides[0].MesoDelivered != 9_600_000 {
		t.Errorf("side 0 mesoDelivered: got %d, want 9600000", p.Sides[0].MesoDelivered)
	}
	if len(p.Sides[0].Items) != 1 || p.Sides[0].Items[0].AssetId != 55 {
		t.Errorf("side 0 items: got %+v", p.Sides[0].Items)
	}
}

// TestUnmarshalAcceptToTradeStep pins the task-205 escrow-at-staging custody
// accept (design §5A.2). The failure this guards against is a payload
// registered in unmarshal.go under the wrong action string: the step then
// decodes into the zero value of some other payload type and the escrow row is
// written with an empty snapshot.
func TestUnmarshalAcceptToTradeStep(t *testing.T) {
	raw := `{
		"stepId": "accept_to_trade-1",
		"status": "pending",
		"action": "accept_to_trade",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"escrowId": "22222222-2222-2222-2222-222222222222",
			"roomId": "33333333-3333-3333-3333-333333333333",
			"ownerId": 100,
			"tradeSlot": 3,
			"sourceInventoryType": 2,
			"assetId": 55,
			"snapshot": {
				"slot": 7,
				"templateId": 2000000,
				"quantity": 42,
				"strength": 11,
				"flag": 8,
				"owner": "Chronicle",
				"cashId": 4815162342,
				"petId": 909,
				"expiration": "2031-04-05T06:07:08Z"
			}
		},
		"createdAt": "2026-08-10T00:00:00Z",
		"updatedAt": "2026-08-10T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AcceptToTrade {
		t.Fatalf("expected action AcceptToTrade, got %q", step.Action)
	}
	p, ok := step.Payload.(AcceptToTradePayload)
	if !ok {
		t.Fatalf("expected AcceptToTradePayload, got %T", step.Payload)
	}
	if p.OwnerId != 100 {
		t.Errorf("ownerId: expected 100, got %d", p.OwnerId)
	}
	if p.TradeSlot != 3 {
		t.Errorf("tradeSlot: expected 3, got %d", p.TradeSlot)
	}
	if p.Snapshot.TemplateId != 2000000 {
		t.Errorf("templateId: expected 2000000, got %d", p.Snapshot.TemplateId)
	}
	if p.Snapshot.Quantity != 42 {
		t.Errorf("quantity: expected 42, got %d", p.Snapshot.Quantity)
	}
	if p.Snapshot.Strength != 11 {
		t.Errorf("strength: expected 11, got %d", p.Snapshot.Strength)
	}
	if p.Snapshot.Owner != "Chronicle" {
		t.Errorf("owner: expected Chronicle, got %q", p.Snapshot.Owner)
	}
	// The cash serial, the pet id and the expiry decode too. They are asserted
	// because a step is REHYDRATED from this JSON after a restart: a tag that
	// stopped decoding would resume the saga with an item stripped of its
	// identity, and the re-grant would silently hand back a lesser item.
	if p.Snapshot.CashId != 4815162342 {
		t.Errorf("cashId: expected 4815162342, got %d", p.Snapshot.CashId)
	}
	if p.Snapshot.PetId != 909 {
		t.Errorf("petId: expected 909, got %d", p.Snapshot.PetId)
	}
	if p.Snapshot.Expiration.IsZero() {
		t.Error("expiration: expected the encoded timestamp, got the zero time")
	}
}

// TestUnmarshalReleaseFromTradeStep pins the custody release. Like
// ReleaseFromMtsHoldingPayload it carries only the row id — the escrow row
// holds everything else, so a release cannot disagree with the accept.
func TestUnmarshalReleaseFromTradeStep(t *testing.T) {
	raw := `{
		"stepId": "release_from_trade-1",
		"status": "pending",
		"action": "release_from_trade",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"escrowId": "22222222-2222-2222-2222-222222222222"
		},
		"createdAt": "2026-08-10T00:00:00Z",
		"updatedAt": "2026-08-10T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != ReleaseFromTrade {
		t.Fatalf("expected action ReleaseFromTrade, got %q", step.Action)
	}
	p, ok := step.Payload.(ReleaseFromTradePayload)
	if !ok {
		t.Fatalf("expected ReleaseFromTradePayload, got %T", step.Payload)
	}
	if p.EscrowId.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("escrowId: got %s", p.EscrowId)
	}
}

// TestUnmarshalTransferToTradeStep pins the composite atlas-trades submits.
func TestUnmarshalTransferToTradeStep(t *testing.T) {
	raw := `{
		"stepId": "transfer_to_trade-1",
		"status": "pending",
		"action": "transfer_to_trade",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"escrowId": "22222222-2222-2222-2222-222222222222",
			"roomId": "33333333-3333-3333-3333-333333333333",
			"characterId": 100,
			"tradeSlot": 1,
			"sourceInventoryType": 2,
			"sourceSlot": 7,
			"assetId": 55,
			"quantity": 3
		},
		"createdAt": "2026-08-10T00:00:00Z",
		"updatedAt": "2026-08-10T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != TransferToTrade {
		t.Fatalf("expected action TransferToTrade, got %q", step.Action)
	}
	p, ok := step.Payload.(TransferToTradePayload)
	if !ok {
		t.Fatalf("expected TransferToTradePayload, got %T", step.Payload)
	}
	if p.AssetId != 55 {
		t.Errorf("assetId: expected 55, got %d", p.AssetId)
	}
	if p.Quantity != 3 {
		t.Errorf("quantity: expected 3, got %d", p.Quantity)
	}
}

func TestUnmarshalStep_OpenNpcShop(t *testing.T) {
	raw := []byte(`{
		"stepId": "open_npc_shop",
		"status": "pending",
		"action": "open_npc_shop",
		"payload": {"characterId": 1234, "worldId": 0, "channelId": 1, "npcTemplateId": 9090000}
	}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(OpenNpcShopPayload)
	if !ok {
		t.Fatalf("payload type = %T, want OpenNpcShopPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.NpcTemplateId != 9090000 || p.ChannelId != channel.Id(1) {
		t.Errorf("payload = %+v", p)
	}
}

func TestUnmarshalExtendAssetExpirationStep(t *testing.T) {
	data := []byte(`{"stepId":"extend_asset_expiration","status":"pending","action":"extend_asset_expiration","payload":{"characterId":12345,"inventoryType":1,"slot":-11,"expiration":"2026-09-12T00:00:00Z","extenderTemplateId":5500001}}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(ExtendAssetExpirationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ExtendAssetExpirationPayload", s.Payload)
	}
	if p.CharacterId != 12345 {
		t.Errorf("CharacterId = %d, want 12345", p.CharacterId)
	}
	if p.InventoryType != 1 {
		t.Errorf("InventoryType = %d, want 1", p.InventoryType)
	}
	if p.Slot != -11 {
		t.Errorf("Slot = %d, want -11", p.Slot)
	}
	if p.ExtenderTemplateId != 5500001 {
		t.Errorf("ExtenderTemplateId = %d, want 5500001", p.ExtenderTemplateId)
	}
	want := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	if !p.Expiration.Equal(want) {
		t.Errorf("Expiration = %v, want %v", p.Expiration, want)
	}
}

func TestExpirationExtenderUseSagaTypeValue(t *testing.T) {
	if ExpirationExtenderUse != Type("expiration_extender_use") {
		t.Fatalf("ExpirationExtenderUse = %q, want %q", ExpirationExtenderUse, "expiration_extender_use")
	}
}

func TestPetReviveSagaTypeValue(t *testing.T) {
	if PetRevive != Type("pet_revive") {
		t.Fatalf("PetRevive = %q, want %q", PetRevive, "pet_revive")
	}
}

func TestRevivePetActionValue(t *testing.T) {
	if RevivePet != Action("revive_pet") {
		t.Fatalf("RevivePet = %q, want %q", RevivePet, "revive_pet")
	}
}

func TestUnmarshalRevivePetPayload(t *testing.T) {
	raw := []byte(`{"stepId":"revive_pet","status":"pending","action":"revive_pet",` +
		`"payload":{"characterId":42,"petId":7,"sourceTemplateId":5180000}}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(RevivePetPayload)
	if !ok {
		t.Fatalf("payload type = %T, want RevivePetPayload", s.Payload)
	}
	want := RevivePetPayload{CharacterId: 42, PetId: 7, SourceTemplateId: 5180000}
	if p != want {
		t.Fatalf("payload = %+v, want %+v", p, want)
	}
}

func TestUnmarshalStartItemConversationPayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "start_item_conversation",
		"status": "pending",
		"action": "start_item_conversation",
		"payload": {
			"characterId": 1234,
			"accountId": 77,
			"itemId": 2430008,
			"npcTemplateId": 2084002,
			"slot": 5,
			"worldId": 0,
			"channelId": 1,
			"mapId": 100000000,
			"instance": "00000000-0000-0000-0000-000000000000"
		}
	}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(StartItemConversationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want StartItemConversationPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.ItemId != 2430008 || p.NpcTemplateId != 2084002 || p.Slot != 5 || p.AccountId != 77 {
		t.Errorf("payload round-trip: %+v", p)
	}
}

func TestUnmarshalStartNpcConversationPayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "start_npc_conversation",
		"status": "pending",
		"action": "start_npc_conversation",
		"payload": {
			"characterId": 1234,
			"accountId": 77,
			"npcTemplateId": 9090002,
			"worldId": 0,
			"channelId": 1,
			"mapId": 100000000,
			"instance": "00000000-0000-0000-0000-000000000000"
		}
	}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(StartNpcConversationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want StartNpcConversationPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.NpcTemplateId != 9090002 || p.AccountId != 77 {
		t.Errorf("payload round-trip: %+v", p)
	}
}

func TestUnmarshalPlayJukeboxStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"play_jukebox","payload":{"worldId":0,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","itemId":5100000,"playerName":"Chronicle","durationMs":45000},"createdAt":"2026-08-21T00:00:00Z","updatedAt":"2026-08-21T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(PlayJukeboxPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.ItemId != 5100000 || p.PlayerName != "Chronicle" || p.DurationMs != 45000 {
		t.Fatalf("payload = %+v", p)
	}
	if p.WorldId != 0 || p.ChannelId != 1 || p.MapId != 100000000 {
		t.Fatalf("field coordinates = %+v", p)
	}
}

func TestUnmarshalMoveEnvironmentStep(t *testing.T) {
	data := []byte(`{
		"stepId": "move-environment-gate01",
		"status": "pending",
		"action": "move_environment",
		"payload": {
			"worldId": 0,
			"channelId": 1,
			"mapId": 910010000,
			"instance": "00000000-0000-0000-0000-000000000000",
			"kind": "OBSTACLE",
			"name": "gate01",
			"state": 3
		}
	}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	if s.Action != MoveEnvironment {
		t.Fatalf("action = %v, want %v", s.Action, MoveEnvironment)
	}
	p, ok := s.Payload.(MoveEnvironmentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want MoveEnvironmentPayload", s.Payload)
	}
	if p.WorldId != world.Id(0) || p.ChannelId != channel.Id(1) || p.MapId != _map.Id(910010000) {
		t.Fatalf("field coordinates = %+v", p)
	}
	if p.Instance != uuid.Nil {
		t.Fatalf("instance = %v, want uuid.Nil", p.Instance)
	}
	if p.Kind != field.ObjectKindObstacle {
		t.Fatalf("kind = %v, want %v", p.Kind, field.ObjectKindObstacle)
	}
	if p.Name != "gate01" {
		t.Fatalf("name = %v, want gate01", p.Name)
	}
	if p.State != uint32(3) {
		t.Fatalf("state = %v, want 3", p.State)
	}
}

func TestUnmarshalResetEnvironmentStep(t *testing.T) {
	data := []byte(`{
		"stepId": "reset-environment-1",
		"status": "pending",
		"action": "reset_environment",
		"payload": {
			"worldId": 0,
			"channelId": 1,
			"mapId": 910010000,
			"instance": "00000000-0000-0000-0000-000000000000"
		}
	}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	if s.Action != ResetEnvironment {
		t.Fatalf("action = %v, want %v", s.Action, ResetEnvironment)
	}
	p, ok := s.Payload.(ResetEnvironmentPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ResetEnvironmentPayload", s.Payload)
	}
	if p.MapId != _map.Id(910010000) {
		t.Fatalf("mapId = %v, want 910010000", p.MapId)
	}
}

func TestUnmarshalSetBackEffectStep(t *testing.T) {
	data := []byte(`{"stepId":"back-effect-step","status":"pending","action":"set_back_effect","payload":{"worldId":0,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","effect":0,"fieldId":100000000,"pageId":1,"duration":1000}}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(SetBackEffectPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.MapId != 100000000 || p.Effect != 0 || p.PageId != 1 || p.Duration != 1000 {
		t.Fatalf("payload = %+v", p)
	}
}

func TestUnmarshalClearBackEffectStep(t *testing.T) {
	data := []byte(`{"stepId":"back-effect-step","status":"pending","action":"clear_back_effect","payload":{"worldId":0,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000"}}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(ClearBackEffectPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.MapId != 100000000 {
		t.Fatalf("payload = %+v", p)
	}
}
