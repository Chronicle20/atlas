package inventory

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipItemCommand{
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(characterId uint32, source int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &unequipItemCommand{
		CharacterId: characterId,
		Source:      source,
	}
	return producer.SingleMessageProvider(key, value)
}

type ItemAddProvider func(quantity uint32, slot int16) model.Provider[[]kafka.Message]

func inventoryItemAddProvider(characterId uint32) func(itemId uint32) ItemAddProvider {
	return func(itemId uint32) ItemAddProvider {
		return func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
			key := producer.CreateKey(int(characterId))
			value := &inventoryChangedEvent[inventoryChangedItemAddBody]{
				CharacterId: characterId,
				Slot:        slot,
				Type:        ChangedTypeAdd,
				Body: inventoryChangedItemAddBody{
					ItemId:   itemId,
					Quantity: quantity,
				},
			}
			return producer.SingleMessageProvider(key, value)
		}
	}
}

type ItemUpdateProvider func(quantity uint32, slot int16) model.Provider[[]kafka.Message]

func inventoryItemUpdateProvider(characterId uint32) func(itemId uint32) ItemUpdateProvider {
	return func(itemId uint32) ItemUpdateProvider {
		return func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
			key := producer.CreateKey(int(characterId))
			value := &inventoryChangedEvent[inventoryChangedItemUpdateBody]{
				CharacterId: characterId,
				Slot:        slot,
				Type:        ChangedTypeUpdate,
				Body: inventoryChangedItemUpdateBody{
					ItemId:   itemId,
					Quantity: quantity,
				},
			}
			return producer.SingleMessageProvider(key, value)
		}
	}
}

func noOpInventoryItemMoveProvider(_ uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(_ int16) model.Provider[[]kafka.Message] {
		return func() ([]kafka.Message, error) {
			return nil, nil
		}
	}
}

func inventoryItemMoveProvider(characterId uint32) func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
		return func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
			return func(slot int16) model.Provider[[]kafka.Message] {
				key := producer.CreateKey(int(characterId))
				value := &inventoryChangedEvent[inventoryChangedItemMoveBody]{
					CharacterId: characterId,
					Slot:        slot,
					Type:        ChangedTypeMove,
					Body: inventoryChangedItemMoveBody{
						ItemId:  itemId,
						OldSlot: oldSlot,
					},
				}
				return producer.SingleMessageProvider(key, value)
			}
		}
	}
}

func inventoryItemRemoveProvider(characterId uint32, itemId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventoryChangedEvent[inventoryChangedItemRemoveBody]{
		CharacterId: characterId,
		Slot:        slot,
		Type:        ChangedTypeRemove,
		Body: inventoryChangedItemRemoveBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
