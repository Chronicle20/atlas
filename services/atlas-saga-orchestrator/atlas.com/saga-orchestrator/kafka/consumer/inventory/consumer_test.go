package inventory

import (
	inventory2 "atlas-saga-orchestrator/kafka/message/inventory"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestHandleInventoryCreatedEvent_TypeGuard verifies the handler ignores events
// of the wrong type. This is a smoke test for the type-guard branch — the
// AcceptEvent integration is exercised in createandequip_integration_test.go.
func TestHandleInventoryCreatedEvent_TypeGuard(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.Background()
	e := inventory2.StatusEvent[inventory2.CreatedStatusEventBody]{
		TransactionId: uuid.New(),
		CharacterId:   100,
		Type:          inventory2.StatusEventTypeDeleted, // wrong type
		Body:          inventory2.CreatedStatusEventBody{},
	}
	// Should return immediately without panicking. AcceptEvent will not be
	// called because the type guard fails first.
	handleInventoryCreatedEvent(logrus.FieldLogger(l), ctx, e)
}

func TestHandleInventoryCreationFailedEvent_TypeGuard(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.Background()
	e := inventory2.StatusEvent[inventory2.CreationFailedStatusEventBody]{
		TransactionId: uuid.New(),
		CharacterId:   100,
		Type:          inventory2.StatusEventTypeCreated, // wrong type
		Body:          inventory2.CreationFailedStatusEventBody{Reason: "boom"},
	}
	handleInventoryCreationFailedEvent(logrus.FieldLogger(l), ctx, e)
}
