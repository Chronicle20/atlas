package message

import (
	consumer2 "atlas-messages/kafka/consumer"
	message2 "atlas-messages/message"
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
			rf(consumer2.NewConfig(l)("chat_command")(EnvCommandTopicChat)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopicChat)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleGeneralChat))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMultiChat))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleWhisperChat))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMessengerChat))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePetChat))); err != nil {
			return err
		}
		return nil
	}
}

func handleGeneralChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[generalChatBody]) {
	if e.Type != ChatTypeGeneral {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	_ = message2.NewProcessor(l, ctx).HandleGeneral(f, e.ActorId, e.Message, e.Body.BalloonOnly)
}

func handleMultiChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[multiChatBody]) {
	if e.Type != ChatTypeBuddy && e.Type != ChatTypeParty && e.Type != ChatTypeGuild && e.Type != ChatTypeAlliance {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	_ = message2.NewProcessor(l, ctx).HandleMulti(f, e.ActorId, e.Message, e.Type, e.Body.Recipients)
}

func handleWhisperChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[whisperChatBody]) {
	if e.Type != ChatTypeWhisper {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	_ = message2.NewProcessor(l, ctx).HandleWhisper(f, e.ActorId, e.Message, e.Body.RecipientName)
}

func handleMessengerChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[messengerChatBody]) {
	if e.Type != ChatTypeMessenger {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	_ = message2.NewProcessor(l, ctx).HandleMessenger(f, e.ActorId, e.Message, e.Body.Recipients)
}

func handlePetChat(l logrus.FieldLogger, ctx context.Context, e chatCommand[petChatBody]) {
	if e.Type != ChatTypePet {
		return
	}
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	_ = message2.NewProcessor(l, ctx).HandlePet(f, e.ActorId, e.Message, e.Body.OwnerId, e.Body.PetSlot, e.Body.Type, e.Body.Action, e.Body.Balloon)
}
