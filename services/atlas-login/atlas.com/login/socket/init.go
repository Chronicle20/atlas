package socket

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const idleThreshold = 30 * time.Second

// NewListenerContext builds the context for a tenant's socket-listener
// registration: the tenant this listener serves, plus this pod's own
// environment (env.Self()) — mirrors socket/handler's per-request session
// context origination (FR-2.2). CreateSocketService's Create/
// DestroyByIdWithSpan/SendPing callbacks below are wired directly from
// this context, bypassing socket/handler.AdaptHandler entirely, so the
// environment has to be originated here too — not only on the per-packet
// handler path — or every session-lifecycle Kafka event (e.g.
// EnvEventTopicSessionStatus) leaves a PR pod with an empty ENVIRONMENT
// header.
func NewListenerContext(ctx context.Context, t tenant.Model) context.Context {
	tctx := tenant.WithContext(ctx, t)
	return env.WithContext(tctx, env.Self())
}

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, port int, names map[uint16]string) {
	t := tenant.MustFromContext(ctx)
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, port int, names map[uint16]string) {
		routine.Go(l, ctx, func(_ context.Context) {
			l.Infof("Creating login socket service for [%s] [%d.%d] on port [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), port)

			hasMapleEncryption := true
			if t.Region() == "JMS" {
				hasMapleEncryption = false
				l.Debugf("Service does not expect Maple encryption.")
			}

			locale := byte(8)
			if t.Region() == "JMS" {
				locale = 3
			}

			l.Debugf("Service locale [%d].", locale)

			wg.Add(1)
			routine.Go(l, ctx, func(_ context.Context) {
				defer wg.Done()

				sp := session.NewProcessor(l, ctx)

				err := socket.Run(l, ctx, wg,
					socket.SetHandlers(hp),
					socket.SetPort(port),
					socket.SetCreator(sp.Create(locale)),
					socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
					socket.SetDestroyer(sp.DestroyByIdWithSpan),
					socket.SetReadWriter(rw),
					socket.SetPacketTracer(NewPacketTracer(l, t, names)),
					socket.SetIdleNotifier(session.SendPing(l, ctx, wp), idleThreshold),
				)
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						return
					}
					l.WithError(err).Errorf("Socket service encountered error")
				}
			})

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		})
	}
}
