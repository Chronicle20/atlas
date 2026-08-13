package saga

// trade_compensation_test.go — the reverse-walk arm of task-205 settlement.
//
// The invariant under test is CONSERVATION: a settlement that fails partway
// must leave both participants exactly where they started. Without the
// reverse-walk the per-action switch in CompensateFailedStep routes these steps
// to compensateStorageOperation, which performs no rollback at all — so
// "release A, release B, accept→B, accept→A fails" would leave A's item
// soft-deleted and B holding both.
//
// The tests drive DispatchTradeTransactionRollbacks directly, avoiding the
// EmitSagaFailed Kafka path (no broker in the test environment), mirroring
// mts_dupe_safety_test.go.

import (
	charactermock "atlas-saga-orchestrator/character/mock"
	compartmentmock "atlas-saga-orchestrator/compartment/mock"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	tradeCharA = uint32(100)
	tradeCharB = uint32(200)
	// A stages 5x 2000000 (USE, asset 55); B stages 1x 1302000 (EQUIP, asset 77).
	tradeTemplateA = uint32(2000000)
	tradeTemplateB = uint32(1302000)
	tradeAssetA    = uint32(55)
	tradeAssetB    = uint32(77)
	tradeTypeA     = byte(2)
	tradeTypeB     = byte(1)
)

// The escrow rows the two staged items live in. Fixed rather than random so a
// failing assertion names a recognisable id.
var (
	tradeEscrowA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000055")
	tradeEscrowB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000077")
)

// tradeRegrantCall captures a RequestAcceptAsset (re-grant) dispatch.
type tradeRegrantCall struct {
	CharacterId   uint32
	InventoryType byte
	TemplateId    uint32
	AssetData     asset2.AssetData
}

// tradeDestroyCall captures a RequestDestroyItem (un-accept) dispatch.
type tradeDestroyCall struct {
	CharacterId uint32
	TemplateId  uint32
	Quantity    uint32
	RemoveAll   bool
}

// tradeMesoCall captures an AwardMesosAndEmit (meso reversal) dispatch.
type tradeMesoCall struct {
	CharacterId uint32
	Amount      int32
}

// tradeRollbackHarness wires the mocks and returns the recorded dispatches.
type tradeRollbackHarness struct {
	compensator Compensator
	tctx        context.Context
	regrants    *[]tradeRegrantCall
	destroys    *[]tradeDestroyCall
	mesos       *[]tradeMesoCall
	tradeMock   *stagingTradeMock
}

func newTradeRollbackHarness(t *testing.T) tradeRollbackHarness {
	t.Helper()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	tctx := tenant.WithContext(context.Background(), te)

	regrants := &[]tradeRegrantCall{}
	destroys := &[]tradeDestroyCall{}
	mesos := &[]tradeMesoCall{}

	compMock := &compartmentmock.ProcessorMock{
		RequestAcceptAssetFunc: func(_ uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error {
			*regrants = append(*regrants, tradeRegrantCall{CharacterId: characterId, InventoryType: inventoryType, TemplateId: templateId, AssetData: assetData})
			return nil
		},
		RequestDestroyItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
			*destroys = append(*destroys, tradeDestroyCall{CharacterId: characterId, TemplateId: templateId, Quantity: quantity, RemoveAll: removeAll})
			return nil
		},
	}
	charMock := &charactermock.ProcessorMock{
		AwardMesosAndEmitFunc: func(_ uuid.UUID, _ channel.Model, characterId uint32, _ uint32, _ string, amount int32, _ bool) error {
			*mesos = append(*mesos, tradeMesoCall{CharacterId: characterId, Amount: amount})
			return nil
		},
	}

	tradeMock := &stagingTradeMock{}

	return tradeRollbackHarness{
		compensator: NewCompensator(logger, tctx).WithCompartmentProcessor(compMock).WithCharacterProcessor(charMock).WithTradeProcessor(tradeMock),
		tctx:        tctx,
		regrants:    regrants,
		destroys:    destroys,
		mesos:       mesos,
		tradeMock:   tradeMock,
	}
}

// tradeSettlementSagaBuilder reproduces the step ids and payloads
// expandTradeSettlement emits for the two-sided fixture, so the reverse-walk is
// exercised against the real forward shape. Statuses are supplied per test.
func tradeSettlementSagaBuilder(transactionId uuid.UUID, releaseA, releaseB, acceptB, acceptA, deduct, credit Status) *Builder {
	return NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(TradeTransaction).
		SetInitiatedBy("trade-compensation-test").
		AddStep("release_from_trade_100_55", releaseA, ReleaseFromTrade, ReleaseFromTradePayload{
			TransactionId: transactionId, EscrowId: tradeEscrowA,
		}).
		AddStep("release_from_trade_200_77", releaseB, ReleaseFromTrade, ReleaseFromTradePayload{
			TransactionId: transactionId, EscrowId: tradeEscrowB,
		}).
		AddStep("accept_to_character_200_55", acceptB, AcceptToCharacter, AcceptToCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharB, InventoryType: tradeTypeA, TemplateId: tradeTemplateA,
			AssetData: asset2.AssetData{Quantity: 5, Owner: "Chronicle"},
		}).
		AddStep("accept_to_character_100_77", acceptA, AcceptToCharacter, AcceptToCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharA, InventoryType: tradeTypeB, TemplateId: tradeTemplateB,
			AssetData: asset2.AssetData{Quantity: 1, Owner: "Chronicle", WeaponAttack: 17},
		}).
		AddStep("award_mesos_credit_200", credit, AwardMesos, AwardMesosPayload{
			CharacterId: tradeCharB, WorldId: 1, ChannelId: 1, ActorId: tradeCharA, ActorType: "CHARACTER", Amount: 9_600_000,
		})
}

// stageTradeSaga builds the saga, parks it in the cache (claimLateCompensation
// reads it from there) and moves the lifecycle to Compensating.
func stageTradeSaga(t *testing.T, h tradeRollbackHarness, transactionId uuid.UUID, b *Builder) Saga {
	t.Helper()
	s, err := b.Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(h.tctx, s))
	require.True(t, GetCache().TryTransition(h.tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))
	t.Cleanup(func() { GetCache().Remove(h.tctx, transactionId) })
	return s
}

// TestTradeRollback_AcceptFailsAfterBothReleases is the half-swap shape: both
// items left escrow and one landed with B, then B→A's accept failed.
// Conservation requires B's newly received copy be destroyed AND both custody
// rows be restored — otherwise A's item is gone and B holds two.
//
// Note what the release inverse is now: a RESTORE, not a re-grant. Under
// escrow-at-staging the item was never in a compartment to give back — putting
// it back into custody is both sufficient and safer, because it cannot race the
// accept that may already have delivered the same item to the counterparty
// (design §5A.7).
func TestTradeRollback_AcceptFailsAfterBothReleases(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed, // both escrow releases done
		Completed, Failed, // accept→B done, accept→A failed
		Pending, Pending, // meso credit never ran
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	// The one completed accept is undone: B loses the copy it just received.
	require.Len(t, *h.destroys, 1)
	require.Equal(t, tradeDestroyCall{CharacterId: tradeCharB, TemplateId: tradeTemplateA, Quantity: 5, RemoveAll: false}, (*h.destroys)[0])

	// Both completed releases are undone: each custody row comes back.
	require.Len(t, h.tradeMock.restoreCalls, 2)
	require.ElementsMatch(t, []uuid.UUID{tradeEscrowA, tradeEscrowB}, h.tradeMock.restoreCalls)

	// Nothing is re-granted to a compartment: the items belong to escrow.
	require.Empty(t, *h.regrants, "a settlement rollback must restore custody, not re-grant to inventories")

	// No meso leg completed, so nothing to reverse.
	require.Empty(t, *h.mesos)
}

// TestTradeRollback_CreditReversedWhenTheSagaFailsAfterIt pins the meso half.
//
// Settlement is CREDIT-ONLY now — the debit happened at stage time — so the only
// meso inverse a rollback can owe is a negation of a credit that already landed.
// Failing to reverse it would leave the receiver holding meso for a trade that
// was rolled back.
func TestTradeRollback_CreditReversedWhenTheSagaFailsAfterIt(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Completed, // the whole item swap landed
		Pending, Completed, // the credit landed too
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	require.Len(t, *h.mesos, 1)
	require.Equal(t, tradeMesoCall{CharacterId: tradeCharB, Amount: -9_600_000}, (*h.mesos)[0],
		"the credit must be reversed with the opposite sign, and no debit re-applied")

	// The item swap is unwound too: both accepts destroyed, both rows restored.
	require.Len(t, *h.destroys, 2)
	require.Len(t, h.tradeMock.restoreCalls, 2)
}

// TestTradeRollback_FailureAfterFirstReleaseOnly pins the earliest failure
// shape: only A's row was released before the second release failed. Exactly one
// restore must fire — for A's row — and nothing may be destroyed.
func TestTradeRollback_FailureAfterFirstReleaseOnly(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Failed, // A's row released, B's release failed
		Pending, Pending,
		Pending, Pending,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	require.Empty(t, *h.destroys, "nothing was created, so nothing may be destroyed")
	require.Empty(t, *h.mesos)
	require.Len(t, h.tradeMock.restoreCalls, 1)
	require.Equal(t, tradeEscrowA, h.tradeMock.restoreCalls[0])
}

// TestTradeRollback_IsIdempotent pins that a second walk over the same saga
// dispatches nothing. The meso inverse is a negation and is NOT idempotent
// downstream, so a repeated walk would debit B 9,600,000 twice.
func TestTradeRollback_IsIdempotent(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Completed,
		Pending, Completed,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)
	firstRestores := len(h.tradeMock.restoreCalls)
	firstDestroys := len(*h.destroys)
	firstMesos := len(*h.mesos)
	require.NotZero(t, firstRestores)
	require.NotZero(t, firstDestroys)
	require.NotZero(t, firstMesos)

	h.compensator.DispatchTradeTransactionRollbacks(s)

	require.Len(t, h.tradeMock.restoreCalls, firstRestores, "a second walk must not restore again")
	require.Len(t, *h.destroys, firstDestroys, "a second walk must not destroy again")
	require.Len(t, *h.mesos, firstMesos, "a second walk must not re-credit again — meso negation is not idempotent")
}

// TestCompensateFailedStepRoutesTradeToTheReverseWalk pins the DISPATCH gate,
// not the walk: a TradeTransaction must reach compensateTradeTransaction rather
// than falling through the per-action switch to compensateStorageOperation,
// whose contract is "no rollback is performed". This is the gap that made the
// half-swap possible.
func TestCompensateFailedStepRoutesTradeToTheReverseWalk(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Failed,
		Pending, Pending,
	))

	// CompensateFailedStep emits the FAILED event at the end; with no broker the
	// emit errors, but the reverse-walk has already run by then. The assertion
	// is on the dispatched inverses, not the return value.
	_ = h.compensator.CompensateFailedStep(s)

	require.NotEmpty(t, h.tradeMock.restoreCalls, "a failed trade step must reach the reverse-walk, not compensateStorageOperation's no-op path")
	require.NotEmpty(t, *h.destroys)
}

// The asset-id step-id pairing tests that used to sit here were DELETED, not
// ported. They pinned the coupling that let the reverse-walk find the accept
// step holding a release's snapshot, which existed only because a settlement
// released from a CHARACTER and had to reconstruct the item to give it back.
// Under escrow-at-staging a settlement releases from custody, and its inverse is
// simply restoring that custody row (design §5A.7) — there is no snapshot to
// pair and nothing for those tests to assert.

// ---------------------------------------------------------------------------
// Late-success absorption (fix round 2).
//
// The reverse-walk only reverses steps that are Completed when it runs. A
// settlement that times out with a step IN FLIGHT therefore leaves that step
// unreversed — and when its success event lands afterwards, the effect is real.
// For accept_to_character that is a DUPLICATE (the walk already re-granted the
// asset to its owner); for release_from_character it is silent LOSS. Both must
// route through CompensateLateStep to a registered inverse.
// ---------------------------------------------------------------------------

// TestLateSuccessfulAcceptDestroysTheDuplicate pins the dupe shape: the walk
// re-granted asset 55 to A, then B's accept lands late. B's copy must be
// destroyed or the item exists twice.
func TestLateSuccessfulAcceptDestroysTheDuplicate(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	// accept_to_character_200_55 is still Pending when the timeout walk runs.
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Pending, Pending,
		Pending, Pending,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)
	require.Len(t, h.tradeMock.restoreCalls, 2, "the walk restores both custody rows")
	require.Empty(t, *h.destroys)

	// Now the in-flight accept succeeds, after the saga went terminal.
	lateStep, ok := s.StepAt(2)
	require.True(t, ok)
	require.Equal(t, "accept_to_character_200_55", lateStep.StepId())

	compensated, err := h.compensator.CompensateLateStep(s, lateStep)
	require.NoError(t, err)
	require.True(t, compensated, "a late accept in a trade saga must have a registered inverse")

	require.Len(t, *h.destroys, 1)
	require.Equal(t, tradeDestroyCall{CharacterId: tradeCharB, TemplateId: tradeTemplateA, Quantity: 5, RemoveAll: false}, (*h.destroys)[0])
}

// TestLateSuccessfulReleaseRestoresTheCustodyRow pins the loss shape: the walk skipped
// B's still-pending release, then it succeeds late and soft-deletes B's item
// with nothing to restore it.
func TestLateSuccessfulReleaseRestoresTheCustodyRow(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Pending, // B's release still in flight
		Pending, Pending,
		Pending, Pending,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)
	require.Len(t, h.tradeMock.restoreCalls, 1, "only A's completed release is reversed by the walk")

	lateStep, ok := s.StepAt(1)
	require.True(t, ok)
	require.Equal(t, "release_from_trade_200_77", lateStep.StepId())

	compensated, err := h.compensator.CompensateLateStep(s, lateStep)
	require.NoError(t, err)
	require.True(t, compensated, "a late release in a trade saga must have a registered inverse")

	require.Len(t, h.tradeMock.restoreCalls, 2)
	require.Equal(t, tradeEscrowB, h.tradeMock.restoreCalls[1], "B's custody row must be restored")
	require.Empty(t, *h.regrants, "a late release restores custody; it does not re-grant to a compartment")
}

// TestLateSuccessDoesNotDoubleCompensateAWalkedStep pins that the late path and
// the reverse-walk share one claim: a step the walk already reversed cannot be
// reversed again when its (duplicate) success event arrives.
func TestLateSuccessDoesNotDoubleCompensateAWalkedStep(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Failed,
		Pending, Pending,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)
	destroysAfterWalk := len(*h.destroys)
	require.Equal(t, 1, destroysAfterWalk)

	walked, ok := s.StepAt(2) // accept_to_character_200_55, already reversed
	require.True(t, ok)
	compensated, err := h.compensator.CompensateLateStep(s, walked)
	require.NoError(t, err)
	require.False(t, compensated, "the walk already claimed this step's inverse")
	require.Len(t, *h.destroys, destroysAfterWalk, "no second destroy may be dispatched")
}

// TestLateCompensableRegistrationIsTradeScoped pins the blast radius: the two
// newly registered actions must NOT become late-compensable for any other saga
// type that routes through the same absorb path.
func TestLateCompensableRegistrationIsTradeScoped(t *testing.T) {
	txId := uuid.New()
	build := func(sagaType Type) Saga {
		s, err := NewBuilder().
			SetTransactionId(txId).
			SetSagaType(sagaType).
			SetInitiatedBy("scope-test").
			AddStep("release_from_trade", Completed, ReleaseFromTrade, ReleaseFromTradePayload{EscrowId: tradeEscrowA}).
			Build()
		require.NoError(t, err)
		return s
	}

	for _, other := range []Type{StorageOperation, MtsOperation, InventoryTransaction, NoteSend} {
		require.Falsef(t, isLateCompensable(build(other), ReleaseFromTrade),
			"%s must keep its pre-task-205 absorb behaviour for release_from_trade", other)
		require.Falsef(t, isLateCompensable(build(other), AcceptToCharacter),
			"%s must keep its pre-task-205 absorb behaviour for accept_to_character", other)
	}

	trade := build(TradeTransaction)
	require.True(t, isLateCompensable(trade, ReleaseFromTrade))
	require.True(t, isLateCompensable(trade, AcceptToCharacter))

	// release_from_character is NOT trade-scoped any more: a settlement releases
	// from custody, never from a compartment. Registering it would hand every
	// storage/cash-shop/MTS saga an inverse it does not have.
	require.False(t, isLateCompensable(trade, ReleaseFromCharacter))

	// The base set is unchanged for every saga type, trade included.
	for _, a := range []Action{AwardMesos, AwardAsset, DestroyAsset} {
		require.True(t, isLateCompensable(trade, a))
		require.True(t, isLateCompensable(build(StorageOperation), a))
	}
}
