package saga

// parcel_compensation_test.go — the orchestrator arm of the Duey parcel
// compensation suite (task-241 design §10). Mirrors mts_dupe_safety_test.go
// and mts_integration_test.go: the compensation tests exercise
// DispatchParcelSendRollbacks / DispatchParcelReceiveRollbacks and
// CompensateLateStep directly to avoid the EmitSagaFailed Kafka path (no
// broker in the test environment).
//
// TestParcelSendCompensation covers the parcel_send reverse walk (§10's
// "earns its keep" case: a re-awarded item loses its stats and a
// released/re-accepted one does not) plus the meso-only RISK-2 guard and the
// late-accept absorb path.
//
// TestParcelReceiveCompensation covers the parcel_receive reverse walk: a
// failed accept_to_character restores the parcel from custody, and a failed
// award_mesos (after both custody steps completed) restores the parcel AND
// undoes the character grant so the item is not left in the recipient's
// inventory.

import (
	charactermock "atlas-saga-orchestrator/character/mock"
	compartmentmock "atlas-saga-orchestrator/compartment/mock"
	"atlas-saga-orchestrator/kafka/message"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	parcelsvc "atlas-saga-orchestrator/parcel"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// parcelTestMock captures RestoreParcelAndEmit / RemoveParcelAndEmit
// dispatches so the tests can assert the exact custody command and parcel id.
// It implements the full parcel.Processor interface; the AndEmit methods the
// reverse-walk and late-inverse paths touch record calls, the rest are no-ops.
type parcelTestMock struct {
	restoreCalls   int
	restoreParcels []uuid.UUID
	removeCalls    int
	removeParcels  []uuid.UUID
	acceptCalls    int
	releaseCalls   int
}

func (m *parcelTestMock) AcceptToParcelAndEmit(_ uuid.UUID, _ parcelsvc.AcceptToParcelParams) error {
	m.acceptCalls++
	return nil
}

func (m *parcelTestMock) AcceptToParcel(_ *message.Buffer) func(uuid.UUID, parcelsvc.AcceptToParcelParams) error {
	return func(_ uuid.UUID, _ parcelsvc.AcceptToParcelParams) error { return nil }
}

func (m *parcelTestMock) ReleaseFromParcelAndEmit(_ uuid.UUID, _ uuid.UUID, _ uint32) error {
	m.releaseCalls++
	return nil
}

func (m *parcelTestMock) ReleaseFromParcel(_ *message.Buffer) func(uuid.UUID, uuid.UUID, uint32) error {
	return func(_ uuid.UUID, _ uuid.UUID, _ uint32) error { return nil }
}

func (m *parcelTestMock) RestoreParcelAndEmit(_ uuid.UUID, parcelId uuid.UUID) error {
	m.restoreCalls++
	m.restoreParcels = append(m.restoreParcels, parcelId)
	return nil
}

func (m *parcelTestMock) RestoreParcel(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(_ uuid.UUID, _ uuid.UUID) error { return nil }
}

func (m *parcelTestMock) RemoveParcelAndEmit(_ uuid.UUID, parcelId uuid.UUID) error {
	m.removeCalls++
	m.removeParcels = append(m.removeParcels, parcelId)
	return nil
}

func (m *parcelTestMock) RemoveParcel(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(_ uuid.UUID, _ uuid.UUID) error { return nil }
}

// Ensure the mock satisfies parcel.Processor at compile time.
var _ parcelsvc.Processor = (*parcelTestMock)(nil)

// parcelCompensationTestCtx builds a tenant context for these tests.
func parcelCompensationTestCtx(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), te)
}

// ---------------------------------------------------------------------------
// TestParcelSendCompensation
// ---------------------------------------------------------------------------

func TestParcelSendCompensation(t *testing.T) {
	const (
		characterId     = uint32(9101)
		senderAccountId = uint32(5001)
		recipientId     = uint32(9102)
		inventoryType   = byte(1)
		templateId      = uint32(1302000)
		mesoAmount      = uint32(1000)
		feePaid         = uint32(100)
	)

	t.Run("send fails at accept re-credits mesos and re-grants item with stats intact", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		tctx := parcelCompensationTestCtx(t)

		transactionId := uuid.New()
		parcelId := uuid.New()

		var acceptCalls []struct {
			CharacterId   uint32
			InventoryType byte
			TemplateId    uint32
			AssetData     asset2.AssetData
		}
		compMock := &compartmentmock.ProcessorMock{
			RequestAcceptAssetFunc: func(_ uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error {
				acceptCalls = append(acceptCalls, struct {
					CharacterId   uint32
					InventoryType byte
					TemplateId    uint32
					AssetData     asset2.AssetData
				}{characterId, inventoryType, templateId, assetData})
				return nil
			},
		}
		charMock := &charactermock.ProcessorMock{
			AwardMesosAndEmitFunc: func(_ uuid.UUID, _ channel.Model, _ uint32, _ uint32, _ string, _ int32, _ bool) error {
				return nil
			},
		}

		// Saga shape mirrors expandTransferToParcel:
		//   step 0: award_mesos             ← Completed (sender debited mesoAmount+fee)
		//   step 1: release_from_character  ← Completed (item left inventory)
		//   step 2: accept_to_parcel        ← Failed (the custody create tx rolled back)
		s, err := NewBuilder().
			SetTransactionId(transactionId).
			SetSagaType(ParcelSend).
			SetInitiatedBy("parcel-send-compensation-test").
			AddStep("award_mesos", Completed, AwardMesos, AwardMesosPayload{
				CharacterId: characterId,
				WorldId:     0,
				ChannelId:   0,
				ActorId:     characterId,
				ActorType:   "SYSTEM",
				Amount:      -int32(mesoAmount + feePaid),
			}).
			AddStep("release_from_character", Completed, ReleaseFromCharacter, ReleaseFromCharacterPayload{
				TransactionId: transactionId,
				CharacterId:   characterId,
				InventoryType: inventoryType,
				AssetId:       42,
				Quantity:      1,
			}).
			AddStep("accept_to_parcel", Failed, AcceptToParcel, AcceptToParcelPayload{
				TransactionId: transactionId,
				ParcelId:      parcelId,
				CharacterId:   characterId,
				WorldId:       0,
				RecipientId:   recipientId,
				MesoAmount:    mesoAmount,
				FeePaid:       feePaid,
				HasItem:       true,
				TemplateId:    templateId,
				Quantity:      1,
				Strength:      5,
				Owner:         "Alice",
			}).
			Build()
		require.NoError(t, err, "saga build should not fail")
		require.NoError(t, GetCache().Put(tctx, s))
		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

		compensator := NewCompensator(logger, tctx).
			WithCompartmentProcessor(compMock).
			WithCharacterProcessor(charMock)
		compensator.DispatchParcelSendRollbacks(s)

		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed))
		GetCache().Remove(tctx, transactionId)

		// INVARIANT: the item is re-granted to EXACTLY ONE place, carrying the
		// AcceptToParcel snapshot's stats and owner — a re-awarded item loses
		// its stats and a released/re-accepted one does not (design §10).
		require.Equal(t, 1, len(acceptCalls), "must re-grant the item exactly once")
		regrant := acceptCalls[0]
		assert.Equal(t, characterId, regrant.CharacterId)
		assert.Equal(t, inventoryType, regrant.InventoryType)
		assert.Equal(t, templateId, regrant.TemplateId)
		assert.Equal(t, uint16(5), regrant.AssetData.Strength, "re-grant must carry the snapshot's Strength")
		assert.Equal(t, "Alice", regrant.AssetData.Owner, "re-grant must carry the snapshot's Owner")
	})

	t.Run("send meso-only fails at accept re-credits mesos and attempts no item re-grant", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		tctx := parcelCompensationTestCtx(t)

		transactionId := uuid.New()
		parcelId := uuid.New()

		var acceptCalls int
		compMock := &compartmentmock.ProcessorMock{
			RequestAcceptAssetFunc: func(_ uuid.UUID, _ uint32, _ byte, _ uint32, _ asset2.AssetData) error {
				acceptCalls++
				return nil
			},
		}
		var awardCalls []int32
		charMock := &charactermock.ProcessorMock{
			AwardMesosAndEmitFunc: func(_ uuid.UUID, _ channel.Model, _ uint32, _ uint32, _ string, amount int32, _ bool) error {
				awardCalls = append(awardCalls, amount)
				return nil
			},
		}

		// Meso-only send (design §12 RISK-2): no release_from_character step at
		// all — the composite expansion never touches inventory for AssetId==0.
		s, err := NewBuilder().
			SetTransactionId(transactionId).
			SetSagaType(ParcelSend).
			SetInitiatedBy("parcel-send-meso-only-compensation-test").
			AddStep("award_mesos", Completed, AwardMesos, AwardMesosPayload{
				CharacterId: characterId,
				WorldId:     0,
				ChannelId:   0,
				ActorId:     characterId,
				ActorType:   "SYSTEM",
				Amount:      -int32(mesoAmount + feePaid),
			}).
			AddStep("accept_to_parcel", Failed, AcceptToParcel, AcceptToParcelPayload{
				TransactionId: transactionId,
				ParcelId:      parcelId,
				CharacterId:   characterId,
				WorldId:       0,
				RecipientId:   recipientId,
				MesoAmount:    mesoAmount,
				FeePaid:       feePaid,
				HasItem:       false,
			}).
			Build()
		require.NoError(t, err, "saga build should not fail")
		require.NoError(t, GetCache().Put(tctx, s))
		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

		compensator := NewCompensator(logger, tctx).
			WithCompartmentProcessor(compMock).
			WithCharacterProcessor(charMock)
		compensator.DispatchParcelSendRollbacks(s)

		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed))
		GetCache().Remove(tctx, transactionId)

		require.Equal(t, 1, len(awardCalls), "must re-credit the meso debit exactly once")
		assert.Equal(t, int32(mesoAmount+feePaid), awardCalls[0], "re-credit must be +mesoAmount+fee (inverse of the debit)")
		assert.Equal(t, 0, acceptCalls, "a meso-only snapshot (HasItem=false) must never produce an item re-grant")
	})

	t.Run("send late accept dispatches REMOVE_PARCEL", func(t *testing.T) {
		ResetCache()
		logger, _ := logtest.NewNullLogger()
		tctx := parcelCompensationTestCtx(t)

		parcelId := uuid.New()
		s, err := NewBuilder().
			SetSagaType(ParcelSend).
			SetInitiatedBy("test").
			AddStep("accept_to_parcel", Pending, AcceptToParcel, AcceptToParcelPayload{ParcelId: parcelId}).
			Build()
		require.NoError(t, err)
		require.NoError(t, GetCache().Put(tctx, s))

		parcelMockP := &parcelTestMock{}
		c := NewCompensator(logger, tctx).WithParcelProcessor(parcelMockP)

		step, _ := s.GetCurrentStep()
		compensated, err := c.CompensateLateStep(s, step)
		require.NoError(t, err)
		assert.True(t, compensated)
		require.Equal(t, 1, parcelMockP.removeCalls, "a late accept_to_parcel success must dispatch exactly one RemoveParcel")
		assert.Equal(t, parcelId, parcelMockP.removeParcels[0])
	})
}

// ---------------------------------------------------------------------------
// TestParcelReceiveCompensation
// ---------------------------------------------------------------------------

func TestParcelReceiveCompensation(t *testing.T) {
	const (
		recipientId   = uint32(9201)
		inventoryType = byte(1)
		templateId    = uint32(1302000)
	)

	t.Run("receive fails at accept_to_character dispatches RESTORE_PARCEL", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		tctx := parcelCompensationTestCtx(t)

		transactionId := uuid.New()
		parcelId := uuid.New()

		parcelMockP := &parcelTestMock{}
		compMock := &compartmentmock.ProcessorMock{}

		// Saga shape mirrors expandWithdrawFromParcel:
		//   step 0: release_from_parcel  ← Completed (custody row transitioned)
		//   step 1: accept_to_character  ← Failed (the grant never landed)
		s, err := NewBuilder().
			SetTransactionId(transactionId).
			SetSagaType(ParcelReceive).
			SetInitiatedBy("parcel-receive-compensation-test").
			AddStep("release_from_parcel", Completed, ReleaseFromParcel, ReleaseFromParcelPayload{
				TransactionId: transactionId,
				ParcelId:      parcelId,
				RecipientId:   recipientId,
			}).
			AddStep("accept_to_character", Failed, AcceptToCharacter, AcceptToCharacterPayload{
				TransactionId: transactionId,
				CharacterId:   recipientId,
				InventoryType: inventoryType,
				TemplateId:    templateId,
				AssetData:     asset2.AssetData{Quantity: 1},
			}).
			Build()
		require.NoError(t, err, "saga build should not fail")
		require.NoError(t, GetCache().Put(tctx, s))
		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

		compensator := NewCompensator(logger, tctx).
			WithParcelProcessor(parcelMockP).
			WithCompartmentProcessor(compMock)
		compensator.DispatchParcelReceiveRollbacks(s)

		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed))
		GetCache().Remove(tctx, transactionId)

		require.Equal(t, 1, parcelMockP.restoreCalls, "a failed accept_to_character must dispatch exactly one RestoreParcel")
		assert.Equal(t, parcelId, parcelMockP.restoreParcels[0])
	})

	t.Run("receive fails at award_mesos restores parcel and does not leave item in inventory", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		tctx := parcelCompensationTestCtx(t)

		transactionId := uuid.New()
		parcelId := uuid.New()

		parcelMockP := &parcelTestMock{}
		var destroyCalls []struct {
			CharacterId uint32
			TemplateId  uint32
			Quantity    uint32
		}
		compMock := &compartmentmock.ProcessorMock{
			RequestDestroyItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ bool) error {
				destroyCalls = append(destroyCalls, struct {
					CharacterId uint32
					TemplateId  uint32
					Quantity    uint32
				}{characterId, templateId, quantity})
				return nil
			},
		}

		// Both custody steps Completed, award_mesos Failed.
		s, err := NewBuilder().
			SetTransactionId(transactionId).
			SetSagaType(ParcelReceive).
			SetInitiatedBy("parcel-receive-award-mesos-compensation-test").
			AddStep("release_from_parcel", Completed, ReleaseFromParcel, ReleaseFromParcelPayload{
				TransactionId: transactionId,
				ParcelId:      parcelId,
				RecipientId:   recipientId,
			}).
			AddStep("accept_to_character", Completed, AcceptToCharacter, AcceptToCharacterPayload{
				TransactionId: transactionId,
				CharacterId:   recipientId,
				InventoryType: inventoryType,
				TemplateId:    templateId,
				AssetData:     asset2.AssetData{Quantity: 1},
			}).
			AddStep("award_mesos", Failed, AwardMesos, AwardMesosPayload{
				CharacterId: recipientId,
				WorldId:     0,
				ChannelId:   0,
				ActorId:     recipientId,
				ActorType:   "SYSTEM",
				Amount:      500,
			}).
			Build()
		require.NoError(t, err, "saga build should not fail")
		require.NoError(t, GetCache().Put(tctx, s))
		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

		compensator := NewCompensator(logger, tctx).
			WithParcelProcessor(parcelMockP).
			WithCompartmentProcessor(compMock)
		compensator.DispatchParcelReceiveRollbacks(s)

		require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed))
		GetCache().Remove(tctx, transactionId)

		assert.Equal(t, 1, parcelMockP.restoreCalls, "the reverse walk must restore the parcel")
		assert.Equal(t, parcelId, parcelMockP.restoreParcels[0])
		require.Equal(t, 1, len(destroyCalls), "the reverse walk must destroy the granted item so it is not left in the recipient's inventory")
		assert.Equal(t, recipientId, destroyCalls[0].CharacterId)
		assert.Equal(t, templateId, destroyCalls[0].TemplateId)
		assert.Equal(t, uint32(1), destroyCalls[0].Quantity)
	})
}
