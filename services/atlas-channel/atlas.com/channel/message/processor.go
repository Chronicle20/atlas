package message

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GeneralChat(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return producer.ProviderImpl(l)(span)(EnvCommandTopicGeneralChat)(generalChatCommandProvider(tenant)(worldId, channelId, mapId, characterId, message, balloonOnly))
	}
}
