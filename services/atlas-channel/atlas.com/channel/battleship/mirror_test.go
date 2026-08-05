package battleship

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestRideMirrorLifecycle(t *testing.T) {
	m := GetRideMirror()
	t1 := testTenant(t)
	t2 := testTenant(t)
	t.Cleanup(func() { m.EvictTenant(t1.Id()); m.EvictTenant(t2.Id()) })

	if _, ok := m.Get(t1, 100); ok {
		t.Fatal("empty mirror should miss")
	}

	m.Put(t1, 100, RideState{SkillLevel: 7, StateTTL: 35 * time.Minute})
	rs, ok := m.Get(t1, 100)
	if !ok || rs.SkillLevel != 7 || rs.StateTTL != 35*time.Minute {
		t.Fatalf("Get = (%+v, %v), want skillLevel 7 ttl 35m", rs, ok)
	}

	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant isolation violated: t2 sees t1's rider")
	}

	m.Remove(t1, 100)
	if _, ok := m.Get(t1, 100); ok {
		t.Fatal("Remove did not clear the entry")
	}
	m.Remove(t1, 100) // idempotent

	m.Put(t1, 200, RideState{SkillLevel: 1})
	m.EvictTenant(t1.Id())
	if _, ok := m.Get(t1, 200); ok {
		t.Fatal("EvictTenant did not clear entries")
	}
}
