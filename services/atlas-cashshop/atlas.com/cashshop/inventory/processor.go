package inventory

import (
	inventory3 "atlas-cashshop/character/inventory"
	"atlas-cashshop/kafka/message/inventory"
	"atlas-cashshop/kafka/producer"
	inventory2 "atlas-cashshop/kafka/producer/inventory"
	"context"
	"github.com/sirupsen/logrus"
)

func IncreaseCapacity(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
		return func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
			return producer.ProviderImpl(l)(ctx)(inventory.EnvCommandTopic)(inventory2.IncreaseCapacityCommandProvider(characterId, byte(inventoryType), amount))
		}
	}
}
