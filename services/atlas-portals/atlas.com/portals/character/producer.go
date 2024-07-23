package character

import (
	"atlas-portals/kafka"
	"atlas-portals/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitEnableActions(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(worldId byte, channelId byte, characterId uint32) {
		event := &statusEvent[statusEventStatChangedBody]{
			Tenant:      tenant,
			CharacterId: characterId,
			Type:        EventCharacterStatusTypeStatChanged,
			WorldId:     worldId,
			Body: statusEventStatChangedBody{
				ChannelId:       channelId,
				ExclRequestSent: true,
			},
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}
