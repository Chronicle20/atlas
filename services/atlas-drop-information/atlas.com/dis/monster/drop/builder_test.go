package drop

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewMonsterDropBuilder(t *testing.T) {
	tenantId := uuid.New()
	id := uint32(123)

	builder := NewMonsterDropBuilder(tenantId, id)

	if builder == nil {
		t.Fatal("Expected builder to be non-nil")
	}

	model := builder.Build()

	if model.TenantId() != tenantId {
		t.Errorf("Expected TenantId %s, got %s", tenantId, model.TenantId())
	}

	if model.Id() != id {
		t.Errorf("Expected Id %d, got %d", id, model.Id())
	}
}

func TestMonsterDropBuilderFluentAPI(t *testing.T) {
	tenantId := uuid.New()

	model := NewMonsterDropBuilder(tenantId, 0).
		SetMonsterId(100100).
		SetItemId(2000000).
		SetMinimumQuantity(1).
		SetMaximumQuantity(5).
		SetQuestId(1001).
		SetChance(50000).
		Build()

	if model.MonsterId() != 100100 {
		t.Errorf("Expected MonsterId 100100, got %d", model.MonsterId())
	}

	if model.ItemId() != 2000000 {
		t.Errorf("Expected ItemId 2000000, got %d", model.ItemId())
	}

	if model.MinimumQuantity() != 1 {
		t.Errorf("Expected MinimumQuantity 1, got %d", model.MinimumQuantity())
	}

	if model.MaximumQuantity() != 5 {
		t.Errorf("Expected MaximumQuantity 5, got %d", model.MaximumQuantity())
	}

	if model.QuestId() != 1001 {
		t.Errorf("Expected QuestId 1001, got %d", model.QuestId())
	}

	if model.Chance() != 50000 {
		t.Errorf("Expected Chance 50000, got %d", model.Chance())
	}
}

func TestMonsterDropBuilderDefaults(t *testing.T) {
	tenantId := uuid.New()

	model := NewMonsterDropBuilder(tenantId, 0).Build()

	// All numeric fields should default to 0
	if model.MonsterId() != 0 {
		t.Errorf("Expected default MonsterId 0, got %d", model.MonsterId())
	}

	if model.ItemId() != 0 {
		t.Errorf("Expected default ItemId 0, got %d", model.ItemId())
	}

	if model.MinimumQuantity() != 0 {
		t.Errorf("Expected default MinimumQuantity 0, got %d", model.MinimumQuantity())
	}

	if model.MaximumQuantity() != 0 {
		t.Errorf("Expected default MaximumQuantity 0, got %d", model.MaximumQuantity())
	}

	if model.QuestId() != 0 {
		t.Errorf("Expected default QuestId 0, got %d", model.QuestId())
	}

	if model.Chance() != 0 {
		t.Errorf("Expected default Chance 0, got %d", model.Chance())
	}
}

func TestMonsterDropBuilderChaining(t *testing.T) {
	tenantId := uuid.New()
	builder := NewMonsterDropBuilder(tenantId, 0)

	// Test that each setter returns the builder for chaining
	result := builder.SetMonsterId(100)
	if result != builder {
		t.Error("SetMonsterId should return the same builder instance")
	}

	result = builder.SetItemId(200)
	if result != builder {
		t.Error("SetItemId should return the same builder instance")
	}

	result = builder.SetMinimumQuantity(1)
	if result != builder {
		t.Error("SetMinimumQuantity should return the same builder instance")
	}

	result = builder.SetMaximumQuantity(5)
	if result != builder {
		t.Error("SetMaximumQuantity should return the same builder instance")
	}

	result = builder.SetQuestId(1000)
	if result != builder {
		t.Error("SetQuestId should return the same builder instance")
	}

	result = builder.SetChance(50000)
	if result != builder {
		t.Error("SetChance should return the same builder instance")
	}
}
