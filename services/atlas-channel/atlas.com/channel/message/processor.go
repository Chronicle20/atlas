package message

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"context"
	"github.com/sirupsen/logrus"
)

func GeneralChat(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return producer.ProviderImpl(l)(ctx)(EnvCommandTopicGeneralChat)(generalChatCommandProvider(tenant)(worldId, channelId, mapId, characterId, message, balloonOnly))
	}
}
