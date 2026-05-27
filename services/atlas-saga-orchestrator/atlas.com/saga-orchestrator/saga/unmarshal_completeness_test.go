package saga

import (
	"fmt"
	"strings"
	"testing"
)

// TestStepUnmarshal_EveryActionRepresented asserts Step[any].UnmarshalJSON
// (saga/model.go) has a switch case for every Action constant declared in
// libs/atlas-saga/model.go.
//
// The orchestrator carries its own Step[T].UnmarshalJSON with a parallel
// switch separate from libs/atlas-saga/unmarshal.go. Drift between the two
// is silent: the consumer fails to decode the saga command, drops the
// message, and any saga touching the un-cased action wedges in pending.
// PR #491 hit this with AwaitInventoryCreated — see commit 5be8e31ad.
//
// We reuse the centralised allActions slice from event_acceptance_test.go
// (same package) as the source of truth. The expectation: missing case ⇒
// switch hits its default and returns "unknown action: <name>". Payload
// validation errors are tolerated — coverage of the switch is the only
// invariant this test protects.
func TestStepUnmarshal_EveryActionRepresented(t *testing.T) {
	for _, a := range allActions {
		t.Run(string(a), func(t *testing.T) {
			payload := fmt.Sprintf(`{"action":%q,"payload":{}}`, string(a))
			var step Step[any]
			err := step.UnmarshalJSON([]byte(payload))
			if err != nil && strings.Contains(err.Error(), "unknown action") {
				t.Fatalf("Step[any].UnmarshalJSON missing case for Action %q — add a case to services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go before the default branch, mirroring the existing pattern. err=%v", a, err)
			}
		})
	}
}
