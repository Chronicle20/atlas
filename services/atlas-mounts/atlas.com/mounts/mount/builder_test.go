package mount

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuilderDefaults(t *testing.T) {
	m, err := NewModelBuilder(uuid.New(), 1, uuid.New()).Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if m.Level() != 1 {
		t.Errorf("Level = %d, want 1", m.Level())
	}
	if m.Exp() != 0 {
		t.Errorf("Exp = %d, want 0", m.Exp())
	}
	if m.Tiredness() != 0 {
		t.Errorf("Tiredness = %d, want 0", m.Tiredness())
	}
	if m.LastTirednessTickAt() != nil {
		t.Errorf("LastTirednessTickAt = %v, want nil", m.LastTirednessTickAt())
	}
}

func TestCloneIsImmutable(t *testing.T) {
	tenantId := uuid.New()
	id := uuid.New()
	tick := time.Now().UTC().Truncate(time.Second)

	m, err := NewModelBuilder(tenantId, 42, id).
		SetLevel(3).
		SetExp(120).
		SetTiredness(33).
		SetLastTirednessTickAt(&tick).
		Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	cloned, err := Clone(m).SetLevel(5).Build()
	if err != nil {
		t.Fatalf("Clone Build returned error: %v", err)
	}

	// Original unchanged.
	if m.Level() != 3 {
		t.Errorf("original Level = %d, want 3 (mutated by Clone)", m.Level())
	}

	// Clone equals m except for level.
	if cloned.Level() != 5 {
		t.Errorf("cloned Level = %d, want 5", cloned.Level())
	}
	if cloned.TenantId() != m.TenantId() {
		t.Errorf("cloned TenantId = %v, want %v", cloned.TenantId(), m.TenantId())
	}
	if cloned.CharacterId() != m.CharacterId() {
		t.Errorf("cloned CharacterId = %d, want %d", cloned.CharacterId(), m.CharacterId())
	}
	if cloned.Id() != m.Id() {
		t.Errorf("cloned Id = %v, want %v", cloned.Id(), m.Id())
	}
	if cloned.Exp() != m.Exp() {
		t.Errorf("cloned Exp = %d, want %d", cloned.Exp(), m.Exp())
	}
	if cloned.Tiredness() != m.Tiredness() {
		t.Errorf("cloned Tiredness = %d, want %d", cloned.Tiredness(), m.Tiredness())
	}
	if cloned.LastTirednessTickAt() == nil || !cloned.LastTirednessTickAt().Equal(*m.LastTirednessTickAt()) {
		t.Errorf("cloned LastTirednessTickAt = %v, want %v", cloned.LastTirednessTickAt(), m.LastTirednessTickAt())
	}
}
