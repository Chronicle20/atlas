package compartment

import (
	"atlas-consumables/kafka/message/compartment"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CreationFailedValidator must accept only CREATION_FAILED for the matching
// transaction — never CREATED (that success signal lives on the asset topic).
func TestCreationFailedValidator(t *testing.T) {
	txn := uuid.New()
	other := uuid.New()
	v := CreationFailedValidator(txn)
	l := logrus.New()
	ctx := context.Background()

	cases := []struct {
		name string
		e    compartment.StatusEvent[compartment.CreateResultEventBody]
		want bool
	}{
		{"failed matching txn", compartment.StatusEvent[compartment.CreateResultEventBody]{TransactionId: txn, Type: compartment.StatusEventTypeCreationFailed}, true},
		{"failed other txn", compartment.StatusEvent[compartment.CreateResultEventBody]{TransactionId: other, Type: compartment.StatusEventTypeCreationFailed}, false},
		{"created matching txn is not a failure", compartment.StatusEvent[compartment.CreateResultEventBody]{TransactionId: txn, Type: compartment.StatusEventTypeCreated}, false},
	}
	for _, c := range cases {
		if got := v(l, ctx, c.e); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
