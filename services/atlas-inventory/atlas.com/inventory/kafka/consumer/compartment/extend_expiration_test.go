package compartment

import (
	"context"
	"testing"

	compartment2 "atlas-inventory/kafka/message/compartment"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func extendExpirationTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

// A command of the wrong type must be a no-op: every handler on this shared
// topic sees every message, so the type guard is what keeps an APPLY_LOCK
// from being processed as an extension.
func TestHandleExtendExpirationCommandIgnoresOtherTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := extendExpirationTestContext(t)
	c := compartment2.Command[compartment2.ExtendExpirationCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   12345,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          compartment2.CommandApplyLock,
		Body:          compartment2.ExtendExpirationCommandBody{Slot: -11},
	}
	// A nil *gorm.DB would panic if the handler proceeded past the guard.
	handleExtendExpirationCommand(nil)(l, ctx, c)
}
