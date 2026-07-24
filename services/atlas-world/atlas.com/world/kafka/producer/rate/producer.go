package rate

import (
	rate2 "atlas-world/kafka/message/rate"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func WorldRateChangedEventProvider(tenant tenant.Model, worldId world.Id, rateType rate2.RateType, multiplier float64) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id().String())
	value := &rate2.WorldRateEvent{
		Type:       rate2.TypeRateChanged,
		WorldId:    worldId,
		RateType:   rateType,
		Multiplier: multiplier,
	}
	return producer.SingleMessageProvider(key, value)
}
