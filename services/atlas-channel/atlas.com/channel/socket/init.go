package socket

import (
	"atlas-channel/channel"
	"atlas-channel/server"
	"atlas-channel/session"
	"context"
	"github.com/Chronicle20/atlas-socket"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, port string) {
	return func(hp socket.HandlerProducer, rw socket.OpReadWriter, sc server.Model, ipAddress string, portStr string) {
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			port, err := strconv.Atoi(portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service [port] is configured incorrectly")
				return
			}

			l.Infof("Creating channel socket service for [%s] on port [%d].", sc.String(), port)

			hasMapleEncryption := true
			if sc.Tenant().Region == "JMS" {
				hasMapleEncryption = false
				l.Debugf("Service does not expect Maple encryption.")
			}

			locale := byte(8)
			if sc.Tenant().Region == "JMS" {
				locale = 3
			}

			l.Debugf("Service locale [%d].", locale)

			go func() {
				wg.Add(1)
				defer wg.Done()

				err = socket.Run(l, hp,
					socket.SetPort(port),
					socket.SetSessionCreator(session.Create(l, session.GetRegistry(), sc.Tenant())(sc.WorldId(), sc.ChannelId(), locale)),
					socket.SetSessionMessageDecryptor(session.Decrypt(l, session.GetRegistry(), sc.Tenant())(true, hasMapleEncryption)),
					socket.SetSessionDestroyer(session.DestroyByIdWithSpan(l, session.GetRegistry(), sc.Tenant().Id)),
					socket.SetReadWriter(rw),
				)

				if err != nil {
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			span := opentracing.StartSpan("startup")
			defer span.Finish()
			err = channel.Register(l, span, sc.Tenant())(sc.WorldId(), sc.ChannelId(), ipAddress, portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)

			span = opentracing.StartSpan("teardown")
			defer span.Finish()
			err = channel.Unregister(l, span, sc.Tenant())(sc.WorldId(), sc.ChannelId())
			if err != nil {
				l.WithError(err).Errorf("Socket service unregistration error.")
			}
		}()
	}
}
