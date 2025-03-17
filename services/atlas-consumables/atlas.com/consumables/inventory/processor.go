package inventory

import (
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func ConsumeItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			err := producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(consumeCommandProvider(characterId, inventoryType, transactionId, slot))
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func CancelItemReservation(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			err := producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cancelReservationCommandProvider(characterId, inventoryType, transactionId, slot))
			if err != nil {
				return err
			}
			return nil
		}
	}
}
