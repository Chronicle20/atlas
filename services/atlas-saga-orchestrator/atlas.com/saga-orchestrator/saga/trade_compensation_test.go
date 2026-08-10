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

	return tradeRollbackHarness{
		compensator: NewCompensator(logger, tctx).WithCompartmentProcessor(compMock).WithCharacterProcessor(charMock),
		tctx:        tctx,
		regrants:    regrants,
		destroys:    destroys,
		mesos:       mesos,
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
		AddStep("release_from_character_100_55", releaseA, ReleaseFromCharacter, ReleaseFromCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharA, InventoryType: tradeTypeA, AssetId: tradeAssetA, Quantity: 5,
		}).
		AddStep("release_from_character_200_77", releaseB, ReleaseFromCharacter, ReleaseFromCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharB, InventoryType: tradeTypeB, AssetId: tradeAssetB, Quantity: 1,
		}).
		AddStep("accept_to_character_200_55", acceptB, AcceptToCharacter, AcceptToCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharB, InventoryType: tradeTypeA, TemplateId: tradeTemplateA,
			AssetData: asset2.AssetData{Quantity: 5, Owner: "Chronicle"},
		}).
		AddStep("accept_to_character_100_77", acceptA, AcceptToCharacter, AcceptToCharacterPayload{
			TransactionId: transactionId, CharacterId: tradeCharA, InventoryType: tradeTypeB, TemplateId: tradeTemplateB,
			AssetData: asset2.AssetData{Quantity: 1, Owner: "Chronicle", WeaponAttack: 17},
		}).
		AddStep("award_mesos_deduct_100", deduct, AwardMesos, AwardMesosPayload{
			CharacterId: tradeCharA, WorldId: 1, ChannelId: 1, ActorId: tradeCharB, ActorType: "CHARACTER", Amount: -10_000_000,
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
// items left their owners and one landed with B, then B→A's accept failed.
// Conservation requires B's newly received copy be destroyed AND both owners'
// items be re-granted — otherwise A's item is gone and B holds two.
func TestTradeRollback_AcceptFailsAfterBothReleases(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed, // both releases done
		Completed, Failed, // accept→B done, accept→A failed
		Pending, Pending, // meso legs never ran
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	// The one completed accept is undone: B loses the copy it just received.
	require.Len(t, *h.destroys, 1)
	require.Equal(t, tradeDestroyCall{CharacterId: tradeCharB, TemplateId: tradeTemplateA, Quantity: 5, RemoveAll: false}, (*h.destroys)[0])

	// Both completed releases are undone: each owner gets their own item back,
	// with the snapshot from the paired accept step so stats survive.
	require.Len(t, *h.regrants, 2)
	byCharacter := map[uint32]tradeRegrantCall{}
	for _, r := range *h.regrants {
		byCharacter[r.CharacterId] = r
	}
	require.Equal(t, tradeTemplateA, byCharacter[tradeCharA].TemplateId, "A must get its OWN item back, not B's")
	require.Equal(t, tradeTypeA, byCharacter[tradeCharA].InventoryType)
	require.Equal(t, uint32(5), byCharacter[tradeCharA].AssetData.Quantity)
	require.Equal(t, tradeTemplateB, byCharacter[tradeCharB].TemplateId, "B must get its OWN item back")
	require.Equal(t, tradeTypeB, byCharacter[tradeCharB].InventoryType)
	require.Equal(t, uint16(17), byCharacter[tradeCharB].AssetData.WeaponAttack, "the equip snapshot's stats must survive the rollback")

	// No meso leg completed, so nothing to reverse.
	require.Empty(t, *h.mesos)
}

// TestTradeRollback_CreditFailsAfterDeduct pins the meso half-swap: A was
// debited the full staged amount and B's credit then failed. A must be made
// whole, and B must not be credited by the rollback.
func TestTradeRollback_CreditFailsAfterDeduct(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Completed, // the whole item swap landed
		Completed, Failed, // deduct done, credit failed
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	// Only the completed deduct is reversed, and with the opposite sign.
	require.Len(t, *h.mesos, 1)
	require.Equal(t, tradeMesoCall{CharacterId: tradeCharA, Amount: 10_000_000}, (*h.mesos)[0])

	// The item swap is unwound too: both accepts destroyed, both releases re-granted.
	require.Len(t, *h.destroys, 2)
	require.Len(t, *h.regrants, 2)
}

// TestTradeRollback_FailureAfterFirstReleaseOnly pins the earliest failure
// shape: only A's item left inventory before the second release failed. Exactly
// one re-grant must fire — to A — and nothing may be destroyed.
func TestTradeRollback_FailureAfterFirstReleaseOnly(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Failed, // A released, B's release failed
		Pending, Pending,
		Pending, Pending,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)

	require.Empty(t, *h.destroys, "nothing was created, so nothing may be destroyed")
	require.Empty(t, *h.mesos)
	require.Len(t, *h.regrants, 1)
	require.Equal(t, tradeCharA, (*h.regrants)[0].CharacterId)
	require.Equal(t, tradeTemplateA, (*h.regrants)[0].TemplateId)
}

// TestTradeRollback_IsIdempotent pins that a second walk over the same saga
// dispatches nothing. The meso inverses are negations and are NOT idempotent
// downstream, so a repeated walk would hand A back 10,000,000 twice.
func TestTradeRollback_IsIdempotent(t *testing.T) {
	h := newTradeRollbackHarness(t)
	txId := uuid.New()
	s := stageTradeSaga(t, h, txId, tradeSettlementSagaBuilder(txId,
		Completed, Completed,
		Completed, Completed,
		Completed, Failed,
	))

	h.compensator.DispatchTradeTransactionRollbacks(s)
	firstRegrants := len(*h.regrants)
	firstDestroys := len(*h.destroys)
	firstMesos := len(*h.mesos)
	require.NotZero(t, firstRegrants)
	require.NotZero(t, firstDestroys)
	require.NotZero(t, firstMesos)

	h.compensator.DispatchTradeTransactionRollbacks(s)

	require.Len(t, *h.regrants, firstRegrants, "a second walk must not re-grant again")
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

	require.NotEmpty(t, *h.regrants, "a failed trade step must reach the reverse-walk, not compensateStorageOperation's no-op path")
	require.NotEmpty(t, *h.destroys)
}

// TestTradeStepIdsCarryTheAssetIdLink pins the coupling the reverse-walk relies
// on to pair a release with the accept holding its snapshot: expandTradeSettlement
// appends the asset id to BOTH step ids, and tradeStepAssetId parses it back.
// If the id format changes without this, releases silently lose their snapshot
// and are skipped — i.e. the item is not re-granted.
func TestTradeStepIdsCarryTheAssetIdLink(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.NoError(t, err)

	seen := map[uint32]int{}
	for _, s := range steps {
		switch s.Action() {
		case ReleaseFromCharacter:
			assetId, ok := tradeStepAssetId(s.StepId())
			require.Truef(t, ok, "release step id %q must carry a parseable asset id", s.StepId())
			pl := s.Payload().(ReleaseFromCharacterPayload)
			require.Equalf(t, pl.AssetId, assetId, "step id %q must encode the payload's asset id", s.StepId())
			seen[assetId]++
		case AcceptToCharacter:
			assetId, ok := tradeStepAssetId(s.StepId())
			require.Truef(t, ok, "accept step id %q must carry a parseable asset id", s.StepId())
			seen[assetId]++
		}
	}
	require.Equal(t, map[uint32]int{tradeAssetA: 2, tradeAssetB: 2}, seen,
		"each staged asset id must appear on exactly one release and one accept step id")
}

// TestTradeStepAssetIdRejectsUnparseableIds pins that the parser reports failure
// rather than returning a plausible 0 — a 0 would silently pair every release
// with the wrong snapshot.
func TestTradeStepAssetIdRejectsUnparseableIds(t *testing.T) {
	for _, bad := range []string{"release_from_character", "accept_to_character_100_", "trade_settlement", "", "award_mesos_deduct_x"} {
		_, ok := tradeStepAssetId(bad)
		require.Falsef(t, ok, "step id %q must not yield an asset id", bad)
	}
	id, ok := tradeStepAssetId("release_from_character_100_55")
	require.True(t, ok)
	require.Equal(t, uint32(55), id)
}
