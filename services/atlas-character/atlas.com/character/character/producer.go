package character

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitCreatedEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, name string) {
	p := producer.ProduceEvent(l, span, lookupTopic(l)(EnvEventTopicCharacterCreated))
	return func(characterId uint32, worldId byte, name string) {
		event := &createdEvent{
			Tenant:      tenant,
			CharacterId: characterId,
			WorldId:     worldId,
			Name:        name,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}
