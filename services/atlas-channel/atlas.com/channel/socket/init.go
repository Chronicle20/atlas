package socket

import (
	"atlas-channel/channel"
	"atlas-channel/configuration"
	"atlas-channel/session"
	"atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup) func(config configuration.Server, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer, wid byte, cid byte, ipAddress string, port string) {
	return func(config configuration.Server, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer, wid byte, cid byte, ipAddress string, portStr string) {
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			t, err := tenant.New(l)(config)
			if err != nil {
				return
			}

			port, err := strconv.Atoi(portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service [port] is configured incorrectly")
				return
			}

			l.Infof("Creating channel socket service for [%s] [%d.%d] world [%d] channel [%d] on port [%d].", t.Region, t.MajorVersion, t.MinorVersion, wid, cid, port)

			hasMapleEncryption := true
			if config.Region == "JMS" {
				hasMapleEncryption = false
				l.Debugf("Service does not expect Maple encryption.")
			}

			locale := byte(8)
			if config.Region == "JMS" {
				locale = 3
			}

			l.Debugf("Service locale [%d].", locale)

			fl := l.WithField("tenant", t.Id.String()).WithField("region", t.Region).WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion, t.MinorVersion))

			go func() {
				wg.Add(1)
				defer wg.Done()

				err = socket.Run(fl, handlerProducer(fl)(config.Handlers, vm, hm, wp),
					socket.SetPort(port),
					socket.SetSessionCreator(session.Create(fl, session.GetRegistry())(t, locale)),
					socket.SetSessionMessageDecryptor(session.Decrypt(fl, session.GetRegistry())(hasMapleEncryption)),
					socket.SetSessionDestroyer(session.DestroyByIdWithSpan(fl, session.GetRegistry())),
				)
				if err != nil {
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			span := opentracing.StartSpan("startup")
			defer span.Finish()
			err = channel.Register(l, span, t)(wid, cid, ipAddress, portStr)
			if err != nil {
				l.WithError(err).Errorf("Socket service registration error.")
			}

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)

			span = opentracing.StartSpan("teardown")
			defer span.Finish()
			err = channel.Unregister(l, span, t)(wid, cid)
			if err != nil {
				l.WithError(err).Errorf("Socket service unregistration error.")
			}
		}()
	}
}

func handlerProducer(l logrus.FieldLogger) func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer) socket.MessageHandlerProducer {
	return func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer) socket.MessageHandlerProducer {
		handlers := make(map[uint16]request.Handler)

		for _, hc := range handlerConfig {
			var v handler.MessageValidator
			var ok bool
			if v, ok = vm[hc.Validator]; !ok {
				l.Warnf("Unable to locate validator [%s] for handler[%s].", hc.Validator, hc.Handler)
				continue
			}

			var h handler.MessageHandler
			if h, ok = hm[hc.Handler]; !ok {
				l.Warnf("Unable to locate handler [%s].", hc.Handler)
				continue
			}

			op, err := strconv.ParseUint(hc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Warnf("Unable to configure handler [%s] for opcode [%s].", hc.Handler, hc.OpCode)
				continue
			}

			l.Debugf("Configuring opcode [%s] with validator [%s] and handler [%s].", hc.OpCode, hc.Validator, hc.Handler)
			handlers[uint16(op)] = handler.AdaptHandler(l, hc.Handler, v, h, wp)
		}

		return func() map[uint16]request.Handler {
			return handlers
		}
	}
}
