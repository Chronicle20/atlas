package map_command

import (
	mapKafka "atlas-saga-orchestrator/kafka/message/map"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func WeatherStartCommandProvider(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.WeatherStartCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.WeatherStartCommandBody{
			ItemId:     itemId,
			Message:    message,
			DurationMs: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
