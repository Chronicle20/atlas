package saga

// timer_stale_snapshot_test.go pins the post-guard re-read in handleSagaTimeout.
//
// handleSagaTimeout used to read the saga ONCE, before taking the
// Pending → Compensating guard, and then drive both the compensation
// reverse-walk and the Failed emission from that pre-guard snapshot. A step
// whose normal completion landed in the window between the read and the guard
// is still Pending in the snapshot, so the walk skips its inverse — and no late
// event can ever repair it, because the normal completion path already consumed
// the event. The trade settlement saga is the first saga type to route items in
// BOTH directions plus meso through that walk, so a skipped inverse is
// dupe-capable there.
//
// The window is not directly reachable from a test (the read and the guard are
// adjacent statements), so this drives it through the Cache seam: a decorator
// answers the FIRST GetById with the stale copy and every later one from the
// real store. With the re-read in place everything after the guard sees the
// completed step; without it, it does not. `failedStep` is the observable —
// EmitSagaFailed derives it from GetCurrentStep on the same variable the three
// walks are dispatched from, three lines above.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// staleFirstReadCache answers the first GetById with a pre-supplied snapshot and
// delegates every other call — including the lifecycle transitions — to the real
// cache. That models exactly one thing: a step completing between
// handleSagaTimeout's read and its guard.
type staleFirstReadCache struct {
	Cache
	stale Saga
	reads int
}

func (c *staleFirstReadCache) GetById(ctx context.Context, transactionId uuid.UUID) (Saga, bool) {
	c.reads++
	if c.reads == 1 {
		return c.stale, true
	}
	return c.Cache.GetById(ctx, transactionId)
}

func TestSagaTimeoutRereadsAfterTakingTheCompensatingGuard(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	tx := uuid.New()

	// The authoritative (post-window) saga: step one has ALREADY completed.
	fresh, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("timeout-reread-test").
		AddStep("step_one", Completed, AwardMesos, AwardMesosPayload{CharacterId: 1, Amount: -10}).
		AddStep("step_two", Pending, AwardMesos, AwardMesosPayload{CharacterId: 2, Amount: 10}).
		Build()
	if err != nil {
		t.Fatalf("build fresh: %v", err)
	}

	// The pre-guard snapshot: step one still reads as Pending.
	stale, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("timeout-reread-test").
		AddStep("step_one", Pending, AwardMesos, AwardMesosPayload{CharacterId: 1, Amount: -10}).
		AddStep("step_two", Pending, AwardMesos, AwardMesosPayload{CharacterId: 2, Amount: 10}).
		Build()
	if err != nil {
		t.Fatalf("build stale: %v", err)
	}

	ResetCache()
	t.Cleanup(ResetCache)
	real := GetCache()
	if err := real.Put(ctx, fresh); err != nil {
		t.Fatalf("put: %v", err)
	}
	SetCache(&staleFirstReadCache{Cache: real, stale: stale})

	var failedStep string
	origEmit := emitSagaFailedByIdsFn
	emitSagaFailedByIdsFn = func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, _ string, _, _ uint32, _, _, step string) error {
		failedStep = step
		return nil
	}
	t.Cleanup(func() { emitSagaFailedByIdsFn = origEmit })

	handleSagaTimeout(logger, ctx, tx, 30*time.Second)

	if failedStep != "step_two" {
		t.Fatalf("failedStep: got %q, want %q — the post-guard walk and emission read the STALE snapshot, "+
			"in which step_one is still Pending; its inverse would never be dispatched and no late event will "+
			"arrive to repair it", failedStep, "step_two")
	}
}
