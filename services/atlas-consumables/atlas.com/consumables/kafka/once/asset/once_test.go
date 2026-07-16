package asset

import (
	"context"
	"testing"

	"atlas-consumables/kafka/message/asset"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CreationValidator must accept only the CREATED event for the matching
// transaction, and reject other transactions or other event types.
func TestCreationValidator(t *testing.T) {
	txn := uuid.New()
	other := uuid.New()
	v := CreationValidator(txn)
	l := logrus.New()
	ctx := context.Background()

	cases := []struct {
		name string
		e    asset.StatusEvent[asset.CreatedStatusEventBody]
		want bool
	}{
		{"created matching txn", asset.StatusEvent[asset.CreatedStatusEventBody]{TransactionId: txn, Type: asset.StatusEventTypeCreated}, true},
		{"created other txn", asset.StatusEvent[asset.CreatedStatusEventBody]{TransactionId: other, Type: asset.StatusEventTypeCreated}, false},
		{"non-created type matching txn", asset.StatusEvent[asset.CreatedStatusEventBody]{TransactionId: txn, Type: "UPDATED"}, false},
		{"empty type matching txn", asset.StatusEvent[asset.CreatedStatusEventBody]{TransactionId: txn}, false},
	}
	for _, c := range cases {
		if got := v(l, ctx, c.e); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
