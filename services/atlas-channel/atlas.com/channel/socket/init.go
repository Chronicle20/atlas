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
	"net"
	"sync"
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

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) {
		chakra.GetRegistry().StartSweeper(l, ctx)

		routine.Go(l, ctx, func(_ context.Context) {
			l.Infof("Creating channel socket service for [%s] on port [%d].", sc.String(), port)

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

			routine.Go(l, ctx, func(_ context.Context) {
				sp := session.NewProcessor(l, ctx)
				err := socket.Run(l, ctx, wg,
					socket.SetHandlers(hp),
					socket.SetPort(port),
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

			err := channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port)
			if err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		})
	}
}
