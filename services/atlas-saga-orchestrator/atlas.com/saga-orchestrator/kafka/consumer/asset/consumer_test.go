package asset

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	notice "atlas-saga-orchestrator/kafka/message/conversation_reward_notice"
	"atlas-saga-orchestrator/saga"
	"context"
	"sync"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func mustTenantCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func putTestSaga(t *testing.T, ctx context.Context, s saga.Saga) {
	t.Helper()
	require.NoError(t, saga.GetCache().Put(ctx, s))
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

func TestHandleAssetCreatedEvent_WrongType(t *testing.T) {
	// Setup
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create an event with wrong type - should return early
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          "wrong_type", // Not StatusEventTypeCreated
		TransactionId: uuid.New(),
		CharacterId:   12345,
	}

	// Execute - should return early without processing
	handleAssetCreatedEvent(logger, tctx, event)

	// No assertions needed - function should return early
}

func TestHandleAssetCreatedEvent_SagaNotFound(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create an event with a transaction ID that doesn't exist
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: uuid.New(), // This saga doesn't exist
		CharacterId:   12345,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify debug log for saga not found emitted by AcceptEvent
	var foundDebugLog bool
	for _, entry := range hook.Entries {
		if entry.Level == logrus.DebugLevel {
			if r, ok := entry.Data["reason"]; ok && r == saga.SkipReasonSagaNotFound {
				foundDebugLog = true
				break
			}
		}
	}
	assert.True(t, foundDebugLog, "Should have debug log for saga not found")
}

func TestHandleAssetCreatedEvent_RegularCreation(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a regular AwardAsset step (not CreateAndEquipAsset)
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("asset-creation-test").
		AddStep("award-step", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1000000,
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(tctx, testSaga)

	// Create an asset created event
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   12345,
		TemplateId:    1000000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify no unexpected error logs (Kafka topic errors are expected in test env)
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel {
			// Skip expected Kafka-related errors in test environment
			if entry.Message == "Unable to emit event on topic [EVENT_TOPIC_SAGA_STATUS]." ||
				entry.Message == "Failed to emit saga completion event." {
				continue
			}
			t.Errorf("Unexpected error log: %s", entry.Message)
		}
	}
}

func TestHandleAssetCreatedEvent_CreateAndEquipAsset(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a CreateAndEquipAsset step
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("create-and-equip-test").
		AddStep("create-and-equip-step", saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1302000, // Equipment item
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(tctx, testSaga)

	// Create an asset created event
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   12345,
		TemplateId:    1302000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify the auto-equip step was added
	var foundAutoEquipLog bool
	for _, entry := range hook.Entries {
		if entry.Message == "Successfully added auto-equip step for CreateAndEquipAsset action to be executed next." {
			foundAutoEquipLog = true
			assert.Contains(t, entry.Data, "auto_equip_step_id")
			assert.Contains(t, entry.Data, "inventory_type")
			assert.Equal(t, uint32(12345), entry.Data["character_id"])
			break
		}
	}
	assert.True(t, foundAutoEquipLog, "Should have log for auto-equip step addition")
}

func TestHandleAssetCreatedEvent_CharacterMismatch(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Create a saga with a CreateAndEquipAsset step for character 12345
	transactionId := uuid.New()
	testSaga, err := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("mismatch-test").
		AddStep("create-and-equip-step", saga.Pending, saga.CreateAndEquipAsset, saga.CreateAndEquipAssetPayload{
			CharacterId: 12345,
			Item: saga.ItemPayload{
				TemplateId: 1302000,
				Quantity:   1,
			},
		}).
		Build()
	assert.NoError(t, err)

	// Store saga in cache
	_ = saga.GetCache().Put(tctx, testSaga)

	// Create an event with a different character ID
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: transactionId,
		CharacterId:   99999, // Different character
		TemplateId:    1302000,
		Slot:          5,
	}

	// Execute
	handleAssetCreatedEvent(logger, tctx, event)

	// Verify error log for character mismatch
	var foundMismatchLog bool
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel && entry.Message == "Character ID mismatch in CreateAndEquipAsset creation event." {
			foundMismatchLog = true
			break
		}
	}
	assert.True(t, foundMismatchLog, "Should have error log for character mismatch")
}

func TestHandleAssetQuantityUpdatedEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expectProcess bool
	}{
		{
			name:          "Process quantity changed event",
			eventType:     asset2.StatusEventTypeQuantityChanged,
			expectProcess: true,
		},
		{
			name:          "Skip wrong event type",
			eventType:     "wrong_type",
			expectProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			event := asset2.StatusEvent[asset2.QuantityChangedEventBody]{
				Type:          tt.eventType,
				TransactionId: uuid.New(),
				CharacterId:   12345,
			}

			// Execute - function should complete without panic
			handleAssetQuantityUpdatedEvent(logger, tctx, event)
		})
	}
}

// TestHandleAssetCreatedEvent_ActionMismatch proves that an ASSET_CREATED
// event does NOT advance a step whose action is not AwardAsset or
// CreateAndEquipAsset (the §9.1 bug class on the asset topic).
func TestHandleAssetCreatedEvent_ActionMismatch(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.ChangeJob, saga.ChangeJobPayload{CharacterId: 1, JobId: 400}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleAssetCreatedEvent(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   1,
		TemplateId:    2070015,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "ChangeJob step must not be completed by ASSET_CREATED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}

// assertDebugReason asserts at least one debug entry in the hook has the
// given `reason` field. Avoids brittle index coupling.
func assertDebugReason(t *testing.T, hook *test.Hook, want string) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if r, ok := e.Data["reason"]; ok && r == want {
			return
		}
	}
	t.Fatalf("expected a debug log with reason=%q; got: %+v", want, hook.AllEntries())
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

func TestHandleAssetMovedEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expectProcess bool
	}{
		{
			name:          "Process moved event",
			eventType:     asset2.StatusEventTypeMoved,
			expectProcess: true,
		},
		{
			name:          "Skip wrong event type",
			eventType:     "wrong_type",
			expectProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			event := asset2.StatusEvent[asset2.MovedStatusEventBody]{
				Type:          tt.eventType,
				TransactionId: uuid.New(),
				CharacterId:   12345,
			}

			// Execute - function should complete without panic
			handleAssetMovedEvent(logger, tctx, event)
		})
	}
}
