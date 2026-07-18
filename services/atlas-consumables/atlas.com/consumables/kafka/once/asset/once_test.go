package asset

import (
	"atlas-consumables/kafka/message/asset"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GrantConfirmedValidator must accept both the fresh-stack (CREATED) and the
// merge-into-existing-stack (QUANTITY_CHANGED) forms of a successful grant for
// the matching transaction AND rolled item, and reject everything else — most
// importantly the box's own asset events, which share the transaction but carry
// a different templateId.
func TestGrantConfirmedValidator(t *testing.T) {
	txn := uuid.New()
	other := uuid.New()
	const rewardItem = uint32(2041303)
	const boxItem = uint32(2022503)
	v := GrantConfirmedValidator(txn, rewardItem)
	l := logrus.New()
	ctx := context.Background()

	ev := func(id uuid.UUID, tmpl uint32, typ string) asset.StatusEvent[asset.CreatedStatusEventBody] {
		return asset.StatusEvent[asset.CreatedStatusEventBody]{TransactionId: id, TemplateId: tmpl, Type: typ}
	}
	cases := []struct {
		name string
		e    asset.StatusEvent[asset.CreatedStatusEventBody]
		want bool
	}{
		{"created fresh stack", ev(txn, rewardItem, asset.StatusEventTypeCreated), true},
		{"merged into existing stack", ev(txn, rewardItem, asset.StatusEventTypeQuantityChanged), true},
		{"box event same txn different item", ev(txn, boxItem, asset.StatusEventTypeQuantityChanged), false},
		{"reward item other txn", ev(other, rewardItem, asset.StatusEventTypeCreated), false},
		{"unrelated type", ev(txn, rewardItem, "MOVED"), false},
	}
	for _, c := range cases {
		if got := v(l, ctx, c.e); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
