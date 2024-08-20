package inventory

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Unequip(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) error {
	return func(characterId uint32, source int16, destination int16) error {
		return producer.ProviderImpl(l)(span)(EnvCommandTopicUnequipItem)(unequipItemCommandProvider(tenant, characterId, source, destination))
	}
}

func Equip(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) error {
	return func(characterId uint32, source int16, destination int16) error {
		return producer.ProviderImpl(l)(span)(EnvCommandTopicEquipItem)(equipItemCommandProvider(tenant, characterId, source, destination))
	}
}

func Move(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		return producer.ProviderImpl(l)(span)(EnvCommandTopicMoveItem)(moveItemCommandProvider(tenant, characterId, inventoryType, source, destination))
	}
}

func Drop(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
		return producer.ProviderImpl(l)(span)(EnvCommandTopicDropItem)(dropItemCommandProvider(tenant, characterId, inventoryType, source, quantity))
	}
}
