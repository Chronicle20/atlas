package drop

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewReactorDropBuilder(t *testing.T) {
	tenantId := uuid.New()
	id := uint32(123)

	builder := NewReactorDropBuilder(tenantId, id)

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

func TestReactorDropBuilderFluentAPI(t *testing.T) {
	tenantId := uuid.New()

	model, err := NewReactorDropBuilder(tenantId, 0).
		SetReactorId(1001).
		SetItemId(2000000).
		SetQuestId(1001).
		SetChance(50000).
		Build()

	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if model.ReactorId() != 1001 {
		t.Errorf("Expected ReactorId 1001, got %d", model.ReactorId())
	}

	if model.ItemId() != 2000000 {
		t.Errorf("Expected ItemId 2000000, got %d", model.ItemId())
	}

	if model.QuestId() != 1001 {
		t.Errorf("Expected QuestId 1001, got %d", model.QuestId())
	}

	if model.Chance() != 50000 {
		t.Errorf("Expected Chance 50000, got %d", model.Chance())
	}
}

func TestReactorDropBuilderDefaults(t *testing.T) {
	tenantId := uuid.New()

	model, err := NewReactorDropBuilder(tenantId, 0).Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	// All numeric fields should default to 0
	if model.ReactorId() != 0 {
		t.Errorf("Expected default ReactorId 0, got %d", model.ReactorId())
	}

	if model.ItemId() != 0 {
		t.Errorf("Expected default ItemId 0, got %d", model.ItemId())
	}

	if model.QuestId() != 0 {
		t.Errorf("Expected default QuestId 0, got %d", model.QuestId())
	}

	if model.Chance() != 0 {
		t.Errorf("Expected default Chance 0, got %d", model.Chance())
	}
}

func TestReactorDropBuilderValidation_NilTenantId(t *testing.T) {
	_, err := NewReactorDropBuilder(uuid.Nil, 0).Build()
	if err == nil {
		t.Error("Expected error for nil tenantId, got nil")
	}
}

func TestReactorDropBuilderChaining(t *testing.T) {
	tenantId := uuid.New()
	builder := NewReactorDropBuilder(tenantId, 0)

	// Test that each setter returns the builder for chaining
	result := builder.SetReactorId(1001)
	if result != builder {
		t.Error("SetReactorId should return the same builder instance")
	}

	result = builder.SetItemId(200)
	if result != builder {
		t.Error("SetItemId should return the same builder instance")
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
