package collection

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderRequiresIdentity(t *testing.T) {
	_, err := NewModelBuilder().Build()
	if err == nil {
		t.Fatal("expected error when characterId is zero")
	}
}

func TestBuilderRoundtrip(t *testing.T) {
	tid := uuid.New()
	m, err := NewModelBuilder().
		SetTenantId(tid).
		SetCharacterId(42).
		SetCoverCardId(2380000).
		SetBookLevel(3).
		SetNormalCount(7).
		SetSpecialCount(2).
		SetExpBonusPercent(3).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if m.CharacterId() != 42 || m.CoverCardId() != 2380000 ||
		m.BookLevel() != 3 || m.NormalCount() != 7 || m.SpecialCount() != 2 ||
		m.ExpBonusPercent() != 3 || m.TenantId() != tid {
		t.Fatalf("roundtrip mismatch: %+v", m)
	}
	if total := m.TotalUniqueCards(); total != 9 {
		t.Fatalf("expected total 9, got %d", total)
	}
}
