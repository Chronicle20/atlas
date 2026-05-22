package main

import (
	"atlas-login/account"
	"atlas-login/configuration"
	"atlas-login/configuration/projection"
	account2 "atlas-login/kafka/consumer/account"
	session2 "atlas-login/kafka/consumer/account/session"
	"atlas-login/kafka/consumer/seed"
	"atlas-login/listener"
	"atlas-login/logger"
	"atlas-login/session"
	"atlas-login/socket"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tasks"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-opcodes"
	account3 "github.com/Chronicle20/atlas/libs/atlas-packet/account/serverbound"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	loginCB "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	loginSB "github.com/Chronicle20/atlas/libs/atlas-packet/login/serverbound"
	socketcb "github.com/Chronicle20/atlas/libs/atlas-packet/socket/clientbound"
	socketsb "github.com/Chronicle20/atlas/libs/atlas-packet/socket/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	restserver "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	socket2 "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

	serviceId := uuid.MustParse(os.Getenv("SERVICE_ID"))
	var consumerGroupId = consumergroup.Resolve(consumerGroupIdTemplate, serviceId.String())

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	account2.InitConsumers(l)(cmf)(consumerGroupId)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	seed.InitConsumers(l)(cmf)(consumerGroupId)

	// Boot the configuration projection: subscribe to the two
	// config-status topics, gate on caught-up so we don't drive the
	// listener registry from a half-loaded state, then run the apply loop
	// in a goroutine.
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	sub := &projection.Subscriber{
		State:        state,
		CaughtUp:     caughtUp,
		ServiceTopic: os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"),
		TenantTopic:  os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"),
		ServiceId:    serviceId,
	}
	if err := sub.Start(tdm.Context(), l, consumerGroupId); err != nil {
		l.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}

	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), 30*time.Second)
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up within 30s.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up; starting listener apply loop.")

	// Bridge the projection snapshot back into the legacy configuration
	// package vars so existing GetServiceConfig / GetTenantConfig callers
	// (handlers, the session timeout task, the account-session consumer)
	// continue to work without the REST-based Init.
	publishSnapshot := func() {
		svc, tenants := state.Snapshot()
		configuration.PublishSnapshot(svc, tenants)
	}
	publishSnapshot()

	listenerRegistry := listener.NewRegistry(l, listener.Dependencies{
		// atlas-login sessions are stateless after handshake; SessionsForKey
		// has no per-tenant index to walk. Phase 2 of drain becomes a no-op
		// and phase 4 still cancels the ctx so handlers stop.
		SessionsForKey:     func(listener.Key) []listener.Session { return nil },
		SendShutdownNotice: func(listener.Session) {},
		DestroySession:     func(listener.Session) error { return nil },
		RemoveHandler:      consumer.GetManager().RemoveHandler,
	}, listener.Config{
		DrainDeadline: parseDrainDeadline(),
	})

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before drain begins. k8s removes the pod
	// from service endpoints once readiness fails, giving in-flight
	// requests a chance to land on a healthy peer.
	var shuttingDown atomic.Bool
	ready := func() bool { return caughtUp.CaughtUpNow() && !shuttingDown.Load() }

	// Teardown order: flip /readyz first, drain listeners next, then the
	// downstream teardowns that destroy state in-flight handlers might touch.
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})
	tdm.TeardownFunc(func() {
		l.Info("Draining all listeners.")
		listenerRegistry.DrainAll()
	})

	build := buildListener(l, tdm, state, validatorMap, handlerMap, writerList)
	go (&projection.ApplyLoop{
		State:       state,
		CaughtUp:    caughtUp,
		Registry:    listenerRegistry,
		AddBody:     build,
		ServerModel: serverModelFn,
		Interval:    250 * time.Millisecond,
	}).Run(tdm.Context(), l)

	// Republish the legacy configuration vars on a slow ticker so
	// operator-driven config changes flow to the handlers that still read
	// from the package-level cache. Cheap: it's a value-copy of a small
	// map. Stops when the teardown ctx cancels.
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-tdm.Context().Done():
				return
			case <-t.C:
				publishSnapshot()
			}
		}
	}()

	tt, err := func() (time.Duration, error) {
		c, err := configuration.GetServiceConfig()
		if err != nil {
			return 0, err
		}
		task, err := c.FindTask(session.TimeoutTask)
		if err != nil {
			return 0, err
		}
		return time.Millisecond * time.Duration(task.Interval), nil
	}()
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	go tasks.Register(l, tdm.Context())(session.NewTimeout(l, tt))

	tdm.TeardownFunc(session.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	restserver.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(restserver.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(restserver.MountReadiness("/readyz", ready)).
		Run()

	tdm.Wait()

	l.Infoln("Service shutdown.")
}

// serverModelFn is the ServerModelFn the apply loop hands to listener.Add.
// Idempotent: tenant.Register tolerates duplicate ids by overwriting the
// registry entry.
func serverModelFn(key listener.Key, cfg projection.ListenerConfig) listener.ServerModel {
	t, err := tenant.Register(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
	if err != nil {
		// tenant.Register only errors when Create errors; Create currently
		// can't fail, but if it ever did we still need a Model. Fall back
		// to a synthesized one so the listener can at least start.
		t, _ = tenant.Create(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
	}
	// atlas-login binds the socket to all interfaces; the advertised IP
	// is captured here only for parity with the channel-side ServerModel.
	return listener.NewServerModel(t, "", cfg.Port)
}

// buildListener returns the per-tenant AddBody the projection apply loop
// invokes inside listener.Registry.Add. The closure captures shared
// dependencies (validator/handler/writer maps, the projection state) so
// each invocation can read the tenant's full socket config without
// thrashing through the global consumer.GetManager() singleton's locks
// any more than necessary.
func buildListener(
	l logrus.FieldLogger,
	tdm *service.Manager,
	state *projection.State,
	validatorMap map[string]handler.MessageValidator,
	handlerMap map[string]handler.MessageHandler,
	writerList []string,
) projection.AddBody {
	return func(ctx context.Context, key listener.Key, cfg projection.ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error) {
		_, tenants := state.Snapshot()
		tenantCfg, ok := tenants[key.TenantId]
		if !ok {
			return nil, fmt.Errorf("tenant %s missing from projection state", key.TenantId)
		}

		t, err := tenant.Register(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
		if err != nil {
			return nil, err
		}
		tctx := tenant.WithContext(ctx, t)

		if err := account.NewProcessor(l, tctx).InitializeRegistry(); err != nil {
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

		wp := produceWriterProducer(fl)(tenantCfg.Socket.Writers, writerList, rw)
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)

		rh := consumer.GetManager().RegisterHandler
		var handles []listener.HandlerHandle
		register := func(hh []listener.HandlerHandle, err error) error {
			if err != nil {
				return err
			}
			handles = append(handles, hh...)
			return nil
		}

		if err := register(account2.InitHandlers(fl)(t)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(session2.InitHandlers(fl)(t)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(seed.InitHandlers(fl)(t)(wp)(rh)); err != nil {
			return nil, err
		}

		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, cfg.Port)

		return handles, nil
	}
}

// parseDrainDeadline reads DRAIN_DEADLINE_MS from env (default 2000ms,
// clamped to a 5s ceiling for atlas-login — sessions are stateless after
// handshake). The listener.Registry enforces the same ceiling internally;
// this parse exists so the operator log shows the effective value we
// picked.
func parseDrainDeadline() time.Duration {
	const def = 2000 * time.Millisecond
	const ceiling = 5 * time.Second
	v := os.Getenv("DRAIN_DEADLINE_MS")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	d := time.Duration(n) * time.Millisecond
	if d > ceiling {
		return ceiling
	}
	return d
}

func produceWriterProducer(l logrus.FieldLogger) func(writers []opcodes.WriterConfig, writerList []string, w socket2.OpWriter) writer.Producer {
	return func(writers []opcodes.WriterConfig, writerList []string, w socket2.OpWriter) writer.Producer {
		return opcodes.BuildWriterProducer(l, writers, writerList, w)
	}
}

func produceWriters() []string {
	return []string{
		loginCB.LoginAuthWriter,
		loginCB.AuthSuccessWriter,
		loginCB.AuthTemporaryBanWriter,
		loginCB.AuthPermanentBanWriter,
		loginCB.AuthLoginFailedWriter,
		loginCB.ServerListRecommendationsWriter,
		loginCB.ServerListEntryWriter,
		loginCB.ServerListEndWriter,
		loginCB.SelectWorldWriter,
		loginCB.ServerStatusWriter,
		charcb.CharacterListWriter,
		charcb.CharacterNameResponseWriter,
		charcb.AddCharacterEntryWriter,
		charcb.DeleteCharacterResponseWriter,
		loginCB.PinOperationWriter,
		loginCB.PinUpdateWriter,
		loginCB.PicResultWriter,
		loginCB.ServerIPWriter,
		loginCB.ServerLoadWriter,
		loginCB.SetAccountResultWriter,
		charcb.CharacterViewAllWriter,
		socketcb.PingWriter,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[handler.DebugHandle] = handler.DebugHandleFunc
	handlerMap[handler.CreateSecurityHandle] = handler.CreateSecurityHandleFunc
	handlerMap[loginSB.LoginHandle] = handler.LoginHandleFunc
	handlerMap[loginSB.ServerListRequestHandle] = handler.ServerListRequestHandleFunc
	handlerMap[loginSB.ServerStatusHandle] = handler.ServerStatusHandleFunc
	handlerMap[loginSB.WorldCharacterListHandle] = handler.CharacterListWorldHandleFunc
	handlerMap[charsb.CharacterCheckNameHandle] = handler.CharacterCheckNameHandleFunc
	handlerMap[charsb.CreateCharacterHandle] = handler.CreateCharacterHandleFunc
	handlerMap[charsb.DeleteCharacterHandle] = handler.DeleteCharacterHandleFunc
	handlerMap[loginSB.AfterLoginHandle] = handler.AfterLoginHandleFunc
	handlerMap[account3.RegisterPinHandle] = handler.RegisterPinHandleFunc
	handlerMap[loginSB.RegisterPicHandle] = handler.RegisterPicHandleFunc
	handlerMap[account3.AcceptTosHandle] = handler.AcceptTosHandleFunc
	handlerMap[loginSB.CharacterSelectedHandle] = handler.CharacterSelectedHandleFunc
	handlerMap[loginSB.CharacterSelectedPicHandle] = handler.CharacterSelectedPicHandleFunc
	handlerMap[loginSB.WorldSelectHandle] = handler.WorldSelectHandleFunc
	handlerMap[account3.SetGenderHandle] = handler.SetGenderHandleFunc
	handlerMap[loginSB.CharacterViewAllHandle] = handler.CharacterViewAllHandleFunc
	handlerMap[loginSB.CharacterViewAllSelectedHandle] = handler.CharacterViewAllSelectedHandleFunc
	handlerMap[loginSB.CharacterViewAllSelectedPicRegisterHandle] = handler.CharacterViewAllSelectedPicRegisterHandleFunc
	handlerMap[loginSB.CharacterViewAllSelectedPicHandle] = handler.CharacterViewAllSelectedPicHandleFunc
	handlerMap[loginSB.CharacterViewAllPongHandle] = handler.CharacterViewAllPongHandleFunc
	handlerMap[handler.ClientStartHandle] = handler.ClientStartHandleFunc
	handlerMap[socketsb.PongHandle] = handler.PongHandleFunc
	handlerMap[socketsb.StartErrorHandle] = handler.StartErrorHandleFunc
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
