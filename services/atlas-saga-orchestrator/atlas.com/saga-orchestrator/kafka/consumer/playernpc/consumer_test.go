package playernpc

import (
	playernpcmsg "atlas-saga-orchestrator/kafka/message/playernpc"
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTenantContext(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func putTestSaga(t *testing.T, ctx context.Context, s saga.Saga) {
	t.Helper()
	require.NoError(t, saga.GetCache().Put(ctx, s))
}

func newDeploySaga(t *testing.T, ctx context.Context, tx uuid.UUID, characterId uint32) {
	t.Helper()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.RemoteNpcUse).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.DeployPlayerNpc, saga.DeployPlayerNpcPayload{CharacterId: characterId}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: characterId, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)
}

// TestHandleCommandSucceededEvent_IgnoresWrongType guards the shared-topic
// fan-out: every handler registered on EVENT_TOPIC_PLAYER_NPC_STATUS sees
// every event, so the type check is what stops a COMMAND_FAILED (or a
// DEPLOYED/UPDATED/REMOVED/REPOSITIONED domain event) from completing a
// deploy_player_npc step.
func TestHandleCommandSucceededEvent_IgnoresWrongType(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleCommandSucceededEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandFailed,
		Body: playernpcmsg.StatusCommandOutcomeBody{TransactionId: uuid.New(), CharacterId: 1234},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a non-COMMAND_SUCCEEDED event: %v", hook.Entries)
	}
}

// TestHandleCommandSucceededEvent_NilTransactionIgnored: the ordinary GM
// deploy path produces commands with uuid.Nil, and any status event carrying
// that value must never advance a saga.
func TestHandleCommandSucceededEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleCommandSucceededEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandSucceeded,
		Body: playernpcmsg.StatusCommandOutcomeBody{TransactionId: uuid.Nil, CharacterId: 1234},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleCommandFailedEvent_NilTransactionIgnored mirrors the success-path
// nil-transaction guard for the failure handler.
func TestHandleCommandFailedEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleCommandFailedEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandFailed,
		Body: playernpcmsg.StatusCommandOutcomeBody{TransactionId: uuid.Nil, CharacterId: 1234, Code: "pool_exhausted"},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleCommandSucceededEvent_CompletesDeployStep verifies the
// COMMAND_SUCCEEDED→success polarity for a deploy_player_npc step.
func TestHandleCommandSucceededEvent_CompletesDeployStep(t *testing.T) {
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	newDeploySaga(t, ctx, tx, 1234)

	handleCommandSucceededEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandSucceeded,
		Body: playernpcmsg.StatusCommandOutcomeBody{TransactionId: tx, CharacterId: 1234, CommandType: playernpcmsg.CommandTypeDeploy},
	})

	got, err := saga.NewProcessor(l, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "deploy_player_npc step must be completed by COMMAND_SUCCEEDED")
}

// TestHandleCommandFailedEvent_FailsDeployStep verifies the
// COMMAND_FAILED→failure polarity. The FR-6.3 errorCode assertion lives in
// errorcode_result_test.go, which needs the retainingCache seam because a
// single-step saga's failure cascades synchronously into compensation and
// evicts the cache before a plain GetById could inspect it.
func TestHandleCommandFailedEvent_FailsDeployStep(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	newDeploySaga(t, ctx, tx, 1234)

	handleCommandFailedEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandFailed,
		Body: playernpcmsg.StatusCommandOutcomeBody{
			TransactionId: tx,
			CharacterId:   1234,
			CommandType:   playernpcmsg.CommandTypeDeploy,
			Code:          "pool_exhausted",
		},
	})

	foundMarkedFailed := false
	for _, e := range hook.AllEntries() {
		if e.Message == "Marked earliest pending step as [failed]." && e.Data["step_id"] == "s1" {
			foundMarkedFailed = true
		}
	}
	assert.True(t, foundMarkedFailed, "deploy_player_npc step must be driven to Failed by COMMAND_FAILED: %+v", hook.AllEntries())
}
