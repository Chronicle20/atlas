package main

import (
	"atlas-login/account"
	"atlas-login/configuration"
	handler2 "atlas-login/configuration/tenant/socket/handler"
	writer2 "atlas-login/configuration/tenant/socket/writer"
	account2 "atlas-login/kafka/consumer/account"
	session2 "atlas-login/kafka/consumer/account/session"
	"atlas-login/kafka/consumer/seed"
	"atlas-login/logger"
	"atlas-login/session"
	"atlas-login/socket"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tasks"
	"atlas-login/tracing"
	"fmt"
	"os"
	"strconv"
	"time"

	account3 "github.com/Chronicle20/atlas-packet/account"
	"github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-packet/login"
	socket3 "github.com/Chronicle20/atlas-packet/socket"
	"github.com/Chronicle20/atlas-service"

	"github.com/Chronicle20/atlas-kafka/consumer"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	sw "github.com/Chronicle20/atlas-socket/writer"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const serviceName = "atlas-login"
const consumerGroupIdTemplate = "ChannelConnect Service - %s"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))
	config, err := configuration.GetServiceConfig()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	var consumerGroupId = fmt.Sprintf(consumerGroupIdTemplate, config.Id.String())
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	account2.InitConsumers(l)(cmf)(consumerGroupId)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	seed.InitConsumers(l)(cmf)(consumerGroupId)

	sctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(tdm.Context(), "startup")

	for _, ten := range config.Tenants {
		tenantId := uuid.MustParse(ten.Id)
		tenantConfig, err := configuration.GetTenantConfig(tenantId)
		if err != nil {
			continue
		}

		var t tenant.Model
		t, err = tenant.Register(tenantId, tenantConfig.Region, tenantConfig.MajorVersion, tenantConfig.MinorVersion)
		if err != nil {
			continue
		}
		tctx := tenant.WithContext(sctx, t)

		err = account.NewProcessor(l, tctx).InitializeRegistry()
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

		wp := produceWriterProducer(fl)(tenantConfig.Socket.Writers, writerList, rw)
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantConfig.Socket.Handlers, validatorMap, handlerMap)

		if err := account2.InitHandlers(fl)(t)(wp)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := session2.InitHandlers(fl)(t)(wp)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := seed.InitHandlers(fl)(t)(wp)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}

		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, ten.Port)
	}
	span.End()

	tt, err := config.FindTask(session.TimeoutTask)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	go tasks.Register(l, tdm.Context())(session.NewTimeout(l, time.Millisecond*time.Duration(tt.Interval)))

	tdm.TeardownFunc(session.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}

func produceWriterProducer(l logrus.FieldLogger) func(writers []writer2.RestModel, writerList []string, w socket2.OpWriter) writer.Producer {
	return func(writers []writer2.RestModel, writerList []string, w socket2.OpWriter) writer.Producer {
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
		writer.Ping,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[handler.DebugHandle] = handler.DebugHandleFunc
	handlerMap[handler.CreateSecurityHandle] = handler.CreateSecurityHandleFunc
	handlerMap[login.LoginHandle] = handler.LoginHandleFunc
	handlerMap[login.ServerListRequestHandle] = handler.ServerListRequestHandleFunc
	handlerMap[login.ServerStatusHandle] = handler.ServerStatusHandleFunc
	handlerMap[login.WorldCharacterListHandle] = handler.CharacterListWorldHandleFunc
	handlerMap[character.CharacterCheckNameHandle] = handler.CharacterCheckNameHandleFunc
	handlerMap[character.CreateCharacterHandle] = handler.CreateCharacterHandleFunc
	handlerMap[character.DeleteCharacterHandle] = handler.DeleteCharacterHandleFunc
	handlerMap[login.AfterLoginHandle] = handler.AfterLoginHandleFunc
	handlerMap[account3.RegisterPinHandle] = handler.RegisterPinHandleFunc
	handlerMap[login.RegisterPicHandle] = handler.RegisterPicHandleFunc
	handlerMap[account3.AcceptTosHandle] = handler.AcceptTosHandleFunc
	handlerMap[login.CharacterSelectedHandle] = handler.CharacterSelectedHandleFunc
	handlerMap[login.CharacterSelectedPicHandle] = handler.CharacterSelectedPicHandleFunc
	handlerMap[login.WorldSelectHandle] = handler.WorldSelectHandleFunc
	handlerMap[account3.SetGenderHandle] = handler.SetGenderHandleFunc
	handlerMap[login.CharacterViewAllHandle] = handler.CharacterViewAllHandleFunc
	handlerMap[login.CharacterViewAllSelectedHandle] = handler.CharacterViewAllSelectedHandleFunc
	handlerMap[login.CharacterViewAllSelectedPicRegisterHandle] = handler.CharacterViewAllSelectedPicRegisterHandleFunc
	handlerMap[login.CharacterViewAllSelectedPicHandle] = handler.CharacterViewAllSelectedPicHandleFunc
	handlerMap[login.CharacterViewAllPongHandle] = handler.CharacterViewAllPongHandleFunc
	handlerMap[handler.ClientStartHandle] = handler.ClientStartHandleFunc
	handlerMap[socket3.PongHandle] = handler.PongHandleFunc
	handlerMap[socket3.StartErrorHandle] = handler.StartErrorHandleFunc
	return handlerMap
}

func produceValidators() map[string]handler.MessageValidator {
	validatorMap := make(map[string]handler.MessageValidator)
	validatorMap[handler.NoOpValidator] = handler.NoOpValidatorFunc
	validatorMap[handler.LoggedInValidator] = handler.LoggedInValidatorFunc
	return validatorMap
}

func getWriterProducer(l logrus.FieldLogger) func(writerConfig []writer2.RestModel, wl []string, w socket2.OpWriter) writer.Producer {
	return func(writerConfig []writer2.RestModel, wl []string, w socket2.OpWriter) writer.Producer {
		rwm := make(map[string]writer.BodyFunc)
		for _, wc := range writerConfig {
			op, err := strconv.ParseUint(wc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
				continue
			}

			for _, wn := range wl {
				if wn == wc.Writer {
					rwm[wc.Writer] = sw.MessageGetter(w.Write(uint16(op)), wc.Options)
				}
			}
		}
		return sw.ProducerGetter(rwm)
	}
}

func handlerProducer(l logrus.FieldLogger) func(adapter handler.Adapter) func(handlerConfig []handler2.RestModel, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
	return func(adapter handler.Adapter) func(handlerConfig []handler2.RestModel, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
		return func(handlerConfig []handler2.RestModel, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
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
