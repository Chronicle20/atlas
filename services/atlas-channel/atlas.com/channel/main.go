package main

import (
	"atlas-channel/account"
	channel3 "atlas-channel/channel"
	"atlas-channel/configuration/projection"
	account2 "atlas-channel/kafka/consumer/account"
	"atlas-channel/kafka/consumer/asset"
	"atlas-channel/kafka/consumer/buddylist"
	"atlas-channel/kafka/consumer/buff"
	"atlas-channel/kafka/consumer/cashshop"
	cashshopCompartment "atlas-channel/kafka/consumer/cashshop/compartment"
	"atlas-channel/kafka/consumer/chair"
	"atlas-channel/kafka/consumer/chalkboard"
	"atlas-channel/kafka/consumer/channel"
	"atlas-channel/kafka/consumer/character"
	"atlas-channel/kafka/consumer/compartment"
	"atlas-channel/kafka/consumer/consumable"
	"atlas-channel/kafka/consumer/conversation_reward_notice"
	"atlas-channel/kafka/consumer/drop"
	"atlas-channel/kafka/consumer/expression"
	"atlas-channel/kafka/consumer/fame"
	"atlas-channel/kafka/consumer/gachapon"
	"atlas-channel/kafka/consumer/guild"
	"atlas-channel/kafka/consumer/guild/thread"
	"atlas-channel/kafka/consumer/instance_transport"
	"atlas-channel/kafka/consumer/invite"
	"atlas-channel/kafka/consumer/map"
	merchantConsumer "atlas-channel/kafka/consumer/merchant"
	"atlas-channel/kafka/consumer/message"
	"atlas-channel/kafka/consumer/messenger"
	mistConsumer "atlas-channel/kafka/consumer/mist"
	"atlas-channel/kafka/consumer/monster"
	mbconsumer "atlas-channel/kafka/consumer/monsterbook"
	monsterDomain "atlas-channel/monster"
	note3 "atlas-channel/kafka/consumer/note"
	"atlas-channel/kafka/consumer/npc/conversation"
	"atlas-channel/kafka/consumer/npc/shop"
	"atlas-channel/kafka/consumer/party"
	"atlas-channel/kafka/consumer/party/member"
	"atlas-channel/kafka/consumer/party_quest"
	"atlas-channel/kafka/consumer/pet"
	"atlas-channel/kafka/consumer/quest"
	"atlas-channel/kafka/consumer/reactor"
	"atlas-channel/kafka/consumer/route"
	"atlas-channel/kafka/consumer/saga"
	session2 "atlas-channel/kafka/consumer/session"
	"atlas-channel/kafka/consumer/skill"
	storage3 "atlas-channel/kafka/consumer/storage"
	"atlas-channel/kafka/consumer/system_message"
	"atlas-channel/listener"
	"atlas-channel/logger"
	"atlas-channel/server"
	"atlas-channel/session"
	_ "atlas-channel/skill/handler/registrations"
	"atlas-channel/socket"
	"atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"atlas-channel/tasks"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	buddy2 "github.com/Chronicle20/atlas/libs/atlas-packet/buddy"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	channelCB "github.com/Chronicle20/atlas/libs/atlas-packet/channel/clientbound"
	channelSB "github.com/Chronicle20/atlas/libs/atlas-packet/channel/serverbound"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	mbcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound/monsterbook"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	mbsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound/monsterbook"
	chatCB "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	chatSB "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	dropcb "github.com/Chronicle20/atlas/libs/atlas-packet/drop/clientbound"
	dropsb "github.com/Chronicle20/atlas/libs/atlas-packet/drop/serverbound"
	famecb "github.com/Chronicle20/atlas/libs/atlas-packet/fame/clientbound"
	famesb "github.com/Chronicle20/atlas/libs/atlas-packet/fame/serverbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	guildcb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	guildsb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/serverbound"
	interaction2 "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	interactionsb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/serverbound"
	invcb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/clientbound"
	invsb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	messengercb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/clientbound"
	messengersb "github.com/Chronicle20/atlas/libs/atlas-packet/messenger/serverbound"
	monstercb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
	notesb "github.com/Chronicle20/atlas/libs/atlas-packet/note/serverbound"
	npccb "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	npcsb "github.com/Chronicle20/atlas/libs/atlas-packet/npc/serverbound"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	partysb "github.com/Chronicle20/atlas/libs/atlas-packet/party/serverbound"
	petcb "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	petsb "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	portal2 "github.com/Chronicle20/atlas/libs/atlas-packet/portal/serverbound"
	questcb "github.com/Chronicle20/atlas/libs/atlas-packet/quest/clientbound"
	questsb "github.com/Chronicle20/atlas/libs/atlas-packet/quest/serverbound"
	reactorcb "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/clientbound"
	reactorsb "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/serverbound"
	socketcb "github.com/Chronicle20/atlas/libs/atlas-packet/socket/clientbound"
	socketsb "github.com/Chronicle20/atlas/libs/atlas-packet/socket/serverbound"
	stat2 "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	storagecb "github.com/Chronicle20/atlas/libs/atlas-packet/storage/clientbound"
	storagesb "github.com/Chronicle20/atlas/libs/atlas-packet/storage/serverbound"
	ui2 "github.com/Chronicle20/atlas/libs/atlas-packet/ui/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-service"

	channel2 "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-opcodes"
	restserver "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	socket2 "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const serviceName = "atlas-channel"
const consumerGroupIdTemplate = "Channel Service - %s"

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

	monsterDomain.InitNextSkillInbox()

	account2.InitConsumers(l)(cmf)(consumerGroupId)
	asset.InitConsumers(l)(cmf)(consumerGroupId)
	buddylist.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	channel.InitConsumers(l)(cmf)(consumerGroupId)
	conversation.InitConsumers(l)(cmf)(consumerGroupId)
	shop.InitConsumers(l)(cmf)(consumerGroupId)
	expression.InitConsumers(l)(cmf)(consumerGroupId)
	guild.InitConsumers(l)(cmf)(consumerGroupId)
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	invite.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	member.InitConsumers(l)(cmf)(consumerGroupId)
	message.InitConsumers(l)(cmf)(consumerGroupId)
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	mbconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	mistConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	party.InitConsumers(l)(cmf)(consumerGroupId)
	party_quest.InitConsumers(l)(cmf)(consumerGroupId)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	fame.InitConsumers(l)(cmf)(consumerGroupId)
	thread.InitConsumers(l)(cmf)(consumerGroupId)
	chair.InitConsumers(l)(cmf)(consumerGroupId)
	drop.InitConsumers(l)(cmf)(consumerGroupId)
	reactor.InitConsumers(l)(cmf)(consumerGroupId)
	skill.InitConsumers(l)(cmf)(consumerGroupId)
	buff.InitConsumers(l)(cmf)(consumerGroupId)
	chalkboard.InitConsumers(l)(cmf)(consumerGroupId)
	messenger.InitConsumers(l)(cmf)(consumerGroupId)
	pet.InitConsumers(l)(cmf)(consumerGroupId)
	consumable.InitConsumers(l)(cmf)(consumerGroupId)
	conversation_reward_notice.InitConsumers(l)(cmf)(consumerGroupId)
	system_message.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	cashshopCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	note3.InitConsumers(l)(cmf)(consumerGroupId)
	quest.InitConsumers(l)(cmf)(consumerGroupId)
	route.InitConsumers(l)(cmf)(consumerGroupId)
	instance_transport.InitConsumers(l)(cmf)(consumerGroupId)
	saga.InitConsumers(l)(cmf)(consumerGroupId)
	storage3.InitConsumers(l)(cmf)(consumerGroupId)
	gachapon.InitConsumers(l)(cmf)(consumerGroupId)
	merchantConsumer.InitConsumers(l)(cmf)(consumerGroupId)

	// Boot the configuration projection: subscribe to the two config-status
	// topics, gate on caught-up so we don't drive the listener registry
	// from a half-loaded state, then run the apply loop in a goroutine.
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

	// 5-minute window because in a fresh PR env atlas-pr-bootstrap takes
	// a couple of minutes to write the initial tenant + service configs
	// after this pod boots; the projection can't catch up until those
	// events are emitted by atlas-configurations and drained to Kafka.
	// Override via PROJECTION_CATCHUP_TIMEOUT_S (positive integer seconds).
	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), parseProjectionCatchupTimeout())
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up; starting listener apply loop.")

	listenerRegistry := listener.NewRegistry(l, listener.Dependencies{
		UnregisterChannel: func(ch channel2.Model) error {
			return channel3.NewProcessor(l, tdm.Context()).Unregister(ch)
		},
		SessionsForKey: func(key server.Key) []listener.Session {
			// TODO: wire session.Processor lookup-by-key once available.
			// Returning nil yields an empty drain phase 2, which is safe
			// — phase 4 still cancels the ctx so handlers stop.
			return nil
		},
		SendShutdownNotice: func(listener.Session) {},
		DestroySession:     func(listener.Session) error { return nil },
		RemoveHandler:      consumer.GetManager().RemoveHandler,
	}, listener.Config{
		DrainDeadline: parseDrainDeadline(),
	})

	// Drop tenant-scoped caches once the last listener for a tenant drains
	// so a later re-Add of the same tenant starts clean. Fires per-tenant,
	// at most once per drain-to-zero transition (listener.Registry guards
	// with a ref count).
	listener.RegisterEvictor(func(t tenant.Model) {
		tid := t.Id()
		account.GetRegistry().EvictTenant(tid)
		monsterDomain.GetStatusMirror().EvictTenant(tid)
		if inbox := monsterDomain.GetNextSkillInbox(); inbox != nil {
			inbox.EvictTenant(tid)
		}
		tenant.Unregister(tid)
	})

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before drain begins. k8s removes the pod
	// from service endpoints once readiness fails, giving in-flight
	// requests a chance to land on a healthy peer.
	var shuttingDown atomic.Bool
	ready := func() bool { return caughtUp.CaughtUpNow() && !shuttingDown.Load() }

	// Teardown order matters here:
	//   1. Flip /readyz → 503 so k8s stops sending new traffic.
	//   2. Drain every listener (in-flight kafka handlers stop touching state).
	//   3. Producer close, session teardown, tracing flush.
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

	go tasks.Register(l, tdm.Context())(channel3.NewHeartbeat(l, tdm.Context(), time.Second*10))

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
// registry entry, and server.Register replaces any prior entry at this key.
func serverModelFn(key server.Key, cfg projection.ListenerConfig) server.Model {
	t, err := tenant.Register(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
	if err != nil {
		// tenant.Register only errors when Create errors; Create currently
		// can't fail, but if it ever did we still need a Model. Fall back
		// to a synthesized one so the listener can at least start.
		t, _ = tenant.Create(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
	}
	return server.Register(t, channel2.NewModel(key.WorldId, key.ChannelId), cfg.IPAddress, cfg.Port)
}

// buildListener returns the per-(t,w,c) AddBody the projection apply loop
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
	return func(ctx context.Context, key server.Key, cfg projection.ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error) {
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

		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}

		sc := h.ServerModel

		fl := l.
			WithField("tenant", t.Id().String()).
			WithField("region", t.Region()).
			WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())).
			WithField("world.id", sc.WorldId()).
			WithField("channel.id", sc.ChannelId())

		wp := produceWriterProducer(fl)(tenantCfg.Socket.Writers, writerList, rw)

		rh := consumer.GetManager().RegisterHandler
		var handles []listener.HandlerHandle
		register := func(hh []listener.HandlerHandle, err error) error {
			if err != nil {
				return err
			}
			handles = append(handles, hh...)
			return nil
		}

		if err := register(account2.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(asset.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(buddylist.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(channel.InitHandlers(fl)(sc)(cfg.IPAddress, cfg.Port)(rh)); err != nil {
			return nil, err
		}
		if err := register(character.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(expression.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(guild.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(compartment.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(invite.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(_map.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(message.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(monster.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(mbconsumer.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(mistConsumer.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(conversation.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(shop.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(member.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(party.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(party_quest.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(session2.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(fame.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(thread.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(chair.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(drop.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(reactor.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(skill.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(buff.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(chalkboard.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(messenger.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(pet.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(consumable.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(conversation_reward_notice.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(system_message.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(cashshop.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(cashshopCompartment.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(note3.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(quest.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(route.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(instance_transport.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(saga.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(storage3.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(gachapon.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
		if err := register(merchantConsumer.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}

		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, sc, cfg.IPAddress, cfg.Port)

		return handles, nil
	}
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from
// env (positive integer seconds) and returns the catch-up window for the
// configuration projection at startup. Default is 5 minutes, which covers
// the fresh-PR-env case where atlas-pr-bootstrap is still writing the
// initial tenant + service configs when this pod boots.
func parseProjectionCatchupTimeout() time.Duration {
	const def = 5 * time.Minute
	v := os.Getenv("PROJECTION_CATCHUP_TIMEOUT_S")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}

// parseDrainDeadline reads DRAIN_DEADLINE_MS from env (default 5000ms,
// clamped to a 10s ceiling). The listener.Registry enforces the same
// ceiling internally; this parse exists so the operator log shows the
// effective value we picked.
func parseDrainDeadline() time.Duration {
	const def = 5000 * time.Millisecond
	const ceiling = 10 * time.Second
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
		fieldcb.SetFieldWriter,
		npccb.NpcSpawnWriter,
		npccb.NpcSpawnRequestControllerWriter,
		npccb.NpcActionWriter,
		stat2.StatChangedWriter,
		channelCB.ChannelChangeWriter,
		cashcb.CashShopOpenWriter,
		cashcb.CashShopOperationWriter,
		cashcb.CashQueryResultWriter,
		monstercb.MonsterSpawnWriter,
		monstercb.MonsterDestroyWriter,
		monstercb.MonsterControlWriter,
		monstercb.MonsterMovementWriter,
		monstercb.MonsterMovementAckWriter,
		charcb.CharacterSpawnWriter,
		chatCB.GeneralChatWriter,
		charcb.CharacterMovementWriter,
		charcb.CharacterInfoWriter,
		invcb.InventoryChangeWriter,
		charcb.CharacterAppearanceUpdateWriter,
		charcb.CharacterDespawnWriter,
		partycb.PartyOperationWriter,
		chatCB.MultiChatWriter,
		charcb.CharacterKeyMapWriter,
		buddy2.BuddyOperationWriter,
		charcb.CharacterExpressionWriter,
		npccb.NpcConversationWriter,
		guildcb.GuildOperationWriter,
		guildcb.GuildEmblemChangedWriter,
		guildcb.GuildNameChangedWriter,
		famecb.FameResponseWriter,
		charcb.CharacterStatusMessageWriter,
		guildcb.GuildBBSWriter,
		charcb.CharacterShowChairWriter,
		charcb.CharacterSitResultWriter,
		dropcb.DropSpawnWriter,
		dropcb.DropDestroyWriter,
		reactorcb.ReactorSpawnWriter,
		reactorcb.ReactorDestroyWriter,
		charcb.CharacterSkillChangeWriter,
		charcb.CharacterAttackMeleeWriter,
		charcb.CharacterAttackRangedWriter,
		charcb.CharacterAttackMagicWriter,
		charcb.CharacterAttackEnergyWriter,
		charcb.CharacterDamageWriter,
		charcb.CharacterBuffGiveWriter,
		charcb.CharacterBuffGiveForeignWriter,
		charcb.CharacterBuffCancelWriter,
		charcb.CharacterBuffCancelForeignWriter,
		charcb.CharacterSkillCooldownWriter,
		charcb.CharacterEffectWriter,
		charcb.CharacterEffectForeignWriter,
		chatCB.WorldMessageWriter,
		monstercb.MonsterHealthWriter,
		partycb.PartyMemberHPWriter,
		charcb.ChalkboardUseWriter,
		chatCB.WhisperWriter,
		messengercb.MessengerOperationWriter,
		petcb.PetActivatedWriter,
		petcb.PetMovementWriter,
		petcb.PetCommandResponseWriter,
		petcb.PetChatWriter,
		charcb.CharacterItemUpgradeWriter,
		character2.CharacterSkillMacroWriter,
		petcb.PetExcludeResponseWriter,
		petcb.PetCashFoodResultWriter,
		charcb.CharacterKeyMapAutoHpWriter,
		charcb.CharacterKeyMapAutoMpWriter,
		npccb.NPCShopWriter,
		npccb.NPCShopOperationWriter,
		invcb.CompartmentMergeWriter,
		invcb.CompartmentSortWriter,
		notecb.NoteOperationWriter,
		fieldcb.KiteSpawnWriter,
		fieldcb.KiteErrorWriter,
		fieldcb.KiteDestroyWriter,
		fieldcb.ClockWriter,
		fieldcb.FieldTransportStateWriter,
		storagecb.StorageOperationWriter,
		charcb.CharacterHintWriter,
		reactorcb.ReactorHitWriter,
		npccb.GuideTalkWriter,
		questcb.ScriptProgressWriter,
		socketcb.PingWriter,
		fieldcb.FieldEffectWriter,
		ui2.UiOpenWriter,
		ui2.UiLockWriter,
		ui2.UiDisableWriter,
		monstercb.MonsterStatSetWriter,
		monstercb.MonsterStatResetWriter,
		monstercb.MonsterDamageWriter,
		fieldcb.FieldEffectWeatherWriter,
		merchantcb.HiredMerchantOperationWriter,
		interactioncb.CharacterInteractionWriter,
		interaction2.MiniRoomWriter,
		mbcb.MonsterBookSetCardWriter,
		mbcb.MonsterBookSetCoverWriter,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[socketsb.CharacterLoggedInHandle] = handler.CharacterLoggedInHandleFunc
	handlerMap[npcsb.NPCActionHandle] = handler.NPCActionHandleFunc
	handlerMap[portal2.PortalScriptHandle] = handler.PortalScriptHandleFunc
	handlerMap[fieldsb.MapChangeHandle] = handler.MapChangeHandleFunc
	handlerMap[charsb.CharacterMoveHandle] = handler.CharacterMoveHandleFunc
	handlerMap[channelSB.ChannelChangeRequestHandle] = handler.ChannelChangeHandleFunc
	handlerMap[cashsb.CashShopEntryHandle] = handler.CashShopEntryHandleFunc
	handlerMap[monstersb.MonsterMovementHandle] = handler.MonsterMovementHandleFunc
	handlerMap[chatSB.CharacterChatGeneralHandle] = handler.CharacterChatGeneralHandleFunc
	handlerMap[charsb.CharacterInfoRequestHandle] = handler.CharacterInfoRequestHandleFunc
	handlerMap[invsb.CharacterInventoryMoveHandle] = handler.CharacterInventoryMoveHandleFunc
	handlerMap[partysb.PartyOperationHandle] = handler.PartyOperationHandleFunc
	handlerMap[partysb.PartyInviteRejectHandle] = handler.PartyInviteRejectHandleFunc
	handlerMap[chatSB.CharacterChatMultiHandle] = handler.CharacterChatMultiHandleFunc
	handlerMap[charsb.CharacterKeyMapChangeHandle] = handler.CharacterKeyMapChangeHandleFunc
	handlerMap[buddy2.BuddyOperationHandle] = handler.BuddyOperationHandleFunc
	handlerMap[charsb.CharacterExpressionHandle] = handler.CharacterExpressionHandleFunc
	handlerMap[npcsb.NPCStartConversationHandle] = handler.NPCStartConversationHandleFunc
	handlerMap[npcsb.NPCContinueConversationHandle] = handler.NPCContinueConversationHandleFunc
	handlerMap[guildsb.GuildOperationHandle] = handler.GuildOperationHandleFunc
	handlerMap[guildsb.GuildInviteRejectHandle] = handler.GuildInviteRejectHandleFunc
	handlerMap[famesb.FameChangeHandle] = handler.FameChangeHandleFunc
	handlerMap[charsb.CharacterDistributeApHandle] = handler.CharacterDistributeApHandleFunc
	handlerMap[charsb.CharacterAutoDistributeApHandle] = handler.CharacterAutoDistributeApHandleFunc
	handlerMap[guildsb.GuildBBSHandle] = handler.GuildBBSHandleFunc
	handlerMap[charsb.CharacterChairPortableHandle] = handler.CharacterChairPortableHandleFunc
	handlerMap[charsb.CharacterChairInteractionHandle] = handler.CharacterChairFixedHandleFunc
	handlerMap[dropsb.DropPickUpHandle] = handler.DropPickUpHandleFunc
	handlerMap[charsb.CharacterDropMesoHandle] = handler.CharacterDropMesoHandleFunc
	handlerMap[handler.CharacterMeleeAttackHandle] = handler.CharacterMeleeAttackHandleFunc
	handlerMap[handler.CharacterRangedAttackHandle] = handler.CharacterRangedAttackHandleFunc
	handlerMap[handler.CharacterMagicAttackHandle] = handler.CharacterMagicAttackHandleFunc
	handlerMap[handler.CharacterTouchAttackHandle] = handler.CharacterTouchAttackHandleFunc
	handlerMap[charsb.CharacterHealOverTimeHandle] = handler.CharacterHealOverTimeHandleFunc
	handlerMap[packetmodel.CharacterDamageHandle] = handler.CharacterDamageHandleFunc
	handlerMap[charsb.CharacterDistributeSpHandle] = handler.CharacterDistributeSpHandleFunc
	handlerMap[handler.CharacterUseSkillHandle] = handler.CharacterUseSkillHandleFunc
	handlerMap[charsb.CharacterBuffCancelHandle] = handler.CharacterBuffCancelHandleFunc
	handlerMap[cashsb.CharacterCashItemUseHandle] = handler.CharacterCashItemUseHandleFunc
	handlerMap[charsb.ChalkboardCloseHandle] = handler.ChalkboardCloseHandleHandleFunc
	handlerMap[chatSB.CharacterChatWhisperHandle] = handler.CharacterChatWhisperHandleFunc
	handlerMap[messengersb.MessengerOperationHandle] = handler.MessengerOperationHandleFunc
	handlerMap[petsb.PetMovementHandle] = handler.PetMovementHandleFunc
	handlerMap[petsb.PetSpawnHandle] = handler.PetSpawnHandleFunc
	handlerMap[petsb.PetCommandHandle] = handler.PetCommandHandleFunc
	handlerMap[petsb.PetChatHandle] = handler.PetChatHandleFunc
	handlerMap[petsb.PetDropPickUpHandle] = handler.PetDropPickUpHandleFunc
	handlerMap[petsb.PetFoodHandle] = handler.PetFoodHandleFunc
	handlerMap[invsb.CharacterItemUseHandle] = handler.CharacterItemUseHandleFunc
	handlerMap[charsb.CharacterItemCancelHandle] = handler.CharacterItemCancelHandleFunc
	handlerMap[invsb.CharacterItemUseTownScrollHandle] = handler.CharacterItemUseTownScrollHandleFunc
	handlerMap[invsb.CharacterItemUseScrollHandle] = handler.CharacterItemUseScrollHandleFunc
	handlerMap[character2.CharacterSkillMacroHandle] = handler.CharacterSkillMacroHandleFunc
	handlerMap[petsb.PetItemExcludeHandle] = handler.PetItemExcludeHandleFunc
	handlerMap[petsb.PetItemUseHandle] = handler.PetItemUseHandleFunc
	handlerMap[cashsb.CashShopOperationHandle] = handler.CashShopOperationHandleFunc
	handlerMap[cashsb.CashShopCheckWalletHandle] = handler.CashShopCheckWalletHandleFunc
	handlerMap[npcsb.NPCShopHandle] = handler.NPCShopHandleFunc
	handlerMap[invsb.CompartmentMergeRequestHandle] = handler.CompartmentMergeHandleFunc
	handlerMap[invsb.CompartmentSortRequestHandle] = handler.CompartmentSortHandleFunc
	handlerMap[invsb.CharacterItemUseSummonBagHandle] = handler.CharacterItemUseSummonBagHandleFunc
	handlerMap[notesb.NoteOperationHandle] = handler.NoteOperationHandleFunc
	handlerMap[questsb.QuestActionHandle] = handler.QuestActionHandleFunc
	handlerMap[storagesb.StorageOperationHandle] = handler.StorageOperationHandleFunc
	handlerMap[reactorsb.ReactorHitHandle] = handler.ReactorHitHandleFunc
	handlerMap[socketsb.PongHandle] = handler.PongHandleFunc
	handlerMap[charsb.MonsterDamageFriendlyHandle] = handler.MonsterDamageFriendlyHandleFunc
	handlerMap[interactionsb.CharacterInteractionHandle] = handler.CharacterInteractionHandleFunc
	handlerMap[merchantsb.HiredMerchantOperationHandle] = handler.HiredMerchantOperationHandleFunc
	handlerMap[mbsb.MonsterBookCoverHandler] = handler.MonsterBookCoverHandleFunc
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
