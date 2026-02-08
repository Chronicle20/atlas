package storage_test

import (
	"atlas-channel/storage"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

func TestStorageRestModelUnmarshalWithAssets(t *testing.T) {
	jsonBody := []byte(`{
		"data": {
			"type": "storages",
			"id": "0eaf71e9-2ba7-4443-96bb-886f7dc8213c",
			"attributes": {
				"world_id": 0,
				"account_id": 1,
				"capacity": 4,
				"mesos": 485
			},
			"relationships": {
				"assets": {
					"data": [
						{"type": "storage_assets", "id": "8"}
					]
				}
			}
		},
		"included": [
			{
				"type": "storage_assets",
				"id": "8",
				"attributes": {
					"slot": 0,
					"templateId": 1322005,
					"expiration": "0001-01-01T00:00:00Z",
					"quantity": 1,
					"ownerId": 0,
					"flag": 0,
					"rechargeable": 0,
					"strength": 0,
					"dexterity": 0,
					"intelligence": 0,
					"luck": 0,
					"hp": 0,
					"mp": 0,
					"weaponAttack": 19,
					"magicAttack": 0,
					"weaponDefense": 0,
					"magicDefense": 0,
					"accuracy": 0,
					"avoidability": 0,
					"hands": 0,
					"speed": 0,
					"jump": 0,
					"slots": 7,
					"locked": false,
					"spikes": false,
					"karmaUsed": false,
					"cold": false,
					"canBeTraded": false,
					"levelType": 0,
					"level": 0,
					"experience": 0,
					"hammersApplied": 0,
					"cashId": "0",
					"commodityId": 0,
					"purchaseBy": 0,
					"petId": 0
				}
			}
		]
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	if restModel.GetID() != "0eaf71e9-2ba7-4443-96bb-886f7dc8213c" {
		t.Errorf("Storage ID mismatch: expected %s, got %s", "0eaf71e9-2ba7-4443-96bb-886f7dc8213c", restModel.GetID())
	}
	if restModel.WorldId != 0 {
		t.Errorf("WorldId mismatch: expected 0, got %d", restModel.WorldId)
	}
	if restModel.AccountId != 1 {
		t.Errorf("AccountId mismatch: expected 1, got %d", restModel.AccountId)
	}
	if restModel.Capacity != 4 {
		t.Errorf("Capacity mismatch: expected 4, got %d", restModel.Capacity)
	}
	if restModel.Mesos != 485 {
		t.Errorf("Mesos mismatch: expected 485, got %d", restModel.Mesos)
	}

	if len(restModel.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(restModel.Assets))
	}

	asset := restModel.Assets[0]

	if asset.Id != 8 {
		t.Errorf("Asset ID mismatch: expected 8, got %d", asset.Id)
	}
	if asset.Slot != 0 {
		t.Errorf("Asset Slot mismatch: expected 0, got %d", asset.Slot)
	}
	if asset.TemplateId != 1322005 {
		t.Errorf("Asset TemplateId mismatch: expected 1322005, got %d", asset.TemplateId)
	}
	if asset.WeaponAttack != 19 {
		t.Errorf("WeaponAttack mismatch: expected 19, got %d", asset.WeaponAttack)
	}
	if asset.Slots != 7 {
		t.Errorf("Slots mismatch: expected 7, got %d", asset.Slots)
	}
}

func TestStorageRestModelUnmarshalMultipleAssets(t *testing.T) {
	jsonBody := []byte(`{
		"data": {
			"type": "storages",
			"id": "test-storage-id",
			"attributes": {
				"world_id": 1,
				"account_id": 100,
				"capacity": 10,
				"mesos": 5000
			},
			"relationships": {
				"assets": {
					"data": [
						{"type": "storage_assets", "id": "1"},
						{"type": "storage_assets", "id": "2"},
						{"type": "storage_assets", "id": "3"}
					]
				}
			}
		},
		"included": [
			{
				"type": "storage_assets",
				"id": "1",
				"attributes": {
					"slot": 0,
					"templateId": 1322005,
					"expiration": "0001-01-01T00:00:00Z",
					"quantity": 1,
					"ownerId": 0,
					"flag": 0,
					"rechargeable": 0,
					"strength": 0,
					"dexterity": 0,
					"intelligence": 0,
					"luck": 0,
					"hp": 0,
					"mp": 0,
					"weaponAttack": 10,
					"magicAttack": 0,
					"weaponDefense": 0,
					"magicDefense": 0,
					"accuracy": 0,
					"avoidability": 0,
					"hands": 0,
					"speed": 0,
					"jump": 0,
					"slots": 5,
					"locked": false,
					"spikes": false,
					"karmaUsed": false,
					"cold": false,
					"canBeTraded": false,
					"levelType": 0,
					"level": 0,
					"experience": 0,
					"hammersApplied": 0,
					"cashId": "0",
					"commodityId": 0,
					"purchaseBy": 0,
					"petId": 0
				}
			},
			{
				"type": "storage_assets",
				"id": "2",
				"attributes": {
					"slot": 1,
					"templateId": 2000000,
					"expiration": "0001-01-01T00:00:00Z",
					"quantity": 50,
					"ownerId": 0,
					"flag": 0,
					"rechargeable": 0,
					"strength": 0,
					"dexterity": 0,
					"intelligence": 0,
					"luck": 0,
					"hp": 0,
					"mp": 0,
					"weaponAttack": 0,
					"magicAttack": 0,
					"weaponDefense": 0,
					"magicDefense": 0,
					"accuracy": 0,
					"avoidability": 0,
					"hands": 0,
					"speed": 0,
					"jump": 0,
					"slots": 0,
					"locked": false,
					"spikes": false,
					"karmaUsed": false,
					"cold": false,
					"canBeTraded": false,
					"levelType": 0,
					"level": 0,
					"experience": 0,
					"hammersApplied": 0,
					"cashId": "0",
					"commodityId": 0,
					"purchaseBy": 0,
					"petId": 0
				}
			},
			{
				"type": "storage_assets",
				"id": "3",
				"attributes": {
					"slot": 2,
					"templateId": 4000000,
					"expiration": "0001-01-01T00:00:00Z",
					"quantity": 100,
					"ownerId": 456,
					"flag": 0,
					"rechargeable": 0,
					"strength": 0,
					"dexterity": 0,
					"intelligence": 0,
					"luck": 0,
					"hp": 0,
					"mp": 0,
					"weaponAttack": 0,
					"magicAttack": 0,
					"weaponDefense": 0,
					"magicDefense": 0,
					"accuracy": 0,
					"avoidability": 0,
					"hands": 0,
					"speed": 0,
					"jump": 0,
					"slots": 0,
					"locked": false,
					"spikes": false,
					"karmaUsed": false,
					"cold": false,
					"canBeTraded": false,
					"levelType": 0,
					"level": 0,
					"experience": 0,
					"hammersApplied": 0,
					"cashId": "0",
					"commodityId": 0,
					"purchaseBy": 0,
					"petId": 0
				}
			}
		]
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	if len(restModel.Assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(restModel.Assets))
	}

	assetIdMap := make(map[uint32]storage.AssetRestModel)
	for _, a := range restModel.Assets {
		if a.Id == 0 {
			t.Errorf("Asset ID should not be 0")
		}
		assetIdMap[a.Id] = a
	}

	// Verify asset 1 (equipable)
	if a, ok := assetIdMap[1]; ok {
		if a.TemplateId != 1322005 {
			t.Errorf("Asset 1 TemplateId mismatch: expected 1322005, got %d", a.TemplateId)
		}
		if a.WeaponAttack != 10 {
			t.Errorf("Asset 1 WeaponAttack mismatch: expected 10, got %d", a.WeaponAttack)
		}
		if a.Slots != 5 {
			t.Errorf("Asset 1 Slots mismatch: expected 5, got %d", a.Slots)
		}
	} else {
		t.Error("Missing asset with ID 1")
	}

	// Verify asset 2 (consumable)
	if a, ok := assetIdMap[2]; ok {
		if a.TemplateId != 2000000 {
			t.Errorf("Asset 2 TemplateId mismatch: expected 2000000, got %d", a.TemplateId)
		}
		if a.Quantity != 50 {
			t.Errorf("Asset 2 Quantity mismatch: expected 50, got %d", a.Quantity)
		}
	} else {
		t.Error("Missing asset with ID 2")
	}

	// Verify asset 3 (etc)
	if a, ok := assetIdMap[3]; ok {
		if a.TemplateId != 4000000 {
			t.Errorf("Asset 3 TemplateId mismatch: expected 4000000, got %d", a.TemplateId)
		}
		if a.Quantity != 100 {
			t.Errorf("Asset 3 Quantity mismatch: expected 100, got %d", a.Quantity)
		}
		if a.OwnerId != 456 {
			t.Errorf("Asset 3 OwnerId mismatch: expected 456, got %d", a.OwnerId)
		}
	} else {
		t.Error("Missing asset with ID 3")
	}
}

func TestStorageRestModelUnmarshalEmptyAssets(t *testing.T) {
	jsonBody := []byte(`{
		"data": {
			"type": "storages",
			"id": "empty-storage-id",
			"attributes": {
				"world_id": 0,
				"account_id": 1,
				"capacity": 8,
				"mesos": 0
			},
			"relationships": {
				"assets": {
					"data": []
				}
			}
		}
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	if restModel.Capacity != 8 {
		t.Errorf("Capacity mismatch: expected 8, got %d", restModel.Capacity)
	}

	if len(restModel.Assets) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(restModel.Assets))
	}
}

func TestAssetRestModelExpiration(t *testing.T) {
	jsonBody := []byte(`{
		"data": {
			"type": "storages",
			"id": "test-exp-id",
			"attributes": {
				"world_id": 0,
				"account_id": 1,
				"capacity": 4,
				"mesos": 0
			},
			"relationships": {
				"assets": {
					"data": [
						{"type": "storage_assets", "id": "1"}
					]
				}
			}
		},
		"included": [
			{
				"type": "storage_assets",
				"id": "1",
				"attributes": {
					"slot": 0,
					"templateId": 1322005,
					"expiration": "2025-12-31T23:59:59Z",
					"quantity": 1,
					"ownerId": 0,
					"flag": 0,
					"rechargeable": 0,
					"strength": 0,
					"dexterity": 0,
					"intelligence": 0,
					"luck": 0,
					"hp": 0,
					"mp": 0,
					"weaponAttack": 0,
					"magicAttack": 0,
					"weaponDefense": 0,
					"magicDefense": 0,
					"accuracy": 0,
					"avoidability": 0,
					"hands": 0,
					"speed": 0,
					"jump": 0,
					"slots": 0,
					"locked": false,
					"spikes": false,
					"karmaUsed": false,
					"cold": false,
					"canBeTraded": false,
					"levelType": 0,
					"level": 0,
					"experience": 0,
					"hammersApplied": 0,
					"cashId": "0",
					"commodityId": 0,
					"purchaseBy": 0,
					"petId": 0
				}
			}
		]
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	if len(restModel.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(restModel.Assets))
	}

	expectedTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	if !restModel.Assets[0].Expiration.Equal(expectedTime) {
		t.Errorf("Expiration mismatch: expected %v, got %v", expectedTime, restModel.Assets[0].Expiration)
	}
}
