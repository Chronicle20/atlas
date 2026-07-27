package field

import (
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestViciousHammerFailureBodyResolvesModeAndErrorCode verifies the DOM-25
// contract: the failure body func emits neither the dispatcher mode byte nor
// the notice selector as a literal — both are resolved per tenant from the
// writer's "operations" and "errorCodes" config tables. The domain service
// only ever picks a semantic reason (task-129).
func TestViciousHammerFailureBodyResolvesModeAndErrorCode(t *testing.T) {
	l := logrus.New()
	l.SetOutput(io.Discard)
	ctx := context.Background()

	// A v83-shaped writer config: FAILURE mode = 62, wire codes 0/1/2/3.
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"OPEN": float64(0), "SUCCESS": float64(61), "FAILURE": float64(62),
		},
		"errorCodes": map[string]interface{}{
			"UNKNOWN":        float64(0),
			"NOT_UPGRADABLE": float64(1),
			"CAP_REACHED":    float64(2),
			"HORNTAIL":       float64(3),
		},
	}

	cases := []struct {
		reason   ViciousHammerFailureReason
		wantCode byte
	}{
		{ViciousHammerReasonNotUpgradable, 1},
		{ViciousHammerReasonCapReached, 2},
		{ViciousHammerReasonHorntail, 3},
		{ViciousHammerReasonUnknown, 0},
	}
	for _, c := range cases {
		t.Run(string(c.reason), func(t *testing.T) {
			got := ViciousHammerFailureBody(c.reason)(l, ctx)(options)
			// mode byte (62) + Encode4(errorCode) little-endian.
			want := []byte{62, c.wantCode, 0, 0, 0}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

// TestViciousHammerFailureBodyUnconfiguredReasonDegrades confirms a reason
// missing from the tenant "errorCodes" table degrades to 99 (generic client
// "Unknown error") rather than panicking or silently sending 0.
func TestViciousHammerFailureBodyUnconfiguredReasonDegrades(t *testing.T) {
	l := logrus.New()
	l.SetOutput(io.Discard)
	options := map[string]interface{}{
		"operations": map[string]interface{}{"FAILURE": float64(62)},
		"errorCodes": map[string]interface{}{"NOT_UPGRADABLE": float64(1)},
	}
	got := ViciousHammerFailureBody(ViciousHammerReasonHorntail)(l, context.Background())(options)
	want := []byte{62, 99, 0, 0, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
		}
	}
}
