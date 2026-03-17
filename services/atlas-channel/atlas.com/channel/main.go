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

	buddy2 "github.com/Chronicle20/atlas-packet/buddy"
	cash2 "github.com/Chronicle20/atlas-packet/cash"
	channel4 "github.com/Chronicle20/atlas-packet/channel"
	character2 "github.com/Chronicle20/atlas-packet/character"
	chat2 "github.com/Chronicle20/atlas-packet/chat"
	drop2 "github.com/Chronicle20/atlas-packet/drop"
	fame2 "github.com/Chronicle20/atlas-packet/fame"
	field2 "github.com/Chronicle20/atlas-packet/field"
	guild2 "github.com/Chronicle20/atlas-packet/guild"
	interaction2 "github.com/Chronicle20/atlas-packet/interaction"
	inventory2 "github.com/Chronicle20/atlas-packet/inventory"
	merchant2 "github.com/Chronicle20/atlas-packet/merchant"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	messenger2 "github.com/Chronicle20/atlas-packet/messenger"
	monster2 "github.com/Chronicle20/atlas-packet/monster"
	note4 "github.com/Chronicle20/atlas-packet/note"
	npc2 "github.com/Chronicle20/atlas-packet/npc"
	party2 "github.com/Chronicle20/atlas-packet/party"
	pet2 "github.com/Chronicle20/atlas-packet/pet"
	portal2 "github.com/Chronicle20/atlas-packet/portal"
	quest2 "github.com/Chronicle20/atlas-packet/quest"
	reactor2 "github.com/Chronicle20/atlas-packet/reactor"
	socket3 "github.com/Chronicle20/atlas-packet/socket"
	stat2 "github.com/Chronicle20/atlas-packet/stat"
	storage2 "github.com/Chronicle20/atlas-packet/storage"
	ui2 "github.com/Chronicle20/atlas-packet/ui"
	"github.com/Chronicle20/atlas-service"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-opcodes"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
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
		field2.SetFieldWriter,
		npc2.NpcSpawnWriter,
		npc2.NpcSpawnRequestControllerWriter,
		npc2.NpcActionWriter,
		stat2.StatChangedWriter,
		channel4.ChannelChangeWriter,
		cash2.CashShopOpenWriter,
		cash2.CashShopOperationWriter,
		cash2.CashQueryResultWriter,
		monster2.MonsterSpawnWriter,
		monster2.MonsterDestroyWriter,
		monster2.MonsterControlWriter,
		monster2.MonsterMovementWriter,
		monster2.MonsterMovementAckWriter,
		character2.CharacterSpawnWriter,
		chat2.GeneralChatWriter,
		character2.CharacterMovementWriter,
		character2.CharacterInfoWriter,
		inventory2.InventoryChangeWriter,
		character2.CharacterAppearanceUpdateWriter,
		character2.CharacterDespawnWriter,
		party2.PartyOperationWriter,
		chat2.MultiChatWriter,
		character2.CharacterKeyMapWriter,
		buddy2.BuddyOperationWriter,
		character2.CharacterExpressionWriter,
		npc2.NpcConversationWriter,
		guild2.GuildOperationWriter,
		guild2.GuildEmblemChangedWriter,
		guild2.GuildNameChangedWriter,
		fame2.FameResponseWriter,
		character2.CharacterStatusMessageWriter,
		guild2.GuildBBSWriter,
		character2.CharacterShowChairWriter,
		character2.CharacterSitResultWriter,
		drop2.DropSpawnWriter,
		drop2.DropDestroyWriter,
		reactor2.ReactorSpawnWriter,
		reactor2.ReactorDestroyWriter,
		character2.CharacterSkillChangeWriter,
		character2.CharacterAttackMeleeWriter,
		character2.CharacterAttackRangedWriter,
		character2.CharacterAttackMagicWriter,
		character2.CharacterAttackEnergyWriter,
		character2.CharacterDamageWriter,
		character2.CharacterBuffGiveWriter,
		character2.CharacterBuffGiveForeignWriter,
		character2.CharacterBuffCancelWriter,
		character2.CharacterBuffCancelForeignWriter,
		character2.CharacterSkillCooldownWriter,
		character2.CharacterEffectWriter,
		character2.CharacterEffectForeignWriter,
		chat2.WorldMessageWriter,
		monster2.MonsterHealthWriter,
		party2.PartyMemberHPWriter,
		character2.ChalkboardUseWriter,
		chat2.WhisperWriter,
		messenger2.MessengerOperationWriter,
		pet2.PetActivatedWriter,
		pet2.PetMovementWriter,
		pet2.PetCommandResponseWriter,
		pet2.PetChatWriter,
		character2.CharacterItemUpgradeWriter,
		character2.CharacterSkillMacroWriter,
		pet2.PetExcludeResponseWriter,
		pet2.PetCashFoodResultWriter,
		character2.CharacterKeyMapAutoHpWriter,
		character2.CharacterKeyMapAutoMpWriter,
		npc2.NPCShopWriter,
		npc2.NPCShopOperationWriter,
		inventory2.CompartmentMergeWriter,
		inventory2.CompartmentSortWriter,
		note4.NoteOperationWriter,
		field2.KiteSpawnWriter,
		field2.KiteErrorWriter,
		field2.KiteDestroyWriter,
		field2.ClockWriter,
		field2.FieldTransportStateWriter,
		storage2.StorageOperationWriter,
		character2.CharacterHintWriter,
		reactor2.ReactorHitWriter,
		npc2.GuideTalkWriter,
		quest2.ScriptProgressWriter,
		socket3.PingWriter,
		field2.FieldEffectWriter,
		ui2.UiOpenWriter,
		ui2.UiLockWriter,
		ui2.UiDisableWriter,
		monster2.MonsterStatSetWriter,
		monster2.MonsterStatResetWriter,
		monster2.MonsterDamageWriter,
		field2.FieldEffectWeatherWriter,
		merchant2.HiredMerchantOperationWriter,
		interaction2.CharacterInteractionWriter,
		interaction2.MiniRoomWriter,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[socket3.CharacterLoggedInHandle] = handler.CharacterLoggedInHandleFunc
	handlerMap[npc2.NPCActionHandle] = handler.NPCActionHandleFunc
	handlerMap[portal2.PortalScriptHandle] = handler.PortalScriptHandleFunc
	handlerMap[field2.MapChangeHandle] = handler.MapChangeHandleFunc
	handlerMap[character2.CharacterMoveHandle] = handler.CharacterMoveHandleFunc
	handlerMap[channel4.ChannelChangeRequestHandle] = handler.ChannelChangeHandleFunc
	handlerMap[cash2.CashShopEntryHandle] = handler.CashShopEntryHandleFunc
	handlerMap[monster2.MonsterMovementHandle] = handler.MonsterMovementHandleFunc
	handlerMap[chat2.CharacterChatGeneralHandle] = handler.CharacterChatGeneralHandleFunc
	handlerMap[character2.CharacterInfoRequestHandle] = handler.CharacterInfoRequestHandleFunc
	handlerMap[inventory2.CharacterInventoryMoveHandle] = handler.CharacterInventoryMoveHandleFunc
	handlerMap[party2.PartyOperationHandle] = handler.PartyOperationHandleFunc
	handlerMap[party2.PartyInviteRejectHandle] = handler.PartyInviteRejectHandleFunc
	handlerMap[chat2.CharacterChatMultiHandle] = handler.CharacterChatMultiHandleFunc
	handlerMap[character2.CharacterKeyMapChangeHandle] = handler.CharacterKeyMapChangeHandleFunc
	handlerMap[buddy2.BuddyOperationHandle] = handler.BuddyOperationHandleFunc
	handlerMap[character2.CharacterExpressionHandle] = handler.CharacterExpressionHandleFunc
	handlerMap[npc2.NPCStartConversationHandle] = handler.NPCStartConversationHandleFunc
	handlerMap[npc2.NPCContinueConversationHandle] = handler.NPCContinueConversationHandleFunc
	handlerMap[guild2.GuildOperationHandle] = handler.GuildOperationHandleFunc
	handlerMap[guild2.GuildInviteRejectHandle] = handler.GuildInviteRejectHandleFunc
	handlerMap[fame2.FameChangeHandle] = handler.FameChangeHandleFunc
	handlerMap[character2.CharacterDistributeApHandle] = handler.CharacterDistributeApHandleFunc
	handlerMap[character2.CharacterAutoDistributeApHandle] = handler.CharacterAutoDistributeApHandleFunc
	handlerMap[guild2.GuildBBSHandle] = handler.GuildBBSHandleFunc
	handlerMap[character2.CharacterChairPortableHandle] = handler.CharacterChairPortableHandleFunc
	handlerMap[character2.CharacterChairInteractionHandle] = handler.CharacterChairFixedHandleFunc
	handlerMap[drop2.DropPickUpHandle] = handler.DropPickUpHandleFunc
	handlerMap[character2.CharacterDropMesoHandle] = handler.CharacterDropMesoHandleFunc
	handlerMap[handler.CharacterMeleeAttackHandle] = handler.CharacterMeleeAttackHandleFunc
	handlerMap[handler.CharacterRangedAttackHandle] = handler.CharacterRangedAttackHandleFunc
	handlerMap[handler.CharacterMagicAttackHandle] = handler.CharacterMagicAttackHandleFunc
	handlerMap[handler.CharacterTouchAttackHandle] = handler.CharacterTouchAttackHandleFunc
	handlerMap[character2.CharacterHealOverTimeHandle] = handler.CharacterHealOverTimeHandleFunc
	handlerMap[packetmodel.CharacterDamageHandle] = handler.CharacterDamageHandleFunc
	handlerMap[character2.CharacterDistributeSpHandle] = handler.CharacterDistributeSpHandleFunc
	handlerMap[handler.CharacterUseSkillHandle] = handler.CharacterUseSkillHandleFunc
	handlerMap[character2.CharacterBuffCancelHandle] = handler.CharacterBuffCancelHandleFunc
	handlerMap[cash2.CharacterCashItemUseHandle] = handler.CharacterCashItemUseHandleFunc
	handlerMap[character2.ChalkboardCloseHandle] = handler.ChalkboardCloseHandleHandleFunc
	handlerMap[chat2.CharacterChatWhisperHandle] = handler.CharacterChatWhisperHandleFunc
	handlerMap[messenger2.MessengerOperationHandle] = handler.MessengerOperationHandleFunc
	handlerMap[pet2.PetMovementHandle] = handler.PetMovementHandleFunc
	handlerMap[pet2.PetSpawnHandle] = handler.PetSpawnHandleFunc
	handlerMap[pet2.PetCommandHandle] = handler.PetCommandHandleFunc
	handlerMap[pet2.PetChatHandle] = handler.PetChatHandleFunc
	handlerMap[pet2.PetDropPickUpHandle] = handler.PetDropPickUpHandleFunc
	handlerMap[pet2.PetFoodHandle] = handler.PetFoodHandleFunc
	handlerMap[inventory2.CharacterItemUseHandle] = handler.CharacterItemUseHandleFunc
	handlerMap[character2.CharacterItemCancelHandle] = handler.CharacterItemCancelHandleFunc
	handlerMap[inventory2.CharacterItemUseTownScrollHandle] = handler.CharacterItemUseTownScrollHandleFunc
	handlerMap[inventory2.CharacterItemUseScrollHandle] = handler.CharacterItemUseScrollHandleFunc
	handlerMap[character2.CharacterSkillMacroHandle] = handler.CharacterSkillMacroHandleFunc
	handlerMap[pet2.PetItemExcludeHandle] = handler.PetItemExcludeHandleFunc
	handlerMap[pet2.PetItemUseHandle] = handler.PetItemUseHandleFunc
	handlerMap[cash2.CashShopOperationHandle] = handler.CashShopOperationHandleFunc
	handlerMap[cash2.CashShopCheckWalletHandle] = handler.CashShopCheckWalletHandleFunc
	handlerMap[npc2.NPCShopHandle] = handler.NPCShopHandleFunc
	handlerMap[inventory2.CompartmentMergeRequestHandle] = handler.CompartmentMergeHandleFunc
	handlerMap[inventory2.CompartmentSortRequestHandle] = handler.CompartmentSortHandleFunc
	handlerMap[inventory2.CharacterItemUseSummonBagHandle] = handler.CharacterItemUseSummonBagHandleFunc
	handlerMap[note4.NoteOperationHandle] = handler.NoteOperationHandleFunc
	handlerMap[quest2.QuestActionHandle] = handler.QuestActionHandleFunc
	handlerMap[storage2.StorageOperationHandle] = handler.StorageOperationHandleFunc
	handlerMap[reactor2.ReactorHitHandle] = handler.ReactorHitHandleFunc
	handlerMap[socket3.PongHandle] = handler.PongHandleFunc
	handlerMap[character2.MonsterDamageFriendlyHandle] = handler.MonsterDamageFriendlyHandleFunc
	handlerMap[interaction2.CharacterInteractionHandle] = handler.CharacterInteractionHandleFunc
	handlerMap[merchant2.HiredMerchantOperationHandle] = handler.HiredMerchantOperationHandleFunc
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
