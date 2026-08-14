//go:build test

package saga

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	compmock "atlas-saga-orchestrator/compartment/mock"
)

// TestRemoteMerchantCompensationEmitsShopExit verifies the remote_merchant
// reverse-walk (task-221 FR-4.5): when consume_remote_merchant_item fails, the
// already-completed open_npc_shop is inverted with an EXIT command so the
// player is not left standing in a shop they did not pay for.
//
// DispatchCashItemUseRollbacks is exercised directly (mirroring the point-reset
// and pet-evolution compensation tests) to avoid the EmitSagaFailed Kafka path.
func TestRemoteMerchantCompensationEmitsShopExit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	const (
		testCharId = uint32(88221)
		miuMiuItem = uint32(5450000)
		miuMiuNpc  = uint32(9090000)
	)

	var exitCalls []uint32
	origExit := SetEmitNpcShopExitForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32) error {
		exitCalls = append(exitCalls, characterId)
		return nil
	})
	t.Cleanup(func() { SetEmitNpcShopExitForTest(origExit) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(RemoteMerchant).
		SetInitiatedBy("remote-merchant-compensation-test").
		AddStep("open_npc_shop", Completed, OpenNpcShop, OpenNpcShopPayload{
			CharacterId:   testCharId,
			NpcTemplateId: miuMiuNpc,
		}).
		AddStep("consume_remote_merchant_item", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   testCharId,
			InventoryType: 5,
			Slot:          3,
			Quantity:      1,
			TemplateId:    miuMiuItem,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 1, len(exitCalls), "the opened shop should be closed exactly once")
	if len(exitCalls) == 1 {
		assert.Equal(t, testCharId, exitCalls[0], "EXIT must target the test character")
	}
}

// TestRemoteMerchantCompensationSkipsUncompletedShopOpen verifies that a shop
// open that never completed is NOT closed — the reverse-walk only inverts
// Completed mutations.
func TestRemoteMerchantCompensationSkipsUncompletedShopOpen(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var exitCalls []uint32
	origExit := SetEmitNpcShopExitForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32) error {
		exitCalls = append(exitCalls, characterId)
		return nil
	})
	t.Cleanup(func() { SetEmitNpcShopExitForTest(origExit) })

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(RemoteMerchant).
		SetInitiatedBy("remote-merchant-compensation-test").
		AddStep("open_npc_shop", Failed, OpenNpcShop, OpenNpcShopPayload{
			CharacterId:   88222,
			NpcTemplateId: 9090000,
		}).
		Build()
	assert.NoError(t, err)

	NewCompensator(logger, testTenantContext()).DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 0, len(exitCalls), "a shop that never opened must not be closed")
}
