package monster

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// resetStatusMirror resets the singleton for test isolation. Test-only.
func resetStatusMirror() {
	statusMirrorOnce = sync.Once{}
	statusMirror = nil
}

func newTestStatusMirror() *StatusMirror {
	return &StatusMirror{perTenant: map[uuid.UUID]map[uint32]map[string][]StatusEntry{}}
}

func TestStatusMirror_OnAppliedStoresEntry(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	eid := uuid.NewString()
	now := time.Now()
	body := StatusEffectAppliedBody{
		EffectId:         eid,
		SourceType:       "CHARACTER",
		SourceCharacterId: 42,
		SourceSkillId:    1111006,
		SourceSkillLevel: 1,
		Statuses:         map[string]int32{"VENOM": 25},
		Duration:         60000,
		TickInterval:     1000,
	}
	m.OnApplied(tm, 7, body, now)

	if reflectInfo, ok := m.GetReflect(tm, 7, "MAGIC"); ok {
		t.Fatalf("expected no reflect info for VENOM-only effect, got %+v", reflectInfo)
	}
	if c := m.VenomCount(tm, 7); c != 1 {
		t.Fatalf("expected venom count 1 after first apply, got %d", c)
	}
}

func TestStatusMirror_GetReflectFalseForUnknownMonster(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	if _, ok := m.GetReflect(tm, 999, "MAGIC"); ok {
		t.Fatalf("expected miss for unknown monster")
	}
}

func TestStatusMirror_GetReflectReturnsReflectInfo(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	eid := uuid.NewString()
	now := time.Now()
	body := StatusEffectAppliedBody{
		EffectId:         eid,
		SourceType:       "CHARACTER",
		SourceCharacterId: 42,
		SourceSkillId:    2121003, // Mana Reflection (example)
		SourceSkillLevel: 20,
		Statuses:         map[string]int32{"MAGIC_REFLECT": 1},
		Duration:         60000,
		ReflectKind:      "MAGIC",
		ReflectPercent:   30,
		ReflectLtX:       -200,
		ReflectLtY:       -200,
		ReflectRbX:       200,
		ReflectRbY:       200,
		ReflectMaxDamage: 9999,
	}
	m.OnApplied(tm, 7, body, now)

	ri, ok := m.GetReflect(tm, 7, "MAGIC")
	if !ok {
		t.Fatalf("expected reflect info present")
	}
	if ri.Kind != "MAGIC" || ri.Percent != 30 || ri.MaxDamage != 9999 {
		t.Fatalf("unexpected reflect info: %+v", ri)
	}
	if ri.LtX != -200 || ri.RbY != 200 {
		t.Fatalf("unexpected reflect bounds: %+v", ri)
	}
}

func TestStatusMirror_GetReflectSkipsWallClockExpired(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	// Apply an entry whose ExpiresAt is already in the past at call time:
	// `now` is two hours ago and Duration is 60s, so expiresAt < time.Now().
	past := time.Now().Add(-2 * time.Hour)
	body := StatusEffectAppliedBody{
		EffectId:       uuid.NewString(),
		Statuses:       map[string]int32{"MAGIC_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    "MAGIC",
		ReflectPercent: 30,
	}
	m.OnApplied(tm, 7, body, past)
	if ri, ok := m.GetReflect(tm, 7, "MAGIC"); ok {
		t.Fatalf("expected wall-clock-expired reflect to be skipped, got %+v", ri)
	}
}

func TestStatusMirror_GetReflectFiltersByKind(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	now := time.Now()

	// Apply a PHYSICAL reflect (WEAPON_COUNTER) to monster 100.
	m.OnApplied(tm, 100, StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"WEAPON_COUNTER": 1},
		Duration:         60000,
		ReflectKind:      "PHYSICAL",
		ReflectPercent:   25,
		ReflectMaxDamage: 1234,
	}, now)

	// Apply a MAGICAL reflect (MAGIC_COUNTER) to the same monster 100.
	m.OnApplied(tm, 100, StatusEffectAppliedBody{
		EffectId:         uuid.NewString(),
		Statuses:         map[string]int32{"MAGIC_COUNTER": 1},
		Duration:         60000,
		ReflectKind:      "MAGICAL",
		ReflectPercent:   40,
		ReflectMaxDamage: 5678,
	}, now)

	ri, ok := m.GetReflect(tm, 100, "PHYSICAL")
	if !ok {
		t.Fatalf("expected PHYSICAL reflect present")
	}
	if ri.Kind != "PHYSICAL" || ri.Percent != 25 || ri.MaxDamage != 1234 {
		t.Fatalf("PHYSICAL lookup returned wrong entry: %+v", ri)
	}

	ri, ok = m.GetReflect(tm, 100, "MAGICAL")
	if !ok {
		t.Fatalf("expected MAGICAL reflect present")
	}
	if ri.Kind != "MAGICAL" || ri.Percent != 40 || ri.MaxDamage != 5678 {
		t.Fatalf("MAGICAL lookup returned wrong entry: %+v", ri)
	}

	if ri, ok := m.GetReflect(tm, 100, "BOGUS"); ok {
		t.Fatalf("expected miss for BOGUS kind, got %+v", ri)
	}
}

func TestStatusMirror_OnExpiredRemovesEntry(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	eid := uuid.NewString()
	now := time.Now()
	body := StatusEffectAppliedBody{
		EffectId:       eid,
		Statuses:       map[string]int32{"MAGIC_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    "MAGIC",
		ReflectPercent: 30,
	}
	m.OnApplied(tm, 7, body, now)
	if _, ok := m.GetReflect(tm, 7, "MAGIC"); !ok {
		t.Fatalf("precondition: expected reflect present")
	}
	m.OnExpired(tm, 7, eid)
	if _, ok := m.GetReflect(tm, 7, "MAGIC"); ok {
		t.Fatalf("expected reflect removed after expire")
	}
}

func TestStatusMirror_OnCancelledRemovesEntry(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	eid := uuid.NewString()
	now := time.Now()
	body := StatusEffectAppliedBody{
		EffectId:       eid,
		Statuses:       map[string]int32{"MAGIC_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    "MAGIC",
		ReflectPercent: 30,
	}
	m.OnApplied(tm, 7, body, now)
	m.OnCancelled(tm, 7, eid)
	if _, ok := m.GetReflect(tm, 7, "MAGIC"); ok {
		t.Fatalf("expected reflect removed after cancel")
	}
}

func TestStatusMirror_OnMonsterGoneClearsAllEntries(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	now := time.Now()
	reflectId := uuid.NewString()
	venomId := uuid.NewString()
	m.OnApplied(tm, 7, StatusEffectAppliedBody{
		EffectId:       reflectId,
		Statuses:       map[string]int32{"MAGIC_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    "MAGIC",
		ReflectPercent: 30,
	}, now)
	m.OnApplied(tm, 7, StatusEffectAppliedBody{
		EffectId: venomId,
		Statuses: map[string]int32{"VENOM": 25},
		Duration: 30000,
	}, now)

	if _, ok := m.GetReflect(tm, 7, "MAGIC"); !ok {
		t.Fatalf("precondition: expected reflect present")
	}
	if c := m.VenomCount(tm, 7); c != 1 {
		t.Fatalf("precondition: expected venom 1, got %d", c)
	}

	m.OnMonsterGone(tm, 7)

	if _, ok := m.GetReflect(tm, 7, "MAGIC"); ok {
		t.Fatalf("expected reflect cleared on monster-gone")
	}
	if c := m.VenomCount(tm, 7); c != 0 {
		t.Fatalf("expected venom 0 on monster-gone, got %d", c)
	}
}

func TestStatusMirror_VenomCountTracksMultipleApplies(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		m.OnApplied(tm, 7, StatusEffectAppliedBody{
			EffectId: uuid.NewString(),
			Statuses: map[string]int32{"VENOM": int32(10 + i)},
			Duration: 30000,
		}, now)
	}
	if c := m.VenomCount(tm, 7); c != 3 {
		t.Fatalf("expected 3 venom stacks, got %d", c)
	}
}

func TestStatusMirror_MultiTenantIsolation(t *testing.T) {
	m := newTestStatusMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)
	now := time.Now()
	m.OnApplied(t1, 7, StatusEffectAppliedBody{
		EffectId:       uuid.NewString(),
		Statuses:       map[string]int32{"MAGIC_REFLECT": 1},
		Duration:       60000,
		ReflectKind:    "MAGIC",
		ReflectPercent: 30,
	}, now)

	if _, ok := m.GetReflect(t2, 7, "MAGIC"); ok {
		t.Fatalf("tenant isolation: t2 should not see t1's reflect")
	}
}

func TestStatusMirror_ConcurrentReadsWrites(t *testing.T) {
	m := newTestStatusMirror()
	tm := newTestTenant(t)
	now := time.Now()

	const writers = 8
	const readers = 8
	const iter = 200

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for i := 0; i < writers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iter; j++ {
				eid := uuid.NewString()
				uniqueId := uint32(id%4 + 1)
				m.OnApplied(tm, uniqueId, StatusEffectAppliedBody{
					EffectId:       eid,
					Statuses:       map[string]int32{"VENOM": 10},
					Duration:       30000,
					ReflectKind:    "MAGIC",
					ReflectPercent: 30,
				}, now)
				if j%5 == 0 {
					m.OnExpired(tm, uniqueId, eid)
				}
				if j%17 == 0 {
					m.OnMonsterGone(tm, uniqueId)
				}
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iter; j++ {
				uniqueId := uint32(id%4 + 1)
				_, _ = m.GetReflect(tm, uniqueId, "MAGIC")
				_ = m.VenomCount(tm, uniqueId)
			}
		}(i)
	}
	wg.Wait()
}

func TestStatusMirror_Singleton(t *testing.T) {
	resetStatusMirror()
	a := GetStatusMirror()
	b := GetStatusMirror()
	if a == nil || a != b {
		t.Fatalf("singleton not stable")
	}
}
