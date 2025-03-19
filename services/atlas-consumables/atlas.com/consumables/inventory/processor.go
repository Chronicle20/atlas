package inventory

import (
	inventory2 "atlas-consumables/kafka/message/inventory"
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Reserves struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

func RequestReserve(l logrus.FieldLogger) func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []Reserves) error {
	return func(ctx context.Context) func(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []Reserves) error {
		return func(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []Reserves) error {
			return producer.ProviderImpl(l)(ctx)(inventory2.EnvCommandTopic)(requestReserveCommandProvider(transactionId, characterId, it, reserves))
		}
	}
}

func Consume(f func(l logrus.FieldLogger) func(ctx context.Context) error) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemReserveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemReserveBody]) {
		_ = f(l)(ctx)
	}
}

func ConsumeItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			return producer.ProviderImpl(l)(ctx)(inventory2.EnvCommandTopic)(consumeCommandProvider(characterId, inventoryType, transactionId, slot))
		}
	}
}

func CancelItemReservation(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			return producer.ProviderImpl(l)(ctx)(inventory2.EnvCommandTopic)(cancelReservationCommandProvider(characterId, inventoryType, transactionId, slot))
		}
	}
}
