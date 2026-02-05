package portal_actions

import (
	"atlas-portals/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	producer2 "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func EnterCommandProvider(f field.Model, portalId uint32, characterId uint32, portalName string) model.Provider[[]kafka.Message] {
	key := producer2.CreateKey(int(characterId))
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
	return producer2.SingleMessageProvider(key, value)
}

func ExecuteScript(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32, portalName string) {
	return func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32, portalName string) {
		return func(f field.Model, portalId uint32, characterId uint32, portalName string) {
			_ = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(EnterCommandProvider(f, portalId, characterId, portalName))
		}
	}
}
