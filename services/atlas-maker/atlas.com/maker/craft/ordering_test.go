package craft_test

import (
	"atlas-maker/craft"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// tenantContext is testContext with a caller-chosen tenant id, so a test can
// still reach craftGuard's own entry afterward -- testContext's tenant id is
// a fresh uuid.New() the caller never sees.
func tenantContext(t *testing.T, tenantId uuid.UUID) context.Context {
	t.Helper()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), te)
}

// TestGuardIsReleasedEvenWhenTerminalEventBeatsTrack exercises the ordering
// the review flagged in processor.go's emit: a terminal event delivered
// synchronously during the produce call -- standing in for
// atlas-saga-orchestrator's terminal event being consumed by this same
// service before Emit ever returns to its caller -- must still find (and
// clear) the guard entry Create is about to leave held.
//
// That only holds if the transaction id was Tracked before Emit ran. Against
// track-after-Emit code this test fails: the release fires while nothing has
// been Tracked yet (a no-op), and the eventual post-Emit Track then installs
// a mapping nothing will ever release again, so AcquireForTest below would
// see the entry still held.
func TestGuardIsReleasedEvenWhenTerminalEventBeatsTrack(t *testing.T) {
	h, equipId, slot := disassembleHarness(t)
	tenantId := uuid.New()
	ctx := tenantContext(t, tenantId)

	d := newDeps()
	d.em.duringEmit = func(s saga.Saga) {
		craft.ReleaseInFlightByTransaction(tenantId, s.TransactionId)
	}

	p := buildCreateProcessorWithContext(t, ctx, h, d)

	txId, err := p.Create(h.characterId, craft.Request{
		Mode:        craft.ModeDisassemble,
		EquipItemId: equipId,
		SlotPos:     slot,
	})
	require.NoError(t, err)
	require.NotZero(t, txId)

	require.True(t, craft.AcquireForTest(tenantId, h.characterId),
		"a terminal event observed during Emit must find the mapping Track installed before Emit ran, and release the entry; it must not still be held afterward")
}

// TestEmitFailureDoesNotLeakGuardEntry covers the other half of the review's
// finding: when Emit fails after Track already ran (Track now always runs
// before Emit), no terminal event will ever arrive to release the mapping,
// so emit itself must unwind it rather than leaking an entry nothing will
// ever clear.
func TestEmitFailureDoesNotLeakGuardEntry(t *testing.T) {
	h, equipId, slot := disassembleHarness(t)
	tenantId := uuid.New()
	ctx := tenantContext(t, tenantId)

	d := newDeps()
	d.em.err = errors.New("produce failed")

	p := buildCreateProcessorWithContext(t, ctx, h, d)

	_, err := p.Create(h.characterId, craft.Request{
		Mode:        craft.ModeDisassemble,
		EquipItemId: equipId,
		SlotPos:     slot,
	})
	require.Error(t, err)

	require.True(t, craft.AcquireForTest(tenantId, h.characterId),
		"a failed Emit must not leak the guard entry it Tracked, since no terminal event will ever arrive to release it")
}
