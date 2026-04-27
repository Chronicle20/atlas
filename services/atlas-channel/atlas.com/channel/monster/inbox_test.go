package monster

import (
	"sync"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// resetInbox resets the singleton for test isolation. Test-only.
func resetInbox() {
	nextSkillInboxOnce = sync.Once{}
	nextSkillInboxInst = nil
}

func TestInbox_PutThenTakeAndClear(t *testing.T) {
	resetInbox()
	InitNextSkillInbox()
	tm := newTestTenant(t)
	d := Decision{SkillId: 100, SkillLevel: 1, DecidedAtMs: 12345}

	GetNextSkillInbox().Put(tm, 7, d)
	got, ok := GetNextSkillInbox().TakeAndClear(tm, 7)
	if !ok {
		t.Fatalf("expected hit on first take")
	}
	if got != d {
		t.Fatalf("got %+v want %+v", got, d)
	}
	if _, ok2 := GetNextSkillInbox().TakeAndClear(tm, 7); ok2 {
		t.Fatalf("expected miss after clear")
	}
}

func TestInbox_PutOverwritesLastWriterWins(t *testing.T) {
	resetInbox()
	InitNextSkillInbox()
	tm := newTestTenant(t)
	GetNextSkillInbox().Put(tm, 7, Decision{SkillId: 100})
	GetNextSkillInbox().Put(tm, 7, Decision{SkillId: 200})

	got, ok := GetNextSkillInbox().TakeAndClear(tm, 7)
	if !ok || got.SkillId != 200 {
		t.Fatalf("expected last-writer-wins (200); got ok=%v skill=%d", ok, got.SkillId)
	}
}

func TestInbox_Evict(t *testing.T) {
	resetInbox()
	InitNextSkillInbox()
	tm := newTestTenant(t)
	GetNextSkillInbox().Put(tm, 7, Decision{SkillId: 100})
	GetNextSkillInbox().Evict(tm, 7)

	if _, ok := GetNextSkillInbox().TakeAndClear(tm, 7); ok {
		t.Fatalf("expected miss after Evict")
	}
}

func TestInbox_MultiTenantIsolation(t *testing.T) {
	resetInbox()
	InitNextSkillInbox()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)

	GetNextSkillInbox().Put(t1, 7, Decision{SkillId: 100})
	GetNextSkillInbox().Put(t2, 7, Decision{SkillId: 200})

	got1, _ := GetNextSkillInbox().TakeAndClear(t1, 7)
	got2, _ := GetNextSkillInbox().TakeAndClear(t2, 7)
	if got1.SkillId != 100 || got2.SkillId != 200 {
		t.Fatalf("tenants leaked: t1=%d t2=%d", got1.SkillId, got2.SkillId)
	}
}
