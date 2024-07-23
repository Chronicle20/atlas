package portal

import (
	"atlas-channel/kafka"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitEnterCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvPortalCommandTopic))
	return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
		c := commandEvent[enterBody]{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			PortalId:  portalId,
			Type:      CommandTypeEnter,
			Body: enterBody{
				CharacterId: characterId,
			},
		}
		p(producer.CreateKey(int(portalId)), c)
	}
}
