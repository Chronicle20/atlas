package inventory

import (
	"atlas-messages/character/inventory/item"
	"atlas-messages/equipment/statistics"
	"context"
	"github.com/sirupsen/logrus"
	"math"
)

func Exists(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) bool {
	return func(ctx context.Context) func(itemId uint32) bool {
		return func(itemId uint32) bool {
			inventoryType := byte(math.Floor(float64(itemId) / 1000000))
			if inventoryType == 1 {
				_, err := statistics.GetById(l)(ctx)(itemId)
				if err != nil {
					return false
				}
				return true
			}

			return true
		}
	}
}

func CreateItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, quantity uint16) (item.Model, error) {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, quantity uint16) (item.Model, error) {
		return func(characterId uint32, itemId uint32, quantity uint16) (item.Model, error) {
			rm, err := requestCreateItem(characterId, itemId, quantity)(l, ctx)
			if err != nil {
				return item.Model{}, err
			}
			return item.Extract(rm)
		}
	}
}
