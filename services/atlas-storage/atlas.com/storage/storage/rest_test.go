package storage_test

import (
	"atlas-storage/asset"
	"atlas-storage/storage"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func TestStorageWithAssetsMarshalUnmarshal(t *testing.T) {
	storageId := uuid.New()

	// Create an equipable asset using the flat builder
	equipableAsset := asset.NewBuilder(storageId, 1322005).
		SetId(8).
		SetSlot(0).
		SetExpiration(time.Time{}).
		SetStrength(15).
		SetDexterity(10).
		SetSlots(7).
		Build()

	// Create a storage model with the asset
	storageModel := storage.NewModelBuilder().
		SetId(storageId).
		SetWorldId(0).
		SetAccountId(1).
		SetCapacity(4).
		SetMesos(485).
		SetAssets([]asset.Model{equipableAsset}).
		MustBuild()

	// Transform to REST model
	restModel, err := storage.Transform(storageModel)
	if err != nil {
		t.Fatalf("Failed to transform storage model: %v", err)
	}

	// Marshal to JSON:API format
	rr := httptest.NewRecorder()
	server.MarshalResponse[storage.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(restModel)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model, status code: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	// Verify the JSON contains the expected structure with relationships
	bodyStr := string(body)
	t.Logf("Marshaled JSON:\n%s", bodyStr)

	// Unmarshal back
	outputRestModel := storage.RestModel{}
	err = jsonapi.Unmarshal(body, &outputRestModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}

	// Verify storage attributes
	if outputRestModel.GetID() != storageModel.Id().String() {
		t.Errorf("Storage ID mismatch: expected %s, got %s", storageModel.Id().String(), outputRestModel.GetID())
	}
	if outputRestModel.WorldId != storageModel.WorldId() {
		t.Errorf("WorldId mismatch: expected %d, got %d", storageModel.WorldId(), outputRestModel.WorldId)
	}
	if outputRestModel.AccountId != storageModel.AccountId() {
		t.Errorf("AccountId mismatch: expected %d, got %d", storageModel.AccountId(), outputRestModel.AccountId)
	}
	if outputRestModel.Capacity != storageModel.Capacity() {
		t.Errorf("Capacity mismatch: expected %d, got %d", storageModel.Capacity(), outputRestModel.Capacity)
	}
	if outputRestModel.Mesos != storageModel.Mesos() {
		t.Errorf("Mesos mismatch: expected %d, got %d", storageModel.Mesos(), outputRestModel.Mesos)
	}

	// Verify assets are included
	if len(outputRestModel.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(outputRestModel.Assets))
	}

	// Verify asset attributes
	outputAsset := outputRestModel.Assets[0]
	if outputAsset.Id != equipableAsset.Id() {
		t.Errorf("Asset ID mismatch: expected %d, got %d", equipableAsset.Id(), outputAsset.Id)
	}
	if outputAsset.Slot != equipableAsset.Slot() {
		t.Errorf("Asset Slot mismatch: expected %d, got %d", equipableAsset.Slot(), outputAsset.Slot)
	}
	if outputAsset.TemplateId != equipableAsset.TemplateId() {
		t.Errorf("Asset TemplateId mismatch: expected %d, got %d", equipableAsset.TemplateId(), outputAsset.TemplateId)
	}
	if outputAsset.Strength != equipableAsset.Strength() {
		t.Errorf("Asset Strength mismatch: expected %d, got %d", equipableAsset.Strength(), outputAsset.Strength)
	}
}

func TestStorageEmptyAssetsMarshalUnmarshal(t *testing.T) {
	storageId := uuid.New()

	// Create a storage model without assets
	storageModel := storage.NewModelBuilder().
		SetId(storageId).
		SetWorldId(1).
		SetAccountId(2).
		SetCapacity(8).
		SetMesos(1000).
		SetAssets([]asset.Model{}).
		MustBuild()

	// Transform to REST model
	restModel, err := storage.Transform(storageModel)
	if err != nil {
		t.Fatalf("Failed to transform storage model: %v", err)
	}

	// Marshal to JSON:API format
	rr := httptest.NewRecorder()
	server.MarshalResponse[storage.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(restModel)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model, status code: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	// Unmarshal back
	outputRestModel := storage.RestModel{}
	err = jsonapi.Unmarshal(body, &outputRestModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}

	// Verify storage attributes
	if outputRestModel.GetID() != storageModel.Id().String() {
		t.Errorf("Storage ID mismatch: expected %s, got %s", storageModel.Id().String(), outputRestModel.GetID())
	}
	if outputRestModel.Capacity != storageModel.Capacity() {
		t.Errorf("Capacity mismatch: expected %d, got %d", storageModel.Capacity(), outputRestModel.Capacity)
	}
	if outputRestModel.Mesos != storageModel.Mesos() {
		t.Errorf("Mesos mismatch: expected %d, got %d", storageModel.Mesos(), outputRestModel.Mesos)
	}

	// Verify no assets
	if len(outputRestModel.Assets) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(outputRestModel.Assets))
	}
}

func TestStorageMultipleAssetsMarshalUnmarshal(t *testing.T) {
	storageId := uuid.New()

	// Create multiple assets of different types
	equipableAsset := asset.NewBuilder(storageId, 1322005).
		SetId(1).
		SetSlot(0).
		SetExpiration(time.Time{}).
		SetStrength(15).
		Build()

	consumableAsset := asset.NewBuilder(storageId, 2000000).
		SetId(2).
		SetSlot(1).
		SetExpiration(time.Time{}).
		SetQuantity(50).
		Build()

	etcAsset := asset.NewBuilder(storageId, 4000000).
		SetId(3).
		SetSlot(2).
		SetExpiration(time.Time{}).
		SetQuantity(99).
		Build()

	// Create a storage model with multiple assets
	storageModel := storage.NewModelBuilder().
		SetId(storageId).
		SetWorldId(0).
		SetAccountId(1).
		SetCapacity(10).
		SetMesos(500).
		SetAssets([]asset.Model{equipableAsset, consumableAsset, etcAsset}).
		MustBuild()

	// Transform to REST model
	restModel, err := storage.Transform(storageModel)
	if err != nil {
		t.Fatalf("Failed to transform storage model: %v", err)
	}

	// Marshal to JSON:API format
	rr := httptest.NewRecorder()
	server.MarshalResponse[storage.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(restModel)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model, status code: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	// Unmarshal back
	outputRestModel := storage.RestModel{}
	err = jsonapi.Unmarshal(body, &outputRestModel)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}

	// Verify correct number of assets
	if len(outputRestModel.Assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(outputRestModel.Assets))
	}

	// Verify each asset has correct ID
	assetIdMap := make(map[uint32]bool)
	for _, a := range outputRestModel.Assets {
		assetIdMap[a.Id] = true
	}

	if !assetIdMap[1] {
		t.Error("Missing asset with ID 1")
	}
	if !assetIdMap[2] {
		t.Error("Missing asset with ID 2")
	}
	if !assetIdMap[3] {
		t.Error("Missing asset with ID 3")
	}
}
