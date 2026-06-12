package producer

import (
	"atlas-mounts/kafka/message/mount"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Provider func(token string) producer.MessageProducer

func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
	return func(ctx context.Context) func(token string) producer.MessageProducer {
		sd := producer.SpanHeaderDecorator(ctx)
		td := producer.TenantHeaderDecorator(ctx)
		return func(token string) producer.MessageProducer {
			return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}

func setEventProvider(worldId world.Id, characterId uint32, body mount.StatusEventBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mount.StatusEvent[mount.StatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        mount.StatusEventTypeSet,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}

func tickEventProvider(worldId world.Id, characterId uint32, body mount.StatusEventBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mount.StatusEvent[mount.StatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        mount.StatusEventTypeTick,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}

func feedEventProvider(worldId world.Id, characterId uint32, body mount.StatusEventBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mount.StatusEvent[mount.StatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        mount.StatusEventTypeFeed,
		Body:        body,
	}
	return producer.SingleMessageProvider(key, value)
}
