package ban

import (
	ban2 "atlas-ban/kafka/message/ban"
	"math/rand"

	kafkago "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(banId uint32) model.Provider[[]kafka.Message] {
	key := kafkago.CreateKey(rand.Int())
	value := &ban2.StatusEvent{
		BanId:  banId,
		Status: ban2.EventStatusCreated,
	}
	return kafkago.SingleMessageProvider(key, value)
}

func deletedEventProvider(banId uint32) model.Provider[[]kafka.Message] {
	key := kafkago.CreateKey(rand.Int())
	value := &ban2.StatusEvent{
		BanId:  banId,
		Status: ban2.EventStatusDeleted,
	}
	return kafkago.SingleMessageProvider(key, value)
}

func expiredEventProvider(banId uint32) model.Provider[[]kafka.Message] {
	key := kafkago.CreateKey(rand.Int())
	value := &ban2.StatusEvent{
		BanId:  banId,
		Status: ban2.EventStatusExpired,
	}
	return kafkago.SingleMessageProvider(key, value)
}
