package inventory

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Unequip(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
		return func(characterId uint32, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicUnequipItem)(unequipItemCommandProvider(characterId, source, destination))
		}
	}
}

func Equip(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
		return func(characterId uint32, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicEquipItem)(equipItemCommandProvider(characterId, source, destination))
		}
	}
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicMoveItem)(moveItemCommandProvider(characterId, inventoryType, source, destination))
		}
	}
}

func Drop(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, inventoryType byte, source int16, quantity int16) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, inventoryType byte, source int16, quantity int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopicDropItem)(dropItemCommandProvider(worldId, channelId, mapId, characterId, inventoryType, source, quantity))
		}
	}
}
