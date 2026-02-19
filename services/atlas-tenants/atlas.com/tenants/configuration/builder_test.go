package configuration_test

import (
	"atlas-tenants/configuration"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := configuration.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	tenantId := uuid.New()
	resourceData := json.RawMessage(`{"data": {"id": "test-1", "name": "Test Route"}}`)

	model, err := configuration.NewModelBuilder().
		SetID(id).
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.ID() != id {
		t.Errorf("model.ID() = %v, want %v", model.ID(), id)
	}
	if model.TenantId() != tenantId {
		t.Errorf("model.TenantId() = %v, want %v", model.TenantId(), tenantId)
	}
	if model.ResourceName() != "routes" {
		t.Errorf("model.ResourceName() = %s, want 'routes'", model.ResourceName())
	}
	if string(model.ResourceData()) != string(resourceData) {
		t.Errorf("model.ResourceData() = %s, want %s", model.ResourceData(), resourceData)
	}
}

func TestBuild_MissingTenantId(t *testing.T) {
	_, err := configuration.NewModelBuilder().
		SetResourceName("routes").
		SetResourceData(json.RawMessage(`{}`)).
		Build()

	if !errors.Is(err, configuration.ErrTenantIdRequired) {
		t.Errorf("Build() error = %v, want ErrTenantIdRequired", err)
	}
}

func TestBuild_MissingResourceName(t *testing.T) {
	_, err := configuration.NewModelBuilder().
		SetTenantId(uuid.New()).
		SetResourceData(json.RawMessage(`{}`)).
		Build()

	if !errors.Is(err, configuration.ErrResourceNameRequired) {
		t.Errorf("Build() error = %v, want ErrResourceNameRequired", err)
	}
}

func TestBuild_Success(t *testing.T) {
	tenantId := uuid.New()
	model, err := configuration.NewModelBuilder().
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(json.RawMessage(`{"data": []}`)).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.TenantId() != tenantId {
		t.Errorf("model.TenantId() = %v, want %v", model.TenantId(), tenantId)
	}
	// ID should be auto-generated
	if model.ID() == uuid.Nil {
		t.Error("model.ID() should not be nil UUID")
	}
}

func TestCloneModel(t *testing.T) {
	tenantId := uuid.New()
	original, err := configuration.NewModelBuilder().
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(json.RawMessage(`{"data": []}`)).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := configuration.CloneModel(original).
		SetResourceName("vessels").
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.ResourceName() != "routes" {
		t.Errorf("original.ResourceName() = %s, want 'routes'", original.ResourceName())
	}

	// Cloned should have new resource name but same other values
	if cloned.ID() != original.ID() {
		t.Errorf("cloned.ID() = %v, want %v", cloned.ID(), original.ID())
	}
	if cloned.TenantId() != tenantId {
		t.Errorf("cloned.TenantId() = %v, want %v", cloned.TenantId(), tenantId)
	}
	if cloned.ResourceName() != "vessels" {
		t.Errorf("cloned.ResourceName() = %s, want 'vessels'", cloned.ResourceName())
	}
}

func TestBuilderFluentChaining(t *testing.T) {
	tenantId := uuid.New()
	resourceData := json.RawMessage(`{"data": {"id": "test-1"}}`)

	model, err := configuration.NewModelBuilder().
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.TenantId() != tenantId {
		t.Errorf("model.TenantId() = %v, want %v", model.TenantId(), tenantId)
	}
	if model.ResourceName() != "routes" {
		t.Errorf("model.ResourceName() = %s, want 'routes'", model.ResourceName())
	}
}

func TestModelString(t *testing.T) {
	tenantId := uuid.New()
	model, err := configuration.NewModelBuilder().
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(json.RawMessage(`{}`)).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	str := model.String()
	if str == "" {
		t.Error("model.String() should not be empty")
	}
	// Should contain resource name
	if !containsSubstring(str, "routes") {
		t.Error("model.String() should contain resource name")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
