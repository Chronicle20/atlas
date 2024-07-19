package socket

import (
	"atlas-login/configuration"
	"atlas-login/session"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

func CreateSocketService(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup) func(config configuration.Server, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, writerList []string) {
	return func(config configuration.Server, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, writerList []string) {
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

			l.Infof("Creating login socket service for [%s] [%d.%d] on port [%d].", t.Region, t.MajorVersion, t.MinorVersion, port)

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

				if t.Region == "GMS" && t.MajorVersion <= 28 {
					owp := func(op uint8) writer.OpWriter {
						return func(w *response.Writer) {
							w.WriteByte(op)
						}
					}
					wp := getWriterProducer[uint8](l)(config.Writers, writerList, owp)

					err = socket.Run(fl, handlerProducer[uint8](fl)(config.Handlers, vm, hm, wp),
						socket.SetPort[uint8](port),
						socket.SetSessionCreator[uint8](session.Create(fl, session.GetRegistry())(t, locale)),
						socket.SetSessionMessageDecryptor[uint8](session.Decrypt(fl, session.GetRegistry())(true, hasMapleEncryption)),
						socket.SetSessionDestroyer[uint8](session.DestroyByIdWithSpan(fl, session.GetRegistry())),
						socket.SetOpReader[uint8](socket.ByteOpReader),
					)
				} else {
					owp := func(op uint16) writer.OpWriter {
						return func(w *response.Writer) {
							w.WriteShort(op)
						}
					}
					wp := getWriterProducer[uint16](l)(config.Writers, writerList, owp)

					err = socket.Run(fl, handlerProducer[uint16](fl)(config.Handlers, vm, hm, wp),
						socket.SetPort[uint16](port),
						socket.SetSessionCreator[uint16](session.Create(fl, session.GetRegistry())(t, locale)),
						socket.SetSessionMessageDecryptor[uint16](session.Decrypt(fl, session.GetRegistry())(true, hasMapleEncryption)),
						socket.SetSessionDestroyer[uint16](session.DestroyByIdWithSpan(fl, session.GetRegistry())),
						socket.SetOpReader[uint16](socket.ShortOpReader),
					)
				}
				if err != nil {
					l.WithError(err).Errorf("Socket service encountered error")
				}
			}()

			<-ctx.Done()
			l.Infof("Shutting down server on port %d", port)
		}()
	}
}

func handlerProducer[E uint8 | uint16](l logrus.FieldLogger) func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer) socket.MessageHandlerProducer[E] {
	return func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler, wp writer.Producer) socket.MessageHandlerProducer[E] {
		handlers := make(map[E]request.Handler)

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
			handlers[E(op)] = handler.AdaptHandler(l, hc.Handler, v, h, wp)
		}

		return func() map[E]request.Handler {
			return handlers
		}
	}
}

func getWriterProducer[E uint8 | uint16](l logrus.FieldLogger) func(writerConfig []configuration.Writer, wl []string, opwp writer.OpWriterProducer[E]) writer.Producer {
	return func(writerConfig []configuration.Writer, wl []string, opwp writer.OpWriterProducer[E]) writer.Producer {
		rwm := make(map[string]writer.BodyFunc)
		for _, wc := range writerConfig {
			op, err := strconv.ParseUint(wc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
				continue
			}

			for _, wn := range wl {
				if wn == wc.Writer {
					rwm[wc.Writer] = writer.MessageGetter(opwp(E(op)), wc.Options)
				}
			}
		}
		return writer.ProducerGetter(rwm)
	}
}
