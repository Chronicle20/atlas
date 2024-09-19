package socket

import (
	"atlas-channel/channel"
	"atlas-channel/server"
	"atlas-channel/session"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-socket"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"net"
	"strconv"
	"sync"
)

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, port string) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, portStr string) {
		go func() {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service [port] is configured incorrectly")
				return
			}

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

			go func() {
				err = socket.Run(l, ctx, wg,
					socket.SetHandlers(hp),
					socket.SetPort(port),
					socket.SetCreator(session.Create(l, session.GetRegistry(), t)(sc.WorldId(), sc.ChannelId(), locale)),
					socket.SetMessageDecryptor(session.Decrypt(l, session.GetRegistry(), t)(true, hasMapleEncryption)),
					socket.SetDestroyer(session.DestroyByIdWithSpan(l)(ctx)(session.GetRegistry())),
					socket.SetReadWriter(rw),
				)

				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						return
					}
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			sctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, "startup")

			err = channel.Register(l)(sctx)(sc.WorldId(), sc.ChannelId(), ipAddress, portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}

			span.End()

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)

			sctx, span = otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, "teardown")
			defer span.End()

			err = channel.Unregister(l)(sctx)(sc.WorldId(), sc.ChannelId())
			if err != nil {
				l.WithError(err).Errorf("Socket service unregistration error.")
			}
		}()
	}
}
