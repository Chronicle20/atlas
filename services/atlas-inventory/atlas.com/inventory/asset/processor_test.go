package asset

import (
	"atlas-inventory/data/equipment/statistics"
	"atlas-inventory/kafka/message"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// buildTestEquipStats constructs a statistics.Model with known values using the
// exported Extract + RestModel path, which is the only way to build one without
// an exported constructor.
func buildTestEquipStats() statistics.Model {
	ea, _ := statistics.Extract(statistics.RestModel{
		Strength:      10,
		Dexterity:     8,
		Intelligence:  6,
		Luck:          4,
		Hp:            100,
		Mp:            50,
		WeaponAttack:  15,
		MagicAttack:   12,
		WeaponDefense: 20,
		MagicDefense:  18,
		Accuracy:      5,
		Avoidability:  3,
		Speed:         10,
		Jump:          5,
		Slots:         7,
	})
	return ea
}

func TestApplyEquipStats_UseAverageStats_True_WritesVerbatim(t *testing.T) {
	ea := buildTestEquipStats()
	b := NewBuilder(uuid.New(), 1040010)
	applyEquipStats(b, ea, true)
	m := b.Build()

	if m.Strength() != 10 {
		t.Errorf("expected Strength 10, got %d", m.Strength())
	}
	if m.Dexterity() != 8 {
		t.Errorf("expected Dexterity 8, got %d", m.Dexterity())
	}
	if m.Intelligence() != 6 {
		t.Errorf("expected Intelligence 6, got %d", m.Intelligence())
	}
	if m.Luck() != 4 {
		t.Errorf("expected Luck 4, got %d", m.Luck())
	}
	if m.Hp() != 100 {
		t.Errorf("expected Hp 100, got %d", m.Hp())
	}
	if m.Mp() != 50 {
		t.Errorf("expected Mp 50, got %d", m.Mp())
	}
	if m.WeaponAttack() != 15 {
		t.Errorf("expected WeaponAttack 15, got %d", m.WeaponAttack())
	}
	if m.MagicAttack() != 12 {
		t.Errorf("expected MagicAttack 12, got %d", m.MagicAttack())
	}
	if m.WeaponDefense() != 20 {
		t.Errorf("expected WeaponDefense 20, got %d", m.WeaponDefense())
	}
	if m.MagicDefense() != 18 {
		t.Errorf("expected MagicDefense 18, got %d", m.MagicDefense())
	}
	if m.Accuracy() != 5 {
		t.Errorf("expected Accuracy 5, got %d", m.Accuracy())
	}
	if m.Avoidability() != 3 {
		t.Errorf("expected Avoidability 3, got %d", m.Avoidability())
	}
	if m.Speed() != 10 {
		t.Errorf("expected Speed 10, got %d", m.Speed())
	}
	if m.Jump() != 5 {
		t.Errorf("expected Jump 5, got %d", m.Jump())
	}
	if m.Slots() != 7 {
		t.Errorf("expected Slots 7, got %d", m.Slots())
	}
}

func TestApplyEquipStats_UseAverageStats_False_RetainsVariance(t *testing.T) {
	ea := buildTestEquipStats()

	// Run 20 iterations; at least one stat should differ from the base across all trials.
	totalDelta := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		b := NewBuilder(uuid.New(), 1040010)
		applyEquipStats(b, ea, false)
		m := b.Build()

		// Sum absolute deltas across stats that have non-zero base values.
		// Slots is always deterministic, so exclude it from delta check.
		totalDelta += abs16(m.Strength(), ea.Strength())
		totalDelta += abs16(m.Dexterity(), ea.Dexterity())
		totalDelta += abs16(m.Intelligence(), ea.Intelligence())
		totalDelta += abs16(m.Luck(), ea.Luck())
		totalDelta += abs16(m.Hp(), ea.Hp())
		totalDelta += abs16(m.Mp(), ea.Mp())
		totalDelta += abs16(m.WeaponAttack(), ea.WeaponAttack())
		totalDelta += abs16(m.MagicAttack(), ea.MagicAttack())
		totalDelta += abs16(m.WeaponDefense(), ea.WeaponDefense())
		totalDelta += abs16(m.MagicDefense(), ea.MagicDefense())
	}

	if totalDelta == 0 {
		t.Error("expected at least some stat variance over 20 trials, but all rolls equalled the base values")
	}

	// Slots must always equal the base regardless of variance mode.
	b := NewBuilder(uuid.New(), 1040010)
	applyEquipStats(b, ea, false)
	if b.Build().Slots() != ea.Slots() {
		t.Errorf("expected Slots %d (deterministic), got %d", ea.Slots(), b.Build().Slots())
	}
}

func abs16(a, b uint16) int {
	if a >= b {
		return int(a - b)
	}
	return int(b - a)
}

// extendExpirationTenantId is shared by testContext and seedExtendableAsset
// so that assets seeded directly via seedExtendableAsset are visible to a
// processor built from testContext.
var extendExpirationTenantId = uuid.New()

// testDatabase opens a fresh in-memory sqlite database, registers the tenant
// scoping callbacks, and runs the asset package migration. It reuses the
// shared databasetest harness (see resource_test.go's use of it) rather than
// hand-rolling sqlite setup.
func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration)
}

// testContext returns a context carrying extendExpirationTenantId.
func testContext(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(extendExpirationTenantId)
}

// seedExtendableAsset creates an asset directly against db, scoped to
// extendExpirationTenantId, with its builder pre-populated with a baseline
// (permanent, unflagged equip) slot and then customized by configure.
//
// Named distinctly from resource_test.go's seedAsset (same package, a
// simpler tenantId/compartmentId/slot/templateId signature used only by the
// pagination tests) rather than overloading that name.
func seedExtendableAsset(t *testing.T, db *gorm.DB, configure func(b *Builder)) Model {
	t.Helper()
	b := NewBuilder(uuid.New(), 1072001).
		SetSlot(1).
		SetCreatedAt(time.Now())
	configure(b)
	a, err := create(db.WithContext(testContext(t)), extendExpirationTenantId, b.Build())
	if err != nil {
		t.Fatalf("Failed to seed asset: %v", err)
	}
	return a
}

// TestExtendExpirationPreservesFlags verifies the load-bearing invariant:
// ExtendExpiration must change only the expiration column, leaving every
// other flag bit — including one unrelated to locking — untouched.
func TestExtendExpirationPreservesFlags(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	// An unlocked, time-limited equip carrying an unrelated flag bit.
	a := seedExtendableAsset(t, db, func(b *Builder) {
		// AddFlag, not SetFlag: SetFlag takes a raw uint16 while the
		// constants are typed af.Flag (builder.go:111 vs :174).
		b.SetExpiration(base).AddFlag(af.FlagUntradeable)
	})

	mb := message.NewBuffer()
	want := base.Add(168 * time.Hour)
	err := NewProcessor(l, ctx, db).ExtendExpiration(mb)(uuid.New(), 12345)(a, want)
	if err != nil {
		t.Fatalf("ExtendExpiration: %v", err)
	}

	got, err := NewProcessor(l, ctx, db).GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Expiration().Equal(want) {
		t.Errorf("Expiration = %v, want %v", got.Expiration(), want)
	}
	if got.Flag() != a.Flag() {
		t.Errorf("Flag = %d, want %d (unchanged)", got.Flag(), a.Flag())
	}
	if got.Locked() {
		t.Error("FlagLock was set; ExtendExpiration must never touch flags")
	}
}

// TestExtendExpirationRejectsLockedAndPermanent verifies the two rejection
// cases: a locked asset (expiration is a lock window, not a time limit) and
// a permanent asset (no expiration to extend).
func TestExtendExpirationRejectsLockedAndPermanent(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)
	p := NewProcessor(l, ctx, db)
	future := time.Now().UTC().Add(240 * time.Hour)

	locked := seedExtendableAsset(t, db, func(b *Builder) {
		b.SetExpiration(time.Now().UTC().Add(48 * time.Hour)).AddFlag(af.FlagLock)
	})
	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(locked, future); err == nil {
		t.Error("expected rejection for a locked asset")
	}

	permanent := seedExtendableAsset(t, db, func(b *Builder) {})
	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(permanent, future); err == nil {
		t.Error("expected rejection for a permanent asset")
	}
}

// TestExtendExpirationRedeliveryIsIdempotent verifies that redelivering the
// same absolute expiration against an already-extended asset does not stack
// a second extension, and that it still emits UPDATED so the saga step that
// triggered the original write completes on redelivery too.
func TestExtendExpirationRedeliveryIsIdempotent(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)
	p := NewProcessor(l, ctx, db)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	a := seedExtendableAsset(t, db, func(b *Builder) { b.SetExpiration(base) })
	want := base.Add(168 * time.Hour)

	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(a, want); err != nil {
		t.Fatal(err)
	}
	extended, err := p.GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	// Replay the same absolute value against the already-extended asset.
	mb := message.NewBuffer()
	if err := p.ExtendExpiration(mb)(uuid.New(), 12345)(extended, want); err != nil {
		t.Fatalf("redelivery must succeed, got %v", err)
	}
	again, err := p.GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	if !again.Expiration().Equal(want) {
		t.Errorf("Expiration = %v, want %v (redelivery must not stack)", again.Expiration(), want)
	}
	if len(mb.GetAll()) == 0 {
		t.Error("redelivery must still emit UPDATED so the saga step completes")
	}
}
