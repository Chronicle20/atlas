package inventory

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Unequip(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, source int16, destination int16) error {
	return func(characterId uint32, source int16, destination int16) error {
		return producer.ProviderImpl(l)(ctx)(EnvCommandTopicUnequipItem)(unequipItemCommandProvider(tenant, characterId, source, destination))
	}
}

func Equip(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, source int16, destination int16) error {
	return func(characterId uint32, source int16, destination int16) error {
		return producer.ProviderImpl(l)(ctx)(EnvCommandTopicEquipItem)(equipItemCommandProvider(tenant, characterId, source, destination))
	}
}

func Move(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		return producer.ProviderImpl(l)(ctx)(EnvCommandTopicMoveItem)(moveItemCommandProvider(tenant, characterId, inventoryType, source, destination))
	}
}

func Drop(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
		return producer.ProviderImpl(l)(ctx)(EnvCommandTopicDropItem)(dropItemCommandProvider(tenant, characterId, inventoryType, source, quantity))
	}
}
