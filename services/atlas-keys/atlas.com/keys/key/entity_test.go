package key

import (
	"testing"

	"github.com/google/uuid"
)

func TestMake_TransformsEntityToModel(t *testing.T) {
	tenantId := uuid.New()
	e := entity{
		TenantId:    tenantId,
		CharacterId: 12345,
		Key:         18,
		Type:        4,
		Action:      0,
	}

	m, err := Make(e)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m.CharacterId() != e.CharacterId {
		t.Errorf("characterId mismatch: got %d, want %d", m.CharacterId(), e.CharacterId)
	}
	if m.Key() != e.Key {
		t.Errorf("key mismatch: got %d, want %d", m.Key(), e.Key)
	}
	if m.Type() != e.Type {
		t.Errorf("type mismatch: got %d, want %d", m.Type(), e.Type)
	}
	if m.Action() != e.Action {
		t.Errorf("action mismatch: got %d, want %d", m.Action(), e.Action)
	}
}

func TestModel_ToEntity_TransformsModelToEntity(t *testing.T) {
	tenantId := uuid.New()
	m, _ := NewModelBuilder().
		SetCharacterId(54321).
		SetKey(65).
		SetType(6).
		SetAction(106).
		Build()

	e := m.ToEntity(tenantId)

	if e.TenantId != tenantId {
		t.Errorf("tenantId mismatch: got %v, want %v", e.TenantId, tenantId)
	}
	if e.CharacterId != m.CharacterId() {
		t.Errorf("characterId mismatch: got %d, want %d", e.CharacterId, m.CharacterId())
	}
	if e.Key != m.Key() {
		t.Errorf("key mismatch: got %d, want %d", e.Key, m.Key())
	}
	if e.Type != m.Type() {
		t.Errorf("type mismatch: got %d, want %d", e.Type, m.Type())
	}
	if e.Action != m.Action() {
		t.Errorf("action mismatch: got %d, want %d", e.Action, m.Action())
	}
}

func TestMake_ToEntity_RoundTrip(t *testing.T) {
	tenantId := uuid.New()
	original := entity{
		TenantId:    tenantId,
		CharacterId: 99999,
		Key:         29,
		Type:        5,
		Action:      52,
	}

	m, err := Make(original)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	roundTrip := m.ToEntity(tenantId)

	if roundTrip.TenantId != original.TenantId {
		t.Errorf("tenantId mismatch after round trip")
	}
	if roundTrip.CharacterId != original.CharacterId {
		t.Errorf("characterId mismatch after round trip")
	}
	if roundTrip.Key != original.Key {
		t.Errorf("key mismatch after round trip")
	}
	if roundTrip.Type != original.Type {
		t.Errorf("type mismatch after round trip")
	}
	if roundTrip.Action != original.Action {
		t.Errorf("action mismatch after round trip")
	}
}

func TestEntity_TableName(t *testing.T) {
	e := entity{}
	if e.TableName() != "keys" {
		t.Errorf("expected table name 'keys', got %q", e.TableName())
	}
}
