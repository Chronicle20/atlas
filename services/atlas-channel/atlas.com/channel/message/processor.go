package message

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func GeneralChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicGeneralChat)(generalChatCommandProvider(worldId, channelId, mapId, characterId, message, balloonOnly))
		}
	}
}
