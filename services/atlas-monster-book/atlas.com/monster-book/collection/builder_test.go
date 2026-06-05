package collection

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderCoverMobIdRoundTrip(t *testing.T) {
	m := NewModelBuilder().
		SetCharacterId(1).
		SetCoverCardId(2380000).
		SetCoverMobId(100100).
		MustBuild()

	if m.CoverMobId() != 100100 {
		t.Fatalf("Model.CoverMobId() = %d, want 100100", m.CoverMobId())
	}

	e := m.ToEntity()
	if e.CoverMobId != 100100 {
		t.Fatalf("entity.CoverMobId = %d, want 100100", e.CoverMobId)
	}

	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.CoverMobId() != 100100 || back.CoverCardId() != 2380000 {
		t.Fatalf("Make round-trip: mobId=%d cardId=%d", back.CoverMobId(), back.CoverCardId())
	}
}

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
