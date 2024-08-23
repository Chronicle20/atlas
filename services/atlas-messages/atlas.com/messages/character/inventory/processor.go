package inventory

import (
	"atlas-messages/equipment/statistics"
	"atlas-messages/tenant"
	"context"
	"github.com/sirupsen/logrus"
	"math"
)

func Exists(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(itemId uint32) bool {
	return func(itemId uint32) bool {
		inventoryType := byte(math.Floor(float64(itemId) / 1000000))
		if inventoryType == 1 {
			_, err := statistics.GetById(l, ctx, tenant)(itemId)
			if err != nil {
				return false
			}
			return true
		}

		return true
	}
}
