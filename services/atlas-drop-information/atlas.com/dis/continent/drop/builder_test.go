package drop

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewContinentDropBuilder(t *testing.T) {
	tenantId := uuid.New()
	id := uint32(123)

	builder := NewContinentDropBuilder(tenantId, id)

	if builder == nil {
		t.Fatal("Expected builder to be non-nil")
	}

	model, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if model.TenantId() != tenantId {
		t.Errorf("Expected TenantId %s, got %s", tenantId, model.TenantId())
	}

	if model.Id() != id {
		t.Errorf("Expected Id %d, got %d", id, model.Id())
	}
}

func TestContinentDropBuilderFluentAPI(t *testing.T) {
	tenantId := uuid.New()

	model, err := NewContinentDropBuilder(tenantId, 0).
		SetContinentId(-1).
		SetItemId(4001126).
		SetMinimumQuantity(1).
		SetMaximumQuantity(2).
		SetQuestId(0).
		SetChance(8000).
		Build()

	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if model.ContinentId() != -1 {
		t.Errorf("Expected ContinentId -1, got %d", model.ContinentId())
	}

	if model.ItemId() != 4001126 {
		t.Errorf("Expected ItemId 4001126, got %d", model.ItemId())
	}

	if model.MinimumQuantity() != 1 {
		t.Errorf("Expected MinimumQuantity 1, got %d", model.MinimumQuantity())
	}

	if model.MaximumQuantity() != 2 {
		t.Errorf("Expected MaximumQuantity 2, got %d", model.MaximumQuantity())
	}

	if model.QuestId() != 0 {
		t.Errorf("Expected QuestId 0, got %d", model.QuestId())
	}

	if model.Chance() != 8000 {
		t.Errorf("Expected Chance 8000, got %d", model.Chance())
	}
}

func TestContinentDropBuilderDefaults(t *testing.T) {
	tenantId := uuid.New()

	model, err := NewContinentDropBuilder(tenantId, 0).Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	// ContinentId defaults to 0 (not -1)
	if model.ContinentId() != 0 {
		t.Errorf("Expected default ContinentId 0, got %d", model.ContinentId())
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

func TestContinentDropBuilderValidation_NilTenantId(t *testing.T) {
	_, err := NewContinentDropBuilder(uuid.Nil, 0).Build()
	if err == nil {
		t.Error("Expected error for nil tenantId, got nil")
	}
}

func TestContinentDropBuilderNegativeContinentId(t *testing.T) {
	tenantId := uuid.New()

	// Test that negative continent IDs work (they are valid, e.g., -1 for global)
	model, err := NewContinentDropBuilder(tenantId, 0).
		SetContinentId(-1).
		Build()

	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if model.ContinentId() != -1 {
		t.Errorf("Expected ContinentId -1, got %d", model.ContinentId())
	}
}

func TestContinentDropBuilderChaining(t *testing.T) {
	tenantId := uuid.New()
	builder := NewContinentDropBuilder(tenantId, 0)

	// Test that each setter returns the builder for chaining
	result := builder.SetContinentId(-1)
	if result != builder {
		t.Error("SetContinentId should return the same builder instance")
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
