package history

import (
	"testing"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	tenantId := uuid.New()
	m, _ := NewBuilder(tenantId, 100, "testuser").
		SetId(42).
		SetIPAddress("192.168.1.1").
		SetHWID("ABC123").
		SetSuccess(true).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != 42 {
		t.Errorf("Id mismatch. Expected 42, got %v", rm.Id)
	}

	if rm.AccountId != 100 {
		t.Errorf("AccountId mismatch. Expected 100, got %v", rm.AccountId)
	}

	if rm.AccountName != "testuser" {
		t.Errorf("AccountName mismatch. Expected testuser, got %v", rm.AccountName)
	}

	if rm.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress mismatch. Expected 192.168.1.1, got %v", rm.IPAddress)
	}

	if rm.HWID != "ABC123" {
		t.Errorf("HWID mismatch. Expected ABC123, got %v", rm.HWID)
	}

	if rm.Success != true {
		t.Errorf("Success mismatch. Expected true, got %v", rm.Success)
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "login-history" {
		t.Errorf("GetName mismatch. Expected 'login-history', got '%v'", rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	rm := RestModel{Id: 789}
	if rm.GetID() != "789" {
		t.Errorf("GetID mismatch. Expected '789', got '%v'", rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("321")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}

	if rm.Id != 321 {
		t.Errorf("Id mismatch after SetID. Expected 321, got %v", rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("notanumber")
	if err == nil {
		t.Fatal("Expected error for invalid id, got nil")
	}
}
