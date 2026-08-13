package combo

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return m
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func testEligibility() Eligibility {
	return NewEligibility(skill.AranStage1ComboAbilityId, 20, 20)
}

func TestIncrementSeedsThenAdvances(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)

	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("first increment: want (1,true), got (%d,%v)", count, seeded)
	}
	count, seeded = m.Increment(tn, 7, testField(), DefaultIdleWindow, now.Add(time.Second))
	if count != 2 || seeded {
		t.Fatalf("second increment: want (2,false), got (%d,%v)", count, seeded)
	}
}

func TestIncrementClampsAtCap(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	// Reach the cap through the real exported path -- no test-only backdoor.
	for i := int32(0); i < ComboCap; i++ {
		m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	}

	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != ComboCap || seeded {
		t.Fatalf("at cap: want (%d,false), got (%d,%v)", ComboCap, count, seeded)
	}

	// A capped entry must still refresh LastHit on every hit: a player still
	// landing hits at the cap must not decay mid-combat. There is no
	// exported read path for LastHit outside the package, so assert this
	// behaviorally via ExpireIdle: refresh the hit far later, then confirm a
	// sweep that lands well within the NEW lastHit's window -- but well past
	// where the OLD lastHit's window would have expired it -- does not
	// expire the entry.
	later := now.Add(time.Minute)
	count, seeded = m.Increment(tn, 7, testField(), DefaultIdleWindow, later)
	if count != ComboCap || seeded {
		t.Fatalf("refreshing hit at cap: want (%d,false), got (%d,%v)", ComboCap, count, seeded)
	}

	sweepAt := later.Add(DefaultIdleWindow / 2)
	if sweepAt.Sub(now) <= DefaultIdleWindow {
		t.Fatalf("test setup invalid: sweepAt is not past the un-refreshed window, assertion below would be vacuous")
	}
	if got := m.ExpireIdle(sweepAt); len(got) != 0 {
		t.Fatalf("capped entry with a refreshed LastHit must not expire: got %d expiries", len(got))
	}
}

func TestEligibilityTTL(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	if _, ok := m.Eligibility(tn, 7, now.Add(59*time.Second), 60*time.Second); !ok {
		t.Error("within TTL: want fresh, got stale")
	}
	if _, ok := m.Eligibility(tn, 7, now.Add(61*time.Second), 60*time.Second); ok {
		t.Error("past TTL: want stale, got fresh")
	}
	if _, ok := m.Eligibility(tn, 99, now, 60*time.Second); ok {
		t.Error("unknown character: want miss, got hit")
	}
}

func TestClearRemovesEntry(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)
	m.Increment(tn, 7, testField(), DefaultIdleWindow, now)

	m.Clear(tn, 7)

	if _, ok := m.Eligibility(tn, 7, now, 60*time.Second); ok {
		t.Error("after Clear: want miss, got hit")
	}
	count, seeded := m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("after Clear the next increment re-seeds: want (1,true), got (%d,%v)", count, seeded)
	}
}

func TestExpireIdleZeroesOnlyStaleNonZeroEntries(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)

	// stale, count > 0 -> expires
	m.SetEligibility(tn, 1, testField(), testEligibility(), now)
	m.Increment(tn, 1, testField(), 3*time.Second, now)
	// fresh, count > 0 -> untouched
	m.SetEligibility(tn, 2, testField(), testEligibility(), now)
	m.Increment(tn, 2, testField(), 3*time.Second, now.Add(3*time.Second))
	// eligibility only, count == 0 -> emits nothing
	m.SetEligibility(tn, 3, testField(), testEligibility(), now)

	got := m.ExpireIdle(now.Add(4 * time.Second))

	if len(got) != 1 {
		t.Fatalf("want exactly 1 expiry, got %d", len(got))
	}
	if got[0].CharacterId() != 1 {
		t.Errorf("want character 1 expired, got %d", got[0].CharacterId())
	}
	if got[0].ComboId() != skill.AranStage1ComboAbilityId {
		t.Errorf("want combo id %d, got %d", skill.AranStage1ComboAbilityId, got[0].ComboId())
	}
	expiredTenant := got[0].Tenant()
	if expiredTenant.Id() != tn.Id() {
		t.Error("expired entry carries the wrong tenant")
	}
	// second sweep is empty: entry 1's count is already zero (idempotent, no
	// re-expiry), and entry 2 is still within its idle window at this point
	// (elapsed 2s < 3s window) so it stays untouched.
	if again := m.ExpireIdle(now.Add(5 * time.Second)); len(again) != 0 {
		t.Errorf("second sweep: want 0 expiries, got %d", len(again))
	}
}

func TestPerTenantIsolation(t *testing.T) {
	m := &Mirror{}
	a := testTenant(t)
	b := testTenant(t)
	now := time.Unix(1000, 0)

	m.SetEligibility(a, 7, testField(), testEligibility(), now)
	m.SetEligibility(b, 7, testField(), testEligibility(), now)
	m.Increment(a, 7, testField(), DefaultIdleWindow, now)
	m.Increment(a, 7, testField(), DefaultIdleWindow, now)

	count, _ := m.Increment(b, 7, testField(), DefaultIdleWindow, now)
	if count != 1 {
		t.Errorf("tenant b must not see tenant a's count: want 1, got %d", count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := &Mirror{}
	tn := testTenant(t)
	now := time.Unix(1000, 0)
	m.SetEligibility(tn, 7, testField(), testEligibility(), now)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Increment(tn, 7, testField(), DefaultIdleWindow, now)
			m.Eligibility(tn, 7, now, 60*time.Second)
			m.ExpireIdle(now)
		}()
	}
	wg.Wait()
}
