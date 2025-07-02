package character

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/script"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	character2 "atlas-npc-conversations/kafka/message/character"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMesoChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventErrorNotEnoughMeso)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError)))
	}
}

func handleStatusEventErrorNotEnoughMeso(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventErrorBody[character2.NotEnoughMesoErrorStatusBody]]) {
	if e.Type == character2.StatusEventTypeError && e.Body.Error == character2.StatusEventErrorTypeNotEnoughMeso {
		_ = conversation.NewProcessor(l, ctx).ContinueViaEvent(e.CharacterId, script.ModeCharacterMesoGainError, e.Body.Body.Amount)
	}
}

func handleStatusEventError(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventErrorBody[any]]) {
	if e.Type == character2.StatusEventTypeError && e.Body.Error != character2.StatusEventErrorTypeNotEnoughMeso {
		_ = conversation.NewProcessor(l, ctx).ContinueViaEvent(e.CharacterId, script.ModeCharacterError, 0)
	}
}

func handleStatusEventMesoChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.MesoChangedStatusEventBody]) {
	if e.Type != character2.StatusEventTypeMesoChanged {
		return
	}
	_ = conversation.NewProcessor(l, ctx).ContinueViaEvent(e.CharacterId, script.ModeCharacterMesoGained, e.Body.Amount)
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type != character2.StatusEventTypeLogout {
		return
	}
	_ = conversation.NewProcessor(l, ctx).End(e.CharacterId)
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventChannelChangedBody]) {
	if e.Type != character2.StatusEventTypeChannelChanged {
		return
	}
	_ = conversation.NewProcessor(l, ctx).End(e.CharacterId)
}
