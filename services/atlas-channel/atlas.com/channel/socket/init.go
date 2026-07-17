package socket

import (
	"atlas-channel/channel"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const idleThreshold = 30 * time.Second

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int) {
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
