package chat

import (
	"context"
	"testing"
)

// TestAvatarMegaphoneResultBodyResolvesErrorCode verifies the DOM-25 contract:
// AvatarMegaphoneResultBody never emits a literal wire byte — the notice
// selector is resolved per tenant from the writer's "errorCodes" table
// (mirrors TestViciousHammerFailureBodyResolvesModeAndErrorCode).
func TestAvatarMegaphoneResultBodyResolvesErrorCode(t *testing.T) {
	l := testLogger()
	ctx := context.Background()

	// A v83-shaped writer config: WAITING_LINE = 83, LEVEL_GATE = 84.
	options := map[string]interface{}{
		"errorCodes": map[string]interface{}{
			"WAITING_LINE": float64(83),
			"LEVEL_GATE":   float64(84),
		},
	}

	cases := []struct {
		reason   AvatarMegaphoneResultReason
		wantCode byte
	}{
		{AvatarMegaphoneWaitingLine, 83},
		{AvatarMegaphoneLevelGate, 84},
	}
	for _, c := range cases {
		t.Run(string(c.reason), func(t *testing.T) {
			got := AvatarMegaphoneResultBody(c.reason)(l, ctx)(options)
			// code-only wire: single byte, no trailing message.
			want := []byte{c.wantCode}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			if got[0] != want[0] {
				t.Fatalf("code byte: got 0x%02X, want 0x%02X", got[0], want[0])
			}
		})
	}
}

// TestAvatarMegaphoneResultBodyUnconfiguredReasonDegrades confirms a reason
// missing from the tenant "errorCodes" table degrades to 99 (generic client
// "Unknown error") rather than panicking or silently sending 0.
func TestAvatarMegaphoneResultBodyUnconfiguredReasonDegrades(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	options := map[string]interface{}{
		"errorCodes": map[string]interface{}{
			"WAITING_LINE": float64(83),
		},
	}
	got := AvatarMegaphoneResultBody(AvatarMegaphoneLevelGate)(l, ctx)(options)
	want := []byte{99}
	if len(got) != len(want) {
		t.Fatalf("byte count: got %d, want %d", len(got), len(want))
	}
	if got[0] != want[0] {
		t.Fatalf("code byte: got 0x%02X, want 0x%02X", got[0], want[0])
	}
}
