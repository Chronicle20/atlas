package inventory

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitEquipItemCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) {
	p := producer.ProduceEvent(l, span, lookupTopic(l)(EnvCommandTopicEquipItem))
	return func(characterId uint32, source int16, destination int16) {
		event := &equipItemCommand{
			Tenant:      tenant,
			CharacterId: characterId,
			Source:      source,
			Destination: destination,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitUnequipItemCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16) {
	p := producer.ProduceEvent(l, span, lookupTopic(l)(EnvCommandTopicUnequipItem))
	return func(characterId uint32, source int16) {
		event := &unequipItemCommand{
			Tenant:      tenant,
			CharacterId: characterId,
			Source:      source,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}
