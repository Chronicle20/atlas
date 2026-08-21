package maplelife

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

func openEntry(at time.Time) Entry {
	return Entry{
		WorldId: world.Id(0),
		ItemId:  item.Id(5420000),
		Slot:    slot.Position(0),
		Phase:   PhaseOpen,
		At:      at,
	}
}

func TestPutThenGet(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	r.Put(a, 7, openEntry(time.Now()))

	e, ok := r.Get(a, 7)
	if !ok {
		t.Fatal("Get(a, 7): want hit")
	}
	if e.Phase != PhaseOpen {
		t.Errorf("entry = %+v", e)
	}
	if _, ok := r.Get(a, 8); ok {
		t.Error("Get(a, 8): want miss")
	}
	if _, ok := r.Get(b, 7); ok {
		t.Error("Get(b, 7): want miss")
	}
}

func TestPutIsIdempotentPerAccount(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	first := openEntry(time.Now().Add(-time.Minute))
	first.CandidateName = "First"
	second := openEntry(time.Now())
	second.CandidateName = "Second"

	r.Put(a, 7, first)
	r.Put(a, 7, second)

	e, ok := r.Get(a, 7)
	if !ok {
		t.Fatal("Get: want hit")
	}
	if e.CandidateName != "Second" {
		t.Errorf("entry = %+v, want the second Put's values to win", e)
	}
}

func TestTakeRemoves(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	r.Put(a, 7, openEntry(time.Now()))

	e, ok := r.Take(a, 7)
	if !ok {
		t.Fatal("first Take: want hit")
	}
	if e.Phase != PhaseOpen {
		t.Errorf("entry = %+v", e)
	}
	if _, ok := r.Take(a, 7); ok {
		t.Error("second Take: want miss — a CREATED and a FAILED racing must consume exactly once")
	}
}

func TestSubmitTransitionsPhase(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	putAt := time.Now().Add(-time.Minute)
	r.Put(a, 7, openEntry(putAt))

	before := time.Now()
	e, ok := r.Submit(a, 7, "tx-1", "Chronicle")
	after := time.Now()
	if !ok {
		t.Fatal("Submit: want hit")
	}
	if e.Phase != PhaseSubmitted || e.TransactionId != "tx-1" || e.CandidateName != "Chronicle" {
		t.Errorf("returned entry = %+v", e)
	}
	if e.At.Before(before) || e.At.After(after) {
		t.Errorf("At = %v, want refreshed to between %v and %v (not left at the original Put time %v)", e.At, before, after, putAt)
	}

	stored, ok := r.Get(a, 7)
	if !ok {
		t.Fatal("Get after Submit: want hit")
	}
	if stored.Phase != PhaseSubmitted || stored.TransactionId != "tx-1" || stored.CandidateName != "Chronicle" {
		t.Errorf("stored entry = %+v", stored)
	}
}

func TestSubmitWithoutOpenFails(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()

	e, ok := r.Submit(a, 999, "tx-2", "Nobody")
	if ok {
		t.Fatalf("Submit without a prior Open: want miss, got %+v", e)
	}
	if _, ok := r.Get(a, 999); ok {
		t.Error("Submit without a prior Open must not store anything")
	}
}

func TestTakeByTransactionId(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	r.Put(a, 7, openEntry(time.Now()))
	if _, ok := r.Submit(a, 7, "tx-1", "Chronicle"); !ok {
		t.Fatal("Submit: want hit")
	}

	accountId, e, ok := r.TakeByTransactionId(a, "tx-1")
	if !ok {
		t.Fatal("TakeByTransactionId: want hit")
	}
	if accountId != 7 {
		t.Errorf("accountId = %d, want 7", accountId)
	}
	if e.TransactionId != "tx-1" {
		t.Errorf("entry = %+v", e)
	}

	if _, _, ok := r.TakeByTransactionId(a, "tx-1"); ok {
		t.Error("second TakeByTransactionId: want miss")
	}

	if _, _, ok := r.TakeByTransactionId(a, ""); ok {
		t.Error(`TakeByTransactionId(a, ""): want miss, must not match an entry whose TransactionId is also empty`)
	}
}

func TestTakeByTransactionIdIsTenantScoped(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearAccount(a, 7)
		r.ClearAccount(b, 7)
	})

	r.Put(a, 7, openEntry(time.Now()))
	r.Submit(a, 7, "tx-1", "A")
	r.Put(b, 7, openEntry(time.Now()))
	r.Submit(b, 7, "tx-1", "B")

	if _, _, ok := r.TakeByTransactionId(a, "tx-1"); !ok {
		t.Fatal("TakeByTransactionId(a): want hit")
	}

	e, ok := r.Get(b, 7)
	if !ok {
		t.Fatal("tenant B's entry must survive tenant A's TakeByTransactionId")
	}
	if e.CandidateName != "B" {
		t.Errorf("tenant B entry = %+v", e)
	}
}

func TestClearAccount(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()

	r.Put(a, 7, openEntry(time.Now()))
	r.ClearAccount(a, 7)

	if _, ok := r.Get(a, 7); ok {
		t.Error("ClearAccount: want subsequent Get to miss")
	}
}

func TestSweepUsesPhaseSpecificTTL(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearAccount(a, 100)
		r.ClearAccount(a, 200)
		r.ClearAccount(a, 300)
		r.ClearAccount(a, 400)
	})

	now := time.Now()

	x := openEntry(now.Add(-(OpenTTL + time.Second)))
	y := openEntry(now.Add(-time.Minute))
	z := openEntry(now.Add(-(SubmittedTTL + time.Second)))
	z.Phase = PhaseSubmitted
	w := openEntry(now.Add(-5 * time.Second))
	w.Phase = PhaseSubmitted

	r.Put(a, 100, x)
	r.Put(a, 200, y)
	r.Put(a, 300, z)
	r.Put(a, 400, w)

	expired := r.Sweep(a, now)
	if len(expired) != 2 {
		t.Fatalf("Sweep = %+v, want exactly 2 entries (X and Z)", expired)
	}
	got := map[uint32]bool{}
	for _, ex := range expired {
		got[ex.AccountId] = true
	}
	if !got[100] || !got[300] {
		t.Fatalf("Sweep expired accounts = %+v, want 100 and 300", got)
	}

	if _, ok := r.Get(a, 100); ok {
		t.Error("account 100 (expired OPEN) should have been removed")
	}
	if _, ok := r.Get(a, 200); !ok {
		t.Error("account 200 (fresh OPEN, aged 1min < OpenTTL) should still be present")
	}
	if _, ok := r.Get(a, 300); ok {
		t.Error("account 300 (expired SUBMITTED) should have been removed")
	}
	if _, ok := r.Get(a, 400); !ok {
		t.Error("account 400 (fresh SUBMITTED, aged 5s < SubmittedTTL) should still be present")
	}
}

func TestSweepIsTenantScoped(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearAccount(a, 10)
		r.ClearAccount(b, 20)
	})

	now := time.Now()
	r.Put(a, 10, openEntry(now.Add(-(OpenTTL + time.Second))))
	r.Put(b, 20, openEntry(now.Add(-(OpenTTL + time.Second))))

	expiredA := r.Sweep(a, now)
	if len(expiredA) != 1 || expiredA[0].AccountId != 10 {
		t.Fatalf("tenant A Sweep = %+v, want exactly its own account 10", expiredA)
	}

	expiredB := r.Sweep(b, now)
	if len(expiredB) != 1 || expiredB[0].AccountId != 20 {
		t.Fatalf("tenant B Sweep = %+v, want exactly its own account 20 — it must not have been stolen by tenant A's sweep", expiredB)
	}
}

func TestSubmittedTTLOutlivesSagaTimeout(t *testing.T) {
	if SubmittedTTL <= 10*time.Second {
		t.Fatalf("SubmittedTTL = %v, must be > 10s so a timed-out saga's FAILED still finds its record", SubmittedTTL)
	}
}
