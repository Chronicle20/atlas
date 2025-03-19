package inventory

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Unequip(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
		return func(characterId uint32, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(unequipItemCommandProvider(characterId, source, destination))
		}
	}
}

func Equip(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, source int16, destination int16) error {
		return func(characterId uint32, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(equipItemCommandProvider(characterId, source, destination))
		}
	}
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(moveItemCommandProvider(characterId, inventoryType, source, destination))
		}
	}
}

func Drop(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) error {
		return func(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(dropItemCommandProvider(m, characterId, inventoryType, source, quantity))
		}
	}
}
