package transports

import (
	"testing"

	"github.com/google/uuid"
)

// design §7.4: "still underway" is state == in_transit AND the SAME voyage. A
// route that has since departed on the NEXT trip is in_transit again, and
// comparing state alone would wrongly report our voyage as ongoing.
func TestVoyageUnderwayRequiresBothStateAndIdentity(t *testing.T) {
	mine := uuid.New()
	next := uuid.New()

	for _, tc := range []struct {
		name string
		rm   RestModel
		want bool
	}{
		{"in transit, same voyage", RestModel{State: "in_transit", VoyageID: mine.String()}, true},
		{"in transit, next voyage", RestModel{State: "in_transit", VoyageID: next.String()}, false},
		{"awaiting return", RestModel{State: "awaiting_return", VoyageID: ""}, false},
		{"open entry", RestModel{State: "open_entry", VoyageID: ""}, false},
	} {
		if got := VoyageUnderway(tc.rm, mine); got != tc.want {
			t.Fatalf("%s: VoyageUnderway = %v, want %v", tc.name, got, tc.want)
		}
	}
}
