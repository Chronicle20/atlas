package weather

import (
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func WeatherStartEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.WeatherStart]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeWeatherStart,
		Body: mapKafka.WeatherStart{
			ItemId:  itemId,
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func WeatherEndEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.WeatherEnd]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeWeatherEnd,
		Body: mapKafka.WeatherEnd{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
