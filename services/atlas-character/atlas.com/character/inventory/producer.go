package inventory

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(tenant tenant.Model, characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipItemCommand{
		Tenant:      tenant,
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(tenant tenant.Model, characterId uint32, source int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &unequipItemCommand{
		Tenant:      tenant,
		CharacterId: characterId,
		Source:      source,
	}
	return producer.SingleMessageProvider(key, value)
}

func inventoryItemAddProvider(tenant tenant.Model, characterId uint32, itemId uint32, quantity uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventoryChangedEvent[inventoryChangedItemAddBody]{
		Tenant:      tenant,
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

func inventoryItemUpdateProvider(tenant tenant.Model, characterId uint32, itemId uint32, quantity uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventoryChangedEvent[inventoryChangedItemUpdateBody]{
		Tenant:      tenant,
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

func noOpInventoryItemMoveProvider(_ uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(_ int16) model.Provider[[]kafka.Message] {
		return func() ([]kafka.Message, error) {
			return nil, nil
		}
	}
}

func inventoryItemMoveProvider(tenant tenant.Model) func(characterId uint32) func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(characterId uint32) func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
		return func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
			return func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
				return func(slot int16) model.Provider[[]kafka.Message] {
					key := producer.CreateKey(int(characterId))
					value := &inventoryChangedEvent[inventoryChangedItemMoveBody]{
						Tenant:      tenant,
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
}

func inventoryItemRemoveProvider(tenant tenant.Model, characterId uint32, itemId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventoryChangedEvent[inventoryChangedItemRemoveBody]{
		Tenant:      tenant,
		CharacterId: characterId,
		Slot:        slot,
		Type:        ChangedTypeRemove,
		Body: inventoryChangedItemRemoveBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
