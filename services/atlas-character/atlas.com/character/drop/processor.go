package drop

import (
	"atlas-character/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func CreateForEquipment(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(dropEquipmentProvider(worldId, channelId, mapId, itemId, equipmentId, dropType, x, y, ownerId))
		}
	}
}

func CreateForItem(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(dropItemProvider(worldId, channelId, mapId, itemId, quantity, dropType, x, y, ownerId))
		}
	}
}

func CreateForMesos(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(dropMesoProvider(worldId, channelId, mapId, mesos, dropType, x, y, ownerId))
		}
	}
}

func RequestPickUp(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestPickUpCommandProvider(worldId, channelId, mapId, dropId, characterId))
		}
	}
}

func CancelReservation(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cancelReservationCommandProvider(worldId, channelId, mapId, dropId, characterId))
		}
	}
}
