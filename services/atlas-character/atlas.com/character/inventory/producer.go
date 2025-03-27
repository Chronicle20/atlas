package inventory

import (
	"atlas-character/equipable"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[equipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          CommandEquip,
		Body: equipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[unequipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          CommandUnequip,
		Body: unequipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

type ItemAddProvider func(quantity uint32, slot int16) model.Provider[[]kafka.Message]

func inventoryItemAddProvider(characterId uint32) func(inventoryType inventory.Type) func(itemId uint32) ItemAddProvider {
	return func(inventoryType inventory.Type) func(itemId uint32) ItemAddProvider {
		return func(itemId uint32) ItemAddProvider {
			return func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
				key := producer.CreateKey(int(characterId))
				value := &inventoryChangedEvent[inventoryChangedItemAddBody]{
					CharacterId:   characterId,
					InventoryType: int8(inventoryType),
					Slot:          slot,
					Type:          ChangedTypeAdd,
					Body: inventoryChangedItemAddBody{
						ItemId:   itemId,
						Quantity: quantity,
					},
				}
				return producer.SingleMessageProvider(key, value)
			}
		}
	}
}

type ItemUpdateProvider func(quantity uint32, slot int16) model.Provider[[]kafka.Message]

func inventoryItemQuantityUpdateProvider(characterId uint32) func(inventoryType inventory.Type) func(itemId uint32) ItemUpdateProvider {
	return func(inventoryType inventory.Type) func(itemId uint32) ItemUpdateProvider {
		return func(itemId uint32) ItemUpdateProvider {
			return func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
				key := producer.CreateKey(int(characterId))
				value := &inventoryChangedEvent[inventoryChangedItemQuantityUpdateBody]{
					CharacterId:   characterId,
					InventoryType: int8(inventoryType),
					Slot:          slot,
					Type:          ChangedTypeUpdateQuantity,
					Body: inventoryChangedItemQuantityUpdateBody{
						ItemId:   itemId,
						Quantity: quantity,
					},
				}
				return producer.SingleMessageProvider(key, value)
			}
		}
	}
}

func inventoryItemAttributeUpdateProvider(characterId uint32) func(e equipable.Model) model.Provider[[]kafka.Message] {
	return func(e equipable.Model) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(characterId))
		value := &inventoryChangedEvent[inventoryChangedItemAttributeUpdateBody]{
			CharacterId:   characterId,
			InventoryType: int8(inventory.TypeValueEquip),
			Slot:          e.Slot(),
			Type:          ChangedTypeUpdateAttribute,
			Body: inventoryChangedItemAttributeUpdateBody{
				ItemId:        e.ItemId(),
				Strength:      e.Strength(),
				Dexterity:     e.Dexterity(),
				Intelligence:  e.Intelligence(),
				Luck:          e.Luck(),
				HP:            e.HP(),
				MP:            e.MP(),
				WeaponAttack:  e.WeaponAttack(),
				MagicAttack:   e.MagicAttack(),
				WeaponDefense: e.WeaponDefense(),
				MagicDefense:  e.MagicDefense(),
				Accuracy:      e.Accuracy(),
				Avoidability:  e.Avoidability(),
				Hands:         e.Hands(),
				Speed:         e.Speed(),
				Jump:          e.Jump(),
				Slots:         e.Slots(),
			},
		}
		return producer.SingleMessageProvider(key, value)
	}
}

func noOpInventoryItemMoveProvider(_ uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(_ int16) model.Provider[[]kafka.Message] {
		return func() ([]kafka.Message, error) {
			return nil, nil
		}
	}
}

func inventoryItemMoveProvider(characterId uint32) func(inventoryType inventory.Type) func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
	return func(inventoryType inventory.Type) func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
		return func(oldSlot int16) func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
			return func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message] {
				return func(slot int16) model.Provider[[]kafka.Message] {
					key := producer.CreateKey(int(characterId))
					value := &inventoryChangedEvent[inventoryChangedItemMoveBody]{
						CharacterId:   characterId,
						InventoryType: int8(inventoryType),
						Slot:          slot,
						Type:          ChangedTypeMove,
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

func inventoryItemRemoveProvider(characterId uint32) func(inventoryType inventory.Type) func(itemId uint32, slot int16) model.Provider[[]kafka.Message] {
	return func(inventoryType inventory.Type) func(itemId uint32, slot int16) model.Provider[[]kafka.Message] {
		return func(itemId uint32, slot int16) model.Provider[[]kafka.Message] {
			key := producer.CreateKey(int(characterId))
			value := &inventoryChangedEvent[inventoryChangedItemRemoveBody]{
				CharacterId:   characterId,
				InventoryType: int8(inventoryType),
				Slot:          slot,
				Type:          ChangedTypeRemove,
				Body: inventoryChangedItemRemoveBody{
					ItemId: itemId,
				},
			}
			return producer.SingleMessageProvider(key, value)
		}
	}
}

func inventoryItemReserveProvider(characterId uint32) func(inventoryType inventory.Type) func(itemId uint32) func(quantity uint32, slot int16, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	return func(inventoryType inventory.Type) func(itemId uint32) func(quantity uint32, slot int16, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
		return func(itemId uint32) func(quantity uint32, slot int16, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
			return func(quantity uint32, slot int16, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
				key := producer.CreateKey(int(characterId))
				value := &inventoryChangedEvent[inventoryChangedItemReserveBody]{
					CharacterId:   characterId,
					InventoryType: int8(inventoryType),
					Slot:          slot,
					Type:          ChangedTypeReserve,
					Body: inventoryChangedItemReserveBody{
						TransactionId: transactionId,
						ItemId:        itemId,
						Quantity:      quantity,
					},
				}
				return producer.SingleMessageProvider(key, value)
			}
		}
	}
}

func inventoryItemCancelReservationProvider(characterId uint32) func(inventoryType inventory.Type) func(itemId uint32) func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
	return func(inventoryType inventory.Type) func(itemId uint32) func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
		return func(itemId uint32) func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
			return func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
				key := producer.CreateKey(int(characterId))
				value := &inventoryChangedEvent[inventoryChangedItemReservationCancelledBody]{
					CharacterId:   characterId,
					InventoryType: int8(inventoryType),
					Slot:          slot,
					Type:          ChangedTypeReservationCancelled,
					Body: inventoryChangedItemReservationCancelledBody{
						ItemId:   itemId,
						Quantity: quantity,
					},
				}
				return producer.SingleMessageProvider(key, value)
			}
		}
	}
}
