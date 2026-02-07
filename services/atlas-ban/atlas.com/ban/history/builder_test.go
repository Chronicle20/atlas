package history

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuilderValidBuild(t *testing.T) {
	tenantId := uuid.New()

	m, err := NewBuilder(tenantId, 123, "testuser").
		SetIPAddress("192.168.1.1").
		SetHWID("ABC123").
		SetSuccess(true).
		Build()

	if err != nil {
		t.Fatalf("Expected successful build, got error: %v", err)
	}

	if m.TenantId() != tenantId {
		t.Errorf("TenantId mismatch. Expected %v, got %v", tenantId, m.TenantId())
	}

	if m.AccountId() != 123 {
		t.Errorf("AccountId mismatch. Expected 123, got %v", m.AccountId())
	}

	if m.AccountName() != "testuser" {
		t.Errorf("AccountName mismatch. Expected testuser, got %v", m.AccountName())
	}

	if m.IPAddress() != "192.168.1.1" {
		t.Errorf("IPAddress mismatch. Expected 192.168.1.1, got %v", m.IPAddress())
	}

	if m.HWID() != "ABC123" {
		t.Errorf("HWID mismatch. Expected ABC123, got %v", m.HWID())
	}

	if m.Success() != true {
		t.Errorf("Success mismatch. Expected true, got %v", m.Success())
	}
}

func TestBuilderZeroAccountIdValidation(t *testing.T) {
	tenantId := uuid.New()

	_, err := NewBuilder(tenantId, 0, "").Build()

	if err == nil {
		t.Fatal("Expected error for zero accountId, got nil")
	}

	if err.Error() != "accountId is required" {
		t.Errorf("Unexpected error message: %v", err.Error())
	}
}

func TestBuilderAllSetters(t *testing.T) {
	tenantId := uuid.New()
	now := time.Now()

	m, err := NewBuilder(tenantId, 456, "admin").
		SetId(789).
		SetIPAddress("10.0.0.1").
		SetHWID("XYZ789").
		SetSuccess(false).
		SetFailureReason("INCORRECT_PASSWORD").
		SetCreatedAt(now).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.Id() != 789 {
		t.Errorf("Id mismatch. Expected 789, got %v", m.Id())
	}

	if m.AccountId() != 456 {
		t.Errorf("AccountId mismatch. Expected 456, got %v", m.AccountId())
	}

	if m.AccountName() != "admin" {
		t.Errorf("AccountName mismatch. Expected admin, got %v", m.AccountName())
	}

	if m.IPAddress() != "10.0.0.1" {
		t.Errorf("IPAddress mismatch. Expected 10.0.0.1, got %v", m.IPAddress())
	}

	if m.HWID() != "XYZ789" {
		t.Errorf("HWID mismatch. Expected XYZ789, got %v", m.HWID())
	}

	if m.Success() != false {
		t.Errorf("Success mismatch. Expected false, got %v", m.Success())
	}

	if m.FailureReason() != "INCORRECT_PASSWORD" {
		t.Errorf("FailureReason mismatch. Expected INCORRECT_PASSWORD, got %v", m.FailureReason())
	}

	if m.CreatedAt() != now {
		t.Errorf("CreatedAt mismatch. Expected %v, got %v", now, m.CreatedAt())
	}
}

func TestBuilderDefaults(t *testing.T) {
	m, err := NewBuilder(uuid.New(), 1, "user").Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.Id() != 0 {
		t.Errorf("Default id should be 0, got %v", m.Id())
	}

	if m.IPAddress() != "" {
		t.Errorf("Default IPAddress should be empty, got %v", m.IPAddress())
	}

	if m.HWID() != "" {
		t.Errorf("Default HWID should be empty, got %v", m.HWID())
	}

	if m.Success() != false {
		t.Errorf("Default Success should be false, got %v", m.Success())
	}

	if m.FailureReason() != "" {
		t.Errorf("Default FailureReason should be empty, got %v", m.FailureReason())
	}
}
