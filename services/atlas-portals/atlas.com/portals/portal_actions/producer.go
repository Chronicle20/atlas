package portal_actions

import (
	"atlas-portals/kafka/producer"
	"context"

	producer2 "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func EnterCommandProvider(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32, portalName string) model.Provider[[]kafka.Message] {
	key := producer2.CreateKey(int(characterId))
	value := &commandEvent[enterBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		PortalId:  portalId,
		Type:      CommandTypeEnter,
		Body: enterBody{
			CharacterId: characterId,
			PortalName:  portalName,
		},
	}
	return producer2.SingleMessageProvider(key, value)
}

func ExecuteScript(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32, portalName string) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32, portalName string) {
		return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32, portalName string) {
			_ = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(EnterCommandProvider(worldId, channelId, mapId, portalId, characterId, portalName))
		}
	}
}
