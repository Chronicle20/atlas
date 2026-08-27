package account

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderValidBuild(t *testing.T) {
	tenantId := uuid.New()
	name := "testuser"

	m, err := NewBuilder(tenantId, name).
		SetPassword("hashedpassword").
		SetGender(1).
		Build()
	if err != nil {
		t.Fatalf("Expected successful build, got error: %v", err)
	}

	if m.TenantId() != tenantId {
		t.Errorf("TenantId mismatch. Expected %v, got %v", tenantId, m.TenantId())
	}

	if m.Name() != name {
		t.Errorf("Name mismatch. Expected %v, got %v", name, m.Name())
	}

	if m.Password() != "hashedpassword" {
		t.Errorf("Password mismatch. Expected %v, got %v", "hashedpassword", m.Password())
	}

	if m.Gender() != 1 {
		t.Errorf("Gender mismatch. Expected %v, got %v", 1, m.Gender())
	}
}

func TestBuilderEmptyNameValidation(t *testing.T) {
	tenantId := uuid.New()

	_, err := NewBuilder(tenantId, "").Build()

	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}

	if err.Error() != "name is required" {
		t.Errorf("Unexpected error message: %v", err.Error())
	}
}

func TestBuilderNilTenantAllowed(t *testing.T) {
	_, err := NewBuilder(uuid.Nil, "testuser").Build()
	if err != nil {
		t.Fatalf("Expected nil tenant to be allowed for REST input models, got error: %v", err)
	}
}

func TestBuilderAllSetters(t *testing.T) {
	tenantId := uuid.New()

	m, err := NewBuilder(tenantId, "testuser").
		SetId(123).
		SetPassword("password").
		SetPin("1234").
		SetPic("5678").
		SetBirthDate(19900101).
		SetState(StateLoggedIn).
		SetGender(1).
		SetTOS(true).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.Id() != 123 {
		t.Errorf("Id mismatch. Expected 123, got %v", m.Id())
	}

	if m.Pin() != "1234" {
		t.Errorf("Pin mismatch. Expected 1234, got %v", m.Pin())
	}

	if m.Pic() != "5678" {
		t.Errorf("Pic mismatch. Expected 5678, got %v", m.Pic())
	}

	if m.BirthDate() != 19900101 {
		t.Errorf("BirthDate mismatch. Expected 19900101, got %v", m.BirthDate())
	}

	if m.State() != StateLoggedIn {
		t.Errorf("State mismatch. Expected %v, got %v", StateLoggedIn, m.State())
	}

	if m.TOS() != true {
		t.Errorf("TOS mismatch. Expected true, got %v", m.TOS())
	}
}

func TestBuilderDefaults(t *testing.T) {
	m, err := NewBuilder(uuid.New(), "testuser").Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.State() != StateNotLoggedIn {
		t.Errorf("Default state should be StateNotLoggedIn, got %v", m.State())
	}

	if m.Gender() != 0 {
		t.Errorf("Default gender should be 0, got %v", m.Gender())
	}

	if m.TOS() != false {
		t.Errorf("Default TOS should be false, got %v", m.TOS())
	}
}

func TestCharacterSlotBuilder(t *testing.T) {
	tenantId := uuid.New()

	m := NewCharacterSlotBuilder(tenantId, 7, 1).
		SetSlots(5).
		Build()

	if m.TenantId() != tenantId {
		t.Errorf("TenantId mismatch. Expected %v, got %v", tenantId, m.TenantId())
	}
	if m.AccountId() != 7 {
		t.Errorf("AccountId mismatch. Expected 7, got %v", m.AccountId())
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId mismatch. Expected 1, got %v", m.WorldId())
	}
	if m.Slots() != 5 {
		t.Errorf("Slots mismatch. Expected 5, got %v", m.Slots())
	}
}
