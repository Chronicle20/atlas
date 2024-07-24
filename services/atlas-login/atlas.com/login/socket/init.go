package socket

import (
	"atlas-login/session"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-socket"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, t tenant.Model, port string) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, t tenant.Model, portStr string) {
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service [port] is configured incorrectly")
				return
			}

			l.Infof("Creating login socket service for [%s] [%d.%d] on port [%d].", t.Region, t.MajorVersion, t.MinorVersion, port)

			hasMapleEncryption := true
			if t.Region == "JMS" {
				hasMapleEncryption = false
				l.Debugf("Service does not expect Maple encryption.")
			}

			locale := byte(8)
			if t.Region == "JMS" {
				locale = 3
			}

			l.Debugf("Service locale [%d].", locale)

			fl := l.WithField("tenant", t.Id.String()).WithField("region", t.Region).WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion, t.MinorVersion))

			go func() {
				wg.Add(1)
				defer wg.Done()

				err = socket.Run(fl, hp,
					socket.SetPort(port),
					socket.SetSessionCreator(session.Create(fl, session.GetRegistry())(t, locale)),
					socket.SetSessionMessageDecryptor(session.Decrypt(fl, session.GetRegistry(), t)(true, hasMapleEncryption)),
					socket.SetSessionDestroyer(session.DestroyByIdWithSpan(fl, session.GetRegistry(), t.Id)),
					socket.SetReadWriter(rw),
				)

				if err != nil {
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		}()
	}
}
