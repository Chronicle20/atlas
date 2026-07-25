package portal_actions

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func EnterCommandProvider(f field.Model, portalId uint32, characterId uint32, portalName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[enterBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		PortalId:  portalId,
		Type:      CommandTypeEnter,
		Body: enterBody{
			CharacterId: characterId,
			PortalName:  portalName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ExecuteScript(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32, portalName string) {
	return func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32, portalName string) {
		return func(f field.Model, portalId uint32, characterId uint32, portalName string) {
			_ = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(EnterCommandProvider(f, portalId, characterId, portalName))
		}
	}
}
