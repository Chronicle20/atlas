package character

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/script"
	consumer2 "atlas-npc-conversations/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMesoChanged)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventErrorNotEnoughMeso)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError)))
	}
}

func handleStatusEventErrorNotEnoughMeso(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventErrorBody[notEnoughMesoErrorStatusBody]]) {
	if e.Type == StatusEventTypeError && e.Body.Error == StatusEventErrorTypeNotEnoughMeso {
		_ = conversation.ContinueViaEvent(l)(ctx)(e.CharacterId, script.ModeCharacterMesoGainError, e.Body.Body.Amount)
	}
}

func handleStatusEventError(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventErrorBody[any]]) {
	if e.Type == StatusEventTypeError && e.Body.Error != StatusEventErrorTypeNotEnoughMeso {
		_ = conversation.ContinueViaEvent(l)(ctx)(e.CharacterId, script.ModeCharacterError, 0)
	}
}

func handleStatusEventMesoChanged(l logrus.FieldLogger, ctx context.Context, e statusEvent[mesoChangedStatusEventBody]) {
	if e.Type != StatusEventTypeMesoChanged {
		return
	}
	_ = conversation.ContinueViaEvent(l)(ctx)(e.CharacterId, script.ModeCharacterMesoGained, e.Body.Amount)
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventLogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	_ = conversation.End(l)(ctx)(e.CharacterId)
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventChannelChangedBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	_ = conversation.End(l)(ctx)(e.CharacterId)
}
