package field

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"testing"
)

func TestModelConstruction(t *testing.T) {
	// Test data
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)
	instance := uuid.New()

	// Test builder without instance
	builder := NewBuilder(worldId, channelId, mapId)
	model := builder.Build()

	if model.WorldId() != worldId {
		t.Errorf("Expected worldId to be %d, got %d", worldId, model.WorldId())
	}
	if model.ChannelId() != channelId {
		t.Errorf("Expected channelId to be %d, got %d", channelId, model.ChannelId())
	}
	if model.MapId() != mapId {
		t.Errorf("Expected mapId to be %d, got %d", mapId, model.MapId())
	}
	if model.Instance() != uuid.Nil {
		t.Errorf("Expected instance to be Nil, got %s", model.Instance())
	}

	// Test builder with instance
	modelWithInstance := builder.SetInstance(instance).Build()

	if modelWithInstance.WorldId() != worldId {
		t.Errorf("Expected worldId to be %d, got %d", worldId, modelWithInstance.WorldId())
	}
	if modelWithInstance.ChannelId() != channelId {
		t.Errorf("Expected channelId to be %d, got %d", channelId, modelWithInstance.ChannelId())
	}
	if modelWithInstance.MapId() != mapId {
		t.Errorf("Expected mapId to be %d, got %d", mapId, modelWithInstance.MapId())
	}
	if modelWithInstance.Instance() != instance {
		t.Errorf("Expected instance to be %s, got %s", instance, modelWithInstance.Instance())
	}
}

func TestIdGeneration(t *testing.T) {
	// Test data
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)
	instance := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	// Create model
	model := NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Generate ID
	id := model.Id()

	// Expected ID format: "1:2:300000000:00000000-0000-0000-0000-000000000000"
	expected := Id("1:2:300000000:00000000-0000-0000-0000-000000000000")
	if id != expected {
		t.Errorf("Expected ID to be %s, got %s", expected, id)
	}
}

func TestModelReconstruction(t *testing.T) {
	// Test data
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)
	instance := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	// Create original model
	originalModel := NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Generate ID
	id := originalModel.Id()

	// Reconstruct model from ID
	reconstructedModel, ok := FromId(id)
	if !ok {
		t.Fatalf("Failed to reconstruct model from ID: %s", id)
	}

	// Verify reconstructed model matches original
	if reconstructedModel.WorldId() != originalModel.WorldId() {
		t.Errorf("Expected worldId to be %d, got %d", originalModel.WorldId(), reconstructedModel.WorldId())
	}
	if reconstructedModel.ChannelId() != originalModel.ChannelId() {
		t.Errorf("Expected channelId to be %d, got %d", originalModel.ChannelId(), reconstructedModel.ChannelId())
	}
	if reconstructedModel.MapId() != originalModel.MapId() {
		t.Errorf("Expected mapId to be %d, got %d", originalModel.MapId(), reconstructedModel.MapId())
	}
	if reconstructedModel.Instance() != originalModel.Instance() {
		t.Errorf("Expected instance to be %s, got %s", originalModel.Instance(), reconstructedModel.Instance())
	}

	// Verify the ID of the reconstructed model matches the original ID
	if reconstructedModel.Id() != id {
		t.Errorf("Expected reconstructed model ID to be %s, got %s", id, reconstructedModel.Id())
	}
}

func TestInvalidIdReconstruction(t *testing.T) {
	// Test invalid ID format
	invalidId := Id("invalid:id:format")
	_, ok := FromId(invalidId)
	if ok {
		t.Errorf("Expected reconstruction to fail for invalid ID: %s", invalidId)
	}

	// Test incomplete ID
	incompleteId := Id("1:2:300000000")
	_, ok = FromId(incompleteId)
	if ok {
		t.Errorf("Expected reconstruction to fail for incomplete ID: %s", incompleteId)
	}

	// Test ID with invalid UUID
	invalidUuidId := Id("1:2:300000000:not-a-uuid")
	_, ok = FromId(invalidUuidId)
	if ok {
		t.Errorf("Expected reconstruction to fail for ID with invalid UUID: %s", invalidUuidId)
	}
}
