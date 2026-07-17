package ranking

import (
	"errors"
	"testing"
	"time"

	"atlas-rankings/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// characterFixture builds a character.Model via its JSON:API extract path —
// the package exposes no test constructor, and Extract is the production
// decode path anyway.
func characterFixture(t *testing.T, id uint32, worldId byte, jobId uint16, level byte, exp uint32, gm int) character.Model {
	t.Helper()
	rm := character.RestModel{
		AccountId:  1,
		WorldId:    world.Id(worldId),
		Level:      level,
		Experience: exp,
		JobId:      job.Id(jobId),
		Gm:         gm,
	}
	rm.Id = id
	m, err := character.Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

func supplierOf(cs ...character.Model) CharacterSupplier {
	return func() ([]character.Model, error) { return cs, nil }
}

func TestRecomputeRanksAndExcludesGms(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 90, 0, 0), // warrior lvl 90 → overall 1
		characterFixture(t, 2, 0, 200, 80, 0, 0), // magician lvl 80 → overall 2
		characterFixture(t, 3, 0, 100, 70, 0, 0), // warrior lvl 70 → overall 3
		characterFixture(t, 4, 0, 900, 99, 0, 1), // GM — excluded entirely, not counted
	))

	if err := p.Recompute(time.Now()); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	m1, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if m1.OverallRank() != 1 || m1.JobRank() != 1 || m1.OverallRankMove() != 0 {
		t.Fatalf("char 1: %+v", m1)
	}
	m2, _ := p.GetByCharacterId(2)
	if m2.OverallRank() != 2 || m2.JobRank() != 1 {
		t.Fatalf("char 2 (GM must not shift ranks): %+v", m2)
	}
	m3, _ := p.GetByCharacterId(3)
	if m3.OverallRank() != 3 || m3.JobRank() != 2 {
		t.Fatalf("char 3: %+v", m3)
	}
	if _, err := p.GetByCharacterId(4); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GM must have no row, got err=%v", err)
	}
}

func TestRecomputeMovesAcrossTwoCycles(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	first := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
	))
	if err := first.Recompute(time.Now()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Character 1 levels past character 2; character 3 appears.
	second := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 70, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
		characterFixture(t, 3, 0, 200, 10, 0, 0),
	))
	if err := second.Recompute(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	m1, _ := second.GetByCharacterId(1)
	if m1.OverallRank() != 1 || m1.OverallRankMove() != 1 || m1.JobRankMove() != 1 {
		t.Fatalf("char 1 move: %+v", m1)
	}
	m2, _ := second.GetByCharacterId(2)
	if m2.OverallRank() != 2 || m2.OverallRankMove() != -1 {
		t.Fatalf("char 2 move: %+v", m2)
	}
	m3, _ := second.GetByCharacterId(3)
	if m3.OverallRankMove() != 0 || m3.JobRankMove() != 0 {
		t.Fatalf("first-seen char 3 must move 0: %+v", m3)
	}
}

func TestRecomputePrunesDepartedCharacters(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	first := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
	))
	if err := first.Recompute(time.Now()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Character 2 deleted, character 1 became GM.
	second := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 1),
	))
	if err := second.Recompute(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	if _, err := second.GetByCharacterId(1); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("became-GM character must be pruned")
	}
	if _, err := second.GetByCharacterId(2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("deleted character must be pruned")
	}
}

func TestGetByCharacterIdsOmitsUnknown(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
	))
	if err := p.Recompute(time.Now()); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	ms, err := p.GetByCharacterIds([]uint32{1, 999})
	if err != nil {
		t.Fatalf("bulk read: %v", err)
	}
	if len(ms) != 1 || ms[0].CharacterId() != 1 {
		t.Fatalf("unknown ids must be omitted: %+v", ms)
	}
}

func TestIsDue(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf())
	now := time.Now()

	due, err := p.IsDue(time.Hour, now)
	if err != nil || !due {
		t.Fatalf("no cycle row must be due: due=%v err=%v", due, err)
	}

	if err := p.Recompute(now); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	due, err = p.IsDue(time.Hour, now.Add(30*time.Minute))
	if err != nil || due {
		t.Fatalf("30m into a 60m interval must not be due: due=%v err=%v", due, err)
	}

	due, err = p.IsDue(time.Hour, now.Add(61*time.Minute))
	if err != nil || !due {
		t.Fatalf("61m into a 60m interval must be due: due=%v err=%v", due, err)
	}

	// Pin the >= boundary precisely: exactly one interval elapsed must
	// already be due. A `>` implementation would pass the 30m/61m cases
	// above but wrongly report not-due here.
	due, err = p.IsDue(time.Hour, now.Add(time.Hour))
	if err != nil || !due {
		t.Fatalf("exactly 60m into a 60m interval must be due (>= semantics): due=%v err=%v", due, err)
	}
}

// TestRecomputeSkipsPruneOnEmptyScanWithExistingRows proves the Finding-1
// guard: a scan that returns ([], nil) — the same shape an HTTP 200 with an
// empty `data` array produces via requests.DrainProvider — must not wipe an
// already-populated tenant's rankings. Without the guard, upsertBatch no-ops
// on the empty entities slice and pruneBefore(tdb, now) deletes every row
// whose computed_at predates `now`, i.e. every row seeded by cycle one.
func TestRecomputeSkipsPruneOnEmptyScanWithExistingRows(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	now := time.Now()
	first := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 11, 0, 100, 50, 0, 0),
		characterFixture(t, 12, 0, 100, 60, 0, 0),
	))
	if err := first.Recompute(now); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Cycle two's scan comes back empty-without-error — the transient
	// failure mode this guard exists for.
	second := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf())
	next := now.Add(time.Hour)
	if err := second.Recompute(next); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	// Character 12 (level 60) outranks character 11 (level 50): rank 1 vs 2.
	m1, err := second.GetByCharacterId(11)
	if err != nil {
		t.Fatalf("existing row for character 11 must survive an empty-scan cycle: %v", err)
	}
	if m1.OverallRank() != 2 {
		t.Fatalf("character 11's rank must be untouched by the skipped prune: %+v", m1)
	}
	m2, err := second.GetByCharacterId(12)
	if err != nil {
		t.Fatalf("existing row for character 12 must survive an empty-scan cycle: %v", err)
	}
	if m2.OverallRank() != 1 {
		t.Fatalf("character 12's rank must be untouched by the skipped prune: %+v", m2)
	}

	// The cycle must still be recorded and advanced even though the prune
	// was skipped, so IsDue reflects the attempted cycle rather than
	// looping on the stale cycle-one timestamp.
	tdb := db.WithContext(ctx)
	c, err := cycleEntityProvider()(tdb)()
	if err != nil {
		t.Fatalf("cycle row read: %v", err)
	}
	if !c.LastStartedAt.Equal(next) {
		t.Fatalf("cycle 2 start must be recorded: got %v, want %v", c.LastStartedAt, next)
	}
	if c.LastCompletedAt == nil {
		t.Fatal("cycle 2 must still complete even though the prune was skipped")
	}
	if c.CharactersRanked != 0 {
		t.Fatalf("cycle 2 ranked 0 characters: got %d", c.CharactersRanked)
	}
}

// TestRecomputeEmptyTenantStillRecordsCycle proves the flip side of the
// Finding-1 guard: when the rankings table is genuinely empty (no prior
// rows), an empty scan is not held back — the cycle records and completes
// normally, and IsDue still advances. This must keep passing under the
// guard exactly as it did before (TestIsDue already depends on this shape).
func TestRecomputeEmptyTenantStillRecordsCycle(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf())
	now := time.Now()
	if err := p.Recompute(now); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	tdb := db.WithContext(ctx)
	c, err := cycleEntityProvider()(tdb)()
	if err != nil {
		t.Fatalf("cycle row read: %v", err)
	}
	if !c.LastStartedAt.Equal(now) {
		t.Fatalf("cycle start must be recorded: got %v, want %v", c.LastStartedAt, now)
	}
	if c.LastCompletedAt == nil {
		t.Fatal("a legitimately empty tenant must still complete its cycle")
	}
	if c.CharactersRanked != 0 {
		t.Fatalf("expected 0 ranked characters, got %d", c.CharactersRanked)
	}

	var count int64
	if err := tdb.Model(&Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("a genuinely empty tenant must have no rows: got %d", count)
	}
}

// TestRecomputeGmBoundaryIsGreaterThanZero pins Seam 1 precisely: eligibility
// is gm > 0, not gm == 1. A fixture set with gm=0 (included), gm=1, and
// gm=2 (both excluded) is required to distinguish the correct rule from the
// wrong `gm == 1` rule — a suite using only gm=1 could not tell them apart.
func TestRecomputeGmBoundaryIsGreaterThanZero(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0), // gm=0 → included
		characterFixture(t, 2, 0, 100, 60, 0, 1), // gm=1 → excluded
		characterFixture(t, 3, 0, 100, 70, 0, 2), // gm=2 → excluded
	))

	if err := p.Recompute(time.Now()); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	m1, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("gm=0 character must be ranked: %v", err)
	}
	if m1.OverallRank() != 1 {
		t.Fatalf("gm=0 character should be sole rank 1: %+v", m1)
	}

	if _, err := p.GetByCharacterId(2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("gm=1 character must be excluded (gm > 0, not gm == 1), got err=%v", err)
	}
	if _, err := p.GetByCharacterId(3); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("gm=2 character must be excluded (gm > 0), got err=%v", err)
	}
}
