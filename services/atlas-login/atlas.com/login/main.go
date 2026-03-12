package main

import (
	"atlas-login/account"
	"atlas-login/configuration"
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
	"time"

	"github.com/Chronicle20/atlas-opcodes"
	account3 "github.com/Chronicle20/atlas-packet/account"
	"github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-packet/login"
	socket3 "github.com/Chronicle20/atlas-packet/socket"
	"github.com/Chronicle20/atlas-service"

	"github.com/Chronicle20/atlas-kafka/consumer"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
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

func produceWriterProducer(l logrus.FieldLogger) func(writers []opcodes.WriterConfig, writerList []string, w socket2.OpWriter) writer.Producer {
	return func(writers []opcodes.WriterConfig, writerList []string, w socket2.OpWriter) writer.Producer {
		return opcodes.BuildWriterProducer(l, writers, writerList, w)
	}
}

func produceWriters() []string {
	return []string{
		login.LoginAuthWriter,
		login.AuthSuccessWriter,
		login.AuthTemporaryBanWriter,
		login.AuthPermanentBanWriter,
		login.AuthLoginFailedWriter,
		login.ServerListRecommendationsWriter,
		login.ServerListEntryWriter,
		login.ServerListEndWriter,
		login.SelectWorldWriter,
		login.ServerStatusWriter,
		character.CharacterListWriter,
		character.CharacterNameResponseWriter,
		character.AddCharacterEntryWriter,
		character.DeleteCharacterResponseWriter,
		login.PinOperationWriter,
		login.PinUpdateWriter,
		login.PicResultWriter,
		login.ServerIPWriter,
		login.ServerLoadWriter,
		login.SetAccountResultWriter,
		character.CharacterViewAllWriter,
		socket3.PingWriter,
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

func handlerProducer(l logrus.FieldLogger) func(adapter handler.Adapter) func(handlerConfig []opcodes.HandlerConfig, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
	return func(adapter handler.Adapter) func(handlerConfig []opcodes.HandlerConfig, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
		return func(handlerConfig []opcodes.HandlerConfig, vm map[string]handler.MessageValidator, hm map[string]handler.MessageHandler) socket2.HandlerProducer {
			adapt := func(name string, v interface{}, h interface{}, options map[string]interface{}) request.Handler {
				return adapter(name, v.(handler.MessageValidator), h.(handler.MessageHandler), options)
			}
			vmGeneric := make(map[string]interface{})
			for k, v := range vm {
				vmGeneric[k] = v
			}
			hmGeneric := make(map[string]interface{})
			for k, v := range hm {
				hmGeneric[k] = v
			}
			handlers := opcodes.BuildHandlerMap(l, handlerConfig, vmGeneric, hmGeneric, adapt)
			return func() map[uint16]request.Handler {
				return handlers
			}
		}
	}
}
