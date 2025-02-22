package main

import (
	"atlas-channel/account"
	"atlas-channel/configuration"
	handler2 "atlas-channel/configuration/tenant/socket/handler"
	writer2 "atlas-channel/configuration/tenant/socket/writer"
	account2 "atlas-channel/kafka/consumer/account"
	"atlas-channel/kafka/consumer/buddylist"
	"atlas-channel/kafka/consumer/buff"
	"atlas-channel/kafka/consumer/chair"
	"atlas-channel/kafka/consumer/channel"
	"atlas-channel/kafka/consumer/character"
	"atlas-channel/kafka/consumer/drop"
	"atlas-channel/kafka/consumer/expression"
	"atlas-channel/kafka/consumer/fame"
	"atlas-channel/kafka/consumer/guild"
	"atlas-channel/kafka/consumer/guild/thread"
	"atlas-channel/kafka/consumer/inventory"
	"atlas-channel/kafka/consumer/invite"
	"atlas-channel/kafka/consumer/map"
	"atlas-channel/kafka/consumer/message"
	"atlas-channel/kafka/consumer/monster"
	"atlas-channel/kafka/consumer/npc/conversation"
	"atlas-channel/kafka/consumer/party"
	"atlas-channel/kafka/consumer/party/member"
	"atlas-channel/kafka/consumer/reactor"
	session2 "atlas-channel/kafka/consumer/session"
	"atlas-channel/kafka/consumer/skill"
	"atlas-channel/logger"
	"atlas-channel/server"
	"atlas-channel/service"
	"atlas-channel/session"
	"atlas-channel/socket"
	"atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"atlas-channel/tasks"
	"atlas-channel/tracing"
	"context"
	"fmt"
	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	socket2 "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"os"
	"strconv"
	"time"
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
	buddylist.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	channel.InitConsumers(l)(cmf)(consumerGroupId)
	conversation.InitConsumers(l)(cmf)(consumerGroupId)
	expression.InitConsumers(l)(cmf)(consumerGroupId)
	guild.InitConsumers(l)(cmf)(consumerGroupId)
	inventory.InitConsumers(l)(cmf)(consumerGroupId)
	invite.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	member.InitConsumers(l)(cmf)(consumerGroupId)
	message.InitConsumers(l)(cmf)(consumerGroupId)
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	party.InitConsumers(l)(cmf)(consumerGroupId)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	fame.InitConsumers(l)(cmf)(consumerGroupId)
	thread.InitConsumers(l)(cmf)(consumerGroupId)
	chair.InitConsumers(l)(cmf)(consumerGroupId)
	drop.InitConsumers(l)(cmf)(consumerGroupId)
	reactor.InitConsumers(l)(cmf)(consumerGroupId)
	skill.InitConsumers(l)(cmf)(consumerGroupId)
	buff.InitConsumers(l)(cmf)(consumerGroupId)

	sctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")

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

		err = account.InitializeRegistry(l)(tctx)
		if err != nil {
			l.WithError(err).Errorf("Unable to initialize account registry for tenant [%s].", t.String())
		}

		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}

		for _, w := range ten.Worlds {
			for _, c := range w.Channels {
				var sc server.Model
				sc, err = server.New(t, world.Id(w.Id), channel2.Id(c.Id))
				if err != nil {
					continue
				}

				fl := l.
					WithField("tenant", t.Id().String()).
					WithField("region", t.Region()).
					WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())).
					WithField("world.id", sc.WorldId()).
					WithField("channel.id", sc.ChannelId())

				wp := produceWriterProducer(fl)(tenantConfig.Socket.Writers, writerList, rw)
				account2.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				buddylist.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				channel.InitHandlers(fl)(sc)(ten.IPAddress, c.Port)(consumer.GetManager().RegisterHandler)
				character.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				expression.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				guild.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				inventory.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				invite.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				_map.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				message.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				monster.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				conversation.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				member.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				party.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				session2.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				fame.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				thread.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				chair.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				drop.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				reactor.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				skill.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)
				buff.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler)

				hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantConfig.Socket.Handlers, validatorMap, handlerMap)
				socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, sc, ten.IPAddress, c.Port)
			}
		}
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
		writer.SetField,
		writer.SpawnNPC,
		writer.SpawnNPCRequestController,
		writer.NPCAction,
		writer.StatChanged,
		writer.ChannelChange,
		writer.CashShopOpen,
		writer.CashShopOperation,
		writer.CashShopCashQueryResult,
		writer.SpawnMonster,
		writer.DestroyMonster,
		writer.ControlMonster,
		writer.MoveMonster,
		writer.MoveMonsterAck,
		writer.CharacterSpawn,
		writer.CharacterGeneralChat,
		writer.CharacterMovement,
		writer.CharacterInfo,
		writer.CharacterInventoryChange,
		writer.CharacterAppearanceUpdate,
		writer.CharacterDespawn,
		writer.PartyOperation,
		writer.CharacterMultiChat,
		writer.CharacterKeyMap,
		writer.BuddyOperation,
		writer.CharacterExpression,
		writer.NPCConversation,
		writer.GuildOperation,
		writer.GuildEmblemChanged,
		writer.GuildNameChanged,
		writer.FameResponse,
		writer.CharacterStatusMessage,
		writer.GuildBBS,
		writer.CharacterShowChair,
		writer.CharacterSitResult,
		writer.DropSpawn,
		writer.DropDestroy,
		writer.ReactorSpawn,
		writer.ReactorDestroy,
		writer.CharacterSkillChange,
		writer.CharacterAttackMelee,
		writer.CharacterAttackRanged,
		writer.CharacterAttackMagic,
		writer.CharacterAttackEnergy,
		writer.CharacterDamage,
		writer.CharacterBuffGive,
		writer.CharacterBuffGiveForeign,
		writer.CharacterBuffCancel,
		writer.CharacterBuffCancelForeign,
		writer.CharacterSkillCooldown,
		writer.CharacterEffect,
		writer.CharacterEffectForeign,
	}
}

func produceHandlers() map[string]handler.MessageHandler {
	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[handler.CharacterLoggedInHandle] = handler.CharacterLoggedInHandleFunc
	handlerMap[handler.NPCActionHandle] = handler.NPCActionHandleFunc
	handlerMap[handler.PortalScriptHandle] = handler.PortalScriptHandleFunc
	handlerMap[handler.MapChangeHandle] = handler.MapChangeHandleFunc
	handlerMap[handler.CharacterMoveHandle] = handler.CharacterMoveHandleFunc
	handlerMap[handler.ChannelChangeHandle] = handler.ChannelChangeHandleFunc
	handlerMap[handler.CashShopEntryHandle] = handler.CashShopEntryHandleFunc
	handlerMap[handler.MonsterMovementHandle] = handler.MonsterMovementHandleFunc
	handlerMap[handler.CharacterGeneralChatHandle] = handler.CharacterGeneralChatHandleFunc
	handlerMap[handler.CharacterInfoRequestHandle] = handler.CharacterInfoRequestHandleFunc
	handlerMap[handler.CharacterInventoryMoveHandle] = handler.CharacterInventoryMoveHandleFunc
	handlerMap[handler.PartyOperationHandle] = handler.PartyOperationHandleFunc
	handlerMap[handler.PartyInviteRejectHandle] = handler.PartyInviteRejectHandleFunc
	handlerMap[handler.CharacterMultiChatHandle] = handler.CharacterMultiChatHandleFunc
	handlerMap[handler.CharacterKeyMapChangeHandle] = handler.CharacterKeyMapChangeHandleFunc
	handlerMap[handler.BuddyOperationHandle] = handler.BuddyOperationHandleFunc
	handlerMap[handler.CharacterExpressionHandle] = handler.CharacterExpressionHandleFunc
	handlerMap[handler.NPCStartConversationHandle] = handler.NPCStartConversationHandleFunc
	handlerMap[handler.NPCContinueConversationHandle] = handler.NPCContinueConversationHandleFunc
	handlerMap[handler.GuildOperationHandle] = handler.GuildOperationHandleFunc
	handlerMap[handler.GuildInviteRejectHandle] = handler.GuildInviteRejectHandleFunc
	handlerMap[handler.FameChangeHandle] = handler.FameChangeHandleFunc
	handlerMap[handler.CharacterDistributeApHandle] = handler.CharacterDistributeApHandleFunc
	handlerMap[handler.CharacterAutoDistributeApHandle] = handler.CharacterAutoDistributeApHandleFunc
	handlerMap[handler.GuildBBSHandle] = handler.GuildBBSHandleFunc
	handlerMap[handler.CharacterChairPortableHandle] = handler.CharacterChairPortableHandleFunc
	handlerMap[handler.CharacterChairInteractionHandle] = handler.CharacterChairFixedHandleFunc
	handlerMap[handler.DropPickUpHandle] = handler.DropPickUpHandleFunc
	handlerMap[handler.CharacterDropMesoHandle] = handler.CharacterDropMesoHandleFunc
	handlerMap[handler.CharacterMeleeAttackHandle] = handler.CharacterMeleeAttackHandleFunc
	handlerMap[handler.CharacterRangedAttackHandle] = handler.CharacterRangedAttackHandleFunc
	handlerMap[handler.CharacterMagicAttackHandle] = handler.CharacterMagicAttackHandleFunc
	handlerMap[handler.CharacterTouchAttackHandle] = handler.CharacterTouchAttackHandleFunc
	handlerMap[handler.CharacterHealOverTimeHandle] = handler.CharacterHealOverTimeHandleFunc
	handlerMap[handler.CharacterDamageHandle] = handler.CharacterDamageHandleFunc
	handlerMap[handler.CharacterDistributeSpHandle] = handler.CharacterDistributeSpHandleFunc
	handlerMap[handler.CharacterUseSkillHandle] = handler.CharacterUseSkillHandleFunc
	handlerMap[handler.CharacterBuffCancelHandle] = handler.CharacterBuffCancelHandleFunc
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
					rwm[wc.Writer] = writer.MessageGetter(w.Write(uint16(op)), wc.Options)
				}
			}
		}
		return writer.ProducerGetter(rwm)
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
