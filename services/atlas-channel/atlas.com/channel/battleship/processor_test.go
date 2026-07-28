package battleship

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeStore is an in-memory counterStore with TenantCounter's atomicity
// semantics (serialized decrements, no key creation on miss).
type fakeStore struct {
	mu     sync.Mutex
	values map[string]int64
	ttls   map[string]time.Duration
	err    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: map[string]int64{}, ttls: map[string]time.Duration{}}
}

func (s *fakeStore) k(t tenant.Model, key string) string { return t.Id().String() + ":" + key }

func (s *fakeStore) Set(_ context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.values[s.k(t, key)] = value
	s.ttls[s.k(t, key)] = ttl
	return nil
}

func (s *fakeStore) DecrByIfExists(_ context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, false, s.err
	}
	v, ok := s.values[s.k(t, key)]
	if !ok {
		return 0, false, nil
	}
	v -= delta
	s.values[s.k(t, key)] = v
	s.ttls[s.k(t, key)] = ttl
	return v, true, nil
}

// InitIfMissingAndDecrBy mirrors TenantCounter's atomic Lua script: the
// existence check, seed, and decrement all happen under one lock, so
// concurrent callers racing an absent key can never both seed the same
// baseline (losing a decrement) or both independently cross zero (double
// break) — whichever goroutine's call runs first seeds the value; every
// other racing call decrements the value that goroutine already seeded.
func (s *fakeStore) InitIfMissingAndDecrBy(_ context.Context, t tenant.Model, key string, initial int64, delta int64, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	v, ok := s.values[s.k(t, key)]
	if !ok {
		v = initial
	}
	v -= delta
	s.values[s.k(t, key)] = v
	s.ttls[s.k(t, key)] = ttl
	return v, nil
}

func (s *fakeStore) Remove(_ context.Context, t tenant.Model, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	delete(s.values, s.k(t, key))
	delete(s.ttls, s.k(t, key))
	return nil
}

type breakRecorder struct {
	mu        sync.Mutex
	cancels   int
	cooldowns []uint32
}

func setupProcessor(t *testing.T) (Processor, *fakeStore, *breakRecorder, tenant.Model, logrus.FieldLogger) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := testlog.NewNullLogger()

	fs := newFakeStore()
	prevStore := store
	store = fs
	rec := &breakRecorder{}
	prevCancel, prevCooldown, prevEffect, prevLevel := cancelBuffFunc, applyCooldownFunc, effectFunc, characterLevelFunc
	cancelBuffFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32) error {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.cancels++
		return nil
	}
	applyCooldownFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, cooldown uint32, _ uint32) error {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.cooldowns = append(rec.cooldowns, cooldown)
		return nil
	}
	effectFunc = func(_ logrus.FieldLogger, _ context.Context, _ byte) (effect.Model, error) {
		return effect.Extract(effect.RestModel{Cooldown: 90})
	}
	characterLevelFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, error) {
		return 150, nil
	}
	t.Cleanup(func() {
		store = prevStore
		cancelBuffFunc, applyCooldownFunc, effectFunc, characterLevelFunc = prevCancel, prevCooldown, prevEffect, prevLevel
		GetRideMirror().EvictTenant(tm.Id())
	})
	return NewProcessor(l, ctx), fs, rec, tm, l
}

// ShipHP is version-gated (R-4): the client computes the gauge's denominator
// itself via get_max_durability_of_vehicle, and that function changed at v87.
// Both arms below mirror the corresponding client exactly over the reachable
// input range (Battleship is 4th-job, so charLevel >= 120 always).
func TestShipHPFormula(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		major      uint16
		skillLevel byte
		charLevel  byte
		expected   int32
	}{
		// Pre-v87 arm: 400*SLV + max(charLevel-120,0)*200. Identical to the
		// client's 200*(charLevel + 2*SLV - 120) for charLevel >= 120; the
		// clamp keeps sub-120 (unreachable) input from going negative.
		{"v83 sub-120 clamp", "GMS", 83, 1, 100, 400},
		{"v83 exactly 120", "GMS", 83, 10, 120, 4000},
		{"v83 121 adds one step", "GMS", 83, 10, 121, 4200},
		{"v83 max: SLV 10, level 200", "GMS", 83, 10, 200, 20000},
		{"v83 mid: SLV 7, level 150", "GMS", 83, 7, 150, 8800},
		{"v61 same arm as v83", "GMS", 61, 7, 150, 8800},
		{"v84 still the old arm", "GMS", 84, 10, 200, 20000},

		// v87+ arm: max(300*charLevel + 500*(SLV-72), 0).
		{"v87 crosses to the new arm", "GMS", 87, 10, 200, 29000},
		{"v92 new arm", "GMS", 92, 10, 120, 5000},
		{"v95 new arm mid", "GMS", 95, 7, 150, 12500},
		{"v95 SLV 1 at 120", "GMS", 95, 1, 120, 500},
		{"jms185 uses the new arm", "JMS", 185, 10, 200, 29000},
		{"new arm floors at zero", "GMS", 95, 1, 100, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tm, err := tenant.Create(uuid.New(), tc.region, tc.major, 1)
			if err != nil {
				t.Fatalf("tenant.Create: %v", err)
			}
			if got := ShipHP(tm, tc.skillLevel, tc.charLevel); got != tc.expected {
				t.Errorf("ShipHP(%s v%d, SLV %d, charLevel %d) = %d, want %d",
					tc.region, tc.major, tc.skillLevel, tc.charLevel, got, tc.expected)
			}
		})
	}
}

func TestInitShipHPStoresFormulaWithTTL(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	if err := p.InitShipHP(100, 7, 150, 35*time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	if got := fs.values[fs.k(tm, "100")]; got != 8800 {
		t.Fatalf("stored HP = %d, want 8800", got)
	}
	if got := fs.ttls[fs.k(tm, "100")]; got != 35*time.Minute {
		t.Fatalf("TTL = %v, want 35m", got)
	}
}

func TestDrainNotRiding(t *testing.T) {
	p, _, _, _, _ := setupProcessor(t)
	res := p.Drain(field.Model{}, 100, 250)
	if res.Status != DrainNotRiding {
		t.Fatalf("Status = %v, want DrainNotRiding", res.Status)
	}
}

func TestDrainZeroDamageSkipped(t *testing.T) {
	p, _, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	if res := p.Drain(field.Model{}, 100, 0); res.Status != DrainSkipped {
		t.Fatalf("Status = %v, want DrainSkipped", res.Status)
	}
}

func TestDrainDecrementsAndReports(t *testing.T) {
	p, _, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7, StateTTL: 35 * time.Minute})
	if err := p.InitShipHP(100, 7, 150, 35*time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	res := p.Drain(field.Model{}, 100, 300)
	if res.Status != DrainDrained || res.RemainingHP != 8500 {
		t.Fatalf("Drain = %+v, want Drained/8500", res)
	}
	if rec.cancels != 0 || len(rec.cooldowns) != 0 {
		t.Fatal("non-breaking drain must not cancel or apply cooldown")
	}
}

func TestDrainLazyReinit(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7}) // no Redis entry
	res := p.Drain(field.Model{}, 100, 300)
	// full = ShipHP(7, 150) = 8800 → 8500 remaining
	if res.Status != DrainDrained || res.RemainingHP != 8500 {
		t.Fatalf("Drain = %+v, want Drained/8500 after lazy re-init", res)
	}
	if got := fs.values[fs.k(tm, "100")]; got != 8500 {
		t.Fatalf("stored HP = %d, want 8500", got)
	}
}

func TestDrainLazyReinitOverkillBreaks(t *testing.T) {
	p, fs, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 1}) // full = 6400 (skill 1, level 150)
	res := p.Drain(field.Model{}, 100, 9999)
	if res.Status != DrainBroke {
		t.Fatalf("Status = %v, want DrainBroke", res.Status)
	}
	if rec.cancels != 1 || len(rec.cooldowns) != 1 || rec.cooldowns[0] != 90 {
		t.Fatalf("break side effects = cancels %d cooldowns %v, want 1/[90]", rec.cancels, rec.cooldowns)
	}
	if _, ok := fs.values[fs.k(tm, "100")]; ok {
		t.Fatal("ship state must be cleared on break")
	}
	if _, riding := GetRideMirror().Get(tm, 100); riding {
		t.Fatal("mirror must be cleared on break")
	}
}

func TestDrainBreakExactlyOnceUnderConcurrency(t *testing.T) {
	p, _, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7, StateTTL: time.Minute})
	if err := p.InitShipHP(100, 7, 150, time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	// 8800 HP, 10 workers × 1000 damage: crosses zero exactly once.
	var wg sync.WaitGroup
	broke := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := p.Drain(field.Model{}, 100, 1000); res.Status == DrainBroke {
				mu.Lock()
				broke++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if broke != 1 {
		t.Fatalf("DrainBroke observed %d times, want exactly 1", broke)
	}
	if rec.cancels != 1 || len(rec.cooldowns) != 1 {
		t.Fatalf("break side effects ran %d/%d times, want once", rec.cancels, len(rec.cooldowns))
	}
}

// TestDrainLazyReinitBreakExactlyOnceUnderConcurrency exercises the lazy
// re-init branch specifically (no InitShipHP call, so DecrByIfExists always
// misses and every goroutine takes the InitIfMissingAndDecrBy path). Under
// the earlier buggy implementation — compute full-damage locally, then
// plain store.Set — every one of these goroutines observes existed=false
// and independently computes the same full baseline; whichever Set call
// lands last overwrites the rest, so in practice most runs never reach
// DrainBroke at all (each goroutine's own single-hit subtraction 8800-1000
// = 7800 > 0). This test would therefore FAIL (broke != 1, likely 0) under
// that implementation; it passes now because InitIfMissingAndDecrBy seeds
// the counter atomically at most once and serializes every decrement
// through Redis (mirrored by the fakeStore's single mutex-guarded method),
// giving the same "exactly one caller crosses zero" guarantee as the
// steady-state DecrByIfExists path.
func TestDrainLazyReinitBreakExactlyOnceUnderConcurrency(t *testing.T) {
	p, _, rec, tm, _ := setupProcessor(t)
	// No InitShipHP call: the Redis entry is absent, forcing every Drain
	// through the lazy re-init branch.
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7, StateTTL: time.Minute})
	// full = ShipHP(SLV 7, level 150) = 8800; 10 workers x 1000 damage
	// jointly exceeds it, crossing zero exactly once.
	var wg sync.WaitGroup
	broke := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := p.Drain(field.Model{}, 100, 1000); res.Status == DrainBroke {
				mu.Lock()
				broke++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if broke != 1 {
		t.Fatalf("DrainBroke observed %d times via lazy re-init, want exactly 1", broke)
	}
	if rec.cancels != 1 || len(rec.cooldowns) != 1 {
		t.Fatalf("break side effects ran %d/%d times via lazy re-init, want once", rec.cancels, len(rec.cooldowns))
	}
}

// TestDrainLazyReinitNoLostDecrementUnderConcurrency covers the
// individually-non-lethal case (b) from the same bug class: under the
// earlier buggy implementation, concurrent misses each independently
// compute full-damage and the final stored value reflects only whichever
// goroutine's plain Set ran last — silently absorbing every other hit. With
// InitIfMissingAndDecrBy, the final stored value must reflect every hit.
func TestDrainLazyReinitNoLostDecrementUnderConcurrency(t *testing.T) {
	p, fs, rec, tm, _ := setupProcessor(t)
	// No InitShipHP call: forces the lazy re-init branch on every Drain.
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 1, StateTTL: time.Minute})
	// full = ShipHP(SLV 1, level 150) = 6400; 8 workers x 100 damage is
	// non-lethal both per-hit and jointly (6400-800=5600 > 0).
	const workers = 8
	const damage = 100
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res := p.Drain(field.Model{}, 100, damage); res.Status != DrainDrained {
				t.Errorf("Status = %v, want DrainDrained (jointly non-lethal)", res.Status)
			}
		}()
	}
	wg.Wait()
	if want := int64(6400 - workers*damage); fs.values[fs.k(tm, "100")] != want {
		t.Fatalf("stored HP = %d, want %d (no decrement lost across %d concurrent lazy re-inits)",
			fs.values[fs.k(tm, "100")], want, workers)
	}
	if rec.cancels != 0 || len(rec.cooldowns) != 0 {
		t.Fatal("no hit was lethal; break side effects must not have run")
	}
}

func TestDrainRedisErrorDegrades(t *testing.T) {
	p, fs, rec, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	fs.err = errors.New("redis down")
	res := p.Drain(field.Model{}, 100, 300)
	if res.Status != DrainSkipped {
		t.Fatalf("Status = %v, want DrainSkipped on Redis error", res.Status)
	}
	if rec.cancels != 0 || len(rec.cooldowns) != 0 {
		t.Fatal("degraded drain must have no side effects")
	}
}

func TestClearIdempotent(t *testing.T) {
	p, fs, _, tm, _ := setupProcessor(t)
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 7})
	if err := p.InitShipHP(100, 7, 150, time.Minute); err != nil {
		t.Fatalf("InitShipHP: %v", err)
	}
	p.Clear(100)
	p.Clear(100) // second call is a no-op
	if _, ok := fs.values[fs.k(tm, "100")]; ok {
		t.Fatal("Clear did not remove ship state")
	}
	if _, riding := GetRideMirror().Get(tm, 100); riding {
		t.Fatal("Clear did not remove mirror entry")
	}
}

func TestIsRiding(t *testing.T) {
	p, _, _, tm, _ := setupProcessor(t)
	if _, riding := p.IsRiding(100); riding {
		t.Fatal("IsRiding true with empty mirror")
	}
	GetRideMirror().Put(tm, 100, RideState{SkillLevel: 9})
	lvl, riding := p.IsRiding(100)
	if !riding || lvl != 9 {
		t.Fatalf("IsRiding = (%d, %v), want (9, true)", lvl, riding)
	}
}
