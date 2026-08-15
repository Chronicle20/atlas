package saga

import (
	compmock "atlas-saga-orchestrator/compartment/mock"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	karmaCharId        = uint32(99001)
	karmaScissorsItem  = uint32(5064000)
	karmaInventoryType = byte(2)
	karmaSlot          = int16(3)
	karmaScissorsKarma = int32(1)
)

// TestApplyAssetKarmaIsInTheAcceptanceTable: a missing entry default-denies in
// StepAcceptsEvent, so the step would never complete and the saga would time out.
func TestApplyAssetKarmaIsInTheAcceptanceTable(t *testing.T) {
	kinds, ok := acceptanceTable[sharedsaga.ApplyAssetKarma]
	if !ok {
		t.Fatal("acceptanceTable has no entry for ApplyAssetKarma")
	}
	if len(kinds) != 1 || kinds[0] != EventKindAssetUpdated {
		t.Fatalf("acceptanceTable[ApplyAssetKarma] = %v, want [EventKindAssetUpdated]", kinds)
	}
}

// TestKarmaScissorsUseTakesTheCashItemUseReverseWalk: without the saga-type arm
// a failed karma saga only compensates the failing step, leaving the already
// destroyed scissors unrefunded.
func TestKarmaScissorsUseTakesTheCashItemUseReverseWalk(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var calls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			calls = append(calls, createCall{characterId, templateId, quantity})
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(KarmaScissorsUse).
		SetInitiatedBy("karma-compensation-test").
		AddStep("consume_scissors", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: karmaCharId,
			TemplateId:  karmaScissorsItem,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("apply_asset_karma", Failed, ApplyAssetKarma, ApplyAssetKarmaPayload{
			CharacterId:   karmaCharId,
			InventoryType: karmaInventoryType,
			Slot:          karmaSlot,
			ScissorsKarma: karmaScissorsKarma,
		}).
		Build()
	require.NoError(t, err)

	c := NewCompensator(logger, testTenantContext()).WithCompartmentProcessor(compP)
	require.NoError(t, c.CompensateFailedStep(s))

	require.Len(t, calls, 1, "the already-destroyed scissors must be refunded exactly once")
	assert.Equal(t, karmaCharId, calls[0].CharacterId)
	assert.Equal(t, karmaScissorsItem, calls[0].TemplateId)
	assert.Equal(t, uint32(1), calls[0].Quantity)
}

// TestKarmaRollbackClearsTheMark is FR-6.6: a saga failing AFTER the mark is
// applied must not leave a free trade behind.
func TestKarmaRollbackClearsTheMark(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	type applyKarmaCall struct {
		CharacterId   uint32
		InventoryType byte
		Slot          int16
		ScissorsKarma int32
		Clear         bool
	}
	var calls []applyKarmaCall
	compP := &compmock.ProcessorMock{
		RequestApplyKarmaFunc: func(_ uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error {
			calls = append(calls, applyKarmaCall{characterId, inventoryType, slot, scissorsKarma, clear})
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(KarmaScissorsUse).
		SetInitiatedBy("karma-compensation-test").
		AddStep("consume_scissors", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: karmaCharId,
			TemplateId:  karmaScissorsItem,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("apply_asset_karma", Completed, ApplyAssetKarma, ApplyAssetKarmaPayload{
			CharacterId:   karmaCharId,
			InventoryType: karmaInventoryType,
			Slot:          karmaSlot,
			ScissorsKarma: karmaScissorsKarma,
		}).
		Build()
	require.NoError(t, err)

	c := NewCompensator(logger, testTenantContext()).WithCompartmentProcessor(compP)
	c.DispatchCashItemUseRollbacks(s)

	require.Len(t, calls, 1, "the applied mark must be cleared exactly once")
	assert.Equal(t, karmaCharId, calls[0].CharacterId)
	assert.Equal(t, karmaInventoryType, calls[0].InventoryType)
	assert.Equal(t, karmaSlot, calls[0].Slot)
	assert.True(t, calls[0].Clear, "rollback must apply the mark's inverse (clear == true)")
}
