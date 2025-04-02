package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/character"
	inventory2 "atlas-cashshop/character/inventory"
	"atlas-cashshop/inventory"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/producer"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/wallet"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func PurchaseInventoryIncreaseByItem(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, serialNumber uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, serialNumber uint32) error {
		return func(db *gorm.DB) func(characterId uint32, currency uint32, serialNumber uint32) error {
			return func(characterId uint32, currency uint32, serialNumber uint32) error {
				ci, err := commodity.GetById(l)(ctx)(serialNumber)
				if err != nil {
					return err
				}
				inventoryType := inventory2.Type(ci.ItemId() - 9110000/1000)
				return purchaseInventoryIncrease(l)(ctx)(db)(characterId, currency, inventoryType, ci.Price(), 4)
			}
		}
	}
}

func PurchaseInventoryIncreaseByType(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type) error {
		return func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type) error {
			return func(characterId uint32, currency uint32, inventoryType inventory2.Type) error {
				return purchaseInventoryIncrease(l)(ctx)(db)(characterId, currency, inventoryType, 4000, 8)
			}
		}
	}
}

func purchaseInventoryIncrease(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type, cost uint32, amount uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type, cost uint32, amount uint32) error {
		return func(db *gorm.DB) func(characterId uint32, currency uint32, inventoryType inventory2.Type, cost uint32, amount uint32) error {
			return func(characterId uint32, currency uint32, inventoryType inventory2.Type, cost uint32, amount uint32) error {
				ErrInsufficientFunds := errors.New("insufficient funds")
				ErrMaxSlots := errors.New("max slots")
				newCapacity := uint32(0)

				l.Debugf("Character [%d] attempting to purchase inventory [%d] increase using currency [%d]. Cost is [%d].", characterId, inventoryType, currency, cost)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					w, err := wallet.GetByCharacterId(ctx)(tx)(characterId)
					if err != nil {
						return err
					}
					credit := w.Credit()
					points := w.Points()
					prepaid := w.Prepaid()
					balance := uint32(0)
					if currency == 1 {
						balance = w.Credit()
						credit -= cost
					} else if currency == 2 {
						balance = w.Points()
						points -= cost
					} else {
						balance = w.Prepaid()
						points -= cost
					}
					if balance < cost {
						return ErrInsufficientFunds
					}

					c, err := character.GetByIdWithInventory(l)(ctx)()(characterId)
					if err != nil {
						return err
					}
					slots := uint32(0)
					if inventoryType == inventory2.TypeValueEquip {
						slots = c.Inventory().Equipable().Capacity()
					} else if inventoryType == inventory2.TypeValueUse {
						slots = c.Inventory().Use().Capacity()
					} else if inventoryType == inventory2.TypeValueSetup {
						slots = c.Inventory().Setup().Capacity()
					} else if inventoryType == inventory2.TypeValueETC {
						slots = c.Inventory().Etc().Capacity()
					} else {
						slots = c.Inventory().Cash().Capacity()
					}

					if slots+amount > 96 {
						return ErrMaxSlots
					}
					newCapacity = slots + amount

					_, err = wallet.Update(l)(ctx)(tx)(characterId, credit, points, prepaid)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "", 0x00))
					return txErr
				}

				l.Debugf("Character [%d] purchased inventory [%d] increase. New capacity will be [%d].", characterId, inventoryType, newCapacity)
				_ = inventory.IncreaseCapacity(l)(ctx)(characterId, inventoryType, amount)
				_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.InventoryCapacityIncreasedStatusEventProvider(characterId, byte(inventoryType), newCapacity, amount))
				return nil
			}
		}
	}
}
