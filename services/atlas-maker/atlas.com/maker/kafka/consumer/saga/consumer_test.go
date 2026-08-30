package saga

import (
	"atlas-maker/craft"
	msgsaga "atlas-maker/kafka/message/saga"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// registeredHandlers captures every handler.Handler InitHandlers registers,
// keyed by registration order (completed first, then failed -- see
// InitHandlers), without needing a real consumer.Manager or broker.
func registeredHandlers(t *testing.T) []handler.Handler {
	t.Helper()
	var handlers []handler.Handler
	rf := func(topic string, h handler.Handler) (string, error) {
		handlers = append(handlers, h)
		return "registration-id", nil
	}
	err := InitHandlers(logrus.StandardLogger())(rf)
	require.NoError(t, err)
	require.Len(t, handlers, 2, "InitHandlers must register exactly the completed and failed handlers")
	return handlers
}

func contextForTenant(tenantId uuid.UUID) context.Context {
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		panic(err)
	}
	return tenant.WithContext(context.Background(), te)
}

func statusEventMessage(t *testing.T, v any) kafka.Message {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return kafka.Message{Value: b}
}

func TestCompletedEventReleasesGuard(t *testing.T) {
	tenantId := uuid.New()
	txId := uuid.New()

	require.True(t, craft.AcquireForTest(tenantId, 1001))
	craft.TrackForTest(tenantId, 1001, txId)

	handlers := registeredHandlers(t)
	ctx := contextForTenant(tenantId)
	msg := statusEventMessage(t, msgsaga.StatusEvent[msgsaga.StatusEventCompletedBody]{
		TransactionId: txId,
		Type:          msgsaga.StatusEventTypeCompleted,
	})

	_, err := handlers[0](logrus.StandardLogger(), ctx, msg)
	require.NoError(t, err)

	assert.True(t, craft.AcquireForTest(tenantId, 1001), "a COMPLETED terminal event must release the guard")
}

func TestFailedEventReleasesGuard(t *testing.T) {
	tenantId := uuid.New()
	txId := uuid.New()

	require.True(t, craft.AcquireForTest(tenantId, 1002))
	craft.TrackForTest(tenantId, 1002, txId)

	handlers := registeredHandlers(t)
	ctx := contextForTenant(tenantId)
	msg := statusEventMessage(t, msgsaga.StatusEvent[msgsaga.StatusEventFailedBody]{
		TransactionId: txId,
		Type:          msgsaga.StatusEventTypeFailed,
		Body: msgsaga.StatusEventFailedBody{
			ErrorCode:  "SOME_ERROR",
			Reason:     "reason",
			FailedStep: "step",
		},
	})

	_, err := handlers[1](logrus.StandardLogger(), ctx, msg)
	require.NoError(t, err)

	assert.True(t, craft.AcquireForTest(tenantId, 1002), "a FAILED terminal event must release the guard")
}

func TestNonTerminalEventDoesNotRelease(t *testing.T) {
	tenantId := uuid.New()
	txId := uuid.New()

	require.True(t, craft.AcquireForTest(tenantId, 1003))
	craft.TrackForTest(tenantId, 1003, txId)

	handlers := registeredHandlers(t)
	ctx := contextForTenant(tenantId)
	msg := statusEventMessage(t, msgsaga.StatusEvent[msgsaga.StatusEventCompletedBody]{
		TransactionId: txId,
		Type:          "SOME_OTHER_TYPE",
	})

	_, err := handlers[0](logrus.StandardLogger(), ctx, msg)
	require.NoError(t, err)

	assert.False(t, craft.AcquireForTest(tenantId, 1003), "an event whose Type matches neither terminal constant must leave the guard held")
}

func TestReleaseIsTenantScoped(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	txId := uuid.New()

	require.True(t, craft.AcquireForTest(tenantB, 1004))
	craft.TrackForTest(tenantB, 1004, txId)

	handlers := registeredHandlers(t)
	// The event carries tenant A's context (a different transaction id
	// entirely, but even a colliding transaction id must not cross tenants).
	ctx := contextForTenant(tenantA)
	msg := statusEventMessage(t, msgsaga.StatusEvent[msgsaga.StatusEventCompletedBody]{
		TransactionId: txId,
		Type:          msgsaga.StatusEventTypeCompleted,
	})

	_, err := handlers[0](logrus.StandardLogger(), ctx, msg)
	require.NoError(t, err)

	assert.False(t, craft.AcquireForTest(tenantB, 1004), "a terminal event for tenant A must not release tenant B's entry")
}
