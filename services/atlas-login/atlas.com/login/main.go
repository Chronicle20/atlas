package main

import (
	"atlas-login/account"
	"atlas-login/configuration"
	session2 "atlas-login/kafka/consumer/session"
	"atlas-login/logger"
	"atlas-login/service"
	"atlas-login/session"
	"atlas-login/socket"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tasks"
	"atlas-login/tracing"
	"fmt"
	"github.com/Chronicle20/atlas-kafka/consumer"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"strconv"
	"time"
)

const serviceName = "atlas-login"
const consumerGroupId = "Login Service - %s"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(account.StatusConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(session2.AccountSessionStatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))

	sctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(tdm.Context(), "startup")

	for _, s := range config.Data.Attributes.Servers {
		var t tenant.Model
		majorVersion, err := strconv.Atoi(s.Version.Major)
		if err != nil {
			l.WithError(err).Errorf("Socket service [majorVersion] is configured incorrectly")
			continue
		}

		minorVersion, err := strconv.Atoi(s.Version.Minor)
		if err != nil {
			l.WithError(err).Errorf("Socket service [minorVersion] is configured incorrectly")
			continue
		}

		t, err = tenant.Register(uuid.MustParse(s.Tenant), s.Region, uint16(majorVersion), uint16(minorVersion))
		if err != nil {
			continue
		}
		tctx := tenant.WithContext(sctx, t)

		err = account.InitializeRegistry(l, tctx, t)
		if err != nil {
			l.WithError(err).Errorf("Unable to initialize account registry for tenant [%s].", t.String())
		}

		fl := l.
			WithField("tenant", t.Id().String()).
			WithField("region", t.Region()).
			WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()))

		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}

		wp := produceWriterProducer(fl)(s.Writers, writerList, rw)
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(s.Handlers, validatorMap, handlerMap)

		_, _ = cm.RegisterHandler(account.StatusRegister(t)(l))
		_, _ = cm.RegisterHandler(session2.CreatedAccountSessionStatusEventRegister(t, wp)(l))
		_, _ = cm.RegisterHandler(session2.LicenseAgreementAccountSessionStatusEventRegister(t, wp)(l))
		_, _ = cm.RegisterHandler(session2.StateChangedAccountSessionStatusEventRegister(t, wp)(l))
		_, _ = cm.RegisterHandler(session2.ErrorAccountSessionStatusEventRegister(t, wp)(l))

		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, t, s.Port)
	}
	span.End()

	tt, err := config.FindTask(session.TimeoutTask)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	go tasks.Register(l, tdm.Context())(session.NewTimeout(l, time.Millisecond*time.Duration(tt.Attributes.Interval)))

	tdm.TeardownFunc(session.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

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
	handlerMap[handler.CharacterViewAllPongHandle] = handler.CharacterViewAllPongHandleFunc
	handlerMap[handler.ClientStartHandle] = handler.ClientStartHandleFunc
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
