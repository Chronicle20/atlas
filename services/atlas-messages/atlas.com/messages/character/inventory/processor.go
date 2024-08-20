package inventory

import (
	"atlas-messages/equipment/statistics"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"math"
)

func Exists(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(itemId uint32) bool {
	return func(itemId uint32) bool {
		inventoryType := byte(math.Floor(float64(itemId) / 1000000))
		if inventoryType == 1 {
			_, err := statistics.GetById(l, span, tenant)(itemId)
			if err != nil {
				return false
			}
			return true
		}

		return true
	}
}
