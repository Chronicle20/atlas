package collection

import (
	"context"
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/kafka/message"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func tenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func TestComputeBookLevelMatchesCosmicFormula(t *testing.T) {
	// Cosmic loop: level=0; expToNext=1; do { level++; expToNext += level*10 }
	// while (total >= expToNext); return level when condition fails.
	//   total=0  → level=1, expToNext=11; 0>=11? no  → 1
	//   total=1  → level=1, expToNext=11; 1>=11? no  → 1
	//   total=12 → level=1, expToNext=11; 12>=11 yes → level=2, expToNext=31; 12>=31 no → 2
	//   total=31 → level=1 (11), 31>=11 yes; level=2 (31), 31>=31 yes; level=3 (61), 31>=61 no → 3
	cases := map[uint16]uint16{
		0:  1,
		1:  1,
		12: 2,
		31: 3,
	}
	for total, want := range cases {
		if got := computeBookLevel(total); got != want {
			t.Errorf("total %d: want level %d got %d", total, want, got)
		}
	}
}

func TestExpBonusEqualsBookLevel(t *testing.T) {
	if got := computeExpBonusPercent(7); got != 7 {
		t.Errorf("want 7, got %d", got)
	}
}

func TestRecomputeAfterFirstAcquisition(t *testing.T) {
	db := newDB(t)
	if err := card.Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	cp := card.NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	if _, err := cp.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	p := NewProcessor(logrus.New(), ctx, db)
	if err := p.RecomputeAndEmit(mb)(1); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	got, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.NormalCount() != 1 || got.BookLevel() != 1 || got.ExpBonusPercent() != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestGetByCharacterIdReturnsDefaultsForUnknown(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	p := NewProcessor(logrus.New(), ctx, db)
	got, err := p.GetByCharacterId(99)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CharacterId() != 99 || got.BookLevel() != 1 || got.NormalCount() != 0 {
		t.Fatalf("expected default model for unknown character, got %+v", got)
	}
}

func TestSetCoverRejectsUnownedCardBeforeProducerCall(t *testing.T) {
	// SetCoverAndEmit calls message.Emit which constructs a Kafka writer via
	// producer.ProviderImpl. We assert ONLY the rejection path (validation
	// fails before the producer is touched) so the test does not require a
	// live broker. Acceptance/clear paths are covered by integration tests
	// or downstream consumer tests.
	db := newDB(t)
	if err := card.Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	mb := message.NewBuffer()
	cp := card.NewProcessor(logrus.New(), ctx, db)
	if _, err := cp.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	p := NewProcessor(logrus.New(), ctx, db)
	// Unowned card → must error out of validation; producer never invoked.
	if err := p.SetCoverAndEmit(uuid.New(), 1, 2380001); err == nil {
		t.Fatal("expected error for unowned card")
	}
	// Out-of-range cardId → must error out of validation.
	if err := p.SetCoverAndEmit(uuid.New(), 1, 1234); err == nil {
		t.Fatal("expected error for non-card itemId")
	}
}
