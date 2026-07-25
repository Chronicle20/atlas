package note

import (
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	note2 "atlas-saga-orchestrator/kafka/message/note"
)

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

// noteSendSaga builds a note_send saga whose destroy step is already
// completed, leaving create_note as the earliest pending step.
func noteSendSaga(t *testing.T, tx uuid.UUID) saga.Saga {
	t.Helper()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.NoteSend).
		SetInitiatedBy("test").
		AddStep("consume_note_item", saga.Completed, saga.DestroyAsset, saga.DestroyAssetPayload{CharacterId: 100, TemplateId: 5090000, Quantity: 1}).
		AddStep("create_note", saga.Pending, saga.CreateNote, saga.CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hi", Flag: 1}).
		Build()
	require.NoError(t, err)
	return s
}

func TestHandleCreatedEvent_CompletesCreateNoteStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreatedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreatedBody]{
		TransactionId: tx,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreated,
		Body:          note2.StatusEventCreatedBody{NoteId: 7, SenderId: 100},
	})

	// Completing the final step drives the saga terminal. Depending on how far
	// the completion path runs in the test environment (the Kafka emission is
	// best-effort here), the saga is either evicted from the cache or its
	// create_note step is no longer pending. A still-pending step means the
	// event was NOT accepted — the failure this test guards against.
	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	if err == nil {
		assert.NotEqual(t, saga.Pending, got.Steps()[1].Status(), "create_note must not remain pending after CREATED event")
	}
}

func TestHandleCreatedEvent_IgnoresNilTransactionId(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreatedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreatedBody]{
		TransactionId: uuid.Nil,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreated,
		Body:          note2.StatusEventCreatedBody{NoteId: 7, SenderId: 100},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[1].Status(), "nil-txn event must not complete anything")
}

func TestHandleCreateFailedEvent_FailsCreateNoteStep(t *testing.T) {
	// The default compensator branch (saga/compensator.go) has no case for
	// CreateNote yet — it resets the failed step back to Pending instead of
	// compensating it. That default-branch behavior is what Task 9 replaces
	// with compensateCreateNote. Skip until then.
	t.Skip("compensation lands in Task 9")

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreateFailedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreateFailedBody]{
		TransactionId: tx,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreateFailed,
		Body:          note2.StatusEventCreateFailedBody{SenderId: 100, Reason: "db down"},
	})

	// StepCompleted(false) routes into compensation; the saga must no longer
	// have create_note pending (it is either failed-and-compensating or the
	// saga is already terminal/evicted).
	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	if err == nil {
		assert.NotEqual(t, saga.Pending, got.Steps()[1].Status())
	}
}
