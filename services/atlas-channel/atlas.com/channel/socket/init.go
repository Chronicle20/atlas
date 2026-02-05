package socket

import (
	"atlas-channel/channel"
	"atlas-channel/server"
	"atlas-channel/session"
	"context"
	"errors"
	"net"
	"sync"

	"github.com/Chronicle20/atlas-socket"
	"github.com/sirupsen/logrus"
)

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, port int) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, port int) {
		go func() {
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
				sp := session.NewProcessor(l, ctx)
				err := socket.Run(l, ctx, wg,
					socket.SetHandlers(hp),
					socket.SetPort(port),
					socket.SetCreator(sp.Create(sc.Channel(), locale)),
					socket.SetMessageDecryptor(sp.Decrypt(true, hasMapleEncryption)),
					socket.SetDestroyer(sp.DestroyByIdWithSpan),
					socket.SetReadWriter(rw),
				)

				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						return
					}
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			err := channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port)
			if err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		}()
	}
}
