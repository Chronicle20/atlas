package character

import (
	"atlas-messengers/character"
	consumer2 "atlas-messengers/kafka/consumer"
	messageCharacter "atlas-messengers/kafka/message/character"
	"atlas-messengers/messenger"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("character_status_event")(messageCharacter.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(messageCharacter.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged)))
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e messageCharacter.StatusEvent[messageCharacter.StatusEventLoginBody]) {
	if e.Type != messageCharacter.EventCharacterStatusTypeLogin {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	err := character.Login(l)(ctx)(e.TransactionId, f, e.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process login for character [%d].", e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e messageCharacter.StatusEvent[messageCharacter.StatusEventLogoutBody]) {
	if e.Type != messageCharacter.EventCharacterStatusTypeLogout {
		return
	}
	err := character.Logout(l)(ctx)(e.TransactionId, e.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process logout for character [%d].", e.CharacterId)
	}
	m, err := model.FirstProvider(messenger.AllProvider(ctx), model.Filters(messenger.MemberFilter(e.CharacterId)))()
	if err != nil {
		return
	}
	_, err = messenger.Leave(l)(ctx)(e.TransactionId, m.Id(), e.CharacterId)
	if err != nil {
		return
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e messageCharacter.StatusEvent[messageCharacter.StatusEventChannelChangedBody]) {
	if e.Type != messageCharacter.EventCharacterStatusTypeChannelChanged {
		return
	}
	err := character.ChannelChange(l)(ctx)(e.CharacterId, e.Body.ChannelId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process channel changed for character [%d].", e.CharacterId)
	}
}
