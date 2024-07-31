package main

import (
	"atlas-login/configuration"
	"atlas-login/logger"
	"atlas-login/session"
	"atlas-login/socket"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tasks"
	"atlas-login/tenant"
	"atlas-login/tracing"
	"context"
	"fmt"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const serviceName = "atlas-login"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}
	defer func(tc io.Closer) {
		err := tc.Close()
		if err != nil {
			l.WithError(err).Errorf("Unable to close tracer.")
		}
	}(tc)

	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	for _, s := range config.Data.Attributes.Servers {
		var t tenant.Model
		t, err = tenant.NewFromConfiguration(l)(s)
		if err != nil {
			continue
		}

		fl := l.
			WithField("tenant", t.Id.String()).
			WithField("region", t.Region).
			WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion, t.MinorVersion))

		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region == "GMS" && t.MajorVersion <= 28 {
			rw = socket2.ByteReadWriter{}
		}

		wp := produceWriterProducer(fl)(s.Writers, writerList, rw)
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t.Id, wp))(s.Handlers, validatorMap, handlerMap)

		socket.CreateSocketService(fl, ctx, wg)(hp, rw, t, s.Port)
	}

	tt, err := config.FindTask(session.TimeoutTask)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	go tasks.Register(l, ctx)(session.NewTimeout(l, time.Millisecond*time.Duration(tt.Attributes.Interval)))

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-c
	l.Infof("Initiating shutdown with signal %s.", sig)
	cancel()
	wg.Wait()

	span := opentracing.StartSpan("teardown")
	defer span.Finish()
	tenant.ForAll(session.DestroyAll(l, span, session.GetRegistry()))

	l.Infoln("Service shutdown.")
}

func produceWriterProducer(l logrus.FieldLogger) func(writers []configuration.Writer, writerList []string, w socket2.OpWriter) writer.Producer {
	return func(writers []configuration.Writer, writerList []string, w socket2.OpWriter) writer.Producer {
		return getWriterProducer(l)(writers, writerList, w)
	}
}

func produceWriters() []string {
	return []string{
		writer.LoginAuth,
		writer.AuthSuccess,
		writer.AuthTemporaryBan,
		writer.AuthPermanentBan,
		writer.AuthLoginFailed,
		writer.ServerListRecommendations,
		writer.ServerListEntry,
		writer.ServerListEnd,
		writer.SelectWorld,
		writer.ServerStatus,
		writer.CharacterList,
		writer.CharacterNameResponse,
		writer.AddCharacterEntry,
		writer.DeleteCharacterResponse,
		writer.PinOperation,
		writer.PinUpdate,
		writer.PicResult,
		writer.ServerIP,
		writer.ServerLoad,
		writer.SetAccountResult,
		writer.CharacterViewAll,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[handler.DebugHandle] = handler.DebugHandleFunc
	handlerMap[handler.CreateSecurityHandle] = handler.CreateSecurityHandleFunc
	handlerMap[handler.LoginHandle] = handler.LoginHandleFunc
	handlerMap[handler.ServerListRequestHandle] = handler.ServerListRequestHandleFunc
	handlerMap[handler.ServerStatusHandle] = handler.ServerStatusHandleFunc
	handlerMap[handler.CharacterListWorldHandle] = handler.CharacterListWorldHandleFunc
	handlerMap[handler.CharacterCheckNameHandle] = handler.CharacterCheckNameHandleFunc
	handlerMap[handler.CreateCharacterHandle] = handler.CreateCharacterHandleFunc
	handlerMap[handler.DeleteCharacterHandle] = handler.DeleteCharacterHandleFunc
	handlerMap[handler.AfterLoginHandle] = handler.AfterLoginHandleFunc
	handlerMap[handler.RegisterPinHandle] = handler.RegisterPinHandleFunc
	handlerMap[handler.RegisterPicHandle] = handler.RegisterPicHandleFunc
	handlerMap[handler.AcceptTosHandle] = handler.AcceptTosHandleFunc
	handlerMap[handler.CharacterSelectedHandle] = handler.CharacterSelectedHandleFunc
	handlerMap[handler.CharacterSelectedPicHandle] = handler.CharacterSelectedPicHandleFunc
	handlerMap[handler.WorldSelectHandle] = handler.WorldSelectHandleFunc
	handlerMap[handler.SetGenderHandle] = handler.SetGenderHandleFunc
	handlerMap[handler.CharacterViewAllHandle] = handler.CharacterViewAllHandleFunc
	handlerMap[handler.CharacterViewAllSelectedHandle] = handler.CharacterViewAllSelectedHandleFunc
	handlerMap[handler.CharacterViewAllSelectedPicRegisterHandle] = handler.CharacterViewAllSelectedPicRegisterHandleFunc
	handlerMap[handler.CharacterViewAllSelectedPicHandle] = handler.CharacterViewAllSelectedPicHandleFunc
	return handlerMap
}

func produceValidators() map[string]handler.MessageValidator {
	validatorMap := make(map[string]handler.MessageValidator)
	validatorMap[handler.NoOpValidator] = handler.NoOpValidatorFunc
	validatorMap[handler.LoggedInValidator] = handler.LoggedInValidatorFunc
	return validatorMap
}

func getWriterProducer(l logrus.FieldLogger) func(writerConfig []configuration.Writer, wl []string, w socket2.OpWriter) writer.Producer {
	return func(writerConfig []configuration.Writer, wl []string, w socket2.OpWriter) writer.Producer {
		rwm := make(map[string]writer.BodyFunc)
		for _, wc := range writerConfig {
			op, err := strconv.ParseUint(wc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
				continue
			}

			for _, wn := range wl {
				if wn == wc.Writer {
					rwm[wc.Writer] = writer.MessageGetter(w.Write(uint16(op)), wc.Options)
				}
			}
		}
		return writer.ProducerGetter(rwm)
	}
}

func handlerProducer(l logrus.FieldLogger) func(adapter handler.Adapter) func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
	return func(adapter handler.Adapter) func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
		return func(handlerConfig []configuration.Handler, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
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
				handlers[uint16(op)] = adapter(hc.Handler, v, h, hc.Options)
			}

			return func() map[uint16]request.Handler {
				return handlers
			}
		}
	}
}
