package socket

import (
	"atlas-channel/channel"
	"atlas-channel/character/chakra"
	"atlas-channel/character/statreset"
	"atlas-channel/remotemerchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const idleThreshold = 30 * time.Second

// NewListenerContext builds the context for a tenant's socket-listener
// registration: the tenant this listener serves, plus this pod's own
// environment (env.Self()) -- mirrors socket/handler's per-request session
// context origination (FR-2.2). CreateSocketService's Create/Destroy/
// SendPing callbacks below are wired directly from this context, bypassing
// socket/handler.AdaptHandler entirely, so the environment has to be
// originated here too -- not only on the per-packet handler path -- or
// every session-lifecycle Kafka event leaves a PR pod with an empty
// ENVIRONMENT header.
func NewListenerContext(ctx context.Context, t tenant.Model) context.Context {
	tctx := tenant.WithContext(ctx, t)
	return env.WithContext(tctx, env.Self())
}

// WithSelfEnvironment attaches this pod's own environment identity
// (env.Self()) to ctx, with no tenant pairing. It exists so a domain
// package outside env-domain-guard's permitted import list (main.go,
// kafka/, rest/, socket/) can originate the environment on a per-event
// context -- e.g. character/combo's DecayTick, which builds its own
// per-character tenant context in a background sweep and has nowhere
// else to source env.Self() from -- without importing atlas-env
// directly. Callers thread this in as a plain function value rather
// than importing atlas-env themselves.
func WithSelfEnvironment(ctx context.Context) context.Context {
	return env.WithContext(ctx, env.Self())
}

// dualWaitGroup fans Add/Done out to two waitgroups so a caller can track
// the same event in a handle-scoped waitgroup and the process-wide one at
// once. Held as the interface, not *sync.WaitGroup, so it is exercisable
// against a counting fake.
type dualWaitGroup struct{ a, b socket.WaitGrouper }

func (d dualWaitGroup) Add(n int) { d.a.Add(n); d.b.Add(n) }
func (d dualWaitGroup) Done()     { d.a.Done(); d.b.Done() }

// CreateSocketService binds the listener SYNCHRONOUSLY and returns it, so
// listener.Registry.Add can surface a bind failure through its existing
// rollback path and buildListener can install Handle.CloseListener. The
// accept loop and per-connection handling stay asynchronous -- only the
// bind result is observable before this returns (task-244 design.md §4.2).
//
// ipAddress is advertisement-only: it is the address this channel hands to
// clients (see the Register call below), never the bind address. The pod
// this listener runs in is not guaranteed to have the advertised host
// address assigned to any interface -- e.g. a LAN IP the deployment
// advertises to clients outside the pod -- so binding it directly fails
// with EADDRNOTAVAIL. The listener always binds the wildcard address and
// lets the kernel accept connections on whichever interface they arrive at.
//
// wg brackets the accept-loop goroutine only (process-wide bookkeeping).
// sessionWg is fanned out per accepted connection, so a handle-scoped
// waitgroup sees real session lifetime without also counting the
// accept loop, which lives as long as the handle does.
func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg socket.WaitGrouper, sessionWg socket.WaitGrouper) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) (net.Listener, error) {
		// Bind before any other side effect: a failed bind must leave
		// nothing for Registry.Add's rollback to unwind. Bind the wildcard
		// address, not ipAddress -- ipAddress is advertisement-only.
		lis, err := socket.Bind(l, "0.0.0.0", port)
		if err != nil {
			return nil, fmt.Errorf("bind port %d: %w", port, err)
		}

		l.Infof("Creating channel socket service for [%s] on port [%d].", sc.String(), port)

		chakra.GetRegistry().StartSweeper(l, ctx)

		hasMapleEncryption := true
		t := sc.Tenant()
		if t.Region() == "JMS" {
			hasMapleEncryption = false
			l.Debugf("Service does not expect Maple encryption.")
		}

		locale := byte(8)
		if t.Region() == "JMS" {
			locale = 3
		}
		l.Debugf("Service locale [%d].", locale)

		sp := session.NewProcessor(l, ctx)
		fanOut := dualWaitGroup{a: wg, b: sessionWg}

		routine.Go(l, ctx, func(_ context.Context) {
			err := socket.Serve(l, ctx, wg, fanOut, lis,
				socket.SetHandlers(hp),
				socket.SetCreator(sp.Create(sc.Channel(), locale)),
				socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
				socket.SetDestroyer(func(sessionId uuid.UUID) {
					sp.IfPresentById(sessionId, func(s session.Model) error {
						shopscanner.GetRegistry().ClearCharacter(t, s.CharacterId())
						// Without this the throttle map leaks one entry per
						// character ever seen by this pod (task-190).
						statreset.GetRegistry().ClearCharacter(t, s.CharacterId())
						// Channel change and disconnect both destroy the
						// session; without this the window map leaks one
						// entry per character ever seen by this pod
						// (PRD FR-5.5, FR-2.2).
						chakra.GetRegistry().Clear(t, s.CharacterId())
						// Channel change and disconnect both destroy the
						// session; without this the pending-unlock map
						// leaks one entry per character ever seen by this
						// pod (task-221).
						remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
						return nil
					})
					sp.DestroyByIdWithSpan(sessionId)
				}),
				socket.SetReadWriter(rw),
				socket.SetIdleNotifier(session.SendPing(l, ctx, wp), idleThreshold),
			)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				l.WithError(err).Errorf("Socket service encountered error")
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port); err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}
			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		})

		return lis, nil
	}
}
