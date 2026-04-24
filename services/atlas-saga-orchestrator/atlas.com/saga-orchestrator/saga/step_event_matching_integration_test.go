//go:build test

package saga_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	characterMock "atlas-saga-orchestrator/character/mock"
	compartmentMock "atlas-saga-orchestrator/compartment/mock"
	"atlas-saga-orchestrator/kafka/consumer/asset"
	"atlas-saga-orchestrator/kafka/consumer/character"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/saga"
)

// TestThiefAdvancementScenario replays the event sequence from PRD §9.1 and
// asserts the fix from task-021: no ripple STAT_CHANGED advances an
// AwardAsset step, and the three conversation_reward_notice emissions land
// in the expected order with the correct templateIds.
//
// The test injects mock character+compartment processors via
// SetProcessorFactoryForTest so that step-dispatch side effects (Kafka
// command emissions) are no-ops. This isolates the test to the
// step/event matching logic under test.
func TestThiefAdvancementScenario(t *testing.T) {
	// 1. Install notice-emitter stub.
	var noticeMu sync.Mutex
	type noticeCall struct {
		TemplateId uint32
		Quantity   uint32
	}
	var noticeCalls []noticeCall
	origEmit := saga.SetEmitConversationRewardNoticeForTest(
		func(l logrus.FieldLogger, ctx context.Context, characterId uint32, kind string, itemId uint32, quantity uint32) error {
			noticeMu.Lock()
			defer noticeMu.Unlock()
			noticeCalls = append(noticeCalls, noticeCall{itemId, quantity})
			return nil
		},
	)
	defer saga.SetEmitConversationRewardNoticeForTest(origEmit)

	// 2. Inject no-op processor mocks so step-dispatch doesn't try to hit Kafka.
	charMock := &characterMock.ProcessorMock{}
	compMock := &compartmentMock.ProcessorMock{}
	var origFactory func(l logrus.FieldLogger, ctx context.Context) saga.Processor
	origFactory = saga.SetProcessorFactoryForTest(func(l logrus.FieldLogger, ctx context.Context) saga.Processor {
		return origFactory(l, ctx).
			WithCharacterProcessor(charMock).
			WithCompartmentProcessor(compMock)
	})
	defer saga.SetProcessorFactoryForTest(origFactory)

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	// 3. Build the 5-step saga from §9.1.
	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("integration_test").
		AddStep("rebalance_ap-12", saga.Pending, saga.RebalanceAP, saga.RebalanceAPPayload{CharacterId: 12}).
		AddStep("change_job-12", saga.Pending, saga.ChangeJob, saga.ChangeJobPayload{CharacterId: 12, JobId: 400}).
		AddStep("award_item-12", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 12,
			Item:        saga.ItemPayload{TemplateId: 2070015, Quantity: 1},
			ShowEffect:  true,
		}).
		AddStep("award_item-12-3", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 12,
			Item:        saga.ItemPayload{TemplateId: 1472061, Quantity: 1},
			ShowEffect:  true,
		}).
		AddStep("award_item-12-4", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 12,
			Item:        saga.ItemPayload{TemplateId: 1332063, Quantity: 1},
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, saga.GetCache().Put(ctx, s))

	// 4. Replay the §9.1 event sequence.
	// 4.1 STAT_CHANGED (rebalance) → completes step 1.
	character.HandleCharacterStatChangedEventForTest(logger, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		Type:          character2.StatusEventTypeStatChanged,
		TransactionId: tx,
	})
	// 4.2 JOB_CHANGED → completes step 2.
	character.HandleCharacterJobChangedEventForTest(logger, ctx, character2.StatusEvent[character2.JobChangedStatusEventBody]{
		Type:          character2.StatusEventTypeJobChanged,
		TransactionId: tx,
	})
	// 4.3 STAT_CHANGED ripple ([JOB]) — MUST NOT complete step 3.
	character.HandleCharacterStatChangedEventForTest(logger, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		Type:          character2.StatusEventTypeStatChanged,
		TransactionId: tx,
	})
	// 4.4 STAT_CHANGED ripple (AP/SP/HP/MP) — MUST NOT complete step 4.
	character.HandleCharacterStatChangedEventForTest(logger, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		Type:          character2.StatusEventTypeStatChanged,
		TransactionId: tx,
	})
	// 4.5 ASSET_CREATED 2070015 → completes step 3 with correct notice.
	asset.HandleAssetCreatedEventForTest(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   12,
		TemplateId:    2070015,
		AssetId:       1,
		Body:          asset2.CreatedStatusEventBody{AssetData: asset2.AssetData{Quantity: 1}},
	})
	// 4.6 ASSET_CREATED 1472061 → completes step 4.
	asset.HandleAssetCreatedEventForTest(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   12,
		TemplateId:    1472061,
		AssetId:       2,
		Body:          asset2.CreatedStatusEventBody{AssetData: asset2.AssetData{Quantity: 1}},
	})
	// 4.7 ASSET_CREATED 1332063 → completes step 5.
	asset.HandleAssetCreatedEventForTest(logger, ctx, asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		Type:          asset2.StatusEventTypeCreated,
		TransactionId: tx,
		CharacterId:   12,
		TemplateId:    1332063,
		AssetId:       3,
		Body:          asset2.CreatedStatusEventBody{AssetData: asset2.AssetData{Quantity: 1}},
	})

	// 5. Assertions. On success the saga is removed from the cache — if it's
	// gone, all 5 steps completed and the completion hook fired (saga terminal
	// success). If it's still in the cache, some step didn't complete.
	_, err = saga.NewProcessor(logger, ctx).GetById(tx)
	assert.Error(t, err, "saga should be removed from cache after all steps complete")

	// Notices: exactly three, in order [2070015, 1472061, 1332063].
	require.Len(t, noticeCalls, 3, "expected exactly three conversation_reward_notice emissions")
	assert.Equal(t, uint32(2070015), noticeCalls[0].TemplateId)
	assert.Equal(t, uint32(1472061), noticeCalls[1].TemplateId)
	assert.Equal(t, uint32(1332063), noticeCalls[2].TemplateId)

	// Bucket log entries for precise assertions. The warn-once logic
	// (processor.maybeWarnUnmatchedEvent, task-021 Task 17) fires exactly
	// one WarnLevel entry with reason=unmatched_event for the first
	// STAT_CHANGED ripple at step 4.3: at that point the remaining pending
	// steps are all AwardAsset, none of which accept CharacterStatChanged,
	// so the event is genuinely unmatched. The second ripple at step 4.4 is
	// deduped by the warn-once sync.Map. No other saga-layer warns are
	// expected. Infrastructure-layer Kafka warnings (unset env vars,
	// "Unable to emit event on topic") are unrelated to step-matching and
	// are filtered out.
	var (
		sagaLayerWarns       []*logrus.Entry
		noPendingStepEntries []*logrus.Entry
		actionMismatchCount  int
	)
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			msg := e.Message
			if strings.Contains(msg, "environment variable not set") ||
				strings.Contains(msg, "Unable to emit event on topic") {
				continue
			}
			sagaLayerWarns = append(sagaLayerWarns, e)
		}
		if e.Data["reason"] == saga.SkipReasonNoPendingStep {
			noPendingStepEntries = append(noPendingStepEntries, e)
		}
		if e.Data["reason"] == saga.SkipReasonActionMismatch {
			actionMismatchCount++
		}
	}

	assert.Empty(t, noPendingStepEntries, "no 'no_pending_step' logs expected in happy path")
	// Exactly one unmatched_event warn for the first STAT_CHANGED ripple;
	// the second ripple is deduped by warn-once.
	require.Len(t, sagaLayerWarns, 1, "expected exactly one saga-layer warn (unmatched_event, once per (tx, kind))")
	assert.Equal(t, saga.SkipReasonUnmatchedEvent, sagaLayerWarns[0].Data["reason"])
	assert.Equal(t, saga.EventKindCharacterStatChanged, sagaLayerWarns[0].Data["event_kind"])
	assert.Equal(t, 2, actionMismatchCount, "exactly two ripple STAT_CHANGED events expected to skip")
}
