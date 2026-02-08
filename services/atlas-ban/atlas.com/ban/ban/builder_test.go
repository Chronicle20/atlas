package ban

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuilderValidBuild(t *testing.T) {
	tenantId := uuid.New()

	m, err := NewBuilder(tenantId, BanTypeIP, "192.168.1.1").
		SetReason("Cheating").
		SetReasonCode(1).
		SetPermanent(true).
		Build()

	if err != nil {
		t.Fatalf("Expected successful build, got error: %v", err)
	}

	if m.TenantId() != tenantId {
		t.Errorf("TenantId mismatch. Expected %v, got %v", tenantId, m.TenantId())
	}

	if m.Type() != BanTypeIP {
		t.Errorf("BanType mismatch. Expected %v, got %v", BanTypeIP, m.Type())
	}

	if m.Value() != "192.168.1.1" {
		t.Errorf("Value mismatch. Expected 192.168.1.1, got %v", m.Value())
	}

	if m.Reason() != "Cheating" {
		t.Errorf("Reason mismatch. Expected Cheating, got %v", m.Reason())
	}

	if m.ReasonCode() != 1 {
		t.Errorf("ReasonCode mismatch. Expected 1, got %v", m.ReasonCode())
	}

	if m.Permanent() != true {
		t.Errorf("Permanent mismatch. Expected true, got %v", m.Permanent())
	}
}

func TestBuilderEmptyValueValidation(t *testing.T) {
	tenantId := uuid.New()

	_, err := NewBuilder(tenantId, BanTypeIP, "").Build()

	if err == nil {
		t.Fatal("Expected error for empty value, got nil")
	}

	if err.Error() != "value is required" {
		t.Errorf("Unexpected error message: %v", err.Error())
	}
}

func TestBuilderNilTenantAllowed(t *testing.T) {
	_, err := NewBuilder(uuid.Nil, BanTypeIP, "192.168.1.1").Build()

	if err != nil {
		t.Fatalf("Expected nil tenant to be allowed, got error: %v", err)
	}
}

func TestBuilderAllSetters(t *testing.T) {
	tenantId := uuid.New()
	now := time.Now()

	m, err := NewBuilder(tenantId, BanTypeHWID, "ABC123").
		SetId(42).
		SetReason("Bot usage").
		SetReasonCode(3).
		SetPermanent(false).
		SetExpiresAt(1234567890).
		SetIssuedBy("admin").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.Id() != 42 {
		t.Errorf("Id mismatch. Expected 42, got %v", m.Id())
	}

	if m.Type() != BanTypeHWID {
		t.Errorf("BanType mismatch. Expected %v, got %v", BanTypeHWID, m.Type())
	}

	if m.Value() != "ABC123" {
		t.Errorf("Value mismatch. Expected ABC123, got %v", m.Value())
	}

	if m.Reason() != "Bot usage" {
		t.Errorf("Reason mismatch. Expected Bot usage, got %v", m.Reason())
	}

	if m.ReasonCode() != 3 {
		t.Errorf("ReasonCode mismatch. Expected 3, got %v", m.ReasonCode())
	}

	if m.Permanent() != false {
		t.Errorf("Permanent mismatch. Expected false, got %v", m.Permanent())
	}

	if m.ExpiresAt() != 1234567890 {
		t.Errorf("ExpiresAt mismatch. Expected 1234567890, got %v", m.ExpiresAt())
	}

	if m.IssuedBy() != "admin" {
		t.Errorf("IssuedBy mismatch. Expected admin, got %v", m.IssuedBy())
	}

	if m.CreatedAt() != now {
		t.Errorf("CreatedAt mismatch. Expected %v, got %v", now, m.CreatedAt())
	}

	if m.UpdatedAt() != now {
		t.Errorf("UpdatedAt mismatch. Expected %v, got %v", now, m.UpdatedAt())
	}
}

func TestBuilderDefaults(t *testing.T) {
	m, err := NewBuilder(uuid.New(), BanTypeIP, "10.0.0.1").Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if m.Id() != 0 {
		t.Errorf("Default id should be 0, got %v", m.Id())
	}

	if m.Reason() != "" {
		t.Errorf("Default reason should be empty, got %v", m.Reason())
	}

	if m.ReasonCode() != 0 {
		t.Errorf("Default reasonCode should be 0, got %v", m.ReasonCode())
	}

	if m.Permanent() != false {
		t.Errorf("Default permanent should be false, got %v", m.Permanent())
	}

	if m.ExpiresAt() != 0 {
		t.Errorf("Default expiresAt should be 0, got %v", m.ExpiresAt())
	}

	if m.IssuedBy() != "" {
		t.Errorf("Default issuedBy should be empty, got %v", m.IssuedBy())
	}
}

func TestBanTypeConstants(t *testing.T) {
	if BanTypeIP != 0 {
		t.Errorf("BanTypeIP should be 0, got %v", BanTypeIP)
	}
	if BanTypeHWID != 1 {
		t.Errorf("BanTypeHWID should be 1, got %v", BanTypeHWID)
	}
	if BanTypeAccount != 2 {
		t.Errorf("BanTypeAccount should be 2, got %v", BanTypeAccount)
	}
}

func TestIsExpiredPermanent(t *testing.T) {
	m, _ := NewBuilder(uuid.New(), BanTypeIP, "10.0.0.1").
		SetPermanent(true).
		Build()

	if IsExpired(m) {
		t.Error("Permanent ban should never be expired")
	}
}

func TestIsExpiredNotYet(t *testing.T) {
	m, _ := NewBuilder(uuid.New(), BanTypeIP, "10.0.0.1").
		SetPermanent(false).
		SetExpiresAt(time.Now().Unix() + 3600).
		Build()

	if IsExpired(m) {
		t.Error("Ban should not be expired yet")
	}
}

func TestIsExpiredAlready(t *testing.T) {
	m, _ := NewBuilder(uuid.New(), BanTypeIP, "10.0.0.1").
		SetPermanent(false).
		SetExpiresAt(time.Now().Unix() - 3600).
		Build()

	if !IsExpired(m) {
		t.Error("Ban should be expired")
	}
}
