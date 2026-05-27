// Package listener owns the per-tenant listener lifecycle in atlas-login.
// Each Handle wraps the per-tenant startup work (account registry init,
// consumer InitHandlers, socket service) and exposes a four-phase Drain so
// the projection apply loop can remove a listener cleanly when config
// drops it. atlas-login's listener is simpler than atlas-channel's: there
// is no per-(world, channel) fan-out, no upstream registration to undo,
// and no per-tenant session save-and-kick phase (login sessions are
// stateless after handshake).
package listener

import (
	"context"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
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

// Key uniquely identifies a per-tenant listener in atlas-login. Comparable
// by value so it can be used as a map key. atlas-login keys on TenantId
// alone — there is one login socket per tenant.
type Key struct {
	TenantId uuid.UUID
}

// ServerModel carries the per-tenant socket binding (IP/port) plus the
// tenant identity. Mirrors atlas-channel/server.Model in role; kept inside
// the listener package because atlas-login doesn't need a standalone
// server registry.
type ServerModel struct {
	tenant    tenant.Model
	ipAddress string
	port      int
}

// NewServerModel constructs a ServerModel.
func NewServerModel(t tenant.Model, ipAddress string, port int) ServerModel {
	return ServerModel{tenant: t, ipAddress: ipAddress, port: port}
}

// Tenant returns the tenant this listener serves.
func (m ServerModel) Tenant() tenant.Model { return m.tenant }

// IpAddress returns the externally-advertised IP for the socket.
func (m ServerModel) IpAddress() string { return m.ipAddress }

// Port returns the TCP port the socket binds on.
func (m ServerModel) Port() int { return m.port }

// HandlerHandle identifies a registered kafka consumer handler. Returned
// by InitHandlers (post K3 sweep) and stored on Handle so Drain can call
// consumer.Manager.RemoveHandler for each.
type HandlerHandle struct {
	Topic string
	Id    string
}

// Handle is the per-tenant listener state.
type Handle struct {
	Key           Key
	State         State
	Ctx           context.Context
	Cancel        context.CancelFunc
	Wg            *sync.WaitGroup
	ServerModel   ServerModel
	KafkaHandlers []HandlerHandle
}
