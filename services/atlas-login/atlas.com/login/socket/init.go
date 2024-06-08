package socket

import (
	"atlas-login/configuration"
	"atlas-login/session"
	"atlas-login/socket/handler"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup) func(config configuration.Server) {
	return func(config configuration.Server) {
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			t, err := tenant.New(l)(config)
			if err != nil {
				return
			}

			port, err := strconv.Atoi(config.Port)
			if err != nil {
				l.WithError(err).Errorf("Socket service [port] is configured incorrectly")
				return
			}

			hasMapleEncryption := true
			if config.Region == "JMS" {
				hasMapleEncryption = false
			}

			locale := byte(8)
			if config.Region == "JMS" {
				locale = 3
			}

			l.Infof("Creating login socket service for [%s] [%d.%d] on port [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), port)
			l.Debugf("Service locale [%d].", locale)
			l.Debugf("Service does not expect Maple encryption.")

			fl := l.WithField("tenant", t.Id()).WithField("region", t.Region()).WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()))

			go func() {
				wg.Add(1)
				defer wg.Done()

				err = socket.Run(fl, handlerProducer(fl),
					socket.SetPort(port),
					socket.SetSessionCreator(session.Create(fl, session.GetRegistry())(t, locale)),
					socket.SetSessionMessageDecryptor(session.Decrypt(fl, session.GetRegistry())(hasMapleEncryption)),
					socket.SetSessionDestroyer(session.DestroyByIdWithSpan(fl, session.GetRegistry())),
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

func handlerProducer(l logrus.FieldLogger) socket.MessageHandlerProducer {
	handlers := make(map[uint16]request.Handler)
	_ = func(op uint16, name string, v handler.MessageValidator, h handler.MessageHandler) {
		handlers[op] = handler.AdaptHandler(l, name, v, h)
	}
	return func() map[uint16]request.Handler {
		return handlers
	}
}
