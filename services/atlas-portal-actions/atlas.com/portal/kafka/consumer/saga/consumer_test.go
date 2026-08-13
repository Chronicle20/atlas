package saga

import (
	"atlas-portal-actions/action"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	warpDefaultMessage      = "You cannot move there right now."
	transportDefaultMessage = "Unable to board transport at this time."
)

// FR-2.7: a failed portal warp must not report a transport boarding failure.
func TestResolveFailureMessage_WarpDefault(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindWarp}
	assert.Equal(t, warpDefaultMessage, resolveFailureMessage(pa, ""))
}

func TestResolveFailureMessage_TransportDefault(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindTransport}
	assert.Equal(t, transportDefaultMessage, resolveFailureMessage(pa, ""))
}

// A registry entry written before Kind existed must keep today's text.
func TestResolveFailureMessage_EmptyKindDefaultsToTransport(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1}
	assert.Equal(t, transportDefaultMessage, resolveFailureMessage(pa, ""))
}

// An explicit failureMessage from the script still wins over everything.
func TestResolveFailureMessage_ExplicitMessageWins(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindWarp, FailureMessage: "custom"}
	assert.Equal(t, "custom", resolveFailureMessage(pa, "TRANSPORT_CAPACITY_FULL"))
}

// The transport error codes keep their specific messages, and remain
// unreachable for a warp saga (nothing on the warp path emits them).
func TestResolveFailureMessage_ErrorCodesUnchanged(t *testing.T) {
	pa := action.PendingAction{CharacterId: 1, Kind: action.KindTransport}
	assert.Equal(t, "The transport is currently full. Please try again later.",
		resolveFailureMessage(pa, "TRANSPORT_CAPACITY_FULL"))
	assert.Equal(t, "You are already on a transport.",
		resolveFailureMessage(pa, "TRANSPORT_ALREADY_IN_TRANSIT"))
	assert.Equal(t, "Transport service is currently unavailable.",
		resolveFailureMessage(pa, "TRANSPORT_ROUTE_NOT_FOUND"))
	assert.Equal(t, "Transport service is currently unavailable.",
		resolveFailureMessage(pa, "TRANSPORT_SERVICE_ERROR"))
}
