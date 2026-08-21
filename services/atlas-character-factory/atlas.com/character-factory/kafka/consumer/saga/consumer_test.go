package saga

import (
	sagaMessage "atlas-character-factory/kafka/message/saga"
	seedMessage "atlas-character-factory/kafka/message/seed"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

var emitted *producertest.Capture

// TestMain installs the shared capturing producer manager so every test in
// this package can inspect what the saga-status bridge emitted on
// EVENT_TOPIC_SEED_STATUS, without a broker.
func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

func decodeCreated(t *testing.T, raw []byte) seedMessage.StatusEvent[seedMessage.CreatedStatusEventBody] {
	t.Helper()
	var ev seedMessage.StatusEvent[seedMessage.CreatedStatusEventBody]
	require.NoError(t, json.Unmarshal(raw, &ev))
	return ev
}

func decodeFailed(t *testing.T, raw []byte) seedMessage.StatusEvent[seedMessage.FailedStatusEventBody] {
	t.Helper()
	var ev seedMessage.StatusEvent[seedMessage.FailedStatusEventBody]
	require.NoError(t, json.Unmarshal(raw, &ev))
	return ev
}

// TestCompletedBridgeCarriesTransactionId drives a CharacterCreation saga
// COMPLETED event through handleSagaCompletedEvent and asserts the emitted
// seed CREATED event carries the same transactionId, so Task 14's channel
// consumer can correlate the async result back to the pending dialog.
func TestCompletedBridgeCarriesTransactionId(t *testing.T) {
	emitted.Reset()
	l, _ := testlog.NewNullLogger()
	transactionId := uuid.New()

	e := sagaMessage.StatusEvent[sagaMessage.StatusEventCompletedBody]{
		TransactionId: transactionId,
		Type:          sagaMessage.StatusEventTypeCompleted,
		Body: sagaMessage.StatusEventCompletedBody{
			SagaType: string(sharedsaga.CharacterCreation),
			Results: map[string]any{
				"accountId":   uint32(7),
				"characterId": uint32(99),
			},
		},
	}

	handleSagaCompletedEvent(l, context.Background(), e)

	msgs := emitted.Messages(seedMessage.EnvEventTopicStatus)
	require.Len(t, msgs, 1)
	ev := decodeCreated(t, msgs[0].Value)
	require.Equal(t, transactionId.String(), ev.TransactionId)
	require.EqualValues(t, 7, ev.AccountId)
	require.EqualValues(t, 99, ev.Body.CharacterId)
}

// TestFailedBridgeCarriesTransactionId drives a CharacterCreation saga
// FAILED event through handleSagaFailedEvent and asserts the emitted seed
// FAILED event carries the same transactionId.
func TestFailedBridgeCarriesTransactionId(t *testing.T) {
	emitted.Reset()
	l, _ := testlog.NewNullLogger()
	transactionId := uuid.New()

	e := sagaMessage.StatusEvent[sagaMessage.StatusEventFailedBody]{
		TransactionId: transactionId,
		Type:          sagaMessage.StatusEventTypeFailed,
		Body: sagaMessage.StatusEventFailedBody{
			AccountId: 7,
			Reason:    "name_taken",
			SagaType:  string(sharedsaga.CharacterCreation),
		},
	}

	handleSagaFailedEvent(l, context.Background(), e)

	msgs := emitted.Messages(seedMessage.EnvEventTopicStatus)
	require.Len(t, msgs, 1)
	ev := decodeFailed(t, msgs[0].Value)
	require.Equal(t, transactionId.String(), ev.TransactionId)
	require.EqualValues(t, 7, ev.AccountId)
	require.Equal(t, "name_taken", ev.Body.Reason)
}

// TestNonCharacterCreationSagaStillDropped pins the existing sagaType filter
// (consumer.go:53-56,96-104): a saga type other than character_creation must
// not reach the seed topic, unaffected by threading transactionId through.
func TestNonCharacterCreationSagaStillDropped(t *testing.T) {
	emitted.Reset()
	l, _ := testlog.NewNullLogger()
	transactionId := uuid.New()

	completed := sagaMessage.StatusEvent[sagaMessage.StatusEventCompletedBody]{
		TransactionId: transactionId,
		Type:          sagaMessage.StatusEventTypeCompleted,
		Body: sagaMessage.StatusEventCompletedBody{
			SagaType: "inventory_transaction",
			Results: map[string]any{
				"accountId":   uint32(7),
				"characterId": uint32(99),
			},
		},
	}
	handleSagaCompletedEvent(l, context.Background(), completed)
	require.Empty(t, emitted.Messages(seedMessage.EnvEventTopicStatus))

	failed := sagaMessage.StatusEvent[sagaMessage.StatusEventFailedBody]{
		TransactionId: transactionId,
		Type:          sagaMessage.StatusEventTypeFailed,
		Body: sagaMessage.StatusEventFailedBody{
			AccountId: 7,
			Reason:    "some_reason",
			SagaType:  "inventory_transaction",
		},
	}
	handleSagaFailedEvent(l, context.Background(), failed)
	require.Empty(t, emitted.Messages(seedMessage.EnvEventTopicStatus))
}

// TestStatusEventMarshalsTransactionId pins the wire contract: a populated
// TransactionId marshals to "transactionId", and an empty one is omitted
// entirely so an older producer's payload (no transactionId key at all)
// round-trips identically through this same struct.
func TestStatusEventMarshalsTransactionId(t *testing.T) {
	withTx := seedMessage.StatusEvent[seedMessage.CreatedStatusEventBody]{
		AccountId:     7,
		TransactionId: "tx-1",
		Type:          seedMessage.StatusEventTypeCreated,
	}
	raw, err := json.Marshal(withTx)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"transactionId":"tx-1"`)

	withoutTx := seedMessage.StatusEvent[seedMessage.CreatedStatusEventBody]{
		AccountId: 7,
		Type:      seedMessage.StatusEventTypeCreated,
	}
	raw, err = json.Marshal(withoutTx)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "transactionId")
}
