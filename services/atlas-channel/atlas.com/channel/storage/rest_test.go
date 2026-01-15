package storage_test

import (
	"atlas-channel/storage"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

func TestStorageRestModelUnmarshalWithAssets(t *testing.T) {
	// This JSON represents the expected JSON:API format from the storage service
	// with assets as included resources
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
					"referenceId": 100,
					"referenceType": "equipable",
					"referenceData": {
						"ownerId": 0,
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
						"hammersApplied": 0
					}
				}
			}
		]
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	// Verify storage attributes
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

	// Verify assets are properly parsed
	if len(restModel.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(restModel.Assets))
	}

	asset := restModel.Assets[0]

	// CRITICAL: Verify asset ID is correctly parsed (this was the bug)
	if asset.Id != 8 {
		t.Errorf("Asset ID mismatch: expected 8, got %d", asset.Id)
	}
	if asset.Slot != 0 {
		t.Errorf("Asset Slot mismatch: expected 0, got %d", asset.Slot)
	}
	if asset.TemplateId != 1322005 {
		t.Errorf("Asset TemplateId mismatch: expected 1322005, got %d", asset.TemplateId)
	}
	if asset.ReferenceId != 100 {
		t.Errorf("Asset ReferenceId mismatch: expected 100, got %d", asset.ReferenceId)
	}
	if asset.ReferenceType != "equipable" {
		t.Errorf("Asset ReferenceType mismatch: expected equipable, got %s", asset.ReferenceType)
	}

	// Verify reference data was parsed correctly
	if asset.ReferenceData == nil {
		t.Fatal("Expected ReferenceData to be non-nil")
	}
	eqData, ok := asset.ReferenceData.(storage.EquipableRestData)
	if !ok {
		t.Fatalf("Expected ReferenceData to be EquipableRestData, got %T", asset.ReferenceData)
	}
	if eqData.WeaponAttack != 19 {
		t.Errorf("WeaponAttack mismatch: expected 19, got %d", eqData.WeaponAttack)
	}
	if eqData.Slots != 7 {
		t.Errorf("Slots mismatch: expected 7, got %d", eqData.Slots)
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
					"referenceId": 100,
					"referenceType": "equipable",
					"referenceData": {"ownerId": 0, "weaponAttack": 10, "slots": 5}
				}
			},
			{
				"type": "storage_assets",
				"id": "2",
				"attributes": {
					"slot": 1,
					"templateId": 2000000,
					"expiration": "0001-01-01T00:00:00Z",
					"referenceId": 101,
					"referenceType": "consumable",
					"referenceData": {"ownerId": 0, "quantity": 50, "flag": 0, "rechargeable": 0}
				}
			},
			{
				"type": "storage_assets",
				"id": "3",
				"attributes": {
					"slot": 2,
					"templateId": 4000000,
					"expiration": "0001-01-01T00:00:00Z",
					"referenceId": 102,
					"referenceType": "etc",
					"referenceData": {"ownerId": 0, "quantity": 100, "flag": 0}
				}
			}
		]
	}`)

	restModel := storage.StorageRestModel{}
	err := jsonapi.Unmarshal(jsonBody, &restModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal storage rest model: %v", err)
	}

	// Verify correct number of assets
	if len(restModel.Assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(restModel.Assets))
	}

	// Verify each asset has the correct ID (non-zero)
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
		if a.ReferenceType != "equipable" {
			t.Errorf("Asset 1 ReferenceType mismatch: expected equipable, got %s", a.ReferenceType)
		}
	} else {
		t.Error("Missing asset with ID 1")
	}

	// Verify asset 2 (consumable)
	if a, ok := assetIdMap[2]; ok {
		if a.TemplateId != 2000000 {
			t.Errorf("Asset 2 TemplateId mismatch: expected 2000000, got %d", a.TemplateId)
		}
		if a.ReferenceType != "consumable" {
			t.Errorf("Asset 2 ReferenceType mismatch: expected consumable, got %s", a.ReferenceType)
		}
	} else {
		t.Error("Missing asset with ID 2")
	}

	// Verify asset 3 (etc)
	if a, ok := assetIdMap[3]; ok {
		if a.TemplateId != 4000000 {
			t.Errorf("Asset 3 TemplateId mismatch: expected 4000000, got %d", a.TemplateId)
		}
		if a.ReferenceType != "etc" {
			t.Errorf("Asset 3 ReferenceType mismatch: expected etc, got %s", a.ReferenceType)
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

	// Verify storage attributes
	if restModel.Capacity != 8 {
		t.Errorf("Capacity mismatch: expected 8, got %d", restModel.Capacity)
	}

	// Verify no assets
	if len(restModel.Assets) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(restModel.Assets))
	}
}

func TestAssetRestModelUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name          string
		json          string
		expectedType  string
		checkRefData  func(t *testing.T, data interface{})
	}{
		{
			name: "equipable",
			json: `{
				"slot": 0,
				"templateId": 1322005,
				"expiration": "0001-01-01T00:00:00Z",
				"referenceId": 100,
				"referenceType": "equipable",
				"referenceData": {
					"ownerId": 123,
					"strength": 5,
					"dexterity": 3,
					"weaponAttack": 25,
					"slots": 7
				}
			}`,
			expectedType: "equipable",
			checkRefData: func(t *testing.T, data interface{}) {
				eq, ok := data.(storage.EquipableRestData)
				if !ok {
					t.Fatalf("Expected EquipableRestData, got %T", data)
				}
				if eq.OwnerId != 123 {
					t.Errorf("OwnerId mismatch: expected 123, got %d", eq.OwnerId)
				}
				if eq.Strength != 5 {
					t.Errorf("Strength mismatch: expected 5, got %d", eq.Strength)
				}
				if eq.WeaponAttack != 25 {
					t.Errorf("WeaponAttack mismatch: expected 25, got %d", eq.WeaponAttack)
				}
			},
		},
		{
			name: "consumable",
			json: `{
				"slot": 1,
				"templateId": 2000000,
				"expiration": "0001-01-01T00:00:00Z",
				"referenceId": 101,
				"referenceType": "consumable",
				"referenceData": {
					"ownerId": 0,
					"quantity": 50,
					"flag": 1,
					"rechargeable": 0
				}
			}`,
			expectedType: "consumable",
			checkRefData: func(t *testing.T, data interface{}) {
				c, ok := data.(storage.ConsumableRestData)
				if !ok {
					t.Fatalf("Expected ConsumableRestData, got %T", data)
				}
				if c.Quantity != 50 {
					t.Errorf("Quantity mismatch: expected 50, got %d", c.Quantity)
				}
				if c.Flag != 1 {
					t.Errorf("Flag mismatch: expected 1, got %d", c.Flag)
				}
			},
		},
		{
			name: "etc",
			json: `{
				"slot": 2,
				"templateId": 4000000,
				"expiration": "0001-01-01T00:00:00Z",
				"referenceId": 102,
				"referenceType": "etc",
				"referenceData": {
					"ownerId": 456,
					"quantity": 100,
					"flag": 0
				}
			}`,
			expectedType: "etc",
			checkRefData: func(t *testing.T, data interface{}) {
				e, ok := data.(storage.EtcRestData)
				if !ok {
					t.Fatalf("Expected EtcRestData, got %T", data)
				}
				if e.OwnerId != 456 {
					t.Errorf("OwnerId mismatch: expected 456, got %d", e.OwnerId)
				}
				if e.Quantity != 100 {
					t.Errorf("Quantity mismatch: expected 100, got %d", e.Quantity)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var asset storage.AssetRestModel
			err := asset.UnmarshalJSON([]byte(tc.json))
			if err != nil {
				t.Fatalf("Failed to unmarshal asset: %v", err)
			}

			if asset.ReferenceType != tc.expectedType {
				t.Errorf("ReferenceType mismatch: expected %s, got %s", tc.expectedType, asset.ReferenceType)
			}

			tc.checkRefData(t, asset.ReferenceData)
		})
	}
}

func TestAssetRestModelExpiration(t *testing.T) {
	jsonBody := `{
		"slot": 0,
		"templateId": 1322005,
		"expiration": "2025-12-31T23:59:59Z",
		"referenceId": 100,
		"referenceType": "equipable",
		"referenceData": {}
	}`

	var asset storage.AssetRestModel
	err := asset.UnmarshalJSON([]byte(jsonBody))
	if err != nil {
		t.Fatalf("Failed to unmarshal asset: %v", err)
	}

	expectedTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	if !asset.Expiration.Equal(expectedTime) {
		t.Errorf("Expiration mismatch: expected %v, got %v", expectedTime, asset.Expiration)
	}
}
