package inventory

import (
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
