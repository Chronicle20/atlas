package message

import (
	"atlas-messages/kafka/producer"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GeneralChat(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return producer.ProviderImpl(l)(span)(EnvEventTopicGeneralChat)(generalChatEventProvider(tenant)(worldId, channelId, mapId, characterId, message, balloonOnly))
	}
}
