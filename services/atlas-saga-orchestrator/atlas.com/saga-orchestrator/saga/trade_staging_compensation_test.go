package saga

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	compartmentmock "atlas-saga-orchestrator/compartment/mock"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	tradesvc "atlas-saga-orchestrator/trade"

	"atlas-saga-orchestrator/kafka/message"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	stagingCharId    = uint32(100)
	stagingInvType   = byte(1)
	stagingAssetId   = uint32(55)
	stagingTemplate  = uint32(1302000)
	stagingQuantity  = uint32(1)
	stagingWeaponAtk = uint16(17)
)

// stagingTradeMock records the trade-custody dispatches a staging rollback makes.
type stagingTradeMock struct {
	removeCalls  []uuid.UUID
	restoreCalls []uuid.UUID
}

func (m *stagingTradeMock) AcceptToTradeAndEmit(_ uuid.UUID, _ tradesvc.AcceptToTradeParams) error {
	return nil
}

func (m *stagingTradeMock) AcceptToTrade(_ *message.Buffer) func(uuid.UUID, tradesvc.AcceptToTradeParams) error {
	return func(uuid.UUID, tradesvc.AcceptToTradeParams) error { return nil }
}

func (m *stagingTradeMock) ReleaseFromTradeAndEmit(_ uuid.UUID, _ uuid.UUID) error { return nil }

func (m *stagingTradeMock) ReleaseFromTrade(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(uuid.UUID, uuid.UUID) error { return nil }
}

func (m *stagingTradeMock) RestoreTradeEscrowAndEmit(_ uuid.UUID, escrowId uuid.UUID) error {
	m.restoreCalls = append(m.restoreCalls, escrowId)
	return nil
}

func (m *stagingTradeMock) RestoreTradeEscrow(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(uuid.UUID, uuid.UUID) error { return nil }
}

func (m *stagingTradeMock) RemoveTradeEscrowAndEmit(_ uuid.UUID, escrowId uuid.UUID) error {
	m.removeCalls = append(m.removeCalls, escrowId)
	return nil
}

func (m *stagingTradeMock) RemoveTradeEscrow(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(uuid.UUID, uuid.UUID) error { return nil }
}

type stagingHarness struct {
	compensator Compensator
	tctx        context.Context
	regrants    *[]tradeRegrantCall
	tradeMock   *stagingTradeMock
}

func newStagingRollbackHarness(t *testing.T) stagingHarness {
	t.Helper()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	tctx := tenant.WithContext(context.Background(), te)

	regrants := &[]tradeRegrantCall{}
	compMock := &compartmentmock.ProcessorMock{
		RequestAcceptAssetFunc: func(_ uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error {
			*regrants = append(*regrants, tradeRegrantCall{CharacterId: characterId, InventoryType: inventoryType, TemplateId: templateId, AssetData: assetData})
			return nil
		},
	}
	tradeMock := &stagingTradeMock{}

	return stagingHarness{
		compensator: NewCompensator(logger, tctx).WithCompartmentProcessor(compMock).WithTradeProcessor(tradeMock),
		tctx:        tctx,
		regrants:    regrants,
		tradeMock:   tradeMock,
	}
}

// stagingSaga reproduces the two steps expandTransferToTrade emits, with
// per-test statuses.
func stagingSaga(t *testing.T, transactionId, escrowId uuid.UUID, release, accept Status) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(TradeStaging).
		SetInitiatedBy("trade-staging-compensation-test").
		AddStep("release_from_character", release, ReleaseFromCharacter, ReleaseFromCharacterPayload{
			TransactionId: transactionId, CharacterId: stagingCharId, InventoryType: stagingInvType,
			AssetId: stagingAssetId, Quantity: stagingQuantity,
		}).
		AddStep("accept_to_trade", accept, AcceptToTrade, AcceptToTradePayload{
			TransactionId: transactionId, EscrowId: escrowId, RoomId: uuid.New(),
			OwnerId: stagingCharId, TradeSlot: 1, SourceInventoryType: stagingInvType, SourceSlot: 3,
			TemplateId: stagingTemplate, Quantity: stagingQuantity, WeaponAttack: stagingWeaponAtk,
			Owner: "Chronicle",
		}).
		Build()
	require.NoError(t, err)
	return s
}

// TestStagingRollbackRegrantsWhenTheEscrowAcceptFails is the load-bearing test
// of the escrow amendment's staging path.
//
// The asset has already left the player's compartment (release Completed) when
// the escrow accept fails. Nothing else in the system knows the asset existed —
// atlas-inventory soft-deleted it against this saga's transaction id — so if the
// reverse-walk does not re-grant here, staging an item DESTROYS it. That is a
// strictly worse outcome than the reserve-at-staging model this replaced, which
// is why the amendment gave staging its own saga type instead of routing it
// through the two-party swap's pairing logic.
func TestStagingRollbackRegrantsWhenTheEscrowAcceptFails(t *testing.T) {
	h := newStagingRollbackHarness(t)
	transactionId := uuid.New()
	escrowId := uuid.New()

	s := stagingSaga(t, transactionId, escrowId, Completed, Failed)
	require.NoError(t, GetCache().Put(h.tctx, s))
	require.True(t, GetCache().TryTransition(h.tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

	h.compensator.DispatchTradeStagingRollbacks(s)

	require.Len(t, *h.regrants, 1, "the released asset must be re-granted to its owner")
	got := (*h.regrants)[0]
	assert.Equal(t, stagingCharId, got.CharacterId)
	assert.Equal(t, stagingInvType, got.InventoryType)
	assert.Equal(t, stagingTemplate, got.TemplateId)
	assert.Equal(t, stagingQuantity, got.AssetData.Quantity)
	assert.Equal(t, stagingWeaponAtk, got.AssetData.WeaponAttack,
		"the re-grant must carry the AcceptToTrade snapshot's stats, not a bare template")
	assert.Equal(t, "Chronicle", got.AssetData.Owner)

	assert.Empty(t, h.tradeMock.removeCalls,
		"the accept FAILED, so no escrow row exists to remove")

	GetCache().Remove(h.tctx, transactionId)
}

// TestStagingRollbackRemovesTheEscrowRowWhenBothStepsCompleted pins the other
// direction: if the escrow row was created and the saga still fails (a later
// step, or a timeout), the row must be hard-deleted AND the asset re-granted.
// Leaving the row would let a settlement or unwind deliver an item the owner
// also holds again — a duplicate.
func TestStagingRollbackRemovesTheEscrowRowWhenBothStepsCompleted(t *testing.T) {
	h := newStagingRollbackHarness(t)
	transactionId := uuid.New()
	escrowId := uuid.New()

	s := stagingSaga(t, transactionId, escrowId, Completed, Completed)
	require.NoError(t, GetCache().Put(h.tctx, s))
	require.True(t, GetCache().TryTransition(h.tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

	h.compensator.DispatchTradeStagingRollbacks(s)

	require.Len(t, h.tradeMock.removeCalls, 1, "the created escrow row must be removed")
	assert.Equal(t, escrowId, h.tradeMock.removeCalls[0])
	require.Len(t, *h.regrants, 1, "the released asset must still be re-granted")

	GetCache().Remove(h.tctx, transactionId)
}

// TestStagingRollbackDispatchesNothingWhenTheReleaseFailed pins the safe case:
// the release itself failed, so the asset never left the compartment. Re-granting
// it would MINT a duplicate.
func TestStagingRollbackDispatchesNothingWhenTheReleaseFailed(t *testing.T) {
	h := newStagingRollbackHarness(t)
	transactionId := uuid.New()
	escrowId := uuid.New()

	s := stagingSaga(t, transactionId, escrowId, Failed, Pending)
	require.NoError(t, GetCache().Put(h.tctx, s))
	require.True(t, GetCache().TryTransition(h.tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

	h.compensator.DispatchTradeStagingRollbacks(s)

	assert.Empty(t, *h.regrants, "nothing left the compartment, so nothing may be re-granted")
	assert.Empty(t, h.tradeMock.removeCalls, "no escrow row was created")

	GetCache().Remove(h.tctx, transactionId)
}

// TestStagingRollbackIsOnceOnly pins the claimLateCompensation marker: a second
// walk (duplicate delivery, or a late success arriving after the walk) must
// dispatch nothing, or the re-grant mints a duplicate item.
func TestStagingRollbackIsOnceOnly(t *testing.T) {
	h := newStagingRollbackHarness(t)
	transactionId := uuid.New()
	escrowId := uuid.New()

	s := stagingSaga(t, transactionId, escrowId, Completed, Completed)
	require.NoError(t, GetCache().Put(h.tctx, s))
	require.True(t, GetCache().TryTransition(h.tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

	h.compensator.DispatchTradeStagingRollbacks(s)
	h.compensator.DispatchTradeStagingRollbacks(s)

	assert.Len(t, *h.regrants, 1, "a second reverse-walk must not re-grant again")
	assert.Len(t, h.tradeMock.removeCalls, 1, "a second reverse-walk must not remove again")

	GetCache().Remove(h.tctx, transactionId)
}
