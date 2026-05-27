// Package listener owns the per-(tenant, world, channel) listener
// lifecycle in atlas-channel. Each Handle wraps the per-(t,w,c) startup
// work (server.Register, account registry init, consumer InitHandlers,
// socket service) and exposes a four-phase Drain so the projection apply
// loop can remove a listener cleanly when config drops it.
package listener

import (
	"context"
	"sync"

	"atlas-channel/server"
)

// State tracks where a Handle is in its lifecycle. State transitions are
// monotonic — once Removed, the Handle is not revivable; a fresh Add must
// be invoked under a new Handle.
type State int

const (
	// Active is the steady state: the listener is accepting traffic.
	Active State = iota
	// Draining means Drain has begun but teardown is in flight.
	Draining
	// Removed is terminal — all kafka handlers deregistered, ctx canceled.
	Removed
)

// HandlerHandle identifies a registered kafka consumer handler. Returned
// by InitHandlers (post Phase H sweep) and stored on Handle so Drain
// can call consumer.Manager.RemoveHandler for each.
type HandlerHandle struct {
	Topic string
	Id    string
}

// Handle is the per-(t,w,c) listener state.
type Handle struct {
	Key           server.Key
	State         State
	Ctx           context.Context
	Cancel        context.CancelFunc
	Wg            *sync.WaitGroup
	ServerModel   server.Model
	KafkaHandlers []HandlerHandle
}
