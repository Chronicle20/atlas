package main

import (
	"atlas-channel/account"
	"atlas-channel/channel"
	"atlas-channel/configuration"
	"atlas-channel/kafka/consumer/buddylist"
	"atlas-channel/kafka/consumer/character"
	"atlas-channel/kafka/consumer/inventory"
	"atlas-channel/kafka/consumer/invite"
	"atlas-channel/kafka/consumer/map"
	"atlas-channel/kafka/consumer/monster"
	"atlas-channel/kafka/consumer/party"
	"atlas-channel/kafka/consumer/party/member"
	"atlas-channel/logger"
	"atlas-channel/message"
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

const serviceName = "atlas-channel"
const consumerGroupId = "Channel Service - %s"

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
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(_map.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.MovementEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(channel.CommandStatusConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(monster.MovementEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(monster.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(account.StatusConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(message.ChatEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(inventory.ChangedConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(party.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(member.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(invite.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(buddylist.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))

	sctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")

	for _, s := range config.Data.Attributes.Servers {
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

		var t tenant.Model
		t, err = tenant.Register(uuid.MustParse(s.Tenant), s.Region, uint16(majorVersion), uint16(minorVersion))
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

		for _, w := range s.Worlds {
			for _, c := range w.Channels {
				var sc server.Model
				sc, err = server.New(t, w.Id, c.Id)
				if err != nil {
					continue
				}

				fl := l.
					WithField("tenant", t.Id().String()).
					WithField("region", t.Region()).
					WithField("ms.version", fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())).
					WithField("world.id", sc.WorldId()).
					WithField("channel.id", sc.ChannelId())

				wp := produceWriterProducer(fl)(s.Writers, writerList, rw)
				_, _ = cm.RegisterHandler(_map.StatusEventCharacterEnterRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(_map.StatusEventCharacterExitRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(character.StatusEventStatChangedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(character.StatusEventMapChangedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(channel.CommandStatusRegister(sc, config.Data.Attributes.IPAddress, c.Port)(fl))
				_, _ = cm.RegisterHandler(monster.StatusEventCreatedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(monster.StatusEventDestroyedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(monster.StatusEventKilledRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(monster.StatusEventStartControlRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(monster.StatusEventStopControlRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(monster.MovementEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(account.StatusRegister(fl, sc))
				_, _ = cm.RegisterHandler(message.GeneralChatEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(message.MultiChatEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(character.MovementEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(inventory.ChangeEventAddRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(inventory.ChangeEventUpdateRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(inventory.ChangeEventMoveRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(inventory.ChangeEventRemoveRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.CreatedStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.JoinStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.LeftStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.ExpelStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.DisbandStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.ChangeLeaderStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(party.ErrorEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(member.LoginStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(member.LogoutStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(invite.CreatedStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(invite.RejectedStatusEventRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(buddylist.StatusEventBuddyAddedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(buddylist.StatusEventBuddyRemovedRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(buddylist.StatusEventBuddyChannelChangeRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(buddylist.StatusEventBuddyCapacityChangeRegister(sc, wp)(fl))
				_, _ = cm.RegisterHandler(buddylist.StatusEventBuddyErrorRegister(sc, wp)(fl))

				hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(s.Handlers, validatorMap, handlerMap)
				socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, sc, config.Data.Attributes.IPAddress, c.Port)
			}
		}
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
