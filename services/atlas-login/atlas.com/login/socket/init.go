package socket

import (
	"atlas-login/session"
	"atlas-login/tenant"
	"context"
	"github.com/Chronicle20/atlas-socket"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, t tenant.Model, port string) {
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

			go func() {
				wg.Add(1)
				defer wg.Done()

				err = socket.Run(l, hp,
					socket.SetPort(port),
					socket.SetSessionCreator(session.Create(l, session.GetRegistry())(t, locale)),
					socket.SetSessionMessageDecryptor(session.Decrypt(l, session.GetRegistry(), t)(true, hasMapleEncryption)),
					socket.SetSessionDestroyer(session.DestroyByIdWithSpan(l, session.GetRegistry(), t.Id)),
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
