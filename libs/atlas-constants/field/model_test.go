package field

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
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

func TestModelJSONMarshal(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)
	instance := uuid.MustParse("12345678-1234-5678-1234-567812345678")

	model := NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal model: %v", err)
	}

	expected := `{"worldId":1,"channelId":2,"mapId":300000000,"instance":"12345678-1234-5678-1234-567812345678"}`
	if string(data) != expected {
		t.Errorf("Expected JSON to be %s, got %s", expected, string(data))
	}
}

func TestModelJSONUnmarshal(t *testing.T) {
	jsonData := `{"worldId":1,"channelId":2,"mapId":300000000,"instance":"12345678-1234-5678-1234-567812345678"}`

	var model Model
	err := json.Unmarshal([]byte(jsonData), &model)
	if err != nil {
		t.Fatalf("Failed to unmarshal model: %v", err)
	}

	if model.WorldId() != world.Id(1) {
		t.Errorf("Expected worldId to be 1, got %d", model.WorldId())
	}
	if model.ChannelId() != channel.Id(2) {
		t.Errorf("Expected channelId to be 2, got %d", model.ChannelId())
	}
	if model.MapId() != _map.Id(300000000) {
		t.Errorf("Expected mapId to be 300000000, got %d", model.MapId())
	}
	expectedInstance := uuid.MustParse("12345678-1234-5678-1234-567812345678")
	if model.Instance() != expectedInstance {
		t.Errorf("Expected instance to be %s, got %s", expectedInstance, model.Instance())
	}
}

func TestModelJSONRoundTrip(t *testing.T) {
	worldId := world.Id(5)
	channelId := channel.Id(3)
	mapId := _map.Id(100000000)
	instance := uuid.New()

	original := NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal model: %v", err)
	}

	var decoded Model
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal model: %v", err)
	}

	if decoded.WorldId() != original.WorldId() {
		t.Errorf("WorldId mismatch: expected %d, got %d", original.WorldId(), decoded.WorldId())
	}
	if decoded.ChannelId() != original.ChannelId() {
		t.Errorf("ChannelId mismatch: expected %d, got %d", original.ChannelId(), decoded.ChannelId())
	}
	if decoded.MapId() != original.MapId() {
		t.Errorf("MapId mismatch: expected %d, got %d", original.MapId(), decoded.MapId())
	}
	if decoded.Instance() != original.Instance() {
		t.Errorf("Instance mismatch: expected %s, got %s", original.Instance(), decoded.Instance())
	}
}

func TestModelJSONWithNilInstance(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)

	model := NewBuilder(worldId, channelId, mapId).Build()

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal model: %v", err)
	}

	var decoded Model
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal model: %v", err)
	}

	if decoded.Instance() != uuid.Nil {
		t.Errorf("Expected instance to be Nil, got %s", decoded.Instance())
	}
}

func TestModelToDTO(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(300000000)
	instance := uuid.New()

	model := NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
	dto := model.ToDTO()

	if dto.WorldId != worldId {
		t.Errorf("Expected WorldId to be %d, got %d", worldId, dto.WorldId)
	}
	if dto.ChannelId != channelId {
		t.Errorf("Expected ChannelId to be %d, got %d", channelId, dto.ChannelId)
	}
	if dto.MapId != mapId {
		t.Errorf("Expected MapId to be %d, got %d", mapId, dto.MapId)
	}
	if dto.Instance != instance {
		t.Errorf("Expected Instance to be %s, got %s", instance, dto.Instance)
	}
}

func TestModelFromDTO(t *testing.T) {
	dto := DataTransferObject{
		WorldId:   world.Id(1),
		ChannelId: channel.Id(2),
		MapId:     _map.Id(300000000),
		Instance:  uuid.New(),
	}

	model := FromDTO(dto)

	if model.WorldId() != dto.WorldId {
		t.Errorf("Expected WorldId to be %d, got %d", dto.WorldId, model.WorldId())
	}
	if model.ChannelId() != dto.ChannelId {
		t.Errorf("Expected ChannelId to be %d, got %d", dto.ChannelId, model.ChannelId())
	}
	if model.MapId() != dto.MapId {
		t.Errorf("Expected MapId to be %d, got %d", dto.MapId, model.MapId())
	}
	if model.Instance() != dto.Instance {
		t.Errorf("Expected Instance to be %s, got %s", dto.Instance, model.Instance())
	}
}

func TestDTORoundTrip(t *testing.T) {
	original := NewBuilder(world.Id(3), channel.Id(4), _map.Id(200000000)).SetInstance(uuid.New()).Build()
	dto := original.ToDTO()
	restored := FromDTO(dto)

	if original.WorldId() != restored.WorldId() {
		t.Errorf("WorldId mismatch after round trip")
	}
	if original.ChannelId() != restored.ChannelId() {
		t.Errorf("ChannelId mismatch after round trip")
	}
	if original.MapId() != restored.MapId() {
		t.Errorf("MapId mismatch after round trip")
	}
	if original.Instance() != restored.Instance() {
		t.Errorf("Instance mismatch after round trip")
	}
}

func TestModelEquals(t *testing.T) {
	instance := uuid.New()
	model1 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(instance).Build()
	model2 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(instance).Build()

	if !model1.Equals(model2) {
		t.Error("Expected models to be equal")
	}

	// Different instance
	model3 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(uuid.New()).Build()
	if model1.Equals(model3) {
		t.Error("Expected models with different instances to not be equal")
	}

	// Different map
	model4 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(400000000)).SetInstance(instance).Build()
	if model1.Equals(model4) {
		t.Error("Expected models with different mapId to not be equal")
	}

	// Different channel
	model5 := NewBuilder(world.Id(1), channel.Id(3), _map.Id(300000000)).SetInstance(instance).Build()
	if model1.Equals(model5) {
		t.Error("Expected models with different channelId to not be equal")
	}

	// Different world
	model6 := NewBuilder(world.Id(2), channel.Id(2), _map.Id(300000000)).SetInstance(instance).Build()
	if model1.Equals(model6) {
		t.Error("Expected models with different worldId to not be equal")
	}
}

func TestModelSameMap(t *testing.T) {
	instance1 := uuid.New()
	instance2 := uuid.New()

	model1 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(instance1).Build()
	model2 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(instance2).Build()

	// Same map, different instance - should be same map
	if !model1.SameMap(model2) {
		t.Error("Expected models with same world/channel/map to be on SameMap")
	}

	// Different mapId - should not be same map
	model3 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(400000000)).SetInstance(instance1).Build()
	if model1.SameMap(model3) {
		t.Error("Expected models with different mapId to not be on SameMap")
	}
}

func TestModelIsInstanced(t *testing.T) {
	// Non-instanced map
	model1 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).Build()
	if model1.IsInstanced() {
		t.Error("Expected model without instance to not be instanced")
	}

	// Instanced map
	model2 := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(uuid.New()).Build()
	if !model2.IsInstanced() {
		t.Error("Expected model with instance to be instanced")
	}
}

func TestModelWithInstance(t *testing.T) {
	original := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).Build()
	newInstance := uuid.New()

	modified := original.WithInstance(newInstance)

	if modified.WorldId() != original.WorldId() {
		t.Error("WorldId should be preserved")
	}
	if modified.ChannelId() != original.ChannelId() {
		t.Error("ChannelId should be preserved")
	}
	if modified.MapId() != original.MapId() {
		t.Error("MapId should be preserved")
	}
	if modified.Instance() != newInstance {
		t.Error("Instance should be updated")
	}
	// Original should be unchanged
	if original.Instance() != uuid.Nil {
		t.Error("Original model should be unchanged")
	}
}

func TestModelWithoutInstance(t *testing.T) {
	instance := uuid.New()
	original := NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).SetInstance(instance).Build()

	modified := original.WithoutInstance()

	if modified.WorldId() != original.WorldId() {
		t.Error("WorldId should be preserved")
	}
	if modified.ChannelId() != original.ChannelId() {
		t.Error("ChannelId should be preserved")
	}
	if modified.MapId() != original.MapId() {
		t.Error("MapId should be preserved")
	}
	if modified.Instance() != uuid.Nil {
		t.Error("Instance should be Nil")
	}
	// Original should be unchanged
	if original.Instance() != instance {
		t.Error("Original model should be unchanged")
	}
}
