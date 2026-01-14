package tenant_test

import (
	"atlas-tenants/tenant"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := tenant.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()

	model, err := tenant.NewModelBuilder().
		SetId(id).
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.Name() != "Test Tenant" {
		t.Errorf("model.Name() = %s, want 'Test Tenant'", model.Name())
	}
	if model.Region() != "GMS" {
		t.Errorf("model.Region() = %s, want 'GMS'", model.Region())
	}
	if model.MajorVersion() != 83 {
		t.Errorf("model.MajorVersion() = %d, want 83", model.MajorVersion())
	}
	if model.MinorVersion() != 1 {
		t.Errorf("model.MinorVersion() = %d, want 1", model.MinorVersion())
	}
}

func TestBuild_MissingName(t *testing.T) {
	_, err := tenant.NewModelBuilder().
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	if err != tenant.ErrNameRequired {
		t.Errorf("Build() error = %v, want ErrNameRequired", err)
	}
}

func TestBuild_MissingRegion(t *testing.T) {
	_, err := tenant.NewModelBuilder().
		SetName("Test Tenant").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	if err != tenant.ErrRegionRequired {
		t.Errorf("Build() error = %v, want ErrRegionRequired", err)
	}
}

func TestBuild_Success(t *testing.T) {
	model, err := tenant.NewModelBuilder().
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Name() != "Test Tenant" {
		t.Errorf("model.Name() = %s, want 'Test Tenant'", model.Name())
	}
	// ID should be auto-generated
	if model.Id() == uuid.Nil {
		t.Error("model.Id() should not be nil UUID")
	}
}

func TestCloneModel(t *testing.T) {
	original, err := tenant.NewModelBuilder().
		SetName("Original Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := tenant.CloneModel(original).
		SetName("Cloned Tenant").
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Name() != "Original Tenant" {
		t.Errorf("original.Name() = %s, want 'Original Tenant'", original.Name())
	}

	// Cloned should have new name but same other values
	if cloned.Id() != original.Id() {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), original.Id())
	}
	if cloned.Name() != "Cloned Tenant" {
		t.Errorf("cloned.Name() = %s, want 'Cloned Tenant'", cloned.Name())
	}
	if cloned.Region() != "GMS" {
		t.Errorf("cloned.Region() = %s, want 'GMS'", cloned.Region())
	}
	if cloned.MajorVersion() != 83 {
		t.Errorf("cloned.MajorVersion() = %d, want 83", cloned.MajorVersion())
	}
}

func TestBuilderFluentChaining(t *testing.T) {
	model, err := tenant.NewModelBuilder().
		SetName("Fluent Tenant").
		SetRegion("EMS").
		SetMajorVersion(90).
		SetMinorVersion(2).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Name() != "Fluent Tenant" {
		t.Errorf("model.Name() = %s, want 'Fluent Tenant'", model.Name())
	}
	if model.Region() != "EMS" {
		t.Errorf("model.Region() = %s, want 'EMS'", model.Region())
	}
	if model.MajorVersion() != 90 {
		t.Errorf("model.MajorVersion() = %d, want 90", model.MajorVersion())
	}
	if model.MinorVersion() != 2 {
		t.Errorf("model.MinorVersion() = %d, want 2", model.MinorVersion())
	}
}

func TestModelString(t *testing.T) {
	model, err := tenant.NewModelBuilder().
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	str := model.String()
	if str == "" {
		t.Error("model.String() should not be empty")
	}
	// Should contain all key fields
	if !containsSubstring(str, "Test Tenant") {
		t.Error("model.String() should contain tenant name")
	}
	if !containsSubstring(str, "GMS") {
		t.Error("model.String() should contain region")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s[1:], substr) || s[:len(substr)] == substr)
}
