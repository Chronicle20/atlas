//go:build test

package asset

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	notice "atlas-saga-orchestrator/kafka/message/conversation_reward_notice"
	"atlas-saga-orchestrator/saga"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noticeCall captures a single EmitConversationRewardNotice invocation.
type noticeCall struct {
	CharacterId uint32
	Kind        string
	TemplateId  uint32
	Quantity    uint32
}

// installNoticeStub overrides saga's emit function via the
// SetEmitConversationRewardNoticeForTest seam for the duration of the test.
func installNoticeStub(t *testing.T) func() []noticeCall {
	t.Helper()
	var mu sync.Mutex
	calls := []noticeCall{}

	orig := saga.SetEmitConversationRewardNoticeForTest(
		func(l logrus.FieldLogger, ctx context.Context, characterId uint32, kind string, itemId uint32, quantity uint32) error {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, noticeCall{
				CharacterId: characterId, Kind: kind, TemplateId: itemId, Quantity: quantity,
			})
			return nil
		},
	)
	t.Cleanup(func() { saga.SetEmitConversationRewardNoticeForTest(orig) })

	return func() []noticeCall {
		mu.Lock()
		defer mu.Unlock()
		out := make([]noticeCall, len(calls))
		copy(out, calls)
		return out
	}
}

func TestEmitRewardNoticeForCurrentStep_UsesEventTemplateIdAndQuantity(t *testing.T) {
	getCalls := installNoticeStub(t)
	logger, _ := test.NewNullLogger()
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 42,
			Item:        saga.ItemPayload{TemplateId: 1111, Quantity: 99},
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	// Call with event templateId=2222, quantity=3 — notice should reflect these.
	emitRewardNoticeForCurrentStep(logger, ctx, tx, 2222, 3)

	calls := getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, uint32(42), calls[0].CharacterId)
	assert.Equal(t, notice.KindItemGain, calls[0].Kind)
	assert.Equal(t, uint32(2222), calls[0].TemplateId, "templateId must come from the event")
	assert.Equal(t, uint32(3), calls[0].Quantity, "quantity must come from the event")
}

func TestEmitRewardNoticeForCurrentStep_DestroyAssetFromSlotUsesPayloadQuantity(t *testing.T) {
	getCalls := installNoticeStub(t)
	logger, _ := test.NewNullLogger()
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.DestroyAssetFromSlot, saga.DestroyAssetFromSlotPayload{
			CharacterId: 42,
			Quantity:    7,
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	// Pass bogus event quantity — should be ignored for DestroyAssetFromSlot.
	emitRewardNoticeForCurrentStep(logger, ctx, tx, 9999, 999)

	calls := getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, uint32(9999), calls[0].TemplateId, "templateId still comes from the event (payload has none)")
	assert.Equal(t, uint32(7), calls[0].Quantity, "DestroyAssetFromSlot uses payload quantity")
}

func TestHandleAssetCreatedEvent_Match(t *testing.T) {
	_ = installNoticeStub(t)
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	// Add a trailing pending step so the saga doesn't terminate (and get
	// evicted from the cache) after the AwardAsset step completes — we need
	// to inspect the completed step's status post-hoc.
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 1,
			Item:        saga.ItemPayload{TemplateId: 2070015, Quantity: 1},
		}).
		AddStep("s2", saga.Pending, saga.SendMessage, saga.SendMessagePayload{CharacterId: 1, Message: "pinned"}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleAssetCreatedEvent(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   1,
		TemplateId:    2070015,
		AssetId:       77,
		Body:          asset2.CreatedStatusEventBody{AssetData: asset2.AssetData{Quantity: 1}},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status())
}

func TestHandleAssetCreatedEvent_TemplateIdMismatch(t *testing.T) {
	_ = installNoticeStub(t)
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 1,
			Item:        saga.ItemPayload{TemplateId: 2070015, Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleAssetCreatedEvent(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   1,
		TemplateId:    1472061,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "templateId mismatch must not complete the step")

	assertDebugReason(t, hook, saga.SkipReasonTemplateIdMismatch)
}
