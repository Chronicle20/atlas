package compartment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	compartment2 "atlas-inventory/kafka/message/compartment"
)

// TestHandleApplyKarmaCommandIgnoresOtherTypes: every handler registered on the
// shared COMMAND_TOPIC_COMPARTMENT sees every command on that topic, so the type
// guard is load-bearing — without it an APPLY_LOCK body unmarshals into an
// ApplyKarmaCommandBody and mutates the wrong thing.
//
// The package's handlers construct their own compartment.Processor from a
// *gorm.DB and have no injection seam (see idempotency_test.go), so this test
// cannot assert the happy path without adding one. Per the task brief, it
// asserts only the type guard: a nil db proves the handler returns before
// ever touching the database when c.Type doesn't match, which a panic would
// disprove.
func TestHandleApplyKarmaCommandIgnoresOtherTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	handleApplyKarmaCommand(nil)(l, context.Background(), compartment2.Command[compartment2.ApplyKarmaCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   1,
		InventoryType: 1,
		Type:          compartment2.CommandApplyLock,
		Body:          compartment2.ApplyKarmaCommandBody{Slot: 3},
	})
}
