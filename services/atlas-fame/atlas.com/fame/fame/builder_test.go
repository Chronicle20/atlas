package fame

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuild(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	targetId := uint32(67890)
	amount := int8(1)

	m, err := NewBuilder(tenantId, characterId, targetId, amount).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.tenantId != tenantId {
		t.Errorf("expected tenantId %v, got %v", tenantId, m.tenantId)
	}
	if m.characterId != characterId {
		t.Errorf("expected characterId %d, got %d", characterId, m.characterId)
	}
	if m.targetId != targetId {
		t.Errorf("expected targetId %d, got %d", targetId, m.targetId)
	}
	if m.amount != amount {
		t.Errorf("expected amount %d, got %d", amount, m.amount)
	}
}

func TestBuilderBuildNegativeAmount(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	targetId := uint32(67890)
	amount := int8(-1)

	m, err := NewBuilder(tenantId, characterId, targetId, amount).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.amount != amount {
		t.Errorf("expected amount %d, got %d", amount, m.amount)
	}
}

func TestBuilderValidationNilTenantId(t *testing.T) {
	_, err := NewBuilder(uuid.Nil, 12345, 67890, 1).Build()
	if err == nil {
		t.Error("expected error for nil tenantId")
	}
	if err.Error() != "tenantId is required" {
		t.Errorf("expected 'tenantId is required' error, got %v", err)
	}
}

func TestBuilderValidationZeroCharacterId(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 0, 67890, 1).Build()
	if err == nil {
		t.Error("expected error for zero characterId")
	}
	if err.Error() != "characterId is required" {
		t.Errorf("expected 'characterId is required' error, got %v", err)
	}
}

func TestBuilderValidationZeroTargetId(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 12345, 0, 1).Build()
	if err == nil {
		t.Error("expected error for zero targetId")
	}
	if err.Error() != "targetId is required" {
		t.Errorf("expected 'targetId is required' error, got %v", err)
	}
}

func TestBuilderValidationInvalidAmountZero(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 12345, 67890, 0).Build()
	if err == nil {
		t.Error("expected error for zero amount")
	}
	if err.Error() != "amount must be 1 or -1" {
		t.Errorf("expected 'amount must be 1 or -1' error, got %v", err)
	}
}

func TestBuilderValidationInvalidAmountTwo(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 12345, 67890, 2).Build()
	if err == nil {
		t.Error("expected error for amount 2")
	}
	if err.Error() != "amount must be 1 or -1" {
		t.Errorf("expected 'amount must be 1 or -1' error, got %v", err)
	}
}

func TestBuilderValidationInvalidAmountNegativeTwo(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 12345, 67890, -2).Build()
	if err == nil {
		t.Error("expected error for amount -2")
	}
	if err.Error() != "amount must be 1 or -1" {
		t.Errorf("expected 'amount must be 1 or -1' error, got %v", err)
	}
}

func TestBuilderFluentChaining(t *testing.T) {
	tenantId := uuid.New()
	characterId := uint32(12345)
	targetId := uint32(67890)
	amount := int8(1)

	builder := NewBuilder(tenantId, characterId, targetId, amount)

	result := builder.
		SetTenantId(uuid.New()).
		SetCharacterId(11111).
		SetTargetId(22222).
		SetAmount(-1)

	if result != builder {
		t.Error("fluent methods should return the same builder instance")
	}
}

func TestBuilderSetters(t *testing.T) {
	tenantId := uuid.New()
	newTenantId := uuid.New()
	characterId := uint32(12345)
	targetId := uint32(67890)

	m, err := NewBuilder(tenantId, characterId, targetId, 1).
		SetTenantId(newTenantId).
		SetCharacterId(11111).
		SetTargetId(22222).
		SetAmount(-1).
		Build()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.tenantId != newTenantId {
		t.Errorf("expected tenantId %v, got %v", newTenantId, m.tenantId)
	}
	if m.characterId != 11111 {
		t.Errorf("expected characterId 11111, got %d", m.characterId)
	}
	if m.targetId != 22222 {
		t.Errorf("expected targetId 22222, got %d", m.targetId)
	}
	if m.amount != -1 {
		t.Errorf("expected amount -1, got %d", m.amount)
	}
}
