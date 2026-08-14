package remotemerchant

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

func TestTake_ReturnsAndRemovesOnce(t *testing.T) {
	ten := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearCharacter(ten, 1234) })

	r.Put(ten, 1234, Entry{ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now()})

	e, ok := r.Take(ten, 1234)
	if !ok {
		t.Fatal("Take: want hit")
	}
	if e.ItemId != item.Id(5450000) || e.Slot != slot.Position(3) {
		t.Errorf("entry = %+v", e)
	}
	if _, ok := r.Take(ten, 1234); ok {
		t.Error("second Take: want miss — an entry must unlock exactly once")
	}
}

func TestTake_MissForUnknownCharacter(t *testing.T) {
	ten := mustTenant(t)
	if _, ok := GetRegistry().Take(ten, 999999); ok {
		t.Error("Take on an unregistered character: want miss (this is what keeps the NPC-talk path byte-identical)")
	}
}

func TestTake_TenantScoped(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearCharacter(a, 1234) })

	r.Put(a, 1234, Entry{ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now()})
	if _, ok := r.Take(b, 1234); ok {
		t.Error("Take from another tenant: want miss")
	}
}

func TestSweep_EvictsExpiredOnly(t *testing.T) {
	ten := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearCharacter(ten, 1)
		r.ClearCharacter(ten, 2)
	})

	now := time.Now()
	r.Put(ten, 1, Entry{ItemId: item.Id(5450000), Slot: slot.Position(1), At: now.Add(-2 * TTL)})
	r.Put(ten, 2, Entry{ItemId: item.Id(5450000), Slot: slot.Position(2), At: now})

	expired := r.Sweep(ten, now)
	if len(expired) != 1 || expired[0].CharacterId != 1 {
		t.Fatalf("Sweep = %+v, want exactly character 1", expired)
	}
	if _, ok := r.Take(ten, 1); ok {
		t.Error("swept entry still present")
	}
	if _, ok := r.Take(ten, 2); !ok {
		t.Error("fresh entry was swept")
	}
}

// TestSweep_MultiTenantSweepersDoNotStealEachOthersEntries reproduces the
// round-2 code review finding: on a pod serving more than one tenant,
// atlas-channel runs one sweep goroutine per (tenant, world, channel)
// listener key against this single shared Registry. Before Sweep took a
// tenant argument, whichever goroutine fired first drained EVERY tenant's
// expired entries — including ones belonging to a tenant it had no business
// touching — so the owning tenant's own sweeper found nothing left to
// unlock. This test simulates exactly that: tenant A's sweeper (calling
// Sweep(a, now)) must never observe, remove, or steal tenant B's expired
// entry, and tenant B's own sweeper must still find it.
func TestSweep_MultiTenantSweepersDoNotStealEachOthersEntries(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearCharacter(a, 10)
		r.ClearCharacter(b, 20)
	})

	now := time.Now()
	r.Put(a, 10, Entry{ItemId: item.Id(5450000), Slot: slot.Position(1), At: now.Add(-2 * TTL)})
	r.Put(b, 20, Entry{ItemId: item.Id(5450000), Slot: slot.Position(2), At: now.Add(-2 * TTL)})

	// Tenant A's sweeper fires first (as in the failure scenario) but must
	// only ever collect its own tenant's expired entries.
	expiredA := r.Sweep(a, now)
	if len(expiredA) != 1 || expiredA[0].CharacterId != 10 {
		t.Fatalf("tenant A Sweep = %+v, want exactly its own character 10", expiredA)
	}

	// Tenant B's entry must have survived tenant A's sweep untouched, so
	// tenant B's own sweeper (which may fire on a later tick, or never fire
	// at all before this test's assertions run) still finds and evicts it.
	expiredB := r.Sweep(b, now)
	if len(expiredB) != 1 || expiredB[0].CharacterId != 20 {
		t.Fatalf("tenant B Sweep = %+v, want exactly its own character 20 — it must not have been stolen by tenant A's sweep", expiredB)
	}
}
