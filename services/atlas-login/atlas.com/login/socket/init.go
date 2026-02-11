package socket

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const idleThreshold = 30 * time.Second

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, port int) {
	t := tenant.MustFromContext(ctx)
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, port int) {
		go func() {
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

			go func() {
				wg.Add(1)
				defer wg.Done()

				sp := session.NewProcessor(l, ctx)

				err := socket.Run(l, ctx, wg,
					socket.SetHandlers(hp),
					socket.SetPort(port),
					socket.SetCreator(sp.Create(locale)),
					socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
					socket.SetDestroyer(sp.DestroyByIdWithSpan),
					socket.SetReadWriter(rw),
					socket.SetIdleNotifier(session.SendPing(l, t, wp), idleThreshold),
				)

				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						return
					}
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		}()
	}
}
