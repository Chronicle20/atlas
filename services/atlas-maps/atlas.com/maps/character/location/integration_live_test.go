//go:build integration

package location

import "testing"

// Scenarios I2, I7, I8 from design.md §8 require multi-service or live
// infrastructure (multiple atlas services, real Kafka, real Redis, network
// faults) and cannot run against the in-memory sqlite + stub info.Processor
// harness used by integration_test.go. They are gated behind the `integration`
// build tag and serve as scaffolding for a future live-stack test runner.
//
// Run with:
//   go test -tags=integration ./character/location/...
//
// Each test below currently t.Skip()s with a description of the missing
// infrastructure.

// TestI2_DisconnectOnTransitMap — design.md §8 row I2.
// "Disconnect on transit map 200090000 → Login lands at 200000100 (Orbis dock).
//  HandleLogin no-op."
//
// Why live infra: the transit map redirect lives in atlas-transports (the
// in-flight route table) and atlas-channel's HandleLogin no-op path. Verifying
// the end-to-end behavior requires atlas-maps + atlas-channel + atlas-transports
// + Kafka so the disconnect → resolve → login flow can be observed.
func TestI2_DisconnectOnTransitMap(t *testing.T) {
	t.Skip("requires live atlas-channel + atlas-transports + Kafka stack")
}

// TestI7_ConcurrentDisconnectDuringChannelChange — design.md §8 row I7.
// "Concurrent disconnect during channel-change → character_locations final
//  value matches disconnect's resolution."
//
// Why live infra: this exercises a race between two services (atlas-channel
// emitting CHANGE_CHANNEL_REQUEST while the socket simultaneously closes,
// triggering DISCONNECT). Reproducing the race deterministically requires a
// real Kafka broker with controllable consumer offsets and at least two
// atlas-channel instances.
func TestI7_ConcurrentDisconnectDuringChannelChange(t *testing.T) {
	t.Skip("requires live atlas-channel + Kafka with race orchestration")
}

// TestI8_AtlasMapsUnreachableDuringSessionBootstrap — design.md §8 row I8.
// "atlas-maps unreachable during session bootstrap → atlas-channel returns
//  error to client; player at character-select."
//
// Why live infra: this scenario asserts atlas-channel's behavior when its
// HTTP call to atlas-maps fails. Reproducing requires the atlas-channel binary,
// a real network endpoint configurable to fail (e.g. a toxiproxy or a stopped
// atlas-maps container), and a client harness to observe the character-select
// fallback.
func TestI8_AtlasMapsUnreachableDuringSessionBootstrap(t *testing.T) {
	t.Skip("requires live atlas-channel + controllable atlas-maps endpoint")
}
