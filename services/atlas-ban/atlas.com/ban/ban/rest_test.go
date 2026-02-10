package ban

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	tenantId := uuid.New()
	m, _ := NewBuilder(tenantId, BanTypeIP, "192.168.1.1").
		SetId(100).
		SetReason("Cheating").
		SetReasonCode(1).
		SetPermanent(true).
		SetIssuedBy("admin").
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != 100 {
		t.Errorf("Id mismatch. Expected 100, got %v", rm.Id)
	}

	if rm.BanType != byte(BanTypeIP) {
		t.Errorf("BanType mismatch. Expected %v, got %v", byte(BanTypeIP), rm.BanType)
	}

	if rm.Value != "192.168.1.1" {
		t.Errorf("Value mismatch. Expected 192.168.1.1, got %v", rm.Value)
	}

	if rm.Reason != "Cheating" {
		t.Errorf("Reason mismatch. Expected Cheating, got %v", rm.Reason)
	}

	if rm.ReasonCode != 1 {
		t.Errorf("ReasonCode mismatch. Expected 1, got %v", rm.ReasonCode)
	}

	if rm.Permanent != true {
		t.Errorf("Permanent mismatch. Expected true, got %v", rm.Permanent)
	}

	if rm.IssuedBy != "admin" {
		t.Errorf("IssuedBy mismatch. Expected admin, got %v", rm.IssuedBy)
	}
}

func TestExtract(t *testing.T) {
	expectedExpiry := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	rm := RestModel{
		Id:         200,
		BanType:    byte(BanTypeHWID),
		Value:      "DEF456",
		Reason:     "Bot usage",
		ReasonCode: 2,
		Permanent:  false,
		ExpiresAt:  expectedExpiry,
		IssuedBy:   "moderator",
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if m.Id() != 200 {
		t.Errorf("Id mismatch. Expected 200, got %v", m.Id())
	}

	if m.Type() != BanTypeHWID {
		t.Errorf("BanType mismatch. Expected %v, got %v", BanTypeHWID, m.Type())
	}

	if m.Value() != "DEF456" {
		t.Errorf("Value mismatch. Expected DEF456, got %v", m.Value())
	}

	if m.Reason() != "Bot usage" {
		t.Errorf("Reason mismatch. Expected Bot usage, got %v", m.Reason())
	}

	if m.ReasonCode() != 2 {
		t.Errorf("ReasonCode mismatch. Expected 2, got %v", m.ReasonCode())
	}

	if m.Permanent() != false {
		t.Errorf("Permanent mismatch. Expected false, got %v", m.Permanent())
	}

	if !m.ExpiresAt().Equal(expectedExpiry) {
		t.Errorf("ExpiresAt mismatch. Expected %v, got %v", expectedExpiry, m.ExpiresAt())
	}

	if m.IssuedBy() != "moderator" {
		t.Errorf("IssuedBy mismatch. Expected moderator, got %v", m.IssuedBy())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "bans" {
		t.Errorf("GetName mismatch. Expected 'bans', got '%v'", rm.GetName())
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

func TestCheckRestModelGetName(t *testing.T) {
	crm := CheckRestModel{}
	if crm.GetName() != "ban-checks" {
		t.Errorf("GetName mismatch. Expected 'ban-checks', got '%v'", crm.GetName())
	}
}

func TestCheckRestModelSetID(t *testing.T) {
	crm := &CheckRestModel{}
	err := crm.SetID("42")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}

	if crm.Id != 42 {
		t.Errorf("Id mismatch after SetID. Expected 42, got %v", crm.Id)
	}
}

func TestTransformCheckNilModel(t *testing.T) {
	result := TransformCheck(nil)

	if result.Banned != false {
		t.Errorf("Expected Banned=false for nil model, got %v", result.Banned)
	}

	if result.BanType != 0 {
		t.Errorf("Expected BanType=0 for nil model, got %v", result.BanType)
	}
}

func TestTransformCheckBannedModel(t *testing.T) {
	m, _ := NewBuilder(uuid.New(), BanTypeIP, "10.0.0.1").
		SetId(5).
		SetReason("Hacking").
		SetReasonCode(7).
		SetPermanent(true).
		Build()

	result := TransformCheck(&m)

	if result.Banned != true {
		t.Errorf("Expected Banned=true, got %v", result.Banned)
	}

	if result.Id != 5 {
		t.Errorf("Id mismatch. Expected 5, got %v", result.Id)
	}

	if result.BanType != byte(BanTypeIP) {
		t.Errorf("BanType mismatch. Expected %v, got %v", byte(BanTypeIP), result.BanType)
	}

	if result.Reason != "Hacking" {
		t.Errorf("Reason mismatch. Expected Hacking, got %v", result.Reason)
	}

	if result.ReasonCode != 7 {
		t.Errorf("ReasonCode mismatch. Expected 7, got %v", result.ReasonCode)
	}

	if result.Permanent != true {
		t.Errorf("Permanent mismatch. Expected true, got %v", result.Permanent)
	}
}
