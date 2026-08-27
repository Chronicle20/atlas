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

// submittedEntry builds a complete PhaseSubmitted entry, the shape
// handleMapleLifeCreate writes on a successful factory call -- entries are
// no longer created any other way (bug-543-is-the-submit-not-the-open.md;
// PhaseOpen and OpenTTL are gone).
func submittedEntry(at time.Time) Entry {
	return Entry{
		WorldId:       world.Id(0),
		ItemId:        item.Id(5420000),
		Slot:          slot.Position(0),
		Phase:         PhaseSubmitted,
		TransactionId: "tx-1",
		CandidateName: "Chronicle",
		At:            at,
	}
}

func TestPutThenGet(t *testing.T) {
	a, b := mustTenant(t), mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	r.Put(a, 7, submittedEntry(time.Now()))

	e, ok := r.Get(a, 7)
	if !ok {
		t.Fatal("Get(a, 7): want hit")
	}
	if e.Phase != PhaseSubmitted {
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

	first := submittedEntry(time.Now().Add(-time.Minute))
	first.CandidateName = "First"
	second := submittedEntry(time.Now())
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

	r.Put(a, 7, submittedEntry(time.Now()))

	e, ok := r.Take(a, 7)
	if !ok {
		t.Fatal("first Take: want hit")
	}
	if e.Phase != PhaseSubmitted {
		t.Errorf("entry = %+v", e)
	}
	if _, ok := r.Take(a, 7); ok {
		t.Error("second Take: want miss — a CREATED and a FAILED racing must consume exactly once")
	}
}

func TestTakeByTransactionId(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() { r.ClearAccount(a, 7) })

	r.Put(a, 7, submittedEntry(time.Now()))

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

	eA := submittedEntry(time.Now())
	eA.CandidateName = "A"
	r.Put(a, 7, eA)
	eB := submittedEntry(time.Now())
	eB.CandidateName = "B"
	r.Put(b, 7, eB)

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

	r.Put(a, 7, submittedEntry(time.Now()))
	r.ClearAccount(a, 7)

	if _, ok := r.Get(a, 7); ok {
		t.Error("ClearAccount: want subsequent Get to miss")
	}
}

func TestSweepUsesSubmittedTTL(t *testing.T) {
	a := mustTenant(t)
	r := GetRegistry()
	t.Cleanup(func() {
		r.ClearAccount(a, 300)
		r.ClearAccount(a, 400)
	})

	now := time.Now()

	expiredEntry := submittedEntry(now.Add(-(SubmittedTTL + time.Second)))
	freshEntry := submittedEntry(now.Add(-5 * time.Second))

	r.Put(a, 300, expiredEntry)
	r.Put(a, 400, freshEntry)

	expired := r.Sweep(a, now)
	if len(expired) != 1 {
		t.Fatalf("Sweep = %+v, want exactly 1 entry (account 300)", expired)
	}
	if expired[0].AccountId != 300 {
		t.Fatalf("Sweep expired account = %d, want 300", expired[0].AccountId)
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
	r.Put(a, 10, submittedEntry(now.Add(-(SubmittedTTL + time.Second))))
	r.Put(b, 20, submittedEntry(now.Add(-(SubmittedTTL + time.Second))))

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
