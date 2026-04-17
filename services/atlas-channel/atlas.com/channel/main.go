package main

import (
	"atlas-channel/account"
	channel3 "atlas-channel/channel"
	"atlas-channel/configuration"
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
	"atlas-channel/kafka/consumer/monster"
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
	"atlas-channel/logger"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket"
	"atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"atlas-channel/tasks"
	"atlas-channel/tracing"
	"fmt"
	"os"
	"time"

	buddy2 "github.com/Chronicle20/atlas/libs/atlas-packet/buddy"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	channelCB "github.com/Chronicle20/atlas/libs/atlas-packet/channel/clientbound"
	channelSB "github.com/Chronicle20/atlas/libs/atlas-packet/channel/serverbound"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
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
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-opcodes"
	socket2 "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
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

	configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))
	config, err := configuration.GetServiceConfig()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}
	var consumerGroupId = fmt.Sprintf(consumerGroupIdTemplate, config.Id.String())

	validatorMap := produceValidators()
	handlerMap := produceHandlers()
	writerList := produceWriters()

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
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

		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}

		for _, w := range ten.Worlds {
			for _, c := range w.Channels {
				ch := channel2.NewModel(world.Id(w.Id), channel2.Id(c.Id))
				sc := server.Register(t, ch, ten.IPAddress, c.Port)

				fl := l.
					WithField("tenant", t.Id().String()).
					WithField("region", t.Region()).
					WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())).
					WithField("world.id", sc.WorldId()).
					WithField("channel.id", sc.ChannelId())

				wp := produceWriterProducer(fl)(tenantConfig.Socket.Writers, writerList, rw)
				if err = account2.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = asset.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = buddylist.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = channel.InitHandlers(fl)(sc)(ten.IPAddress, c.Port)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = character.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = expression.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = guild.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = compartment.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = invite.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = _map.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = message.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = monster.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = conversation.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = shop.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = member.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = party.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = party_quest.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = session2.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = fame.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = thread.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = chair.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = drop.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = reactor.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = skill.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = buff.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = chalkboard.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = messenger.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = pet.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = consumable.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = system_message.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = cashshop.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = cashshopCompartment.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = note3.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = quest.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = route.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = instance_transport.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = saga.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = storage3.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = gachapon.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}
				if err = merchantConsumer.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
					fl.WithError(err).Fatal("Unable to register kafka handlers.")
				}

				hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantConfig.Socket.Handlers, validatorMap, handlerMap)
				socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, sc, ten.IPAddress, c.Port)
			}
		}
	}
	span.End()

	//tt, err := config.FindTask(session.TimeoutTask)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	//go tasks.Register(l, tdm.Context())(session.NewTimeout(l, time.Millisecond*time.Duration(tt.Interval)))
	go tasks.Register(l, tdm.Context())(channel3.NewHeartbeat(l, tdm.Context(), time.Second*10))

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
